package clickhouse

import (
	"context"
	"testing"
	"time"

	"github.com/probe-lab/akai/db/models"
	"github.com/stretchr/testify/require"
)

// assumes that the db is freshly started from scratch
func Test_BlocksTable(t *testing.T) {
	// variables
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxSize := 1
	blocksTable := make(map[string]struct{}, 0)
	blocksTable[models.Block{}.TableName()] = struct{}{}

	dbCli := generateClickhouseDatabase(t, mainCtx, blocksTable)
	batcher, err := newQueryBatcher[models.Block](blockTableDriver, MaxFlushInterval, maxSize)
	require.NoError(t, err)
	require.Equal(t, batcher.maxSize, maxSize)
	require.Equal(t, batcher.currentLen(), 0)
	require.Equal(t, batcher.BaseQuery(), blockTableDriver.baseQuery)
	require.Equal(t, batcher.isFull(), false)

	// drop anything existing in the testing DB
	dbCli.highMu.Lock()
	err = dropAllBlocksTable(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	require.NoError(t, err)
	dbCli.highMu.Unlock()

	// test data insert
	block := &models.Block{
		Network:     dbCli.currentNetwork.String(),
		Timestamp:   time.Now(),
		Hash:        "0xHASH",
		Key:         "0xKEY",
		Number:      1,
		DASRows:     1,
		DASColumns:  1,
		SampleUntil: time.Now().Add(1 * time.Minute),
	}

	isFull := batcher.addItem(block)
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
	blocks, err := requestAllBlocks(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(blocks))
	require.Equal(t, block.Timestamp.Day(), blocks[0].Timestamp.Day())
	require.Equal(t, block.Timestamp.Minute(), blocks[0].Timestamp.Minute())
	require.Equal(t, block.Timestamp.Second(), blocks[0].Timestamp.Second())
	require.Equal(t, block.Hash, blocks[0].Hash)
	require.Equal(t, block.Key, blocks[0].Key)
	require.Equal(t, block.Number, blocks[0].Number)
	require.Equal(t, block.DASRows, blocks[0].DASRows)
	require.Equal(t, block.DASColumns, blocks[0].DASColumns)
	require.Equal(t, block.SampleUntil.Day(), blocks[0].SampleUntil.Day())
	require.Equal(t, block.SampleUntil.Minute(), blocks[0].SampleUntil.Minute())
	require.Equal(t, block.SampleUntil.Second(), blocks[0].SampleUntil.Second())

	// try adding a second block
	block2 := &models.Block{
		Network:     dbCli.currentNetwork.String(),
		Timestamp:   time.Now(),
		Hash:        "0xHASH2",
		Key:         "0xKEY2",
		Number:      2,
		DASRows:     1,
		DASColumns:  1,
		SampleUntil: block.Timestamp,
	}

	_ = batcher.addItem(block2)
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
	blocks, err = requestBlocksOnTTL(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(blocks))

	// get latest block
	dbCli.highMu.Lock()
	lastBlock, err := requestLastBlock(mainCtx, dbCli.highLevelClient)
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(blocks))
	require.Equal(t, block2.Timestamp.Day(), lastBlock.Timestamp.Day())
	require.Equal(t, block2.Timestamp.Minute(), lastBlock.Timestamp.Minute())
	require.Equal(t, block2.Timestamp.Second(), lastBlock.Timestamp.Second())
	require.Equal(t, block2.Hash, lastBlock.Hash)
	require.Equal(t, block2.Key, lastBlock.Key)
	require.Equal(t, block2.Number, lastBlock.Number)
	require.Equal(t, block2.DASRows, lastBlock.DASRows)
	require.Equal(t, block2.DASColumns, lastBlock.DASColumns)
	require.Equal(t, block2.SampleUntil.Day(), lastBlock.SampleUntil.Day())
	require.Equal(t, block2.SampleUntil.Minute(), lastBlock.SampleUntil.Minute())
	require.Equal(t, block2.SampleUntil.Second(), lastBlock.SampleUntil.Second())

	// drop anything existing in the testing DB
	dbCli.highMu.Lock()
	err = dropAllBlocksTable(mainCtx, dbCli.highLevelClient, dbCli.currentNetwork.String())
	require.NoError(t, err)
	dbCli.highMu.Unlock()
}
