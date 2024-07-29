package models

import "time"

type AgnosticBlock struct {
	Timestamp time.Time
	Root      Root

	BlockNumber uint64
	Rows        uint32
	Columns     uint32

	LastSampled time.Time
	TrackingTTL time.Duration
}
