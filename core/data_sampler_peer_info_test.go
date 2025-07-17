package core

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/models"
)

func Test_sampleByFindPeerInfo_Success(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := composeDemoDHTNetwork(ctx, t, 3)
	defer stopTestNetwork(t, hosts)

	time.Sleep(2 * time.Second)

	targetHost := hosts[1]
	queryHost := hosts[2]

	item := &models.SamplingItem{
		Key:        targetHost.Host().ID().String(),
		Network:    config.DefaultNetwork.String(),
		SampleType: config.SamplePeerInfo.String(),
	}

	visitRound := 1
	timeout := 10 * time.Second

	visit, err := sampleByFindPeerInfo(ctx, queryHost, item, visitRound, timeout)
	require.NoError(t, err)

	peerInfoVisits := visit.GetGenericPeerInfoVisit()
	require.NotNil(t, peerInfoVisits)
	require.Len(t, peerInfoVisits, 1)

	peerInfoVisit := peerInfoVisits[0]
	require.Equal(t, uint64(visitRound), peerInfoVisit.VisitRound)
	require.Equal(t, item.Network, peerInfoVisit.Network)
	require.Equal(t, item.Key, peerInfoVisit.PeerID)
	require.Greater(t, peerInfoVisit.Duration, time.Duration(0))
	require.NotEmpty(t, peerInfoVisit.AgentVersion)
	require.NotEmpty(t, peerInfoVisit.Protocols)
	require.Empty(t, peerInfoVisit.ProtocolVersion) // the target item doesn't have a protocol version set
	require.NotEmpty(t, peerInfoVisit.MultiAddresses)
	require.Empty(t, peerInfoVisit.Error)
	require.False(t, peerInfoVisit.Timestamp.IsZero())
}

func Test_sampleByFindPeerInfo_InvalidPeerIDs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := composeDemoDHTNetwork(ctx, t, 2)
	defer stopTestNetwork(t, hosts)

	queryHost := hosts[1]

	tests := []struct {
		name              string
		peerID            string
		visitRound        int
		timeout           time.Duration
		expectError       bool
		expectEmptyVisit  bool
		expectVisitFields bool
	}{
		{
			name:              "Invalid peer ID format",
			peerID:            "invalid-peer-id",
			visitRound:        1,
			timeout:           5 * time.Second,
			expectError:       true,
			expectEmptyVisit:  true,
			expectVisitFields: false,
		},
		{
			name:              "Non-existent peer ID",
			peerID:            "12D3KooWEMxBCsCXMWvisYDtDSiWVnVMWfegrbEK4Tvu1wfU1ER7",
			visitRound:        2,
			timeout:           2 * time.Second,
			expectError:       true,
			expectEmptyVisit:  false,
			expectVisitFields: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &models.SamplingItem{
				Key:        tt.peerID,
				Network:    config.DefaultNetwork.String(),
				SampleType: config.SamplePeerInfo.String(),
			}

			visit, err := sampleByFindPeerInfo(ctx, queryHost, item, tt.visitRound, tt.timeout)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.expectEmptyVisit {
				require.Equal(t, models.GeneralVisit{}, visit)
				return
			}

			if tt.expectVisitFields {
				peerInfoVisits := visit.GetGenericPeerInfoVisit()
				require.NotNil(t, peerInfoVisits)
				require.Len(t, peerInfoVisits, 1)

				peerInfoVisit := peerInfoVisits[0]
				require.Equal(t, uint64(tt.visitRound), peerInfoVisit.VisitRound)
				require.Equal(t, item.Network, peerInfoVisit.Network)
				require.Equal(t, item.Key, peerInfoVisit.PeerID)
				require.Greater(t, peerInfoVisit.Duration, time.Duration(0))
				require.Empty(t, peerInfoVisit.AgentVersion)
				require.Empty(t, peerInfoVisit.Protocols)
				require.Empty(t, peerInfoVisit.ProtocolVersion)
				require.Empty(t, peerInfoVisit.MultiAddresses)
				require.NotEmpty(t, peerInfoVisit.Error)
				require.False(t, peerInfoVisit.Timestamp.IsZero())
			}
		})
	}
}

func Test_sampleByFindPeerInfo_Timeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := composeDemoDHTNetwork(ctx, t, 3)
	defer stopTestNetwork(t, hosts)

	queryHost := hosts[2]

	// Use a valid but non-existent peer ID to force a timeout
	// This is a real peer ID format but doesn't exist in our test network
	nonExistentPeerID, err := peer.Decode("12D3KooWDpJ7As7PUYKkTgDNTzVr91nwCy9RHAaD5gqPqP7R7E6e")
	require.NoError(t, err)

	item := &models.SamplingItem{
		Key:        nonExistentPeerID.String(),
		Network:    config.DefaultNetwork.String(),
		SampleType: config.SamplePeerInfo.String(),
	}

	visitRound := 1
	timeout := 1 * time.Microsecond

	start := time.Now()
	visit, err := sampleByFindPeerInfo(ctx, queryHost, item, visitRound, timeout)
	elapsed := time.Since(start)

	// The test should complete quickly due to timeout, even if there's an error
	t.Logf("Test elapsed time: %v, timeout: %v", elapsed, timeout)
	require.True(t, elapsed <= timeout*100, "Operation should timeout within reasonable time of the specified timeout, got %v", elapsed)

	if err != nil {
		// If there's an error, the operation should still have timed out quickly
		t.Logf("Error received: %v", err)
		return
	}

	peerInfoVisits := visit.GetGenericPeerInfoVisit()
	require.NotNil(t, peerInfoVisits)
	require.Len(t, peerInfoVisits, 1)

	peerInfoVisit := peerInfoVisits[0]
	require.Equal(t, uint64(visitRound), peerInfoVisit.VisitRound)
	require.Equal(t, item.Network, peerInfoVisit.Network)
	require.Equal(t, item.Key, peerInfoVisit.PeerID)
	require.Greater(t, peerInfoVisit.Duration, time.Duration(0))

	// Check that the operation timed out - either with a timeout error or completed within reasonable time of the timeout
	t.Logf("Test elapsed time: %v, timeout: %v, duration: %v", elapsed, timeout, peerInfoVisit.Duration)
	require.True(t, elapsed <= timeout*2, "Operation should timeout within reasonable time of the specified timeout, got %v", elapsed)
	require.NotEmpty(t, peerInfoVisit.Error)
	require.False(t, peerInfoVisit.Timestamp.IsZero())
}
