package core

import (
	"context"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
)

var DefaultDataSamplerConfig = config.AkaiDataSamplerConfig{
	Network:         config.DefaultNetwork.String(),
	Workers:         10,
	SamplingTimeout: 10 * time.Second,
	DBsyncInterval:  10 * time.Minute,
}

type DataSampler struct {
	cfg config.AkaiDataSamplerConfig

	db db.Database
	h  DHTHost

	blobIDs     *lru.Cache[uint64, *models.AgnosticBlob]
	segmentsIDs *lru.Cache[string, *models.AgnosticSegment]
}

func NewDataSampler(
	cfg config.AkaiDataSamplerConfig,
	database db.Database,
	h DHTHost) (*DataSampler, error) {

	blobsCache, err := lru.New[uint64, *models.AgnosticBlob](cfg.BlobsSetCacheSize) // <block_number>, <block_ID_in_DB>
	if err != nil {
		return nil, err
	}
	segmentsCache, err := lru.New[string, *models.AgnosticSegment](cfg.SegmentsSetCacheSize) // <block_number>, <block_ID_in_DB>
	if err != nil {
		return nil, err
	}

	ds := &DataSampler{
		cfg:         cfg,
		db:          database,
		h:           h,
		blobIDs:     blobsCache,
		segmentsIDs: segmentsCache,
	}

	return ds, nil
}

func (ds *DataSampler) Serve(ctx context.Context) error {
	err := ds.syncWithDatabase(ctx)
	if err != nil {
		return err
	}

	// start the orchester
	go ds.runSampleOrchester(ctx)

	// start the workers
	for samplerID := 1; samplerID <= ds.cfg.Workers; samplerID++ {
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

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

func (ds *DataSampler) runSampler(ctx context.Context, samplerID int) {
	wlog := log.WithField("sampler-id", samplerID)
	wlog.Debug("spawning new sampler")
	defer func() {
		wlog.Info("sampler closed")
	}()

	for {
		select {
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
		}
		log.WithFields(log.Fields{
			"from": first,
			"to":   last,
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
		for _, blob := range sampleableBlobs {
			ds.blobIDs.Add(blob.BlobNumber, &blob)
		}
		log.WithFields(log.Fields{
			"from": first,
			"to":   last,
		}).Info("synced data-sampler's sampleable blobs with the DB")
	} else {
		log.Warn("no sampleable blobs were found at DB")
	}

	return nil
}

func (ds *DataSampler) sampleSegment(ctx context.Context, segment models.AgnosticSegment) error {
	log.WithFields(log.Fields{
		"segment":        segment.Key,
		"sampleable_ttl": segment.SampleUntil,
	}).Debug("sampling segment of blob")

	visit := models.AgnosticVisit{
		Timestamp:     time.Now(),
		Key:           segment.Key,
		IsRetrievable: false,
	}

	// make the sampling
	duration, bytes, err := ds.h.FindValue(ctx, segment.Key)
	if err != nil {
		visit.Error = err.Error()
	}
	if len(bytes) > 0 {
		visit.IsRetrievable = true
	}
	visit.DurationMs = duration.Milliseconds()

	// update the DB with the result of the visit
	return ds.db.PersistNewSegmentVisit(ctx, visit)
}

func (ds *DataSampler) BlobsCache() *lru.Cache[uint64, *models.AgnosticBlob] {
	return ds.BlobsCache()
}

func (ds *DataSampler) SegmentsCache() *lru.Cache[string, *models.AgnosticSegment] {
	return ds.SegmentsCache()
}
