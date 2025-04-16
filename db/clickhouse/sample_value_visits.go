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
		networks     proto.ColStr
		timestamps   proto.ColDateTime
		keys         proto.ColStr
		blockNumbers proto.ColUInt64
		rows         = new(proto.ColUInt32).Nullable()
		columns      = new(proto.ColUInt32).Nullable()
		durations    proto.ColInt64
		retrievables proto.ColBool
		bytes        proto.ColInt32
		errors       proto.ColStr
	)

	for _, visit := range visits {
		visitTypes.Append(visit.VisitType)
		visitRounds.Append(visit.VisitRound)
		networks.Append(visit.Network)
		timestamps.Append(visit.Timestamp)
		keys.Append(visit.Key)
		blockNumbers.Append(visit.BlockNumber)

		dr := proto.Null[uint32]()
		if visit.DASRow > 0 {
			dr = proto.NewNullable[uint32](visit.DASRow)
		}
		rows.Append(dr)

		dc := proto.Null[uint32]()
		if visit.DASColumn > 0 {
			dc = proto.NewNullable[uint32](visit.DASColumn)
		}
		columns.Append(dc)
		durations.Append(visit.DurationMs)
		retrievables.Append(visit.IsRetrievable)
		bytes.Append(visit.Bytes)
		errors.Append(visit.Error)
	}

	return proto.Input{
		{Name: "visit_type", Data: visitTypes},
		{Name: "visit_round", Data: visitRounds},
		{Name: "timestamp", Data: timestamps},
		{Name: "network", Data: networks},
		{Name: "key", Data: keys},
		{Name: "block_number", Data: blockNumbers},
		{Name: "das_row", Data: rows},
		{Name: "das_column", Data: columns},
		{Name: "duration_ms", Data: durations},
		{Name: "is_retrievable", Data: retrievables},
		{Name: "bytes", Data: bytes},
		{Name: "error", Data: errors},
	}
}

func requestSampleValueVisitWithCondition(ctx context.Context, highLevelConn driver.Conn, condition string) ([]models.SampleValueVisit, error) {
	query := fmt.Sprintf(`
		SELECT
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
			error
		FROM %s
		%s
		ORDER BY block_number, das_row, das_column;
		`,
		sampleValueVisitTableDriver.tableName,
		condition,
	)

	// lock the connection
	var response []models.SampleValueVisit
	err := highLevelConn.Select(ctx, &response, query)
	return response, err
}

func requestAllSampleValueVisits(ctx context.Context, highLevelConn driver.Conn, network string) ([]models.SampleValueVisit, error) {
	log.WithFields(log.Fields{
		"table":      sampleValueVisitTableDriver.tableName,
		"query_type": "selecting all content",
	}).Debugf("requesting from the clickhouse db")
	return requestSampleValueVisitWithCondition(
		ctx,
		highLevelConn,
		fmt.Sprintf("WHERE network = '%s'", network),
	)
}

func dropAllSampleValueVisitsTable(ctx context.Context, highLevelConn driver.Conn, network string) error {
	log.WithFields(log.Fields{
		"table":      sampleValueVisitTableDriver.tableName,
		"query_type": "deleting all content",
	}).Debugf("deleting visits from the clickhouse db")

	query := fmt.Sprintf(`DELETE FROM %s WHERE network = '%s';`, sampleValueVisitTableDriver.tableName, network)
	return highLevelConn.Exec(ctx, query)
}
