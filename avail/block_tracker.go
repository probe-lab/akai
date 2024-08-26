package avail

import (
	"context"
	"time"

	"github.com/probe-lab/akai/avail/api"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
	"github.com/thejerf/suture/v4"
)

var DefaultClientConfig = config.AvailAPIConfig{
	Host:    "localhost",
	Port:    5000,
	Timeout: 10 * time.Second,
}

type BlockTracker struct {
	sup *suture.Supervisor

	httpAPICli     *api.HTTPClient
	BlockRequester *BlockRequester
	blockConsumers []BlockConsumer
}

func NewBlockTracker(cfg config.AvailBlockTracker) (*BlockTracker, error) {
	// api
	httpAPICli, err := api.NewHTTPCli(cfg.AvailAPIconfig)
	if err != nil {
		return nil, err
	}

	network := models.NetworkFromStr(cfg.Network)
	// compose block consumers (Text, Akai_Api)
	blockConsumers := make([]BlockConsumer, 0)
	if cfg.TextConsumer {
		textConsumer, err := NewTextConsumer()
		if err != nil {
			return nil, err
		}
		blockConsumers = append(blockConsumers, textConsumer)
	}
	if cfg.AkaiAPIconsumer {
		akaiAPIconsumer, err := NewAkaiAPIconsumer(network, cfg.AkaiAPIconfig)
		if err != nil {
			return nil, err
		}
		blockConsumers = append(blockConsumers, akaiAPIconsumer)
	}

	// block requester
	blockReq, err := NewBlockRequester(httpAPICli, network, blockConsumers)
	if err != nil {
		return nil, err
	}

	bTracker := &BlockTracker{
		sup:            suture.NewSimple("avail-block-tracker"),
		httpAPICli:     httpAPICli,
		BlockRequester: blockReq,
		blockConsumers: blockConsumers,
	}

	log.WithFields(log.Fields{
		"new_block_check_interval": api.BlockIntervalTarget,
		"consumers":                getTypesFromBlockConsumers(bTracker.blockConsumers),
	}).Info("avail block tracker successfully created...")

	return bTracker, nil
}

func (t *BlockTracker) Start(ctx context.Context) error {
	// add each of the consumers to the suture supervisor manager
	for _, consumer := range t.blockConsumers {
		t.sup.Add(consumer)
	}
	t.sup.Add(t.httpAPICli)
	t.sup.Add(t.BlockRequester)
	// add all the existing consumers to the suture
	for _, consumer := range t.blockConsumers {
		t.sup.Add(consumer)
	}
	return t.sup.Serve(ctx)
}
