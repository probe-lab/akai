package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
)

var segmentTableDriver = tableDriver[models.AgnosticSegment]{
	tableName:      models.SegmentTableName,
	tag:            "insert_new_segment",
	baseQuery:      insertSegmentQueryBase(),
	inputConverter: convertSegmentToInput,
}

func insertSegmentQueryBase() string {
	query := `
	INSERT INTO %s (
		timestamp,
		block_number,
		key,
		row,
		column,
		tracking_ttl)
	VALUES`
	return query
}

func convertSegmentToInput(segments []models.AgnosticSegment) proto.Input {
	var (
		timestamps   proto.ColDateTime
		blockNumbers proto.ColUInt64
		keys         proto.ColStr
		rows         proto.ColUInt32
		columns      proto.ColUInt32
		trackingTTLs proto.ColDateTime
	)

	for _, segment := range segments {
		timestamps.Append(segment.Timestamp)
		blockNumbers.Append(segment.BlockNumber)
		keys.Append(segment.Key)
		rows.Append(segment.Row)
		columns.Append(segment.Column)
		trackingTTLs.Append(segment.TrackingTTL)
	}

	return proto.Input{
		{Name: "timestamp", Data: timestamps},
		{Name: "block_number", Data: blockNumbers},
		{Name: "key", Data: keys},
		{Name: "row", Data: rows},
		{Name: "column", Data: columns},
		{Name: "tracking_ttl", Data: trackingTTLs},
	}
}

func requestSegmentWithCondition(ctx context.Context, highLevelConn driver.Conn, condition string) ([]models.AgnosticSegment, error) {
	query := fmt.Sprintf(`
		SELECT 
			timestamp,
			block_number,
			key,
			row,
			column,
			tracking_ttl,
		FROM %s
		%s
		ORDER BY block_number, row, column;
		`,
		segmentTableDriver.tableName,
		condition,
	)

	// lock the connection
	var response []models.AgnosticSegment
	err := highLevelConn.Select(ctx, &response, query)
	return response, err
}

func requestAllSegments(ctx context.Context, highLevelConn driver.Conn) ([]models.AgnosticSegment, error) {
	log.WithFields(log.Fields{
		"table":      segmentTableDriver.tableName,
		"query_type": "selecting all content",
	}).Debugf("requesting from the clickhouse db")
	return requestSegmentWithCondition(ctx, highLevelConn, "")
}

func requestSegmentsOnTTL(ctx context.Context, highLevelConn driver.Conn) ([]models.AgnosticSegment, error) {
	log.WithFields(log.Fields{
		"table":      segmentTableDriver.tableName,
		"query_type": "selecting items on TTL",
	}).Debugf("requesting from the clickhouse db")
	return requestSegmentWithCondition(ctx, highLevelConn, "WHERE tracking_ttl > now()")
}

func dropAllSegmentsTable(ctx context.Context, highLevelConn driver.Conn) error {
	log.WithFields(log.Fields{
		"table":      segmentTableDriver.tableName,
		"query_type": "deleting all content",
	}).Debugf("deleting segments from the clickhouse db")

	query := fmt.Sprintf(`DELETE * FROM %s;`, segmentTableDriver.tableName)
	return highLevelConn.Exec(ctx, query)
}
