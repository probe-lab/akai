package avail

import (
	"context"
	"time"

	"github.com/probe-lab/akai/avail/api"
	log "github.com/sirupsen/logrus"
)

type ConsumerType string

var (
	TextConsumerType     ConsumerType = "text"
	AkaiAPIConsumerType  ConsumerType = "akai_api"
	CallBackConsumerType ConsumerType = "callback"
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

func (tc *TextConsumer) ProccessNewBlock(
	ctx context.Context,
	blockNot *BlockNotification,
	lastReqT time.Time) error {
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

type CallBackConsumer struct {
	blockProccessFn func(context.Context, *BlockNotification, time.Time) error
}

var _ BlockConsumer = (*CallBackConsumer)(nil)

func NewCallBackConsumer(
	ctx context.Context,
	blockProcessFn func(context.Context, *BlockNotification, time.Time) error,
) (*CallBackConsumer, error) {
	return &CallBackConsumer{
		blockProccessFn: blockProcessFn,
	}, nil
}

func (cb *CallBackConsumer) Type() ConsumerType {
	return CallBackConsumerType
}

func (cb *CallBackConsumer) Serve(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (cb *CallBackConsumer) ProccessNewBlock(
	ctx context.Context,
	blockNot *BlockNotification,
	lastReqT time.Time) error {
	return cb.blockProccessFn(ctx, blockNot, lastReqT)
}
