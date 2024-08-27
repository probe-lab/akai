package core

import (
	"context"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	"github.com/probe-lab/akai/api"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db"
	"github.com/probe-lab/akai/db/models"

	log "github.com/sirupsen/logrus"
	"github.com/thejerf/suture/v4"
)

var DefaulDaemonConfig = config.AkaiDaemonConfig{}

type Daemon struct {
	config config.AkaiDaemonConfig
	sup    *suture.Supervisor // to coordinate all the services

	api         *api.Service
	db          db.Database
	dataSampler *DataSampler

	network     models.Network
	blobIDs     *lru.Cache[uint64, *models.AgnosticBlob]
	segmentsIDs *lru.Cache[uint64, *models.AgnosticSegment]
}

func NewDaemon(
	cfg config.AkaiDaemonConfig,
	dbServ db.Database,
	dataSampler *DataSampler,
	apiServ *api.Service) (*Daemon, error) {

	segmentsCache, err := lru.New[uint64, *models.AgnosticSegment](cfg.SegmentsSetCacheSize) // <block_number>, <block_ID_in_DB>
	if err != nil {
		return nil, err
	}
	blobsCache, err := lru.New[uint64, *models.AgnosticBlob](cfg.BlobsSetCacheSize) // <block_number>, <block_ID_in_DB>
	if err != nil {
		return nil, err
	}

	daemon := &Daemon{
		config:      cfg,
		api:         apiServ,
		db:          dbServ,
		sup:         suture.NewSimple("akai-daemon"),
		network:     models.NetworkFromStr(cfg.Network),
		blobIDs:     blobsCache,
		segmentsIDs: segmentsCache,
	}

	// once everything is in place, we MUST override the appHandlers at the ApiServer
	apiServ.UpdateNewBlobHandler(daemon.newBlobHandler)
	apiServ.UpdateNewSegmentHandler(daemon.newSegmentHandler)
	apiServ.UpdateNewSegmentsHandler(daemon.newSegmentsHandler)

	return daemon, nil
}

func (d *Daemon) Start(ctx context.Context) error {
	// prepare services
	err := d.db.Init(ctx, d.db.GetAllTables())
	if err != nil {
		return errors.Wrap(err, "unable to start akai-daemon")
	}

	err = d.configureInternalCaches()
	if err != nil {
		return err
	}

	// add services to
	d.sup.Add(d.db)
	d.sup.Add(d.dataSampler)
	d.sup.Add(d.api)

	return d.sup.Serve(ctx)
}

func (d *Daemon) configureInternalCaches() error {
	// get the networkID for the current network
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	networks, err := d.db.GetNetworks(ctx)
	if err != nil {
		return errors.Wrap(err, "getting recorded networks")
	}
	present := false
	for _, network := range networks {
		if network.String() == d.config.Network {
			d.network = network
			present = true
		}
	}
	if !present {
		log.WithFields(log.Fields{
			"network": networks,
		}).Info("adding networks to DB")
		d.network.NetworkID = uint16(len(networks))
		ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = d.db.PersistNewNetwork(ctx2, d.network)
		if err != nil {
			return err
		}
	}

	// populate the cache of IDs for the ongoing blobs
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lastBlob, err := d.db.GetLastestBlob(ctx)
	if err != nil {
		return err
	}
	if lastBlob.IsComplete() {
		log.Info("synced with the DB at blob", lastBlob.BlockNumber)
		d.blobIDs.Add(lastBlob.BlockNumber, &lastBlob)
	} else {
		log.Warn("no blob was found at DB")
	}

	// populate the cache of IDs for the ongoing segments
	return nil
}

func (d *Daemon) Init(ctx context.Context) error {
	return nil
}

func (d *Daemon) newBlobHandler(ctx context.Context, blob api.Blob) error {
	log.WithFields(log.Fields{
		"block": blob.Number,
		"hash":  blob.Hash,
	}).Info("new blob arrived to the daemon")

	// check blob in cache, add it if it's new
	_, ok := d.blobIDs.Get(blob.Number)
	if ok {
		// we already processed the blob
		return nil
	}
	agBlob := d.newAgnosticBlobFromAPIblob(blob)
	d.blobIDs.Add(agBlob.BlockNumber, &agBlob)

	// add the blob info to the DB
	err := d.db.PersistNewBlob(ctx, agBlob)
	if err != nil {
		return err
	}

	// proccess all the segments through the specific handler

	return nil
}

func (d *Daemon) newSegmentHandler(ctx context.Context, segment api.BlobSegment) error {
	log.WithFields(log.Fields{
		"block": segment.BlockNumber,
		"key":   segment.Key,
	}).Info("new segment arrived to the daemon")
	// add the segment info to the DB
	err := d.db.PersistNewSegment(ctx, d.newAgnosticSegmentFromAPIsegment(segment))
	if err != nil {
		return err
	}
	// add segment to the DataSampler

	return nil
}

func (d *Daemon) newSegmentsHandler(ctx context.Context, segments []api.BlobSegment) error {
	log.WithFields(log.Fields{
		"number": len(segments),
	}).Info("new segments arrived to the daemon")
	// add the segment info to the DB
	err := d.db.PersistNewSegments(ctx, d.newAgnosticSegmentsFromAPIsegments(segments))
	if err != nil {
		return err
	}

	// add segment to the DataSampler

	return nil
}

func (d *Daemon) newAgnosticBlobFromAPIblob(blob api.Blob) models.AgnosticBlob {
	return models.AgnosticBlob{
		NetworkID:   d.network.NetworkID,
		Timestamp:   blob.Timestamp,
		Hash:        blob.Hash,
		BlockNumber: blob.Number,
		Rows:        uint32(blob.Rows),
		Columns:     uint32(blob.Columns),
		SampleUntil: blob.SampleUntil,
	}
}

func (d *Daemon) newAgnosticSegmentFromAPIsegment(segment api.BlobSegment) models.AgnosticSegment {
	return models.AgnosticSegment{
		Timestamp:   segment.Timestamp,
		Key:         segment.Key,
		BlockNumber: segment.BlockNumber,
		Row:         uint32(segment.Row),
		Column:      uint32(segment.Column),
		SampleUntil: segment.SampleUntil,
	}
}

func (d *Daemon) newAgnosticSegmentsFromAPIsegments(segments []api.BlobSegment) []models.AgnosticSegment {
	agSegs := make([]models.AgnosticSegment, len(segments), 0)
	for _, seg := range segments {
		agSeg := d.newAgnosticSegmentFromAPIsegment(seg)
		agSegs = append(agSegs, agSeg)
	}
	return agSegs
}
