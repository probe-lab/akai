package api

import (
	"time"

	"github.com/probe-lab/akai/db/models"
)

type ACK struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

type PongReply struct {
	Msg string `json:"msg"`
}

type SupportedNetworks struct {
	Network models.Network `json:"Networks"`
}

type Blob struct {
	Timestamp   time.Time      `json:"timestamp"`
	Network     models.Network `json:"network"`
	Number      uint64         `json:"number"`
	Hash        string         `json:"hash"`
	ParentHash  string         `json:"parent_hash"`
	Rows        uint64         `json:"rows"`
	Columns     uint64         `json:"columns"`
	Segments    []BlobSegment  `json:"segments"`
	Metadata    map[string]any `json:"metadata"`
	SampleUntil time.Time      `json:"sample_until"`
}

type BlobSegment struct {
	Timestamp   time.Time `json:"timestamp"`
	BlobNumber  uint64    `json:"blob_number"`
	Key         string    `json:"key"`
	Row         uint64    `json:"row"`
	Column      uint64    `json:"column"`
	Bytes       []byte    `json:"bytes"`
	SampleUntil time.Time `json:"sample_until"`
}
