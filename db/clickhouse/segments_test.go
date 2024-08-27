package clickhouse

import (
	"context"
	"testing"
	"time"

	"github.com/probe-lab/akai/db/models"
	"github.com/stretchr/testify/require"
)

// assumes that the db is freshly started from scratch
func Test_SegmentsTable(t *testing.T) {
	// variables
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxSize := 1
	segmentsTable := make(map[string]struct{}, 0)
	segmentsTable[models.AgnosticSegment{}.TableName()] = struct{}{}

	dbCli := generateClickhouseDatabase(t, mainCtx, segmentsTable)
	batcher, err := newQueryBatcher[models.AgnosticSegment](segmentTableDriver, MaxFlushInterval, maxSize)
	require.NoError(t, err)
	require.Equal(t, batcher.maxSize, maxSize)
	require.Equal(t, batcher.currentLen(), 0)
	require.Equal(t, batcher.BaseQuery(), segmentTableDriver.baseQuery)
	require.Equal(t, batcher.isFull(), false)

	// drop anything existing in the testing DB
	dbCli.highMu.Lock()
	err = dropAllSegmentsTable(mainCtx, dbCli.highLevelClient)
	require.NoError(t, err)
	dbCli.highMu.Unlock()

	// test data insert
	segment1 := models.AgnosticSegment{
		Timestamp:   time.Now(),
		Key:         "0xKEY",
		BlobNumber:  1,
		Row:         1,
		Column:      1,
		SampleUntil: time.Now().Add(1 * time.Minute),
	}

	isFull := batcher.addItem(segment1)
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
	segments, err := requestAllSegments(mainCtx, dbCli.highLevelClient)
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(segments))
	require.Equal(t, segment1.Timestamp.Day(), segments[0].Timestamp.Day())
	require.Equal(t, segment1.Timestamp.Minute(), segments[0].Timestamp.Minute())
	require.Equal(t, segment1.Timestamp.Second(), segments[0].Timestamp.Second())
	require.Equal(t, segment1.Key, segments[0].Key)
	require.Equal(t, segment1.BlobNumber, segments[0].BlobNumber)
	require.Equal(t, segment1.SampleUntil.Day(), segments[0].SampleUntil.Day())
	require.Equal(t, segment1.SampleUntil.Minute(), segments[0].SampleUntil.Minute())
	require.Equal(t, segment1.SampleUntil.Second(), segments[0].SampleUntil.Second())

	// try adding a second blob
	segment2 := models.AgnosticSegment{
		Timestamp:   time.Now(),
		Key:         "0xKEY2",
		BlobNumber:  2,
		Row:         1,
		Column:      1,
		SampleUntil: segment1.Timestamp,
	}

	_ = batcher.addItem(segment2)
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
	segments, err = requestSegmentsOnTTL(mainCtx, dbCli.highLevelClient)
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(segments))

	// drop anything existing in the testing DB
	dbCli.highMu.Lock()
	err = dropAllSegmentsTable(mainCtx, dbCli.highLevelClient)
	require.NoError(t, err)
	dbCli.highMu.Unlock()
}
