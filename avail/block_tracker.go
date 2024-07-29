package avail

import (
	"context"

	"github.com/probe-lab/akai/avail/api"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db"
	log "github.com/sirupsen/logrus"
	"github.com/thejerf/suture/v4"
)

type BlockTracker struct {
	sup *suture.Supervisor

	httpAPICli     *api.HTTPClient
	BlockRequester *BlockRequester
	blockConsumers []BlockConsumer
}

func NewBlockTracker(cfg config.AvailBlockTracker) (*BlockTracker, error) {
	// api
	httpAPICli, err := api.NewHTTPCli(cfg.AvailHttpAPIClient)
	if err != nil {
		return nil, err
	}

	// compose block consumers
	var blockConsumers []BlockConsumer
	if cfg.TextConsumer {
		textConsumer, err := NewTextConsumer()
		if err != nil {
			return nil, err
		}
		blockConsumers = append(blockConsumers, textConsumer)
	}

	// block requester
	network := db.Network{}.FromString(cfg.Network)
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
	return t.sup.Serve(ctx)
}
