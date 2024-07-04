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
	types := make([]ConsumerType, len(consumers), len(consumers))
	for i, con := range consumers {
		types[i] = con.Type()
	}
	return types
}

// text consumer
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

func (tc *TextConsumer) ProccessNewBlock(ctx context.Context, block *BlockNotification) error {

	log.WithFields(log.Fields{
		"req_time":  block.RequestTime,
		"hash":      block.BlockInfo.Hash,
		"number":    block.BlockInfo.Number,
		"rows":      block.BlockInfo.Extension.Rows,
		"colums":    block.BlockInfo.Extension.Columns,
		"data_root": block.BlockInfo.Extension.DataRoot,
	}).Info("new avail block...")

	return nil
}

// TODO: add API interface with the Akai service
