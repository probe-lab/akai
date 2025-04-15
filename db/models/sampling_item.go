package models

import (
	"time"

	"github.com/probe-lab/akai/config"
)

var (
	SamplingItemTableName   = "items"
	SamplingitemBatcherSize = 1024
)

type SamplingItem struct {
	Timestamp   time.Time               `ch:"timestamp" json:"timestamp"`
	Network     config.Network          `ch:"network" json:"network"`
	ItemType    config.SamplingItemType `ch:"item_type" json:"item_type"`
	SampleType  config.SamplingType     `ch:"sample_type" json:"sample_type"`
	BlockLink   uint64                  `ch:"block_link" json:"block_link"`
	Key         string                  `ch:"key" json:"key"`
	Hash        string                  `ch:"hash" json:"hash"`
	DASRow      uint32                  `ch:"das_row" json:"das_row"`
	DASColumn   uint32                  `ch:"das_column" json:"das_column"`
	Metadata    string                  `ch:"metadata" json:"metadata"`
	Traceable   bool                    `ch:"traceable" json:"traceable"`
	SampleUntil time.Time               `ch:"sample_until" json:"sample_until"`
	NextVisit   time.Time               // internal for the ItemSet
}

func (s SamplingItem) IsComplete() bool {
	return s.Network.IsComplete() &&
		!s.Timestamp.IsZero() &&
		s.BlockLink > 0 &&
		s.Key != "" &&
		!s.SampleUntil.IsZero()
}

func (s SamplingItem) TableName() string {
	return SamplingItemTableName
}

func (s SamplingItem) QueryValues() map[string]any {
	return map[string]any{
		"timestamp":    s.Timestamp,
		"network":      s.Network.String(),
		"item_type":    s.ItemType.String(),
		"sample_type":  s.SampleType.String(),
		"block_link":   s.BlockLink,
		"key":          s.Key,
		"hash":         s.Hash,
		"das_row":      s.DASRow,
		"das_column":   s.DASColumn,
		"metadata":     s.Metadata,
		"traceable":    s.Traceable,
		"sample_until": s.SampleUntil,
	}
}

func (s SamplingItem) BatchingSize() int {
	return SamplingitemBatcherSize
}

func (s SamplingItem) IsReadyForNextPing() bool {
	return !s.NextVisit.IsZero() && time.Now().After(s.NextVisit)
}

func (s SamplingItem) IsFinished() bool {
	return time.Now().After(s.SampleUntil)
}
