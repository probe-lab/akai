package models

import "time"

var SegmentTableName = "segments"
var segmentBatcherSize = 1024

type AgnosticSegment struct {
	Timestamp   time.Time `ch:"timestamp" json:"timestamp"`
	BlobNumber  uint64    `ch:"blob_number" json:"blob_number"`
	Key         string    `ch:"key" json:"key"`
	Row         uint32    `ch:"row" json:"row"`
	Column      uint32    `ch:"column" json:"column"`
	SampleUntil time.Time `ch:"sample_until" json:"sample_until"`
	NextVisit   time.Time // for the segmentSet
}

func (s AgnosticSegment) IsComplete() bool {
	return s.BlobNumber > 0 &&
		s.Key != "" &&
		!s.SampleUntil.IsZero()
}

func (s AgnosticSegment) TableName() string {
	return SegmentTableName
}

func (s AgnosticSegment) QueryValues() map[string]any {
	return map[string]any{
		"timestamp":    s.Timestamp,
		"key":          s.Key,
		"blob_number":  s.BlobNumber,
		"row":          s.Row,
		"column":       s.Column,
		"sample_until": s.SampleUntil,
	}
}

func (s AgnosticSegment) BatchingSize() int {
	return segmentBatcherSize
}

func (s AgnosticSegment) IsReadyForNextPing() bool {
	return !s.NextVisit.IsZero() && time.Now().After(s.NextVisit)
}

func (s AgnosticSegment) IsFinished() bool {
	return time.Now().After(s.SampleUntil)
}
