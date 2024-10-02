package models

import "time"

var (
	VisitTableName   = "visits"
	visitBatcherSize = segmentBatcherSize * 4
)

type AgnosticVisit struct {
	VisitRound    uint64    `ch:"visit_round" json:"visit_round"`
	Timestamp     time.Time `ch:"timestamp" json:"timestamp"`
	Key           string    `ch:"key" json:"key"`
	BlobNumber    uint64    `ch:"blob_number" json:"blob_number"`
	Row           uint32    `ch:"row" json:"row"`
	Column        uint32    `ch:"column" json:"column"`
	DurationMs    int64     `ch:"duration_ms" json:"duration_ms"`
	IsRetrievable bool      `ch:"is_retrievable" json:"is_retrievable"`
	Providers     uint32    `ch:"providers" json:"providers"`
	Bytes         uint32    `ch:"bytes" json:"bytes"`
	Error         string    `ch:"error" json:"error"`
}

func (b AgnosticVisit) IsComplete() bool {
	return b.Key != "" && !b.Timestamp.IsZero()
}

func (b AgnosticVisit) TableName() string {
	return VisitTableName
}

func (b AgnosticVisit) QueryValues() map[string]any {
	return map[string]any{
		"visit_round":    b.VisitRound,
		"timestamp":      b.Timestamp,
		"key":            b.Key,
		"blob_number":    b.BlobNumber,
		"row":            b.Row,
		"column":         b.Column,
		"duration_ms":    b.DurationMs,
		"is_retrievable": b.IsRetrievable,
		"providers":      b.Providers,
		"bytes":          b.Bytes,
		"error":          b.Error,
	}
}

func (b AgnosticVisit) BatchingSize() int {
	return visitBatcherSize
}
