package avail

import (
	"context"
	"fmt"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

var DefaultNetorkScrapperConfig = &config.AvailNetworkScrapperConfig{
	Network:              config.DefaultAvailNetwork.String(),
	TrackBlocksOnDB:      true,
	NotChannelBufferSize: 1000,
	SamplerNotifyTimeout: 10 * time.Second,
	TrackDuration:        20 * time.Minute,
	TrackInterval:        6 * time.Hour,
	BlockTrackerCfg:      DefaultBlockTracker,
}

// NetworkScrapper is a minor wrapper on top of the BlockTracker that takes care of:
// - keeping track of all the DAS items comming from the Avail network
// - populate the DB with block information
// - perform some catching to not spam the DataSampler
// - notify to the DataSampler whenever there is an new item to track
type NetworkScrapper struct {
	cfg    config.AvailNetworkScrapperConfig
	closeC chan struct{}

	network      config.Network
	samplerNotCh chan []*models.SamplingItem
	db           db.Database

	// the internal cache to keep track of wheter the segments where seen or not previously
	// will be handled by block-number (easier, simpler, uses less resources)
	internalBlockCache *lru.Cache[uint64, struct{}]

	samplingItemsMut   sync.Mutex
	trackSamplingItems bool
}

func NewNetworkScrapper(
	cfg *config.AvailNetworkScrapperConfig,
	db db.Database,
) (*NetworkScrapper, error) {
	if cfg == nil {
		return nil, fmt.Errorf("an empty avail network scrapper config was given")
	}
	cache, err := lru.New[uint64, struct{}](config.AvailBlobsSetCacheSize)
	if err != nil {
		return nil, err
	}
	networkScrapper := &NetworkScrapper{
		closeC:             make(chan struct{}),
		samplerNotCh:       make(chan []*models.SamplingItem, cfg.NotChannelBufferSize),
		internalBlockCache: cache,
		trackSamplingItems: true, // enabled by default
		db:                 db,
		network:            config.NetworkFromStr(cfg.Network),
	}
	return networkScrapper, nil
}

func (s *NetworkScrapper) Serve(ctx context.Context) error {
	mainCtx, cancel := context.WithCancel(ctx)

	// only when the NetworkScrapper is initiated:
	// create the blockTracker with the custom callback function
	// and append it to the networkScrappper
	callBackConsumer, err := NewCallBackConsumer(mainCtx, s.blockTrackerCallBackFn)
	if err != nil {
		cancel()
		return err
	}
	blockTracker, err := NewBlockTracker(
		mainCtx,
		s.cfg.BlockTrackerCfg,
		callBackConsumer,
	)

	// check if we want to have some alternating tracking of samples
	if s.withPeriodicSamplingAlternator() {
		go s.spawPeriodicSamplingAlternator(mainCtx)
	}

	// TODO: this should be updated to the suture.Stop() call, but we need to update?
	go func() {
		defer cancel()
		select {
		case <-ctx.Done():
			log.Warn("network scrapper context died")
		case <-mainCtx.Done():
			log.Warn("network scrapper context died")
		case <-s.closeC:
			err = blockTracker.Close(mainCtx)
			log.Error(err)
		}
		return
	}()

	var errWg errgroup.Group
	errWg.Go(func() error { return blockTracker.Start(mainCtx) })
	return errWg.Wait()
}

func (s *NetworkScrapper) Close(ctx context.Context) error {
	select {
	case s.closeC <- struct{}{}:
	case <-ctx.Done():
	}
	return nil
}

func (s *NetworkScrapper) withPeriodicSamplingAlternator() bool {
	return s.cfg.TrackInterval != time.Duration(0) && s.cfg.TrackDuration != time.Duration(0)
}

func (s *NetworkScrapper) spawPeriodicSamplingAlternator(ctx context.Context) error {
	trackDurationT := time.NewTicker(s.cfg.TrackDuration)
	trackIntervalT := time.NewTicker(s.cfg.TrackInterval)
	for {
		select {

		case <-trackDurationT.C:
			s.alternateSamplingItemTrack()
			trackDurationT.Stop()

		case <-trackIntervalT.C:
			s.alternateSamplingItemTrack()
			trackDurationT.Reset(s.cfg.TrackDuration)

		case <-ctx.Done():
			log.Debug("the context of the AvailNetworkScrapper is done, closing the periodic sampling alternator")
			return nil
		}
	}
}

// alternateSamplingItemTrack is a threadsafe inversion of the existing SamplingItem flag
func (s *NetworkScrapper) alternateSamplingItemTrack() {
	s.samplingItemsMut.Lock()
	defer s.samplingItemsMut.Unlock()
	s.trackSamplingItems = !s.trackSamplingItems
}

func (s *NetworkScrapper) isTrackDASItemsEnabled() bool {
	s.samplingItemsMut.Lock()
	defer s.samplingItemsMut.Unlock()
	return s.trackSamplingItems
}

func (s *NetworkScrapper) blockTrackerCallBackFn(
	ctx context.Context,
	blockNot *BlockNotification,
	lastReqT time.Time) error {

	// translate the Avail Block to General Block info
	block, err := NewBlock(FromAPIBlockHeader(blockNot.BlockInfo))
	if err != nil {
		return err
	}

	// first thing, check if we saw that block-already in the internalBlockCache
	// if so, ignore the blockNot
	if s.internalBlockCache.Contains(block.Number) {
		return nil
	}
	// if not, add it to the cache (so far we don't care about the eviction)
	s.internalBlockCache.Add(block.Number, struct{}{})

	// if we persist blocks enabled, store the block info in the database
	dbBlock := block.ToDBBlock(s.network)
	if s.cfg.TrackBlocksOnDB {
		err = s.db.PersistNewBlock(ctx, &dbBlock)
	}

	// if, and only if, we want to track this particular segments:
	// 1. extract all the Avail DAS cells into DASSamplingItems (enable their ToTrack flag if it's time)
	// 2. store them in the DB
	// 3. notify directly the DataSampler of new items to blockTracker
	if s.isTrackDASItemsEnabled() {
		samplingItems := block.GetDASSamplingItems(s.network)
		err = s.db.PersistNewSamplingItems(ctx, samplingItems)
		return s.notifySampler(ctx, samplingItems)
	}
	return nil
}

func (s *NetworkScrapper) notifySampler(ctx context.Context, samplingItems []*models.SamplingItem) error {
	notCtx, cancel := context.WithTimeout(ctx, s.cfg.SamplerNotifyTimeout)
	defer cancel()

	select {
	case s.samplerNotCh <- samplingItems:
		log.WithField("items", len(samplingItems)).Debug("notified from the Avail NetworkScrapper of new DAS items available")
		return nil
	case <-notCtx.Done():
		return fmt.Errorf("iterrupted the AvailNetworkScrapper -> DataSampler notification due to a timeout")
	}
}

func (s *NetworkScrapper) GetSamplingItemStream() chan []*models.SamplingItem {
	return s.samplerNotCh
}

// syncs up with the database any prior existing sampleable item that we should keep tracking
func (ds *NetworkScrapper) SyncWithDatabase(ctx context.Context) ([]*models.SamplingItem, error) {
	// get all the samples that are still:
	// - trackeable
	// - in TTL to sample
	// - belong to the network

	syncCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	items, err := ds.db.GetSampleableItems(syncCtx)
	if err != nil {
		return nil, err
	}
	if len(items) < 0 {
		log.Warn("no sampleable blobs were found at DB")
		return []*models.SamplingItem{}, nil
	}

	sampleItems := make([]*models.SamplingItem, len(items))
	for i, item := range items {
		if !ds.internalBlockCache.Contains(item.BlockLink) {
			ds.internalBlockCache.Add(item.BlockLink, struct{}{})
		}
		sampleItems[i] = &item
	}
	return sampleItems, nil
}
