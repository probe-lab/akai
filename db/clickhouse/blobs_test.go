package clickhouse

import (
	"context"
	"testing"
	"time"

	"github.com/probe-lab/akai/db/models"
	"github.com/stretchr/testify/require"
)

// assumes that the db is freshly started from scratch
func Test_BlobsTable(t *testing.T) {
	// variables
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxSize := 1
	blobsTable := make(map[string]struct{}, 0)
	blobsTable[models.AgnosticBlob{}.TableName()] = struct{}{}

	dbCli := generateClickhouseDatabase(t, mainCtx, blobsTable)
	batcher, err := newQueryBatcher[models.AgnosticBlob](blobTableDriver, MaxFlushInterval, maxSize)
	require.NoError(t, err)
	require.Equal(t, batcher.maxSize, maxSize)
	require.Equal(t, batcher.currentLen(), 0)
	require.Equal(t, batcher.BaseQuery(), blobTableDriver.baseQuery)
	require.Equal(t, batcher.isFull(), false)

	// drop anything existing in the testing DB
	dbCli.highMu.Lock()
	err = dropAllBlobsTable(mainCtx, dbCli.highLevelClient)
	require.NoError(t, err)
	dbCli.highMu.Unlock()

	// test data insert
	blob := models.AgnosticBlob{
		NetworkID:   uint16(1),
		Timestamp:   time.Now(),
		Hash:        "0xHASH",
		Key:         "0xKEY",
		BlobNumber:  1,
		Rows:        1,
		Columns:     1,
		SampleUntil: time.Now().Add(1 * time.Minute),
	}

	isFull := batcher.addItem(blob)
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
	blobs, err := requestAllBlobs(mainCtx, dbCli.highLevelClient)
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(blobs))
	require.Equal(t, blob.Timestamp.Day(), blobs[0].Timestamp.Day())
	require.Equal(t, blob.Timestamp.Minute(), blobs[0].Timestamp.Minute())
	require.Equal(t, blob.Timestamp.Second(), blobs[0].Timestamp.Second())
	require.Equal(t, blob.Hash, blobs[0].Hash)
	require.Equal(t, blob.Key, blobs[0].Key)
	require.Equal(t, blob.BlobNumber, blobs[0].BlobNumber)
	require.Equal(t, blob.Rows, blobs[0].Rows)
	require.Equal(t, blob.Columns, blobs[0].Columns)
	require.Equal(t, blob.SampleUntil.Day(), blobs[0].SampleUntil.Day())
	require.Equal(t, blob.SampleUntil.Minute(), blobs[0].SampleUntil.Minute())
	require.Equal(t, blob.SampleUntil.Second(), blobs[0].SampleUntil.Second())

	// try adding a second blob
	blob2 := models.AgnosticBlob{
		NetworkID:   uint16(1),
		Timestamp:   time.Now(),
		Hash:        "0xHASH2",
		Key:         "0xKEY2",
		BlobNumber:  2,
		Rows:        1,
		Columns:     1,
		SampleUntil: blob.Timestamp,
	}

	_ = batcher.addItem(blob2)
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
	blobs, err = requestBlobsOnTTL(mainCtx, dbCli.highLevelClient)
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(blobs))

	// get latest blob
	dbCli.highMu.Lock()
	lastBlob, err := requestLatestBlob(mainCtx, dbCli.highLevelClient)
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(blobs))
	require.Equal(t, blob2.Timestamp.Day(), lastBlob.Timestamp.Day())
	require.Equal(t, blob2.Timestamp.Minute(), lastBlob.Timestamp.Minute())
	require.Equal(t, blob2.Timestamp.Second(), lastBlob.Timestamp.Second())
	require.Equal(t, blob2.Hash, lastBlob.Hash)
	require.Equal(t, blob2.Key, lastBlob.Key)
	require.Equal(t, blob2.BlobNumber, lastBlob.BlobNumber)
	require.Equal(t, blob2.Rows, lastBlob.Rows)
	require.Equal(t, blob2.Columns, lastBlob.Columns)
	require.Equal(t, blob2.SampleUntil.Day(), lastBlob.SampleUntil.Day())
	require.Equal(t, blob2.SampleUntil.Minute(), lastBlob.SampleUntil.Minute())
	require.Equal(t, blob2.SampleUntil.Second(), lastBlob.SampleUntil.Second())

	// drop anything existing in the testing DB
	dbCli.highMu.Lock()
	err = dropAllBlobsTable(mainCtx, dbCli.highLevelClient)
	require.NoError(t, err)
	dbCli.highMu.Unlock()
}
