package avail

import (
	"context"
	"fmt"
	"time"

	akai_api "github.com/probe-lab/akai/api"
	"github.com/probe-lab/akai/avail/api"
	"github.com/probe-lab/akai/config"
	log "github.com/sirupsen/logrus"
)

type ConsumerType string

var (
	TextConsumerType    ConsumerType = "text"
	AkaiAPIConsumerType ConsumerType = "akai_api"
)

type BlockConsumer interface {
	Type() ConsumerType
	Serve(context.Context) error
	ProccessNewBlock(context.Context, *BlockNotification, time.Time) error
}

type BlockNotification struct {
	RequestTime time.Time
	BlockInfo   api.V2BlockHeader
}

func getTypesFromBlockConsumers(consumers []BlockConsumer) []ConsumerType {
	types := make([]ConsumerType, len(consumers))
	for i, con := range consumers {
		types[i] = con.Type()
	}
	return types
}

// TextConsumer (most simple text logger consumer)
type TextConsumer struct{}

var _ BlockConsumer = (*TextConsumer)(nil)

func NewTextConsumer() (*TextConsumer, error) {
	return &TextConsumer{}, nil
}

func (tc *TextConsumer) Type() ConsumerType {
	return TextConsumerType
}

func (tc *TextConsumer) Serve(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (tc *TextConsumer) ProccessNewBlock(ctx context.Context, blockNot *BlockNotification, lastReqT time.Time) error {
	block, err := NewBlock(FromAPIBlockHeader(blockNot.BlockInfo))
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"req_time":              blockNot.RequestTime,
		"hash":                  blockNot.BlockInfo.Hash,
		"number":                blockNot.BlockInfo.Number,
		"rows":                  blockNot.BlockInfo.Extension.Rows,
		"colums":                blockNot.BlockInfo.Extension.Columns,
		"data_root":             blockNot.BlockInfo.Extension.DataRoot,
		"cid":                   block.Cid().Hash().B58String(),
		"time-since-last-block": time.Since(lastReqT),
	}).Info("new avail block...")

	return nil
}

// TODO: add API interface with the Akai service
type AkaiAPIconsumer struct {
	config         *config.AkaiAPIClientConfig
	networkConfing *config.NetworkConfiguration
	cli            *akai_api.Client
}

var _ BlockConsumer = (*AkaiAPIconsumer)(nil)

func NewAkaiAPIconsumer(networkConfig *config.NetworkConfiguration, cfg *config.AkaiAPIClientConfig) (*AkaiAPIconsumer, error) {
	apiCli, err := akai_api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	// ensure that the network is the same one
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	apiNet, err := apiCli.GetSupportedNetworks(ctx)
	if err != nil {
		return nil, err
	}
	if networkConfig.Network.String() != apiNet.Network.String() {
		return nil, fmt.Errorf("block consumer and Akai API running in different networks %s - %s", networkConfig.Network.String(), apiNet.Network.String())
	}

	return &AkaiAPIconsumer{
		config:         cfg,
		networkConfing: networkConfig,
		cli:            apiCli,
	}, nil
}

func (ac *AkaiAPIconsumer) Type() ConsumerType {
	return AkaiAPIConsumerType
}

func (ac *AkaiAPIconsumer) Serve(ctx context.Context) error {
	return ac.cli.Serve(ctx)
}

func (ac *AkaiAPIconsumer) ProccessNewBlock(ctx context.Context, blockNot *BlockNotification, lastReqT time.Time) error {
	block, err := NewBlock(FromAPIBlockHeader(blockNot.BlockInfo))
	if err != nil {
		return err
	}

	apiBlob := block.ToAkaiAPIBlob(ac.networkConfing.Network, true)
	ctx, cancel := context.WithTimeout(ctx, ac.config.Timeout)
	defer cancel()
	err = ac.cli.PostNewBlob(ctx, apiBlob)
	if err != nil {
		return err
	}

	return nil
}
