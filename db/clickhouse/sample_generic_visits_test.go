package clickhouse

import (
	"context"
	"testing"
	"time"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/models"
	"github.com/stretchr/testify/require"
)

// assumes that the db is freshly started from scratch
func Test_GenericVisitsTable(t *testing.T) {
	// variables
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxSize := 1
	visitsTable := make(map[string]struct{}, 0)
	visitsTable[models.SampleGenericVisit{}.TableName()] = struct{}{}

	dbCli := generateClickhouseDatabase(t, mainCtx, visitsTable)
	batcher, err := newQueryBatcher[models.SampleGenericVisit](sampleGenericVisistsTableDriver, MaxFlushInterval, maxSize)
	require.NoError(t, err)
	require.Equal(t, batcher.maxSize, maxSize)
	require.Equal(t, batcher.currentLen(), 0)
	require.Equal(t, batcher.BaseQuery(), sampleGenericVisistsTableDriver.baseQuery)
	require.Equal(t, batcher.isFull(), false)

	// drop anything existing in the testing DB
	dbCli.highMu.Lock()
	err = dropAllSampleGenericVisitTable(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	require.NoError(t, err)
	dbCli.highMu.Unlock()

	// test data insert
	visit1 := &models.SampleGenericVisit{
		VisitType:     config.SampleProviders.String(),
		VisitRound:    uint64(0),
		Network:       dbCli.currentNetwork.String(),
		Timestamp:     time.Now(),
		Key:           "0xKEY",
		DurationMs:    60 * 1000,
		ResponseItems: 2,
		Peers:         []string{"Peer1", "Peer2"},
		Error:         "",
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

	// test data retrieval
	dbCli.highMu.Lock()
	visits, err := requestAllSampleGenericVisit(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(visits))
	require.Equal(t, visit1.VisitRound, visits[0].VisitRound)
	require.Equal(t, visit1.VisitType, visits[0].VisitType)
	require.Equal(t, visit1.Timestamp.Day(), visits[0].Timestamp.Day())
	require.Equal(t, visit1.Timestamp.Minute(), visits[0].Timestamp.Minute())
	require.Equal(t, visit1.Timestamp.Second(), visits[0].Timestamp.Second())
	require.Equal(t, visit1.Network, visits[0].Network)
	require.Equal(t, visit1.Key, visits[0].Key)
	require.Equal(t, visit1.ResponseItems, visits[0].ResponseItems)
	require.Equal(t, visit1.Peers, visits[0].Peers)
	require.Equal(t, visit1.Error, visits[0].Error)

	// drop anything existing in the testing DB
	dbCli.highMu.Lock()
	err = dropAllSampleGenericVisitTable(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	require.NoError(t, err)
	dbCli.highMu.Unlock()
}
