package api

import (
	"time"

	"github.com/probe-lab/akai/config"
)

type ACK struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

type PongReply struct {
	Msg string `json:"msg"`
}

type SupportedNetworks struct {
	Network config.Network `json:"Networks"`
}

type Block struct {
	Timestamp   time.Time      `json:"timestamp"`
	Network     string         `json:"network"`
	Number      uint64         `json:"number"`
	Hash        string         `json:"hash"`
	Key         string         `json:"key"`
	ParentHash  string         `json:"parent_hash"`
	Rows        uint64         `json:"rows"`
	Columns     uint64         `json:"columns"`
	Items       []DASItem      `json:"das_items"`
	Metadata    map[string]any `json:"metadata"`
	SampleUntil time.Time      `json:"sample_until"`
}

type DASItem struct {
	Timestamp   time.Time      `json:"timestamp"`
	Network     string         `json:"network"`
	BlockLink   uint64         `json:"block_link"`
	Key         string         `json:"key"`
	Hash        string         `json:"hash"`
	Row         uint64         `json:"row"`
	Column      uint64         `json:"column"`
	Metadata    map[string]any `json:"metadata"`
	SampleUntil time.Time      `json:"sample_until"`
	SampleType  string         `json:"sample_type"`
}
