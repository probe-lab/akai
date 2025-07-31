package core

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type SamplerFunction func(
	context.Context,
	DHTHost,
	*models.SamplingItem,
	int,
	time.Duration) (models.GeneralVisit, error)

var minIterTime = 250 * time.Millisecond

var DefaultDataSamplerConfig = &config.AkaiDataSamplerConfig{
	Network:         config.DefaultNetwork.String(),
	Workers:         1000,
	SamplingTimeout: 10 * time.Second,
	DBsyncInterval:  10 * time.Minute,
	Meter:           otel.GetMeterProvider().Meter("akai_data_sampler"),
}

type DataSampler struct {
	cfg     *config.AkaiDataSamplerConfig
	network config.Network

	h               DHTHost
	networkScrapper NetworkScrapper
	newItemC        chan []*models.SamplingItem
	db              db.Database
	itemSet         *dasItemSet

	samplingTaskC chan SamplingTask

	// metrics
	currentSamplesCount     metric.Int64ObservableGauge
	loopVisitCount          metric.Int64Gauge
	loopTimeDurationSeconds metric.Int64Gauge
	delaySecsToNextVisit    metric.Int64Gauge
}

func NewDataSampler(
	cfg *config.AkaiDataSamplerConfig,
	database db.Database,
	netScrapper NetworkScrapper,
	h DHTHost,
) (*DataSampler, error) {

	newItemC := netScrapper.GetSamplingItemStream()
	if newItemC == nil {
		return nil, fmt.Errorf("unable to get new-sampling-items chan from network-scrapper")
	}

	ds := &DataSampler{
		cfg:             cfg,
		network:         config.NetworkFromStr(cfg.Network),
		db:              database,
		networkScrapper: netScrapper,
		newItemC:        newItemC,
		h:               h,
		itemSet:         newDASItemSet(),
		samplingTaskC:   make(chan SamplingTask, cfg.Workers),
	}

	return ds, nil
}

func (ds *DataSampler) Serve(ctx context.Context) error {
	err := ds.initMetrics()
	if err != nil {
		return err
	}

	go ds.consumeNewSamplingItems(ctx)

	// start the orchester
	go ds.runSampleOrchester(ctx)

	// start the workers
	var samplerWG sync.WaitGroup
	for samplerID := int64(1); samplerID <= ds.cfg.Workers; samplerID++ {
		samplerWG.Add(1)
		go ds.runSampler(ctx, samplerID)
	}

	<-ctx.Done()
	return nil
}

func (ds *DataSampler) consumeNewSamplingItems(ctx context.Context) {
	wlog := log.WithField("data-sampler", "newItemConsumer")
	wlog.Debug("spawning...")
	defer func() {
		wlog.Debug("closed")
	}()

	for {
		select {
		case newItem := <-ds.newItemC:
			wlog.Debugf("new %d items tracked", len(newItem))
			for _, item := range newItem {
				wlog.Debugf("new item %s", item.Key)
				_ = ds.itemSet.addItem(item)
				_, _ = ds.updateNextVisitTime(item)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (ds *DataSampler) runSampleOrchester(ctx context.Context) {
	olog := log.WithField("orcherster", ds.cfg.Network)
	olog.Debug("spawning service")
	defer func() {
		olog.Info("sample orchester closed")
	}()

	// ensure that the ItemSet is not freshly created
	minTimeT := time.NewTicker(minIterTime)
initLoop:
	for !ds.itemSet.isInit() {
		select {
		case <-minTimeT.C:
			olog.Trace("Item_set still not initialized")
			minTimeT.Reset(minIterTime)
		case <-ctx.Done():
			break initLoop
		}
	}
	minTimeT.Reset(minIterTime)

	// metrics
	var currentLoopVisitCounter int64 = 0
	loopStartTime := time.Now()

	// check if we need to update the Item set
	updateSegSet := func() {
		olog.Debugf("reorgananizing (%d) Items based on their next visit time", ds.itemSet.Len())
		ds.itemSet.SortItemList()
		currentLoopVisitCounter = 0
		loopStartTime = time.Now()
	}

	updateSamplerMetrics := func() {
		ds.loopVisitCount.Record(ctx, currentLoopVisitCounter)
		ds.loopTimeDurationSeconds.Record(ctx, int64(time.Since(loopStartTime).Seconds()))

		nextT, sortedNeeded := ds.itemSet.NextVisitTime()
		if sortedNeeded {
			if nextT.IsZero() {
				ds.delaySecsToNextVisit.Record(ctx, int64(0))
			} else {
				ds.delaySecsToNextVisit.Record(ctx, int64(time.Until(nextT).Seconds()))
			}
		} else {
			ds.delaySecsToNextVisit.Record(ctx, int64(time.Until(nextT).Seconds()))
		}
	}

	// orchester main loop
	for {
		select {
		case <-ctx.Done():
			return

		default:
			<-minTimeT.C // ensure minimal interval between resets to not spam the DB nor wasting CPU cicles
			minTimeT.Reset(minIterTime)

			nextItem := ds.itemSet.Next()
			if nextItem == nil {
				log.Debug("there is no next Item to visit")
				updateSegSet()
				updateSamplerMetrics()
				continue
			}

			// check if we have to sort again
			if !nextItem.IsReadyForNextPing() {
				if nextItem.NextVisit.IsZero() {
					// as we organize the Items by nextVisitTime, "zero" time gets first
					// breaking always the loop until all the Cids have been generated
					// and published, thus, let the foor loop find a valid time
					olog.Debugf("not in time to visit %s next visit in zero (%s)", nextItem.Key, time.Until(nextItem.NextVisit))
				}
				olog.Debugf("not in time to visit %s next visit in %s", nextItem.Key, time.Until(nextItem.NextVisit))
				// we have to update anyways
				updateSegSet()
				updateSamplerMetrics()
				continue
			}

			// update the Item for the next visit
			visitRound, hasValidNextVisit := ds.updateNextVisitTime(nextItem)
			if !hasValidNextVisit {
				ds.itemSet.removeItem(nextItem.Key)
			}
			ds.samplingTaskC <- SamplingTask{
				visitRound: visitRound,
				item:       nextItem,
			}

			// metrics
			currentLoopVisitCounter++
			updateSamplerMetrics()

			_, sortNeeded := ds.itemSet.NextVisitTime()
			if sortNeeded {
				updateSegSet()
			}
		}
	}
}

func (ds *DataSampler) runSampler(ctx context.Context, samplerID int64) {
	wlog := log.WithField("sampler-id", samplerID)
	wlog.Debug("spawning new sampler")
	defer func() {
		wlog.Debug("sampler closed")
	}()

	for {
		select {
		case task := <-ds.samplingTaskC:
			wlog.Debugf("sampling %s", task.item.Key)
			err := ds.sampleItem(ctx, task.visitRound, task.item, ds.cfg.SamplingTimeout)
			if err != nil {
				log.Warnf("error persisting sampling visit - %s - key: %s", err, task.item.Key)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (ds *DataSampler) sampleItem(
	ctx context.Context,
	visitRound int,
	item *models.SamplingItem,
	timeout time.Duration,
) error {
	log.WithFields(log.Fields{
		"Item":             item.Key,
		"sampleable_until": item.SampleUntil,
	}).Debug("sampling Item of blob")

	samplerFn, err := samplingFnFromType(config.SamplingTypeFromStr(item.SampleType))
	if err != nil {
		return err
	}

	resVisit, err := samplerFn(ctx, ds.h, item, visitRound, timeout)
	if err != nil {
		return err
	}
	// Iter through all the items in the visits
	// - Generic Visists
	if genericVisits := resVisit.GetGenericVisit(); genericVisits != nil {
		for _, visit := range genericVisits {
			err = ds.db.PersistNewSampleGenericVisit(ctx, visit)
			if err != nil {
				return err
			}
		}
	}
	// - Value Visists
	if valueVisits := resVisit.GetGenericValueVisit(); valueVisits != nil {
		for _, visit := range valueVisits {
			err = ds.db.PersistNewSampleValueVisit(ctx, visit)
			if err != nil {
				return err
			}
		}
	}

	if peerInfoVisits := resVisit.GetGenericPeerInfoVisit(); peerInfoVisits != nil {
		for _, visit := range peerInfoVisits {
			err = ds.db.PersistNewPeerInfoVisit(ctx, visit)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (ds *DataSampler) updateNextVisitTime(item *models.SamplingItem) (visitRound int, validNextVisit bool) {
	// TODO: Consider having all these network parameters in as single struct per network
	switch ds.network.Protocol {
	case config.ProtocolIPFS, config.ProtocolCelestia:
		// constant increment of 15/30 mins
		visitRound, nextVisitTime := computeNextVisitTime(
			item.Timestamp,
			item.NextVisit,
			ds.cfg.DelayBase,
			ds.cfg.DelayMultiplier,
			addConstantDelay,
		)

		item.NextVisit = nextVisitTime
		return visitRound, nextVisitTime.Before(item.SampleUntil)

	case config.ProtocolAvail, config.ProtocolLocal:
		// exponentially increasing delay until end date
		visitRound, nextVisitTime := computeNextVisitTime(
			item.Timestamp,
			item.NextVisit,
			ds.cfg.DelayBase,
			ds.cfg.DelayMultiplier,
			addExponentialDelay,
		)
		item.NextVisit = nextVisitTime
		return visitRound, nextVisitTime.Before(item.SampleUntil)

	default:
		log.Warn("unable to apply delay for next visit, as the network is not supported")
		return 0, false
	}
}

func computeNextVisitTime(
	itemT, itemNextV time.Time,
	delayBase time.Duration,
	delayMultiplier int,
	delayApplier func(*time.Duration, int, time.Duration),
) (visitRound int, nextVisitTime time.Time) {

	multCnt := 1
	delay := delayBase
	nextVisit := itemT.Add(delay)
	multCnt++
	for (nextVisit.Before(itemNextV) && nextVisit.Equal(itemNextV)) || nextVisit.Before(time.Now()) {
		delayApplier(&delay, delayMultiplier, delayBase)
		nextVisit = itemT.Add(delay)
		multCnt = multCnt + 1
	}
	/*
		fmt.Println("next visit time:")
		fmt.Println("timestamp:", itemT)
		fmt.Println("next_visit:", itemNextV)
		fmt.Println("delay_base:", delayBase)
		fmt.Println("delayMultiplier:", delayMultiplier)
		fmt.Println("visit_round:", multCnt)
		fmt.Println("next_visit:", nextVisit)
	*/
	return multCnt, nextVisit
}

func addExponentialDelay(delay *time.Duration, delayMultiplier int, _ time.Duration) {
	*delay = *delay * time.Duration(delayMultiplier)
}

func addConstantDelay(delay *time.Duration, _ int, delayBase time.Duration) {
	*delay = *delay + delayBase
}

// initMetrics initializes various prometheus metrics and stores the meters
// on the [BlockTracker] object.
func (ds *DataSampler) initMetrics() (err error) {
	ds.currentSamplesCount, err = ds.cfg.Meter.Int64ObservableGauge("current_samples")
	if err != nil {
		return fmt.Errorf("new current_samples gauge: %w", err)
	}

	_, err = ds.cfg.Meter.RegisterCallback(func(ctx context.Context, obs metric.Observer) error {
		obs.ObserveInt64(ds.currentSamplesCount, int64(ds.itemSet.Len()))
		return nil
	}, ds.currentSamplesCount)
	if err != nil {
		return fmt.Errorf("register total_block counter callback: %w", err)
	}

	ds.loopVisitCount, err = ds.cfg.Meter.Int64Gauge("orchester_loop_visit_count")
	if err != nil {
		return fmt.Errorf("new orchester_loop_visit_count counter: %w", err)
	}

	ds.loopTimeDurationSeconds, err = ds.cfg.Meter.Int64Gauge("orchester_loop_time_duration_s")
	if err != nil {
		return fmt.Errorf("new orchester_loop_time_duration counter: %w", err)
	}

	ds.delaySecsToNextVisit, err = ds.cfg.Meter.Int64Gauge("orcherster_secs_to_next_visit")
	if err != nil {
		return fmt.Errorf("new orcherster_secs_to_next_visit counter: %w", err)
	}

	return nil
}

// TODO: in the future, this could be extended to make a "smarter" division of the tasks:
// - key-space filltering
// - host selection
// - or similars
type SamplingTask struct {
	visitRound int
	item       *models.SamplingItem
}

// logical sampler functions
func sampleByFindProviders(
	ctx context.Context,
	h DHTHost,
	item *models.SamplingItem,
	visitRound int,
	timeout time.Duration,
) (models.GeneralVisit, error) {
	// generate the base for the generic visit
	visit := models.SampleGenericVisit{
		VisitType:     config.SampleProviders.String(),
		VisitRound:    uint64(visitRound),
		Network:       item.Network,
		Timestamp:     time.Now(),
		Key:           item.Key,
		DurationMs:    int64(0),
		ResponseItems: 0,
		Peers:         make([]string, 0),
		Error:         "",
	}

	contentID, err := cid.Decode(item.Key)
	if err != nil {
		visit.Error = "Item key doesn't represent a valid CID"
	}

	// make the sampling
	duration, providers, err := h.FindProviders(ctx, contentID, timeout)
	if err != nil {
		visit.Error = err.Error()
	}
	// apply rest of values
	visit.DurationMs = duration.Milliseconds()
	visit.ResponseItems = int32(len(providers))
	for _, pr := range providers {
		visit.Peers = append(visit.Peers, pr.ID.String())
	}

	log.WithFields(log.Fields{
		"timestamp":   time.Now(),
		"key":         item.Key,
		"duration":    duration,
		"n_providers": len(providers),
		"providers":   len(visit.Peers),
		"error":       visit.Error,
	}).Debug("find providers operation done")

	return models.GeneralVisit{
		GenericVisit: []*models.SampleGenericVisit{
			&visit,
		},
		GenericValueVisit: nil,
	}, nil
}

func sampleByFindPeers(
	ctx context.Context,
	h DHTHost,
	item *models.SamplingItem,
	visitRound int,
	timeout time.Duration,
) (models.GeneralVisit, error) {
	visit := models.SampleGenericVisit{
		VisitType:     config.SamplePeers.String(),
		VisitRound:    uint64(visitRound),
		Network:       item.Network,
		Timestamp:     time.Now(),
		Key:           item.Key,
		DurationMs:    int64(0),
		ResponseItems: 0,
		Peers:         make([]string, 0),
		Error:         "",
	}

	// make the sampling
	duration, peers, err := h.FindPeers(ctx, item.Key, timeout)
	if err != nil {
		visit.Error = err.Error()
	}

	// apply rest of values
	visit.DurationMs = duration.Milliseconds()
	visit.ResponseItems = int32(len(peers))
	for _, pr := range peers {
		visit.Peers = append(visit.Peers, pr.ID.String())
	}

	log.WithFields(log.Fields{
		"timestamp":   time.Now(),
		"key":         item.Key,
		"duration":    duration,
		"n_providers": len(peers),
		"peer_ids":    ListPeerIDsFromAddrsInfos(peers),
		"error":       visit.Error,
	}).Debug("find peers operation done")

	return models.GeneralVisit{
		GenericVisit: []*models.SampleGenericVisit{
			&visit,
		},
		GenericValueVisit: nil,
	}, nil
}

func sampleByFindPeerInfo(
	ctx context.Context,
	h DHTHost,
	item *models.SamplingItem,
	visitRound int,
	timeout time.Duration,
) (models.GeneralVisit, error) {
	visit := models.PeerInfoVisit{
		VisitRound:      uint64(visitRound),
		Timestamp:       time.Now(),
		Network:         item.Network,
		PeerID:          item.Key,
		Duration:        0,
		AgentVersion:    "",
		Protocols:       make([]protocol.ID, 0),
		ProtocolVersion: "",
		MultiAddresses:  make([]multiaddr.Multiaddr, 0),
		Error:           "",
	}

	peerID, err := peer.Decode(item.Key)
	if err != nil {
		return models.GeneralVisit{}, err
	}

	log.WithFields(log.Fields{"timeout": timeout})

	duration, peerInfo, err := h.FindPeer(ctx, peerID, timeout)
	visit.Duration = duration
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warnf("find peer info operation could not be completed")
		visit.Error = err.Error()
		return models.GeneralVisit{
			GenericPeerInfoVisit: []*models.PeerInfoVisit{
				&visit,
			},
		}, err
	}

	hostInfo, err := h.ConnectAndIdentifyPeer(ctx, peerInfo, 1, 30*time.Second)
	errStr := ""
	if err != nil {
		visit.Error = err.Error()
	} else {
		visit.Error = errStr
	}

	disconnectionErr := h.Host().Network().ClosePeer(peerID)
	if disconnectionErr != nil {
		return models.GeneralVisit{}, err
	}

	log.WithFields(log.Fields{
		"operation":   config.SamplePeerInfo.String(),
		"timestamp":   duration.Milliseconds(),
		"peer_id":     peerInfo.ID.String(),
		"maddres":     peerInfo.Addrs,
		"peer_info":   hostInfo,
		"duration_ms": duration,
		"error":       errStr,
	}).Infof("find peer info operation done")

	// apply remaining values
	if agentVersion, ok := hostInfo["agent_version"].(string); ok {
		visit.AgentVersion = agentVersion
	}
	if protocolVersion, ok := hostInfo["protocol_version"].(string); ok {
		visit.ProtocolVersion = protocolVersion
	}
	visit.Protocols = append(visit.Protocols, hostInfo["protocols"].([]protocol.ID)...)
	visit.MultiAddresses = append(visit.MultiAddresses, peerInfo.Addrs...)

	return models.GeneralVisit{
		GenericPeerInfoVisit: []*models.PeerInfoVisit{
			&visit,
		},
	}, nil
}

func sampleByFindValue(
	ctx context.Context,
	h DHTHost,
	item *models.SamplingItem,
	visitRound int,
	timeout time.Duration,
) (models.GeneralVisit, error) {
	visit := models.SampleValueVisit{
		VisitRound:    uint64(visitRound),
		VisitType:     item.ItemType,
		Network:       item.Network,
		Timestamp:     time.Now(),
		Key:           item.Key,
		BlockNumber:   item.BlockLink,
		DASRow:        item.DASRow,
		DASColumn:     item.DASColumn,
		DurationMs:    int64(0),
		IsRetrievable: false,
		Bytes:         0,
		Error:         "",
	}

	// make the sampling
	duration, bytes, err := h.FindValue(ctx, item.Key, timeout)
	if err != nil {
		visit.Error = err.Error()
	}
	if len(bytes) > 0 {
		visit.IsRetrievable = true
		visit.Bytes = int32(len(bytes))
		// there is an edgy case where the sampling reports a failure
		// but the content was retrieved -> Err = ContextDeadlineExceeded
		// rewrite the error to nil/Empty
		if err == context.DeadlineExceeded {
			log.WithFields(log.Fields{
				"key":   item.Key,
				"bytes": len(bytes),
			}).Warn("key retrieved, but context was exceeded")
			visit.Error = ""
		}
	}
	// apply rest of values
	visit.DurationMs = duration.Milliseconds()

	log.WithFields(log.Fields{
		"timestamp": time.Now(),
		"key":       item.Key,
		"duration":  duration,
		"bytes":     len(bytes),
		"error":     visit.Error,
	}).Debug("find value operation done")

	return models.GeneralVisit{
		GenericVisit: nil,
		GenericValueVisit: []*models.SampleValueVisit{
			&visit,
		},
	}, nil
}

func samplingFnFromType(sampleType config.SamplingType) (SamplerFunction, error) {
	switch sampleType {
	case config.SampleProviders:
		return sampleByFindProviders, nil

	case config.SampleValue:
		return sampleByFindValue, nil

	case config.SamplePeers:
		return sampleByFindPeers, nil

	case config.SamplePeerInfo:
		return sampleByFindPeerInfo, nil

	default:
		return func(
			_ context.Context,
			_ DHTHost,
			_ *models.SamplingItem,
			_ int,
			_ time.Duration,
		) (models.GeneralVisit, error) {
			return models.GeneralVisit{}, nil
		}, nil
	}
}
