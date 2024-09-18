package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
)

var visistsTableDriver = tableDriver[models.AgnosticVisit]{
	tableName:      models.VisitTableName,
	tag:            "insert_new_visit",
	baseQuery:      insertVisitQueryBase(),
	inputConverter: convertVisitToInput,
}

func insertVisitQueryBase() string {
	query := `
	INSERT INTO %s (
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
		error)
	VALUES`
	return query
}

func convertVisitToInput(visits []models.AgnosticVisit) proto.Input {
	var (
		visitRounds  proto.ColUInt64
		timestamps   proto.ColDateTime
		keys         proto.ColStr
		blobNumbers  proto.ColUInt64
		rows         proto.ColUInt32
		columns      proto.ColUInt32
		durations    proto.ColInt64
		retrievables proto.ColBool
		providers    proto.ColUInt32
		bytes        proto.ColUInt32
		errors       proto.ColStr
	)

	for _, visit := range visits {
		visitRounds.Append(visit.VisitRound)
		timestamps.Append(visit.Timestamp)
		keys.Append(visit.Key)
		blobNumbers.Append(visit.BlobNumber)
		rows.Append(visit.Row)
		columns.Append(visit.Column)
		durations.Append(visit.DurationMs)
		retrievables.Append(visit.IsRetrievable)
		providers.Append(visit.Providers)
		bytes.Append(visit.Bytes)
		errors.Append(visit.Error)
	}

	return proto.Input{
		{Name: "visit_round", Data: visitRounds},
		{Name: "timestamp", Data: timestamps},
		{Name: "key", Data: keys},
		{Name: "blob_number", Data: blobNumbers},
		{Name: "row", Data: rows},
		{Name: "column", Data: columns},
		{Name: "duration_ms", Data: durations},
		{Name: "is_retrievable", Data: retrievables},
		{Name: "providers", Data: providers},
		{Name: "bytes", Data: bytes},
		{Name: "error", Data: errors},
	}
}

func requestVisitWithCondition(ctx context.Context, highLevelConn driver.Conn, condition string) ([]models.AgnosticVisit, error) {
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
		visistsTableDriver.tableName,
		condition,
	)

	// lock the connection
	var response []models.AgnosticVisit
	err := highLevelConn.Select(ctx, &response, query)
	return response, err
}

func requestAllVisits(ctx context.Context, highLevelConn driver.Conn) ([]models.AgnosticVisit, error) {
	log.WithFields(log.Fields{
		"table":      visistsTableDriver.tableName,
		"query_type": "selecting all content",
	}).Debugf("requesting from the clickhouse db")
	return requestVisitWithCondition(ctx, highLevelConn, "")
}

func dropAllVisitsTable(ctx context.Context, highLevelConn driver.Conn) error {
	log.WithFields(log.Fields{
		"table":      visistsTableDriver.tableName,
		"query_type": "deleting all content",
	}).Debugf("deleting visits from the clickhouse db")

	query := fmt.Sprintf(`DELETE FROM %s WHERE blob_number >= 0;`, visistsTableDriver.tableName)
	return highLevelConn.Exec(ctx, query)
}
