package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

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
				VisitRound: visitRound,
				Item:       nextItem,
				Quorum:     ds.networkScrapper.GetQuorum(),
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
			wlog.Debugf("sampling %s", task.Item.Key)
			err := ds.sampleItem(ctx, task, ds.cfg.SamplingTimeout)
			if err != nil {
				log.Warnf("error persisting sampling visit - %s - key: %s", err, task.Item.Key)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (ds *DataSampler) sampleItem(
	ctx context.Context,
	task SamplingTask,
	timeout time.Duration,
) error {
	log.WithFields(log.Fields{
		"Item":             task.Item.Key,
		"sampleable_until": task.Item.SampleUntil,
	}).Debug("sampling Item of blob")

	samplerFn, err := samplingFnFromType(config.SamplingTypeFromStr(task.Item.SampleType))
	if err != nil {
		return err
	}

	resVisit, err := samplerFn(ctx, ds.h, task, timeout)
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

	// - PeerInfo Visits
	if peerInfoVisits := resVisit.GetGenericPeerInfoVisit(); peerInfoVisits != nil {
		for _, visit := range peerInfoVisits {
			err = ds.db.PersistNewPeerInfoVisit(ctx, visit)
			if err != nil {
				return err
			}
		}
	}

	// - IPNS record Visits
	if ipnsRecordVisits := resVisit.GetGenericIPNSRecordVisit(); ipnsRecordVisits != nil {
		for _, visit := range ipnsRecordVisits {
			err = ds.db.PersistNewIPNSRecordVisit(ctx, visit)
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
	case config.ProtocolIPFS, config.ProtocolIPNS, config.ProtocolCelestia:
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
