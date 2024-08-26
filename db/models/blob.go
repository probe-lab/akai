package models

import "time"

var BlobTableName = "blobs"
var blobBatcherSize = 2

type AgnosticBlob struct {
	NetworkID   uint16    `ch:"network_id" json:"network_id"`
	Timestamp   time.Time `ch:"timestamp" json:"timestamp"`
	Hash        string    `ch:"hash" json:"hash"`
	Key         string    `ch:"key" json:"key"`
	BlockNumber uint64    `ch:"blob_number" json:"blob_number"`
	Rows        uint32    `ch:"rows" json:"rows"`
	Columns     uint32    `ch:"columns" json:"columns"`
	TrackingTTL time.Time `ch:"tracking_ttl" json:"tracking_ttl"`
}

func (b AgnosticBlob) IsComplete() bool {
	return b.BlockNumber > 0 &&
		b.NetworkID > 0 &&
		!b.TrackingTTL.IsZero()
}

func (b AgnosticBlob) TableName() string {
	return NetworkTableName
}

func (b AgnosticBlob) QueryValues() map[string]any {
	return map[string]any{
		"network_id":   b.NetworkID,
		"timestamp":    b.Timestamp,
		"hash":         b.Hash,
		"key":          b.Key,
		"block_number": b.BlockNumber,
		"rows":         b.Rows,
		"columns":      b.Columns,
		"tracking_ttl": b.TrackingTTL,
	}
}

func (b AgnosticBlob) BatchingSize() int {
	return blobBatcherSize
}
