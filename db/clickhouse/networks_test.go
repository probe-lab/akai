package clickhouse

import (
	"context"
	"testing"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/models"
	"github.com/stretchr/testify/require"
)

// assumes that the db is freshly started from scratch
func Test_NetworksTable(t *testing.T) {
	// variables
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxSize := 1
	networkTable := make(map[string]struct{}, 0)
	networkTable[models.Network{}.TableName()] = struct{}{}

	dbCli := generateClickhouseDatabase(t, mainCtx, networkTable)
	batcher, err := newQueryBatcher[models.Network](networkTableDriver, MaxFlushInterval, maxSize)
	require.NoError(t, err)
	require.Equal(t, batcher.maxSize, maxSize)
	require.Equal(t, batcher.currentLen(), 0)
	require.Equal(t, batcher.BaseQuery(), networkTableDriver.baseQuery)
	require.Equal(t, batcher.isFull(), false)

	// drop anything existing in the testing DB
	dbCli.highMu.Lock()
	err = dropValuesNetworksTable(mainCtx, dbCli.highLevelClient)
	require.NoError(t, err)
	dbCli.highMu.Unlock()

	// test data insert
	network := models.Network{
		NetworkID:   uint16(1),
		Protocol:    config.ProtocolAvail,
		NetworkName: config.NetworkNameTuring,
	}

	isFull := batcher.addItem(network)
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

	require.NoError(t, err)

	// test data retrieval
	dbCli.highMu.Lock()
	networks, err := requestNetworks(mainCtx, dbCli.highLevelClient)
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(networks))
	require.Equal(t, network.NetworkID, networks[0].NetworkID)
	require.Equal(t, network.Protocol, networks[0].Protocol)
	require.Equal(t, network.NetworkName, networks[0].NetworkName)

	// drop anything existing in the testing DB
	dbCli.highMu.Lock()
	err = dropValuesNetworksTable(mainCtx, dbCli.highLevelClient)
	require.NoError(t, err)
	dbCli.highMu.Unlock()
}
