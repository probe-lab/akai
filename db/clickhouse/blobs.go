package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
)

var blobTableDriver = tableDriver[models.AgnosticBlob]{
	tableName:      models.BlobTableName,
	tag:            "insert_new_blob",
	baseQuery:      insertBlobQueryBase(),
	inputConverter: convertBlobToInput,
}

func insertBlobQueryBase() string {
	query := `
	INSERT INTO %s (
		network_id,
		timestamp,
		hash,
		block_number,
		key,
		rows,
		columns,
		sample_until)
	VALUES`
	return query
}

func convertBlobToInput(blobs []models.AgnosticBlob) proto.Input {
	var (
		networkIDs   proto.ColUInt16
		timestamps   proto.ColDateTime
		hashs        proto.ColStr
		keys         proto.ColStr
		blockNumbers proto.ColUInt64
		rows         proto.ColUInt32
		columns      proto.ColUInt32
		sampleUntil  proto.ColDateTime
	)

	for _, blob := range blobs {
		networkIDs.Append((blob.NetworkID))
		timestamps.Append(blob.Timestamp)
		blockNumbers.Append(blob.BlockNumber)
		hashs.Append(blob.Hash)
		keys.Append(blob.Key)
		rows.Append(blob.Rows)
		columns.Append(blob.Columns)
		sampleUntil.Append(blob.SampleUntil)
	}

	return proto.Input{
		{Name: "network_id", Data: networkIDs},
		{Name: "timestamp", Data: timestamps},
		{Name: "hash", Data: hashs},
		{Name: "key", Data: keys},
		{Name: "block_number", Data: blockNumbers},
		{Name: "rows", Data: rows},
		{Name: "columns", Data: columns},
		{Name: "sample_until", Data: sampleUntil},
	}
}

func requestBlobWithCondition(ctx context.Context, highLevelConn driver.Conn, condition string) ([]models.AgnosticBlob, error) {
	query := fmt.Sprintf(`
		SELECT 
			network_id,
			timestamp,
			hash,
			key,
			block_number,
			rows,
			columns,
			sample_until,
		FROM %s
		%s
		ORDER BY block_number;
		`,
		blobTableDriver.tableName,
		condition,
	)

	// lock the connection
	var response []models.AgnosticBlob
	err := highLevelConn.Select(ctx, &response, query)
	return response, err
}

func requestLatestBlob(ctx context.Context, highLevelConn driver.Conn) (models.AgnosticBlob, error) {
	log.WithFields(log.Fields{
		"table":      blobTableDriver.tableName,
		"query_type": "selecting latest blob",
	}).Debugf("requesting from the clickhouse db")

	query := fmt.Sprintf(`
		SELECT 
			network_id,
			timestamp,
			hash,
			key,
			block_number,
			rows,
			columns,
			sample_until,
		FROM %s
		ORDER BY block_number DESC
		LIMIT 1;
		`,
		blobTableDriver.tableName,
	)

	// lock the connection
	var blobs []models.AgnosticBlob
	err := highLevelConn.Select(ctx, &blobs, query)
	if err != nil {
		return models.AgnosticBlob{}, err
	}
	if len(blobs) == 0 {
		return models.AgnosticBlob{}, nil
	}
	return blobs[0], nil
}

func requestAllBlobs(ctx context.Context, highLevelConn driver.Conn) ([]models.AgnosticBlob, error) {
	log.WithFields(log.Fields{
		"table":      blobTableDriver.tableName,
		"query_type": "selecting all content",
	}).Debugf("requesting from the clickhouse db")
	return requestBlobWithCondition(ctx, highLevelConn, "")
}

func requestBlobsOnTTL(ctx context.Context, highLevelConn driver.Conn) ([]models.AgnosticBlob, error) {
	log.WithFields(log.Fields{
		"table":      blobTableDriver.tableName,
		"query_type": "selecting items on TTL",
	}).Debugf("requesting from the clickhouse db")
	return requestBlobWithCondition(ctx, highLevelConn, "WHERE sample_until > now()")
}

func dropAllBlobsTable(ctx context.Context, highLevelConn driver.Conn) error {
	log.WithFields(log.Fields{
		"table":      blobTableDriver.tableName,
		"query_type": "deleting all content",
	}).Debugf("deleting blobs from the clickhouse db")

	query := fmt.Sprintf(`DELETE FROM %s WHERE network_id >= 0;`, blobTableDriver.tableName)
	return highLevelConn.Exec(ctx, query)
}
