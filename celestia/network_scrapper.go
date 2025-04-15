package celestia

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/probe-lab/akai/api"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
)

var DefaultNetworkScrapperConfig = &config.CelestiaNetworkScrapperConfig{
	Network:              config.DefaultAvailNetwork.String(),
	NotChannelBufferSize: 1_000,
	SamplerNotifyTimeout: 20 * time.Second,
	AkaiAPIServiceConfig: api.DefaulServiceConfig,
}

type NetworkScrapper struct {
	cfg     *config.CelestiaNetworkScrapperConfig
	network config.Network
	closeC  chan struct{}

	itemNotCh         chan []*models.SamplingItem
	internalItemCache *lru.Cache[string, struct{}]
	apiSer            *api.Service
	db                db.Database
}

func NewNetworkScrapper(
	cfg *config.CelestiaNetworkScrapperConfig,
	db db.Database,
) (*NetworkScrapper, error) {

	apiSer, err := api.NewService(DefaultNetworkScrapperConfig.AkaiAPIServiceConfig)
	if err != nil {
		return nil, err
	}
	networkScrapper := &NetworkScrapper{
		cfg:       cfg,
		network:   config.NetworkFromStr(cfg.Network),
		closeC:    make(chan struct{}),
		itemNotCh: make(chan []*models.SamplingItem, cfg.NotChannelBufferSize),
		db:        db,
		apiSer:    apiSer,
	}

	apiSer.UpdateNewBlockHandler(networkScrapper.newBlockHandler)
	apiSer.UpdateNewItemHandler(networkScrapper.newItemHandler)
	apiSer.UpdateNewItemsHandler(networkScrapper.newItemsHandler)

	return networkScrapper, nil
}

func (s *NetworkScrapper) Serve(ctx context.Context) error {
	return nil
}

func (s *NetworkScrapper) Close(ctx context.Context) error {
	return nil
}

func (s *NetworkScrapper) notifySampler(ctx context.Context, samplingItems []*models.SamplingItem) error {
	notCtx, cancel := context.WithTimeout(ctx, s.cfg.SamplerNotifyTimeout)
	defer cancel()

	select {
	case s.itemNotCh <- samplingItems:
		log.WithField("items", len(samplingItems)).Debug("notified from the Avail NetworkScrapper of new DAS items available")
		return nil
	case <-notCtx.Done():
		return fmt.Errorf("iterrupted the AvailNetworkScrapper -> DataSampler notification due to a timeout")
	}
}

func (s *NetworkScrapper) GetSamplingItemStream(ctx context.Context) (chan []*models.SamplingItem, error) {
	return s.itemNotCh, nil
}

// syncs up with the database any prior existing sampleable item that we should keep tracking
func (s *NetworkScrapper) SyncWithDatabase(ctx context.Context) ([]*models.SamplingItem, error) {
	syncCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	items, err := s.db.GetSampleableItems(syncCtx)
	if err != nil {
		return nil, err
	}
	if len(items) < 0 {
		log.Warn("no sampleable blobs were found at DB")
		return []*models.SamplingItem{}, nil
	}

	sampleItems := make([]*models.SamplingItem, len(items), 0)
	for i, item := range items {
		if !s.internalItemCache.Contains(item.Key) {
			s.internalItemCache.Add(item.Key, struct{}{})
		}
		sampleItems[i] = item
	}
	return sampleItems, nil
}

func (s *NetworkScrapper) Init(ctx context.Context) error {
	return nil
}

func (s *NetworkScrapper) newBlockHandler(ctx context.Context, Block api.Block) error {
	log.WithFields(log.Fields{
		"block": Block.Number,
		"hash":  Block.Hash,
		"items": len(Block.Items),
	}).Info("new celestia-block arrived to the api")
	log.Warn("we don't support celestia blocks at the moment, only DHT namespaces")
	return nil
}

func (s *NetworkScrapper) newItemHandler(ctx context.Context, apiItem api.DASItem) error {
	log.WithFields(log.Fields{
		"key": apiItem.Key,
	}).Info("new celestia-item arrived to the api")

	// translate the Avail Block to General Block info
	item := s.getSamplingItemFromAPIitem(apiItem)

	// first thing, check if we saw that block-already in the internalBlockCache
	// if so, ignore the blockNot
	if s.internalItemCache.Contains(apiItem.Key) {
		return nil
	}
	// if not, add it to the cache (so far we don't care about the eviction)
	s.internalItemCache.Add(apiItem.Key, struct{}{})

	// add the segment info to the DB
	return s.db.PersistNewSamplingItem(ctx, item)
}

func (s *NetworkScrapper) newItemsHandler(ctx context.Context, apiItems []api.DASItem) error {
	log.WithFields(log.Fields{
		"items": len(apiItems),
	}).Info("new celestia namespace items arrived to the daemon")

	// add the segment info to the DB
	items := s.getSamplingItemFromAPIitems(apiItems)

	// add the segments to the segmentsSet
	for _, item := range items {
		// check if we already saw the namespace
		if s.internalItemCache.Contains(item.Key) {
			return nil
		}
		// if not, add it to the cache (so far we don't care about the eviction)
		s.internalItemCache.Add(item.Key, struct{}{})

		// add the namespace as a sampling item in the DB
		err := s.db.PersistNewSamplingItem(ctx, item)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *NetworkScrapper) getSamplingItemFromAPIitem(apiItem api.DASItem) *models.SamplingItem {
	metadataStr := ""
	if len(apiItem.Metadata) >= 0 {
		bs, err := json.Marshal(apiItem.Metadata)
		if err != nil {
			return nil
		}
		metadataStr = string(bs)
	}
	return &models.SamplingItem{
		Timestamp:   apiItem.Timestamp,
		Network:     s.network,
		ItemType:    config.CelestiaDHTNamesSpaceItemType, // TODO: do we want to select this on the API itself
		SampleType:  config.SamplePeers,                   // TODO: do we want to select this on the API itself
		BlockLink:   apiItem.BlockLink,
		Key:         apiItem.Key,
		Hash:        "",
		DASRow:      uint32(0),
		DASColumn:   uint32(0),
		Metadata:    metadataStr,
		Traceable:   true,
		SampleUntil: apiItem.SampleUntil,
		NextVisit:   time.Time{}, // empty for now, choose at the sampler
	}
}

func (s *NetworkScrapper) getSamplingItemFromAPIitems(segments []api.DASItem) []*models.SamplingItem {
	agSegs := make([]*models.SamplingItem, len(segments))
	for i, seg := range segments {
		agSeg := s.getSamplingItemFromAPIitem(seg)
		agSegs[i] = agSeg
	}
	return agSegs
}
