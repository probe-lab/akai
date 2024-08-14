package api

import "github.com/probe-lab/akai/db"

type ACK struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

type PongReply struct {
	Msg string `json:"msg"`
}

type SupportedNetworks struct {
	Network db.Network `json:"Networks"`
}

type Blob struct {
	Network      db.Network     `json:"network"`
	Number       uint64         `json:"number"`
	Hash         string         `json:"hash"`
	ParentHash   string         `json:"parent-hash"`
	Rows         uint64         `json:"rows"`
	Columns      uint64         `json:"columns"`
	BlobSegments BlobSegments   `json:"segments"`
	Metadata     map[string]any `json:"metadata"`
}

type BlobSegment struct {
	BlockNumber uint64 `json:"block-number"`
	Key         string `json:"key"`
	Row         uint64 `json:"row"`
	Column      uint64 `json:"column"`
	Bytes       []byte `json:"bytes"`
}

type BlobSegments struct {
	Segments []BlobSegment `json:"segments"`
}
