package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/probe-lab/akai/amino"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type SamplerFunction func(context.Context, DHTHost, models.AgnosticSegment) (models.AgnosticVisit, error)

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
	network models.Network

	db        db.Database
	h         DHTHost
	samplerFn SamplerFunction

	blobIDs    *lru.Cache[uint64, *models.AgnosticBlob]
	segSet     *segmentSet
	visitTaskC chan *models.AgnosticSegment

	// metrics
	currentSamplesCount     metric.Int64ObservableGauge
	loopVisitCount          metric.Int64Gauge
	loopTimeDurationSeconds metric.Int64Gauge
	delaySecsToNextVisit    metric.Int64Gauge
}

func NewDataSampler(
	cfg *config.AkaiDataSamplerConfig,
	database db.Database,
	h DHTHost,
	samplerFn SamplerFunction,
) (*DataSampler, error) {
	blobsCache, err := lru.New[uint64, *models.AgnosticBlob](cfg.BlobsSetCacheSize) // <block_number>, <block_ID_in_DB>
	if err != nil {
		return nil, err
	}

	ds := &DataSampler{
		cfg:        cfg,
		network:    models.NetworkFromStr(cfg.Network),
		db:         database,
		h:          h,
		samplerFn:  samplerFn,
		segSet:     newSegmentSet(),
		visitTaskC: make(chan *models.AgnosticSegment, cfg.Workers),
		blobIDs:    blobsCache,
	}

	return ds, nil
}

func (ds *DataSampler) Serve(ctx context.Context) error {
	err := ds.initMetrics()
	if err != nil {
		return err
	}

	err = ds.syncWithDatabase(ctx)
	if err != nil {
		return err
	}

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

func (ds *DataSampler) runSampleOrchester(ctx context.Context) {
	olog := log.WithField("orcherster", ds.cfg.Network)
	olog.Debug("spawning service")
	defer func() {
		olog.Info("sample orchester closed")
	}()

	// ensure that the segmentSet is not freshly created
	minTimeT := time.NewTicker(minIterTime)
initLoop:
	for !ds.segSet.isInit() {
		select {
		case <-minTimeT.C:
			olog.Trace("segment_set still not initialized")
			minTimeT.Reset(minIterTime)
		case <-ctx.Done():
			break initLoop
		}
	}
	minTimeT.Reset(minIterTime)

	// metrics
	var currentLoopVisitCounter int64 = 0
	loopStartTime := time.Now()

	// check if we need to update the segment set
	updateSegSet := func() {
		olog.Debugf("reorgananizing (%d) Segments based on their next visit time", ds.segSet.Len())
		ds.segSet.SortSegmentList()
		currentLoopVisitCounter = 0
		loopStartTime = time.Now()
	}

	updateSamplerMetrics := func() {
		ds.loopVisitCount.Record(ctx, currentLoopVisitCounter)
		ds.loopTimeDurationSeconds.Record(ctx, int64(time.Since(loopStartTime).Seconds()))

		nextT, sortedNeeded := ds.segSet.NextVisitTime()
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

			nextSegment := ds.segSet.Next()
			if nextSegment == nil {
				log.Debug("there is no next segment to visit")
				updateSegSet()
				updateSamplerMetrics()
				continue
			}

			// check if we have to sort again
			if !nextSegment.IsReadyForNextPing() {
				if nextSegment.NextVisit.IsZero() {
					// as we organize the segments by nextVisitTime, "zero" time gets first
					// breaking always the loop until all the Cids have been generated
					// and published, thus, let the foor loop find a valid time
					olog.Debugf("not in time to visit %s next visit in zero (%s)", nextSegment.Key, time.Until(nextSegment.NextVisit))
				}
				olog.Debugf("not in time to visit %s next visit in %s", nextSegment.Key, time.Until(nextSegment.NextVisit))
				// we have to update anyways
				updateSegSet()
				updateSamplerMetrics()
				continue
			}

			// update the segment for the next visit
			hasValidNextVisit := ds.updateNextVisitTime(nextSegment)
			if !hasValidNextVisit {
				ds.segSet.removeSegment(nextSegment.Key)
			}
			ds.visitTaskC <- nextSegment

			// metrics
			currentLoopVisitCounter++
			updateSamplerMetrics()

			_, sortNeeded := ds.segSet.NextVisitTime()
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
		case samplingSegment := <-ds.visitTaskC:
			wlog.Debugf("sampling %s", samplingSegment.Key)
			samplingCtx, cancel := context.WithTimeout(ctx, ds.cfg.SamplingTimeout)
			defer cancel()
			err := ds.sampleSegment(samplingCtx, *samplingSegment)
			if err != nil {
				log.Panicf("error persinting sampling visit - %s", err)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (ds *DataSampler) syncWithDatabase(ctx context.Context) error {
	// sample the blobs
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	sampleableBlobs, err := ds.db.GetSampleableBlobs(ctx)
	if err != nil {
		return err
	}
	if len(sampleableBlobs) > 0 {
		first := sampleableBlobs[0].BlobNumber
		last := sampleableBlobs[0].BlobNumber
		for _, blob := range sampleableBlobs {
			ds.blobIDs.Add(blob.BlobNumber, &blob)
			last = blob.BlobNumber
		}
		log.WithFields(log.Fields{
			"from":  first,
			"to":    last,
			"total": len(sampleableBlobs),
		}).Info("synced data-sampler's sampleable blobs with the DB")
	} else {
		log.Warn("no sampleable blobs were found at DB")
	}

	// sample the segments
	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	sampleableSegs, err := ds.db.GetSampleableSegments(ctx)
	if err != nil {
		return err
	}
	if len(sampleableSegs) > 0 {
		first := sampleableSegs[0].Key
		last := sampleableSegs[0].Key
		for _, seg := range sampleableSegs {
			last = seg.Key
			if ds.segSet.isSegmentAlready(seg.Key) {
				continue
			}
			ds.updateNextVisitTime(&seg)
			ds.segSet.addSegment(&seg)
		}
		log.WithFields(log.Fields{
			"from":  first,
			"to":    last,
			"total": len(sampleableSegs),
		}).Info("synced data-sampler's sampleable segments with the DB")
	} else {
		log.Warn("no sampleable segments were found at DB")
	}

	return nil
}

func (ds *DataSampler) sampleSegment(ctx context.Context, segment models.AgnosticSegment) error {
	log.WithFields(log.Fields{
		"segment":          segment.Key,
		"sampleable_until": segment.SampleUntil,
	}).Debug("sampling segment of blob")

	visit, err := ds.samplerFn(ctx, ds.h, segment)
	if err != nil {
		return err
	}

	// update the DB with the result of the visit
	return ds.db.PersistNewSegmentVisit(ctx, visit)
}

func (ds *DataSampler) BlobsCache() *lru.Cache[uint64, *models.AgnosticBlob] {
	return ds.blobIDs
}

func (ds *DataSampler) updateNextVisitTime(segment *models.AgnosticSegment) (validNextVisit bool) {
	// TODO: COnsider having all these network parameters in as single struct per network
	switch ds.network.Protocol {
	case config.ProtocolIPFS, config.ProtocolCelestia:
		// constant increment of 15/30 mins
		multCnt := 1
		delay := ds.cfg.DelayBase
		nextVisit := segment.Timestamp.Add(delay)
		multCnt++
		for (nextVisit.Before(segment.NextVisit) && nextVisit.Equal(segment.NextVisit)) || nextVisit.Before(time.Now()) {
			delay = delay + ds.cfg.DelayBase
			nextVisit = segment.Timestamp.Add(delay)
			multCnt++
		}

		segment.VisitRound = uint64(multCnt)
		segment.NextVisit = nextVisit
		return nextVisit.Before(segment.SampleUntil)

	case config.ProtocolAvail, config.NetworkNameCustom:
		// exponentially increasing delay until end date
		multCnt := 1
		delay := ds.cfg.DelayBase
		nextVisit := segment.Timestamp.Add(delay)
		multCnt++
		for (nextVisit.Before(segment.NextVisit) && nextVisit.Equal(segment.NextVisit)) || nextVisit.Before(time.Now()) {
			delay = delay * time.Duration(ds.cfg.DelayMultiplier)
			nextVisit = segment.Timestamp.Add(delay)
			multCnt++
		}

		segment.VisitRound = uint64(multCnt)
		segment.NextVisit = nextVisit
		return nextVisit.Before(segment.SampleUntil)

	default:
		log.Panic("unable to apply delay for next visit, as the network is not supported")
		return false
	}
}

// initMetrics initializes various prometheus metrics and stores the meters
// on the [BlockTracker] object.
func (ds *DataSampler) initMetrics() (err error) {
	ds.currentSamplesCount, err = ds.cfg.Meter.Int64ObservableGauge("current_samples")
	if err != nil {
		return fmt.Errorf("new current_samples gauge: %w", err)
	}

	_, err = ds.cfg.Meter.RegisterCallback(func(ctx context.Context, obs metric.Observer) error {
		obs.ObserveInt64(ds.currentSamplesCount, int64(len(ds.segSet.segmentArray)))
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

// logical sampler functions
func sampleByFindProviders(ctx context.Context, h DHTHost, segmnt models.AgnosticSegment) (models.AgnosticVisit, error) {
	visit := models.AgnosticVisit{
		VisitRound:    segmnt.VisitRound,
		Timestamp:     time.Now(),
		Key:           segmnt.Key,
		BlobNumber:    segmnt.BlobNumber,
		Row:           segmnt.Row,
		Column:        segmnt.Column,
		IsRetrievable: false,
		Providers:     0,
		Bytes:         0,
	}

	cid, err := amino.CidFromString(segmnt.Key)
	if err != nil {
		visit.Error = "segment key doesn't represent a valid CID"
	}

	// make the sampling
	duration, providers, err := h.FindProviders(ctx, cid.Cid())
	if err != nil {
		visit.Error = err.Error()
	}
	if len(providers) > 0 {
		visit.IsRetrievable = true
	}
	// apply rest of values
	visit.DurationMs = duration.Milliseconds()
	visit.Bytes = 0
	visit.Providers = uint32(len(providers))

	log.WithFields(log.Fields{
		"timestamp":   time.Now(),
		"key":         segmnt.Key,
		"duration":    duration,
		"n_providers": len(providers),
		"error":       visit.Error,
	}).Debug("find providers operation done")

	return visit, nil
}

func sampleByFindPeers(ctx context.Context, h DHTHost, segmnt models.AgnosticSegment) (models.AgnosticVisit, error) {
	visit := models.AgnosticVisit{
		VisitRound:    segmnt.VisitRound,
		Timestamp:     time.Now(),
		Key:           segmnt.Key,
		BlobNumber:    segmnt.BlobNumber,
		Row:           segmnt.Row,
		Column:        segmnt.Column,
		IsRetrievable: false,
		Providers:     0,
		Bytes:         0,
	}

	// make the sampling
	duration, peers, err := h.FindPeers(ctx, segmnt.Key, 15*time.Second)
	if err != nil {
		visit.Error = err.Error()
	}
	if len(peers) > 0 {
		visit.IsRetrievable = true
	}

	// apply rest of values
	visit.DurationMs = duration.Milliseconds()
	visit.Bytes = 0
	visit.Providers = uint32(len(peers))

	log.WithFields(log.Fields{
		"timestamp":   time.Now(),
		"key":         segmnt.Key,
		"duration":    duration,
		"n_providers": len(peers),
		"peer_ids":    ListPeerIDsFromAddrsInfos(peers),
		"error":       visit.Error,
	}).Debug("find peers operation done")

	return visit, nil
}

func sampleByFindValue(ctx context.Context, h DHTHost, segmnt models.AgnosticSegment) (models.AgnosticVisit, error) {
	visit := models.AgnosticVisit{
		VisitRound:    segmnt.VisitRound,
		Timestamp:     time.Now(),
		Key:           segmnt.Key,
		BlobNumber:    segmnt.BlobNumber,
		Row:           segmnt.Row,
		Column:        segmnt.Column,
		IsRetrievable: false,
	}

	// make the sampling
	duration, bytes, err := h.FindValue(ctx, segmnt.Key)
	if err != nil {
		visit.Error = err.Error()
	}
	if len(bytes) > 0 {
		visit.Providers = 1
		visit.IsRetrievable = true
		visit.Bytes = uint32(len(bytes))
		// there is an edgy case where the sampling reports a failure
		// but the content was retrieved -> Err = ContextDeadlineExceeded
		// rewrite the error to nil/Empty
		if err == context.DeadlineExceeded {
			log.WithFields(log.Fields{
				"key": segmnt.Key,
				"bytes": len(bytes),
			}).Warn("key retrieved, but context was exceeded")
			visit.Error = ""
		}
	}
	// apply rest of values
	visit.DurationMs = duration.Milliseconds()

	log.WithFields(log.Fields{
		"timestamp": time.Now(),
		"key":       segmnt.Key,
		"duration":  duration,
		"bytes":     len(bytes),
		"error":     visit.Error,
	}).Debug("find value operation done")

	return visit, nil
}

func GetSamplingFnFromType(sampleType config.SamplingType) (SamplerFunction, error) {
	switch sampleType {
	case config.SampleProviders:
		return sampleByFindProviders, nil

	case config.SampleValue:
		return sampleByFindValue, nil

	case config.SamplePeers:
		return sampleByFindPeers, nil

	default:
		return func(_ context.Context, _ DHTHost, _ models.AgnosticSegment) (models.AgnosticVisit, error) {
			return models.AgnosticVisit{}, nil
		}, nil
	}
}
