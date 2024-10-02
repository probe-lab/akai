package clickhouse

import (
	"time"

	"github.com/ClickHouse/ch-go/proto"
)

type flusheableBatcher interface {
	// from table driver
	Tag() string
	TableName() string
	BaseQuery() string

	// methods
	currentLen() int
	isFull() bool
	isFlusheable() bool
	getPersistable() (proto.Input, int)
	resetBatcher()
}

// queryBatcher[T] is a generic ready to work synchronous batcher for akai's main modules
type queryBatcher[T any] struct {
	tableDriver[T]
	items []T

	// control values
	maxFlushT  time.Duration
	lastFlushT time.Time

	maxSize      int // max number of items it can hold befor flushing
	currentItems int
	totalItems   int64
}

var _ flusheableBatcher = (*queryBatcher[struct{}])(nil)

func newQueryBatcher[T any](
	driver tableDriver[T],
	maxFlushT time.Duration,
	maxBatchSize int,
) (*queryBatcher[T], error) {
	return &queryBatcher[T]{
		tableDriver:  driver,
		maxFlushT:    maxFlushT,
		lastFlushT:   time.Now(),
		maxSize:      maxBatchSize,
		currentItems: 0,
		items:        make([]T, 0, maxBatchSize),
	}, nil
}

func (b *queryBatcher[T]) currentLen() int {
	return b.currentItems
}

// upper layers should take care of flushing
func (b *queryBatcher[T]) addItem(item T) bool {
	// convert the item to "persitable"
	b.items = append(b.items, item)
	b.currentItems++
	b.totalItems++
	return b.isFull()
}

func (b *queryBatcher[T]) isFull() bool {
	return b.currentLen() >= b.maxSize
}

func (b *queryBatcher[T]) isFlusheable() bool {
	return b.lastFlushT.After(b.lastFlushT.Add(b.maxFlushT))
}

// returns the array of items that are ready to be persisted, and the number of items within
func (b *queryBatcher[T]) getPersistable() (proto.Input, int) {
	// copy items into a single Input
	input := b.inputConverter(b.items)
	// return copy
	return input, b.currentItems
}

// resets the metrics and values within the batcher
func (b *queryBatcher[T]) resetBatcher() {
	b.lastFlushT = time.Now()
	b.items = make([]T, 0, b.maxSize)
	b.currentItems = 0
}
