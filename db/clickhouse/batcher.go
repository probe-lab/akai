package clickhouse

import (
	"github.com/ClickHouse/ch-go/proto"
)

// it is ready to work synchronously
type queryBatcher[T any] struct {
	tableDriver[T]
	items []T

	// control values
	maxSize      int // max number of items it can hold befor flushing
	currentItems int
	totalItems   int64
}

func newQueryBatcher[T any](
	driver tableDriver[T],
	maxBatchSize int) (*queryBatcher[T], error) {

	return &queryBatcher[T]{
		tableDriver:  driver,
		maxSize:      maxBatchSize,
		currentItems: 0,
		items:        make([]T, 0, maxBatchSize),
	}, nil
}

func (b *queryBatcher[T]) currentLen() int {
	return b.currentItems
}

func (b *queryBatcher[T]) baseQuery() string {
	return b.tableDriver.baseQuery
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

// returns the array of items that are ready to be persisted, and the number of items within
func (b *queryBatcher[T]) getPersistable() (proto.Input, int) {
	// copy items into a single Input
	input := b.inputConverter(b.items)
	// return copy
	return input, b.currentItems
}

// resets the metrics and values within the batcher
func (b *queryBatcher[T]) resetBatcher() {
	b.items = make([]T, 0, b.maxSize)
	b.currentItems = 0
}
