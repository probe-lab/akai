package models

import "time"

var VisitTableName = "visits"
var visitBatcherSize = segmentBatcherSize * 4

type AgnosticVisit struct {
	Timestamp     time.Time     `ch:"timestamp" json:"timestamp"`
	Key           string        `ch:"key" json:"key"`
	Duration      time.Duration `ch:"duration" json:"duration"`
	IsRetrievable bool          `ch:"is_retrievable" json:"is_retrievable"`
}

func (b AgnosticVisit) IsComplete() bool {
	return b.Key != "" && !b.Timestamp.IsZero()
}

func (b AgnosticVisit) TableName() string {
	return VisitTableName
}

func (b AgnosticVisit) QueryValues() map[string]any {
	return map[string]any{
		"timestamp":      b.Timestamp,
		"key":            b.Key,
		"duration":       b.Duration,
		"is_retrievable": b.IsRetrievable,
	}
}

func (b AgnosticVisit) BatchingSize() int {
	return visitBatcherSize
}
