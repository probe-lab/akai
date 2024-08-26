package models

import "time"

var SegmentTableName = "segments"
var segmentBatcherSize = 1024

type AgnosticSegment struct {
	Timestamp   time.Time `ch:"timestamp" json:"timestamp"`
	BlockNumber uint64    `ch:"blob_number" json:"blob_number"`
	Key         string    `ch:"key" json:"key"`
	Row         uint32    `ch:"rows" json:"row"`
	Column      uint32    `ch:"columns" json:"column"`
	SampleUntil time.Time `ch:"sample_until" json:"sample_until"`
}

func (b AgnosticSegment) IsComplete() bool {
	return b.BlockNumber > 0 &&
		b.Key != "" &&
		!b.SampleUntil.IsZero()
}

func (b AgnosticSegment) TableName() string {
	return SegmentTableName
}

func (b AgnosticSegment) QueryValues() map[string]any {
	return map[string]any{
		"timestamp":    b.Timestamp,
		"key":          b.Key,
		"block_number": b.BlockNumber,
		"row":          b.Row,
		"column":       b.Column,
		"sample_until": b.SampleUntil,
	}
}

func (b AgnosticSegment) BatchingSize() int {
	return segmentBatcherSize
}
