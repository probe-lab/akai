package models

import "time"

var (
	SampleGenericVisitTableName   = "generic_visits"
	SampleGenericVisitBatcherSize = SamplingitemBatcherSize * 4
)

type SampleGenericVisit struct {
	VisitType     string    `ch:"visit_type" json:"visit_type"`
	VisitRound    uint64    `ch:"visit_round" json:"visit_round"`
	Network       string    `ch:"network" json:"network"`
	Timestamp     time.Time `ch:"timestamp" json:"timestamp"`
	Key           string    `ch:"key" json:"key"`
	DurationMs    int64     `ch:"duration_ms" json:"duration_ms"`
	ResponseItems int32     `ch:"response_items" json:"response_items"`
	Peers         []string  `ch:"peer_ids" json:"peer_ids"`
	Error         string    `ch:"error" json:"error"`
}

func (b SampleGenericVisit) IsComplete() bool {
	return b.Key != "" && !b.Timestamp.IsZero()
}

func (b SampleGenericVisit) TableName() string {
	return SampleGenericVisitTableName
}

func (b SampleGenericVisit) QueryValues() map[string]any {
	return map[string]any{
		"visit_type":     b.VisitType,
		"visit_round":    b.VisitRound,
		"network":        b.Network,
		"timestamp":      b.Timestamp,
		"key":            b.Key,
		"duration_ms":    b.DurationMs,
		"response_items": b.ResponseItems,
		"peer_ids":       b.Peers,
		"error":          b.Error,
	}
}

func (b SampleGenericVisit) BatchingSize() int {
	return SampleGenericVisitBatcherSize
}
