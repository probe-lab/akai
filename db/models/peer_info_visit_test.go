package models

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

func TestPeerInfoVisit_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		visit    PeerInfoVisit
		expected bool
	}{
		{
			name: "complete visit",
			visit: PeerInfoVisit{
				PeerID:    "12D3KooWABC123",
				Timestamp: time.Now(),
			},
			expected: true,
		},
		{
			name: "missing peer ID",
			visit: PeerInfoVisit{
				PeerID:    "",
				Timestamp: time.Now(),
			},
			expected: false,
		},
		{
			name: "missing timestamp",
			visit: PeerInfoVisit{
				PeerID:    "12D3KooWABC123",
				Timestamp: time.Time{},
			},
			expected: false,
		},
		{
			name: "both missing",
			visit: PeerInfoVisit{
				PeerID:    "",
				Timestamp: time.Time{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.visit.IsComplete()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestPeerInfoVisit_TableName(t *testing.T) {
	visit := PeerInfoVisit{}
	require.Equal(t, PeerInfoVisitTableName, visit.TableName())
	require.Equal(t, "peer_info_visits", visit.TableName())
}

func TestPeerInfoVisit_QueryValues(t *testing.T) {
	timestamp := time.Now()
	duration := 5 * time.Second
	protocols := []protocol.ID{"/ipfs/kad/1.0.0", "/ipfs/id/1.0.0"}
	ip4, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/4001")
	ip6, _ := multiaddr.NewMultiaddr("/ip6/::1/tcp/4001")
	multiAddrs := []multiaddr.Multiaddr{ip4, ip6}

	visit := PeerInfoVisit{
		VisitRound:      42,
		Timestamp:       timestamp,
		Network:         "ipfs",
		PeerID:          "12D3KooWABC123",
		Duration:        duration,
		AgentVersion:    "go-ipfs/0.13.0",
		Protocols:       protocols,
		ProtocolVersion: "ipfs/0.1.0",
		MultiAddresses:  multiAddrs,
		Error:           "connection timeout",
	}

	values := visit.QueryValues()

	require.Equal(t, uint64(42), values["visit_round"])
	require.Equal(t, timestamp, values["timestamp"])
	require.Equal(t, "ipfs", values["network"])
	require.Equal(t, "12D3KooWABC123", values["peerID"])
	require.Equal(t, duration, values["duration_ms"])
	require.Equal(t, "go-ipfs/0.13.0", values["agentVersion"])
	require.Equal(t, protocols, values["protocols"])
	require.Equal(t, "ipfs/0.1.0", values["protocolVersion"])
	require.Equal(t, multiAddrs, values["multiAddresses"])
	require.Equal(t, "connection timeout", values["error"])
}

func TestPeerInfoVisit_BatchingSize(t *testing.T) {
	visit := PeerInfoVisit{}
	require.Equal(t, PeerInfoVisitBatcherSize, visit.BatchingSize())
	require.Equal(t, 4096, visit.BatchingSize())
}

func TestPeerInfoVisit_StructureValidation(t *testing.T) {
	visit := PeerInfoVisit{
		VisitRound:      1,
		Timestamp:       time.Now(),
		Network:         "ipfs",
		PeerID:          "12D3KooWABC123",
		Duration:        100 * time.Millisecond,
		AgentVersion:    "go-ipfs/0.13.0",
		Protocols:       []protocol.ID{"/ipfs/kad/1.0.0"},
		ProtocolVersion: "ipfs/0.1.0",
		MultiAddresses:  []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/127.0.0.1/tcp/4001")},
		Error:           "",
	}

	require.True(t, visit.IsComplete())
	require.Equal(t, "peer_info_visits", visit.TableName())
	require.Equal(t, 4096, visit.BatchingSize())

	queryValues := visit.QueryValues()
	require.Len(t, queryValues, 10)
	require.Contains(t, queryValues, "visit_round")
	require.Contains(t, queryValues, "timestamp")
	require.Contains(t, queryValues, "network")
	require.Contains(t, queryValues, "peerID")
	require.Contains(t, queryValues, "duration_ms")
	require.Contains(t, queryValues, "agentVersion")
	require.Contains(t, queryValues, "protocols")
	require.Contains(t, queryValues, "protocolVersion")
	require.Contains(t, queryValues, "multiAddresses")
	require.Contains(t, queryValues, "error")
}
