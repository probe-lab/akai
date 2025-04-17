package avail

import (
	"context"
	"fmt"

	"github.com/probe-lab/akai/avail/api"
	"github.com/probe-lab/akai/config"
	log "github.com/sirupsen/logrus"
	"github.com/thejerf/suture/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var DefaultBlockTracker = &config.AvailBlockTrackerConfig{
	Network:        config.DefaultAvailNetwork.String(),
	AvailAPIconfig: api.DefaultClientConfig,
	Meter:          otel.GetMeterProvider().Meter("avail_block_tracker"),
}

type BlockTracker struct {
	cfg    *config.AvailBlockTrackerConfig
	netCfg *config.NetworkConfiguration
	sup    *suture.Supervisor

	httpAPICli     *api.HTTPClient
	BlockRequester *BlockRequester
	blockConsumers []BlockConsumer

	// metrics
	totalBlocksCount metric.Int64ObservableCounter
	lastBlockGau     metric.Int64ObservableGauge
}

func NewBlockTracker(
	ctx context.Context,
	cfg *config.AvailBlockTrackerConfig,
	blockConsumers ...BlockConsumer) (*BlockTracker, error) {
	// api
	httpAPICli, err := api.NewHTTPCli(cfg.AvailAPIconfig)
	if err != nil {
		return nil, err
	}

	network := config.NetworkFromStr(cfg.Network)
	networkConfig, err := config.ConfigureNetwork(network)
	if err != nil {
		return nil, err
	}

	// block requester
	blockReq, err := NewBlockRequester(httpAPICli, networkConfig, blockConsumers)
	if err != nil {
		return nil, err
	}

	bTracker := &BlockTracker{
		sup:            suture.NewSimple("avail-block-tracker"),
		cfg:            cfg,
		netCfg:         networkConfig,
		httpAPICli:     httpAPICli,
		BlockRequester: blockReq,
		blockConsumers: blockConsumers,
	}

	log.WithFields(log.Fields{
		"new_block_check_interval": config.BlockIntervalTarget,
		"consumers":                getTypesFromBlockConsumers(bTracker.blockConsumers),
	}).Info("avail block tracker successfully created...")

	return bTracker, nil
}

func (t *BlockTracker) Start(ctx context.Context) error {
	// init metrics
	t.initMetrics()

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

func (t *BlockTracker) Close(ctx context.Context) error {
	return nil // TODO: update with sutue.Supervisor.Stop()
}

// initMetrics initializes various prometheus metrics and stores the meters
// on the [BlockTracker] object.
func (t *BlockTracker) initMetrics() (err error) {
	t.totalBlocksCount, err = t.cfg.Meter.Int64ObservableCounter("total_blocks")
	if err != nil {
		return fmt.Errorf("new total_block counter: %w", err)
	}

	_, err = t.cfg.Meter.RegisterCallback(func(ctx context.Context, obs metric.Observer) error {
		obs.ObserveInt64(t.totalBlocksCount, int64(t.BlockRequester.totalRequestedBlocks))
		return nil
	}, t.totalBlocksCount)
	if err != nil {
		return fmt.Errorf("register total_block counter callback: %w", err)
	}

	t.lastBlockGau, err = t.cfg.Meter.Int64ObservableGauge("latest_block")
	if err != nil {
		return fmt.Errorf("new latest_block counter: %w", err)
	}

	_, err = t.cfg.Meter.RegisterCallback(func(ctx context.Context, obs metric.Observer) error {
		obs.ObserveInt64(t.lastBlockGau, int64(t.BlockRequester.lastBlockTracked))
		return nil
	}, t.lastBlockGau)
	if err != nil {
		return fmt.Errorf("register latest_block counter callback: %w", err)
	}

	return nil
}
