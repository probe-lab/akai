package clickhouse

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/probe-lab/akai/db/models"
)

func Test_IPNSRecordVisitsTable(t *testing.T) {
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxSize := 1
	visitsTable := make(map[string]struct{}, 0)
	visitsTable[models.IPNSRecordVisit{}.TableName()] = struct{}{}

	dbCli := generateClickhouseDatabase(t, mainCtx, visitsTable)
	batcher, err := newQueryBatcher[models.IPNSRecordVisit](ipnsRecordVisitsTableDriver, MaxFlushInterval, maxSize)
	require.NoError(t, err)
	require.Equal(t, batcher.maxSize, maxSize)
	require.Equal(t, batcher.currentLen(), 0)
	require.Equal(t, batcher.BaseQuery(), ipnsRecordVisitsTableDriver.baseQuery)
	require.Equal(t, batcher.isFull(), false)

	dbCli.highMu.Lock()
	err = dropAllIPNSRecordVisitsTable(mainCtx, dbCli.highLevelClient)
	require.NoError(t, err)
	dbCli.highMu.Unlock()

	visit1 := &models.IPNSRecordVisit{
		VisitRound:    uint64(1),
		Timestamp:     time.Now(),
		Record:        "k51qzi5uqu5dgox2z23r6e99oqency055a6xt92xzmyvpz8mwx2xx9y98f4yfy",
		RecordType:    "ipns",
		Quorum:        uint8(5),
		SeqNumber:     uint32(10),
		TTL:           30 * time.Second,
		IsValid:       true,
		IsRetrievable: true,
		Result:        "success",
		Duration:      100 * time.Millisecond,
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

	dbCli.highMu.Lock()
	visits, err := requestAllIPNSRecordVisits(mainCtx, dbCli.highLevelClient)
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(visits))
	require.Equal(t, visit1.VisitRound, visits[0].VisitRound)
	require.Equal(t, visit1.Timestamp.Day(), visits[0].Timestamp.Day())
	require.Equal(t, visit1.Timestamp.Minute(), visits[0].Timestamp.Minute())
	require.Equal(t, visit1.Timestamp.Second(), visits[0].Timestamp.Second())
	require.Equal(t, visit1.Record, visits[0].Record)
	require.Equal(t, visit1.RecordType, visits[0].RecordType)
	require.Equal(t, visit1.Quorum, visits[0].Quorum)
	require.Equal(t, visit1.SeqNumber, visits[0].SeqNumber)
	require.Equal(t, visit1.TTL, visits[0].TTL)
	require.Equal(t, visit1.IsValid, visits[0].IsValid)
	require.Equal(t, visit1.IsRetrievable, visits[0].IsRetrievable)
	require.Equal(t, visit1.Result, visits[0].Result)
	require.Equal(t, visit1.Duration, visits[0].Duration)
	require.Equal(t, visit1.Error, visits[0].Error)

	dbCli.highMu.Lock()
	err = dropAllIPNSRecordVisitsTable(mainCtx, dbCli.highLevelClient)
	require.NoError(t, err)
	dbCli.highMu.Unlock()
}

func Test_IPNSRecordVisitsTable_WithError(t *testing.T) {
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxSize := 1
	visitsTable := make(map[string]struct{}, 0)
	visitsTable[models.IPNSRecordVisit{}.TableName()] = struct{}{}

	dbCli := generateClickhouseDatabase(t, mainCtx, visitsTable)
	batcher, err := newQueryBatcher[models.IPNSRecordVisit](ipnsRecordVisitsTableDriver, MaxFlushInterval, maxSize)
	require.NoError(t, err)

	dbCli.highMu.Lock()
	err = dropAllIPNSRecordVisitsTable(mainCtx, dbCli.highLevelClient)
	require.NoError(t, err)
	dbCli.highMu.Unlock()

	visit1 := &models.IPNSRecordVisit{
		VisitRound:    uint64(1),
		Timestamp:     time.Now(),
		Record:        "k51qzi5uqu5dgox2z23r6e99oqency055a6xt92xzmyvpz8mwx2xx9y98f4yfy",
		RecordType:    "ipns",
		Quorum:        uint8(3),
		SeqNumber:     uint32(0),
		TTL:           0 * time.Second,
		IsValid:       false,
		IsRetrievable: false,
		Result:        "failed",
		Duration:      2 * time.Second,
		Error:         "record not found",
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
	visits, err := requestAllIPNSRecordVisits(mainCtx, dbCli.highLevelClient)
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 1, len(visits))
	require.Equal(t, visit1.VisitRound, visits[0].VisitRound)
	require.Equal(t, visit1.Record, visits[0].Record)
	require.Equal(t, visit1.RecordType, visits[0].RecordType)
	require.Equal(t, visit1.Quorum, visits[0].Quorum)
	require.Equal(t, visit1.SeqNumber, visits[0].SeqNumber)
	require.Equal(t, visit1.TTL, visits[0].TTL)
	require.Equal(t, visit1.IsValid, visits[0].IsValid)
	require.Equal(t, visit1.IsRetrievable, visits[0].IsRetrievable)
	require.Equal(t, visit1.Result, visits[0].Result)
	require.Equal(t, visit1.Duration, visits[0].Duration)
	require.Equal(t, visit1.Error, visits[0].Error)

	dbCli.highMu.Lock()
	err = dropAllIPNSRecordVisitsTable(mainCtx, dbCli.highLevelClient)
	require.NoError(t, err)
	dbCli.highMu.Unlock()
}

func Test_IPNSRecordVisitsTable_MultipleBatches(t *testing.T) {
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxSize := 2
	visitsTable := make(map[string]struct{}, 0)
	visitsTable[models.IPNSRecordVisit{}.TableName()] = struct{}{}

	dbCli := generateClickhouseDatabase(t, mainCtx, visitsTable)
	batcher, err := newQueryBatcher[models.IPNSRecordVisit](ipnsRecordVisitsTableDriver, MaxFlushInterval, maxSize)
	require.NoError(t, err)

	dbCli.highMu.Lock()
	err = dropAllIPNSRecordVisitsTable(mainCtx, dbCli.highLevelClient)
	require.NoError(t, err)
	dbCli.highMu.Unlock()

	visit1 := &models.IPNSRecordVisit{
		VisitRound:    uint64(1),
		Timestamp:     time.Now(),
		Record:        "k51qzi5uqu5dgox2z23r6e99oqency055a6xt92xzmyvpz8mwx2xx9y98f4yfy",
		RecordType:    "ipns",
		Quorum:        uint8(5),
		SeqNumber:     uint32(10),
		TTL:           30 * time.Second,
		IsValid:       true,
		IsRetrievable: true,
		Result:        "success",
		Duration:      100 * time.Millisecond,
		Error:         "",
	}

	visit2 := &models.IPNSRecordVisit{
		VisitRound:    uint64(2),
		Timestamp:     time.Now().Add(time.Minute),
		Record:        "k51qzi5uqu5dh3k2z98r6e99oqency055a6xt92xzmyvpz8mwx2xx9y98f4abc",
		RecordType:    "ipns",
		Quorum:        uint8(7),
		SeqNumber:     uint32(20),
		TTL:           60 * time.Second,
		IsValid:       true,
		IsRetrievable: true,
		Result:        "success",
		Duration:      150 * time.Millisecond,
		Error:         "",
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
	visits, err := requestAllIPNSRecordVisits(mainCtx, dbCli.highLevelClient)
	dbCli.highMu.Unlock()
	require.NoError(t, err)
	require.Equal(t, 2, len(visits))

	dbCli.highMu.Lock()
	err = dropAllIPNSRecordVisitsTable(mainCtx, dbCli.highLevelClient)
	require.NoError(t, err)
	dbCli.highMu.Unlock()
}

func Test_insertIPNSRecordVisitQueryBase(t *testing.T) {
	query := insertIPNSRecordVisitQueryBase()
	require.Contains(t, query, "INSERT INTO %s")
	require.Contains(t, query, "visit_round")
	require.Contains(t, query, "timestamp")
	require.Contains(t, query, "record")
	require.Contains(t, query, "record_type")
	require.Contains(t, query, "quorum")
	require.Contains(t, query, "seq_number")
	require.Contains(t, query, "ttl_s")
	require.Contains(t, query, "is_valid")
	require.Contains(t, query, "is_retrievable")
	require.Contains(t, query, "result")
	require.Contains(t, query, "duration_ms")
	require.Contains(t, query, "error")
	require.Contains(t, query, "VALUES")
}

func Test_convertIPNSRecordVisitToInput(t *testing.T) {
	now := time.Now()
	visits := []*models.IPNSRecordVisit{
		{
			VisitRound:    uint64(1),
			Timestamp:     now,
			Record:        "k51qzi5uqu5dgox2z23r6e99oqency055a6xt92xzmyvpz8mwx2xx9y98f4yfy",
			RecordType:    "ipns",
			Quorum:        uint8(5),
			SeqNumber:     uint32(10),
			TTL:           30 * time.Second,
			IsValid:       true,
			IsRetrievable: true,
			Result:        "success",
			Duration:      100 * time.Millisecond,
			Error:         "",
		},
		{
			VisitRound:    uint64(2),
			Timestamp:     now.Add(time.Hour),
			Record:        "k51qzi5uqu5dh3k2z98r6e99oqency055a6xt92xzmyvpz8mwx2xx9y98f4abc",
			RecordType:    "ipns",
			Quorum:        uint8(3),
			SeqNumber:     uint32(20),
			TTL:           60 * time.Second,
			IsValid:       false,
			IsRetrievable: false,
			Result:        "failed",
			Duration:      200 * time.Millisecond,
			Error:         "timeout",
		},
	}

	input := convertIPNSRecordVisitToInput(visits)
	require.Len(t, input, 12)

	expectedFields := []string{
		"visit_round", "timestamp", "record", "record_type",
		"quorum", "seq_number", "ttl_s", "is_valid",
		"is_retrievable", "result", "duration_ms", "error",
	}

	for i, field := range expectedFields {
		require.Equal(t, field, input[i].Name)
	}
}

func Test_ipnsRecordVisitsTableDriver(t *testing.T) {
	driver := ipnsRecordVisitsTableDriver

	require.Equal(t, models.IPNSRecordVisitTableName, driver.tableName)
	require.Equal(t, "insert_new_ipns_record_visit", driver.tag)
	require.NotNil(t, driver.baseQuery)
	require.NotNil(t, driver.inputConverter)

	query := driver.baseQuery
	require.Contains(t, query, "INSERT INTO %s")
	require.Contains(t, query, "visit_round")
	require.Contains(t, query, "VALUES")
}
