package clickhouse

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/models"
	"github.com/stretchr/testify/require"
)

// assumes that the db is freshly started from scratch
func Test_SamplingItemTable(t *testing.T) {
	// variables
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxSize := 1
	segmentsTable := make(map[string]struct{}, 0)
	segmentsTable[models.SamplingItem{}.TableName()] = struct{}{}

	dbCli := generateClickhouseDatabase(t, mainCtx, segmentsTable)
	batcher, err := newQueryBatcher[models.SamplingItem](samplingItemTableDriver, MaxFlushInterval, maxSize)
	require.NoError(t, err)
	require.Equal(t, batcher.maxSize, maxSize)
	require.Equal(t, batcher.currentLen(), 0)
	require.Equal(t, batcher.BaseQuery(), samplingItemTableDriver.baseQuery)
	require.Equal(t, batcher.isFull(), false)

	// drop anything existing in the testing DB
	dbCli.highMu.Lock()
	err = dropAllItemsTable(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	require.NoError(t, err)
	dbCli.highMu.Unlock()

	// test data insert
	item1 := &models.SamplingItem{
		Timestamp:   time.Now(),
		Network:     dbCli.currentNetwork.String(),
		ItemType:    config.AvailDASCellItemType.String(),
		SampleType:  config.SampleValue.String(),
		Key:         "0xKEY",
		Hash:        "0xHASH",
		BlockLink:   1,
		DASRow:      1,
		DASColumn:   1,
		Metadata:    "{Metadata: yes?}",
		Traceable:   true,
		SampleUntil: time.Now().Add(1 * time.Minute),
	}

	isFull := batcher.addItem(item1)
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
	items, err := requestAllSAmplingItems(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	fmt.Println(items)
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(items))
	require.Equal(t, item1.Timestamp.Day(), items[0].Timestamp.Day())
	require.Equal(t, item1.Timestamp.Minute(), items[0].Timestamp.Minute())
	require.Equal(t, item1.Timestamp.Second(), items[0].Timestamp.Second())
	require.Equal(t, item1.Key, items[0].Key)
	require.Equal(t, item1.BlockLink, items[0].BlockLink)
	require.Equal(t, item1.SampleUntil.Day(), items[0].SampleUntil.Day())
	require.Equal(t, item1.SampleUntil.Minute(), items[0].SampleUntil.Minute())
	require.Equal(t, item1.SampleUntil.Second(), items[0].SampleUntil.Second())

	// try adding a second blob{
	item2 := &models.SamplingItem{
		Timestamp:   time.Now(),
		Network:     dbCli.currentNetwork.String(),
		ItemType:    config.AvailDASCellItemType.String(),
		SampleType:  config.SampleValue.String(),
		Key:         "0xKEY2",
		Hash:        "0xHASH2",
		BlockLink:   2,
		DASRow:      1,
		DASColumn:   1,
		Metadata:    "{Metadata: yes?}",
		Traceable:   true,
		SampleUntil: item1.Timestamp,
	}

	_ = batcher.addItem(item2)
	inputable, itemsNum = batcher.getPersistable()
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
	items, err = requestItemsOnTTL(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(items))

	// drop anything existing in the testing DB
	dbCli.highMu.Lock()
	err = dropAllItemsTable(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	require.NoError(t, err)
	dbCli.highMu.Unlock()
}
