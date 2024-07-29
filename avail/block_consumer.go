package avail

import (
	"context"
	"time"

	"github.com/probe-lab/akai/avail/api"
	log "github.com/sirupsen/logrus"
)

type ConsumerType string

var (
	TextConsumerType ConsumerType = "text"
)

type BlockConsumer interface {
	Type() ConsumerType
	Serve(context.Context) error
	ProccessNewBlock(context.Context, *BlockNotification) error
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

func (tc *TextConsumer) ProccessNewBlock(ctx context.Context, blockNot *BlockNotification) error {
	block, err := NewBlock(FromAPIBlockHeader(blockNot.BlockInfo))
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"req_time":  blockNot.RequestTime,
		"hash":      blockNot.BlockInfo.Hash,
		"number":    blockNot.BlockInfo.Number,
		"rows":      blockNot.BlockInfo.Extension.Rows,
		"colums":    blockNot.BlockInfo.Extension.Columns,
		"data_root": blockNot.BlockInfo.Extension.DataRoot,
		"cid":       block.Cid().Hash().B58String(),
	}).Info("new avail block...")

	return nil
}

// TODO: add API interface with the Akai service
