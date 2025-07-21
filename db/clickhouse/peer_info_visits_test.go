package clickhouse

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"

	"github.com/probe-lab/akai/db/models"
)

func mustParseMultiaddrs(addrs []string) []multiaddr.Multiaddr {
	result := make([]multiaddr.Multiaddr, len(addrs))
	for i, addr := range addrs {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			panic(err)
		}
		result[i] = ma
	}
	return result
}

func Test_PeerInfoVisitsTable(t *testing.T) {
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxSize := 1
	visitsTable := make(map[string]struct{}, 0)
	visitsTable[models.PeerInfoVisit{}.TableName()] = struct{}{}

	dbCli := generateClickhouseDatabase(t, mainCtx, visitsTable)
	batcher, err := newQueryBatcher[models.PeerInfoVisit](peerInfoVisitsTableDriver, MaxFlushInterval, maxSize)
	require.NoError(t, err)
	require.Equal(t, batcher.maxSize, maxSize)
	require.Equal(t, batcher.currentLen(), 0)
	require.Equal(t, batcher.BaseQuery(), peerInfoVisitsTableDriver.baseQuery)
	require.Equal(t, batcher.isFull(), false)

	dbCli.highMu.Lock()
	err = dropAllPeerInfoVisitsTable(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	require.NoError(t, err)
	dbCli.highMu.Unlock()

	visit1 := &models.PeerInfoVisit{
		VisitRound:      uint64(1),
		Timestamp:       time.Now(),
		Network:         dbCli.currentNetwork.String(),
		PeerID:          "12D3KooWABC123",
		Duration:        5 * time.Second,
		AgentVersion:    "go-ipfs/0.13.0",
		Protocols:       []protocol.ID{"/ipfs/kad/1.0.0", "/ipfs/id/1.0.0"},
		ProtocolVersion: "ipfs/0.1.0",
		MultiAddresses:  mustParseMultiaddrs([]string{"/ip4/127.0.0.1/tcp/4001", "/ip6/::1/tcp/4001"}),
		Error:           "",
	}

	isFull := batcher.addItem(visit1)
	require.Equal(t, isFull, true)

	inputable, itemsNum := batcher.getPersistable()
	require.Equal(t, itemsNum, maxSize)

	err = dbCli.persistBatch(
		mainCtx,
		batcher.Tag(),
		batcher.BaseQuery(),
		batcher.TableName(),
		inputable,
	)
	batcher.resetBatcher()
	require.NoError(t, err)

	dbCli.highMu.Lock()
	visits, err := requestAllPeerInfoVisits(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(visits))
	require.Equal(t, visit1.VisitRound, visits[0].VisitRound)
	require.Equal(t, visit1.Timestamp.Day(), visits[0].Timestamp.Day())
	require.Equal(t, visit1.Timestamp.Minute(), visits[0].Timestamp.Minute())
	require.Equal(t, visit1.Timestamp.Second(), visits[0].Timestamp.Second())
	require.Equal(t, visit1.Network, visits[0].Network)
	require.Equal(t, visit1.PeerID, visits[0].PeerID)
	require.Equal(t, visit1.Duration, visits[0].Duration)
	require.Equal(t, visit1.AgentVersion, visits[0].AgentVersion)
	require.Equal(t, visit1.Protocols, visits[0].Protocols)
	require.Equal(t, visit1.ProtocolVersion, visits[0].ProtocolVersion)
	require.Equal(t, visit1.MultiAddresses, visits[0].MultiAddresses)
	require.Equal(t, visit1.Error, visits[0].Error)

	dbCli.highMu.Lock()
	err = dropAllPeerInfoVisitsTable(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	require.NoError(t, err)
	dbCli.highMu.Unlock()
}

func Test_PeerInfoVisitsTable_WithError(t *testing.T) {
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxSize := 1
	visitsTable := make(map[string]struct{}, 0)
	visitsTable[models.PeerInfoVisit{}.TableName()] = struct{}{}

	dbCli := generateClickhouseDatabase(t, mainCtx, visitsTable)
	batcher, err := newQueryBatcher[models.PeerInfoVisit](peerInfoVisitsTableDriver, MaxFlushInterval, maxSize)
	require.NoError(t, err)

	dbCli.highMu.Lock()
	err = dropAllPeerInfoVisitsTable(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	require.NoError(t, err)
	dbCli.highMu.Unlock()

	visit1 := &models.PeerInfoVisit{
		VisitRound:      uint64(1),
		Timestamp:       time.Now(),
		Network:         dbCli.currentNetwork.String(),
		PeerID:          "12D3KooWNonExistentPeer",
		Duration:        2 * time.Second,
		AgentVersion:    "",
		Protocols:       []protocol.ID{},
		ProtocolVersion: "",
		MultiAddresses:  mustParseMultiaddrs([]string{}),
		Error:           "peer not found",
	}

	isFull := batcher.addItem(visit1)
	require.Equal(t, isFull, true)

	inputable, itemsNum := batcher.getPersistable()
	require.Equal(t, itemsNum, maxSize)

	err = dbCli.persistBatch(
		mainCtx,
		batcher.Tag(),
		batcher.BaseQuery(),
		batcher.TableName(),
		inputable,
	)
	batcher.resetBatcher()
	require.NoError(t, err)

	dbCli.highMu.Lock()
	visits, err := requestAllPeerInfoVisits(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(visits))
	require.Equal(t, visit1.VisitRound, visits[0].VisitRound)
	require.Equal(t, visit1.Network, visits[0].Network)
	require.Equal(t, visit1.PeerID, visits[0].PeerID)
	require.Equal(t, visit1.Duration, visits[0].Duration)
	require.Equal(t, visit1.AgentVersion, visits[0].AgentVersion)
	require.Equal(t, visit1.Protocols, visits[0].Protocols)
	require.Equal(t, visit1.ProtocolVersion, visits[0].ProtocolVersion)
	require.Equal(t, visit1.MultiAddresses, visits[0].MultiAddresses)
	require.Equal(t, visit1.Error, visits[0].Error)

	dbCli.highMu.Lock()
	err = dropAllPeerInfoVisitsTable(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	require.NoError(t, err)
	dbCli.highMu.Unlock()
}

func Test_PeerInfoVisitsTable_MultipleBatches(t *testing.T) {
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxSize := 2
	visitsTable := make(map[string]struct{}, 0)
	visitsTable[models.PeerInfoVisit{}.TableName()] = struct{}{}

	dbCli := generateClickhouseDatabase(t, mainCtx, visitsTable)
	batcher, err := newQueryBatcher[models.PeerInfoVisit](peerInfoVisitsTableDriver, MaxFlushInterval, maxSize)
	require.NoError(t, err)

	dbCli.highMu.Lock()
	err = dropAllPeerInfoVisitsTable(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	require.NoError(t, err)
	dbCli.highMu.Unlock()

	visit1 := &models.PeerInfoVisit{
		VisitRound:      uint64(1),
		Timestamp:       time.Now(),
		Network:         dbCli.currentNetwork.String(),
		PeerID:          "12D3KooWABC123",
		Duration:        5 * time.Second,
		AgentVersion:    "go-ipfs/0.13.0",
		Protocols:       []protocol.ID{"/ipfs/kad/1.0.0"},
		ProtocolVersion: "ipfs/0.1.0",
		MultiAddresses:  mustParseMultiaddrs([]string{"/ip4/127.0.0.1/tcp/4001"}),
		Error:           "",
	}

	visit2 := &models.PeerInfoVisit{
		VisitRound:      uint64(2),
		Timestamp:       time.Now().Add(time.Minute),
		Network:         dbCli.currentNetwork.String(),
		PeerID:          "12D3KooWDEF456",
		Duration:        3 * time.Second,
		AgentVersion:    "go-ipfs/0.14.0",
		Protocols:       []protocol.ID{"/ipfs/kad/1.0.0", "/ipfs/bitswap/1.2.0"},
		ProtocolVersion: "ipfs/0.1.0",
		MultiAddresses:  mustParseMultiaddrs([]string{"/ip4/192.168.1.1/tcp/4001"}),
		Error:           "",
	}

	isFull := batcher.addItem(visit1)
	require.Equal(t, isFull, false)

	isFull = batcher.addItem(visit2)
	require.Equal(t, isFull, true)

	inputable, itemsNum := batcher.getPersistable()
	require.Equal(t, itemsNum, maxSize)

	err = dbCli.persistBatch(
		mainCtx,
		batcher.Tag(),
		batcher.BaseQuery(),
		batcher.TableName(),
		inputable,
	)
	batcher.resetBatcher()
	require.NoError(t, err)

	dbCli.highMu.Lock()
	visits, err := requestAllPeerInfoVisits(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 2, len(visits))

	dbCli.highMu.Lock()
	err = dropAllPeerInfoVisitsTable(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	require.NoError(t, err)
	dbCli.highMu.Unlock()
}

func Test_PersistNewPeerInfoVisit(t *testing.T) {
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	visitsTable := make(map[string]struct{}, 0)
	visitsTable[models.PeerInfoVisit{}.TableName()] = struct{}{}

	dbCli := generateClickhouseDatabase(t, mainCtx, visitsTable)

	dbCli.highMu.Lock()
	err := dropAllPeerInfoVisitsTable(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	require.NoError(t, err)
	dbCli.highMu.Unlock()

	visit := &models.PeerInfoVisit{
		VisitRound:      uint64(1),
		Timestamp:       time.Now(),
		Network:         dbCli.currentNetwork.String(),
		PeerID:          "12D3KooWABC123",
		Duration:        5 * time.Second,
		AgentVersion:    "go-ipfs/0.13.0",
		Protocols:       []protocol.ID{"/ipfs/kad/1.0.0"},
		ProtocolVersion: "ipfs/0.1.0",
		MultiAddresses:  mustParseMultiaddrs([]string{"/ip4/127.0.0.1/tcp/4001"}),
		Error:           "",
	}

	err = dbCli.PersistNewPeerInfoVisit(mainCtx, visit)
	require.NoError(t, err)

	err = dbCli.flushBatcherIfNeeded(mainCtx, dbCli.qBatchers.peerInfoVisits)
	require.NoError(t, err)

	dbCli.highMu.Lock()
	visits, err := requestAllPeerInfoVisits(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(visits))
	require.Equal(t, visit.VisitRound, visits[0].VisitRound)
	require.Equal(t, visit.PeerID, visits[0].PeerID)
	require.Equal(t, visit.Network, visits[0].Network)

	dbCli.highMu.Lock()
	err = dropAllPeerInfoVisitsTable(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	require.NoError(t, err)
	dbCli.highMu.Unlock()
}
