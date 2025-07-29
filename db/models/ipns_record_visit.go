package models

import (
	"time"
)

var (
	IPNSRecordVisitTableName   = "ipns_record_visits"
	IPNSRecordVisitBatcherSize = 4096
)

type IPNSRecordVisit struct {
	VisitRound    uint64        `ch:"visit_round" json:"visit_round"` // Round number of the crawling visit
	Timestamp     time.Time     `ch:"timestamp" json:"timestamp"`     // When the peer was visited
	Record        string        `ch:"record" json:"record"`           // Unique identifier of the IPNS record
	RecordType    string        `ch:"record_type" json:"record_type"` // Type of IPNS record (e.g., "ipns-record", "dns-link")
	Quorum        uint8         `ch:"quorum" json:"quorum"`           // Quorum target for the DHT lookup
	SeqNumber     uint32        `ch:"seq_number" json:"seq_number"`   // The highest found version for the ipns record
	TTL           time.Duration `ch:"ttl" json:"ttl"`
	IsValid       bool          `ch:"is_valid" json:"is_valid"`
	IsRetrievable bool          `ch:"is_retrievable" json:"is_retrievable"` // Whether the item was retrievable or not from the Network
	Result        string        `ch:"result" json:"result"`                 // Resulting CID/Link representation
	Duration      time.Duration `ch:"duration" json:"duration"`             // Time taken to establish connection/gather info
	Error         string        `ch:"error" json:"error"`                   // Error message if visit failed
}

func (p IPNSRecordVisit) IsComplete() bool { return p.Record != "" && !p.Timestamp.IsZero() }

func (p IPNSRecordVisit) TableName() string { return IPNSRecordVisitTableName }

func (p IPNSRecordVisit) QueryValues() map[string]any {
	return map[string]any{
		"visit_round":    p.VisitRound,
		"timestamp":      p.Timestamp,
		"record":         p.Record,
		"record_type":    p.RecordType,
		"quorum":         p.Quorum,
		"seq_number":     p.SeqNumber,
		"ttl":            p.TTL,
		"is_valid":       p.IsValid,
		"is_retrievable": p.IsRetrievable,
		"result":         p.Result,
		"duration_ms":    p.Duration,
		"error":          p.Error,
	}
}

func (p IPNSRecordVisit) BatchingSize() int { return PeerInfoVisitBatcherSize }
