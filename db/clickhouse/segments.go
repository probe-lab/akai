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
		blob_number,
		key,
		row,
		column,
		sample_until)
	VALUES`
	return query
}

func convertSegmentToInput(segments []models.AgnosticSegment) proto.Input {
	var (
		timestamps  proto.ColDateTime
		blobNumbers proto.ColUInt64
		keys        proto.ColStr
		rows        proto.ColUInt32
		columns     proto.ColUInt32
		sampleUntil proto.ColDateTime
	)

	for _, segment := range segments {
		timestamps.Append(segment.Timestamp)
		blobNumbers.Append(segment.BlobNumber)
		keys.Append(segment.Key)
		rows.Append(segment.Row)
		columns.Append(segment.Column)
		sampleUntil.Append(segment.SampleUntil)
	}

	return proto.Input{
		{Name: "timestamp", Data: timestamps},
		{Name: "blob_number", Data: blobNumbers},
		{Name: "key", Data: keys},
		{Name: "row", Data: rows},
		{Name: "column", Data: columns},
		{Name: "sample_until", Data: sampleUntil},
	}
}

func requestSegmentWithCondition(ctx context.Context, highLevelConn driver.Conn, condition string) ([]models.AgnosticSegment, error) {
	query := fmt.Sprintf(`
		SELECT 
			timestamp,
			blob_number,
			key,
			row,
			column,
			sample_until,
		FROM %s
		%s
		ORDER BY blob_number, row, column;
		`,
		segmentTableDriver.tableName,
		condition,
	)

	// lock the connection
	var response []models.AgnosticSegment
	err := highLevelConn.Select(ctx, &response, query)
	fmt.Println(response)
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
	return requestSegmentWithCondition(ctx, highLevelConn, "WHERE sample_until > now()")
}

func dropAllSegmentsTable(ctx context.Context, highLevelConn driver.Conn) error {
	log.WithFields(log.Fields{
		"table":      segmentTableDriver.tableName,
		"query_type": "deleting all content",
	}).Debugf("deleting segments from the clickhouse db")

	query := fmt.Sprintf(`DELETE FROM %s WHERE blob_number >= 0;`, segmentTableDriver.tableName)
	return highLevelConn.Exec(ctx, query)
}
