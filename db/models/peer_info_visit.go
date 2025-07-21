package models

import (
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"time"
)

var (
	PeerInfoVisitTableName   = "peer_info_visits"
	PeerInfoVisitBatcherSize = 4096
)

type PeerInfoVisit struct {
	VisitRound      uint64                `ch:"visit_round" json:"visit_round"`         // Round number of the crawling visit
	Timestamp       time.Time             `ch:"timestamp" json:"timestamp"`             // When the peer was visited
	Network         string                `ch:"network" json:"network"`                 // Network identifier (e.g., "ipfs", "libp2p")
	PeerID          string                `ch:"peerID" json:"peerID"`                   // Unique identifier of the peer
	Duration        time.Duration         `ch:"duration" json:"duration"`               // Time taken to establish connection/gather info
	AgentVersion    string                `ch:"agentVersion" json:"agentVersion"`       // Software version reported by the peer
	Protocols       []protocol.ID         `ch:"protocols" json:"protocols"`             // List of protocols supported by the peer
	ProtocolVersion string                `ch:"protocolVersion" json:"protocolVersion"` // Version of the main protocol
	MultiAddresses  []multiaddr.Multiaddr `ch:"multiAddresses" json:"multiAddresses"`   // Network addresses where peer can be reached
	Error           string                `ch:"error" json:"error"`                     // Error message if visit failed
}

func (p PeerInfoVisit) IsComplete() bool { return p.PeerID != "" && !p.Timestamp.IsZero() }

func (p PeerInfoVisit) TableName() string { return PeerInfoVisitTableName }

func (p PeerInfoVisit) QueryValues() map[string]any {
	return map[string]any{
		"visit_round":     p.VisitRound,
		"timestamp":       p.Timestamp,
		"network":         p.Network,
		"peerID":          p.PeerID,
		"duration_ms":     p.Duration,
		"agentVersion":    p.AgentVersion,
		"protocols":       p.Protocols,
		"protocolVersion": p.ProtocolVersion,
		"multiAddresses":  p.MultiAddresses,
		"error":           p.Error,
	}
}

func (p PeerInfoVisit) BatchingSize() int { return PeerInfoVisitBatcherSize }
