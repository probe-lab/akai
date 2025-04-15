package models

import "time"

var (
	BlockTableName   = "block"
	blockBatcherSize = 2
)

type Block struct {
	Timestamp   time.Time `ch:"timestamp" json:"timestamp"`
	Network     string    `ch:"network" json:"network"`
	Number      uint64    `ch:"number" json:"number"`
	Hash        string    `ch:"hash" json:"hash"`
	Key         string    `ch:"key" json:"key"`
	Metadata    string    `ch:"metadata" json:"metadata"`
	DASRows     uint32    `ch:"das_rows" json:"das_rows"`
	DASColumns  uint32    `ch:"das_columns" json:"das_columns"`
	SampleUntil time.Time `ch:"sample_until" json:"sample_until"`
}

func (b Block) IsComplete() bool {
	// check only those not-nullable values
	return !b.Timestamp.IsZero() &&
		b.Network != "" &&
		b.Number > 0 &&
		!b.SampleUntil.IsZero()
}

func (b Block) TableName() string {
	return BlockTableName
}

func (b Block) QueryValues() map[string]any {
	return map[string]any{
		"timestamp":    b.Timestamp,
		"network":      b.Network,
		"number":       b.Number,
		"hash":         b.Hash,
		"key":          b.Key,
		"metadata":     b.Metadata,
		"das_rows":     b.DASRows,
		"das_columns":  b.DASColumns,
		"sample_until": b.SampleUntil,
	}
}

func (b Block) BatchingSize() int {
	return blockBatcherSize
}
