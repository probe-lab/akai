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
	baseQuery:      insertSegmentQueryBase(),
	inputConverter: convertVisitToInput,
}

func insertVisitQueryBase() string {
	query := `
	INSERT INTO %s (
		timestamp,
		key,
		duration,
		is_retrievable)
	VALUES`
	return query
}

func convertVisitToInput(visits []models.AgnosticVisit) proto.Input {
	var (
		timestamps   proto.ColDateTime
		keys         proto.ColStr
		durations    proto.ColInt64
		retrievables proto.ColBool
	)

	for _, visit := range visits {
		timestamps.Append(visit.Timestamp)
		keys.Append(visit.Key)
		durations.Append(visit.Duration.Microseconds())
		retrievables.Append(visit.IsRetrievable)
	}

	return proto.Input{
		{Name: "timestamp", Data: timestamps},
		{Name: "key", Data: keys},
		{Name: "duration", Data: durations},
		{Name: "is_retrievable", Data: retrievables},
	}
}

func requestVisitWithCondition(ctx context.Context, highLevelConn driver.Conn, condition string) ([]models.AgnosticVisit, error) {
	query := fmt.Sprintf(`
		SELECT 
			timestamp,
			key,
			duration,
			tracking_ttl,
		FROM %s
		%s
		ORDER BY block_number, row, column;
		`,
		segmentTableDriver.tableName,
		condition,
	)

	// lock the connection
	var response []models.AgnosticVisit
	err := highLevelConn.Select(ctx, &response, query)
	return response, err
}

func requestAllVisits(ctx context.Context, highLevelConn driver.Conn) ([]models.AgnosticVisit, error) {
	log.WithFields(log.Fields{
		"table":      segmentTableDriver.tableName,
		"query_type": "selecting all content",
	}).Debugf("requesting from the clickhouse db")
	return requestVisitWithCondition(ctx, highLevelConn, "")
}

func dropAllVisitsTable(ctx context.Context, highLevelConn driver.Conn) error {
	log.WithFields(log.Fields{
		"table":      segmentTableDriver.tableName,
		"query_type": "deleting all content",
	}).Debugf("deleting visits from the clickhouse db")

	query := fmt.Sprintf(`DELETE * FROM %s;`, segmentTableDriver.tableName)
	return highLevelConn.Exec(ctx, query)
}
