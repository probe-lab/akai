package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
)

var sampleValueVisitTableDriver = tableDriver[models.SampleValueVisit]{
	tableName:      models.SampleValueVisitTableName,
	tag:            "insert_new_sample_closest_visit",
	baseQuery:      insertSampleValueVisitQueryBase(),
	inputConverter: convertSampleValueVisitToInput,
}

func insertSampleValueVisitQueryBase() string {
	query := `
	INSERT INTO %s (
		visit_type,
		visit_round,
		network,
		timestamp,
		key,
		block_number,
		das_row,
		das_column,
		duration_ms,
		is_retrievable,
		bytes,
		error)
	VALUES`
	return query
}

func convertSampleValueVisitToInput(visits []*models.SampleValueVisit) proto.Input {
	var (
		visitTypes   proto.ColStr
		visitRounds  proto.ColUInt64
		timestamps   proto.ColDateTime
		keys         proto.ColStr
		blockNumbers proto.ColUInt64
		rows         proto.ColUInt32
		columns      proto.ColUInt32
		durations    proto.ColInt64
		retrievables proto.ColBool
		bytes        proto.ColInt32
		errors       proto.ColStr
	)

	for _, visit := range visits {
		visitTypes.Append(visit.VisitType)
		visitRounds.Append(visit.VisitRound)
		timestamps.Append(visit.Timestamp)
		keys.Append(visit.Key)
		blockNumbers.Append(visit.BlockNumber)
		rows.Append(visit.DASRow)
		columns.Append(visit.DASColumn)
		durations.Append(visit.DurationMs)
		retrievables.Append(visit.IsRetrievable)
		bytes.Append(visit.Bytes)
		errors.Append(visit.Error)
	}

	return proto.Input{
		{Name: "visit_type", Data: visitTypes},
		{Name: "visit_round", Data: visitRounds},
		{Name: "timestamp", Data: timestamps},
		{Name: "key", Data: keys},
		{Name: "block_number", Data: blockNumbers},
		{Name: "row", Data: rows},
		{Name: "column", Data: columns},
		{Name: "duration_ms", Data: durations},
		{Name: "is_retrievable", Data: retrievables},
		{Name: "bytes", Data: bytes},
		{Name: "error", Data: errors},
	}
}

func requestSampleValueVisitWithCondition(ctx context.Context, highLevelConn driver.Conn, condition string) ([]*models.SampleValueVisit, error) {
	query := fmt.Sprintf(`
		SELECT
			visit_round,
			timestamp,
			key,
			blob_number,
			row,
			column,
			duration_ms,
			is_retrievable,
			providers,
			bytes,
			error
		FROM %s
		%s
		ORDER BY blob_number, row, column;
		`,
		sampleValueVisitTableDriver.tableName,
		condition,
	)

	// lock the connection
	var response []*models.SampleValueVisit
	err := highLevelConn.Select(ctx, &response, query)
	return response, err
}

func requestAllSampleValueVisits(ctx context.Context, highLevelConn driver.Conn) ([]*models.SampleValueVisit, error) {
	log.WithFields(log.Fields{
		"table":      sampleValueVisitTableDriver.tableName,
		"query_type": "selecting all content",
	}).Debugf("requesting from the clickhouse db")
	return requestSampleValueVisitWithCondition(ctx, highLevelConn, "")
}

func dropAllSampleValueVisitsTable(ctx context.Context, highLevelConn driver.Conn) error {
	log.WithFields(log.Fields{
		"table":      sampleValueVisitTableDriver.tableName,
		"query_type": "deleting all content",
	}).Debugf("deleting visits from the clickhouse db")

	query := fmt.Sprintf(`DELETE FROM %s WHERE block_number >= 0;`, sampleValueVisitTableDriver.tableName)
	return highLevelConn.Exec(ctx, query)
}
