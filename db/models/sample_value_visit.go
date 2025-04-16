package models

import "time"

var (
	SampleValueVisitTableName   = "value_visits"
	SampleValueVisitBatcherSize = SamplingitemBatcherSize * 4
)

type SampleValueVisit struct {
	VisitType     string    `ch:"visit_type" json:"visit_type"`
	VisitRound    uint64    `ch:"visit_round" json:"visit_round"`
	Network       string    `ch:"network" json:"network"`
	Timestamp     time.Time `ch:"timestamp" json:"timestamp"`
	Key           string    `ch:"key" json:"key"`
	BlockNumber   uint64    `ch:"block_number" json:"block_number"`
	DASRow        uint32    `ch:"das_row" json:"rdas_row"`
	DASColumn     uint32    `ch:"das_column" json:"das_column"`
	DurationMs    int64     `ch:"duration_ms" json:"duration_ms"`
	IsRetrievable bool      `ch:"is_retrievable" json:"is_retrievable"`
	Bytes         int32     `ch:"bytes" json:"bytes"`
	Error         string    `ch:"error" json:"error"`
}

func (b SampleValueVisit) IsComplete() bool {
	return b.Key != "" && !b.Timestamp.IsZero()
}

func (b SampleValueVisit) TableName() string {
	return SampleValueVisitTableName
}

func (b SampleValueVisit) QueryValues() map[string]any {
	return map[string]any{
		"visit_type":     b.VisitType,
		"visit_round":    b.VisitRound,
		"network":        b.Network,
		"timestamp":      b.Timestamp,
		"key":            b.Key,
		"block_number":   b.BlockNumber,
		"das_row":        b.DASRow,
		"das_column":     b.DASColumn,
		"duration_ms":    b.DurationMs,
		"is_retrievable": b.IsRetrievable,
		"bytes":          b.Bytes,
		"error":          b.Error,
	}
}

func (b SampleValueVisit) BatchingSize() int {
	return SampleValueVisitBatcherSize
}
