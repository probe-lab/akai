package clickhouse

import (
	"context"
	"testing"
	"time"

	"github.com/probe-lab/akai/db/models"
	"github.com/stretchr/testify/require"
)

// assumes that the db is freshly started from scratch
func Test_VisitsTable(t *testing.T) {
	// variables
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxSize := 1
	visitsTable := make(map[string]struct{}, 0)
	visitsTable[models.AgnosticVisit{}.TableName()] = struct{}{}

	dbCli := generateClickhouseDatabase(t, mainCtx, visitsTable)
	batcher, err := newQueryBatcher[models.AgnosticVisit](visistsTableDriver, MaxFlushInterval, maxSize)
	require.NoError(t, err)
	require.Equal(t, batcher.maxSize, maxSize)
	require.Equal(t, batcher.currentLen(), 0)
	require.Equal(t, batcher.BaseQuery(), visistsTableDriver.baseQuery)
	require.Equal(t, batcher.isFull(), false)

	// drop anything existing in the testing DB
	dbCli.highMu.Lock()
	err = dropAllVisitsTable(mainCtx, dbCli.highLevelClient)
	require.NoError(t, err)
	dbCli.highMu.Unlock()

	// test data insert
	visit1 := models.AgnosticVisit{
		Timestamp:     time.Now(),
		Key:           "0xKEY",
		BlobNumber:    1,
		Row:           1,
		Column:        1,
		DurationMs:    60 * 1000,
		IsRetrievable: true,
		Providers:     1,
		Bytes:         128,
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
	visits, err := requestAllVisits(mainCtx, dbCli.highLevelClient)
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(visits))
	require.Equal(t, visit1.Timestamp.Day(), visits[0].Timestamp.Day())
	require.Equal(t, visit1.Timestamp.Minute(), visits[0].Timestamp.Minute())
	require.Equal(t, visit1.Timestamp.Second(), visits[0].Timestamp.Second())
	require.Equal(t, visit1.Key, visits[0].Key)
	require.Equal(t, visit1.BlobNumber, visits[0].BlobNumber)
	require.Equal(t, visit1.IsRetrievable, visits[0].IsRetrievable)
	require.Equal(t, visit1.Providers, visits[0].Providers)
	require.Equal(t, visit1.Bytes, visits[0].Bytes)
	require.Equal(t, visit1.Error, visits[0].Error)

	// drop anything existing in the testing DB
	dbCli.highMu.Lock()
	err = dropAllVisitsTable(mainCtx, dbCli.highLevelClient)
	require.NoError(t, err)
	dbCli.highMu.Unlock()
}
