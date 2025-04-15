package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
)

var blockTableDriver = tableDriver[models.Block]{
	tableName:      models.BlockTableName,
	tag:            "insert_new_block",
	baseQuery:      insertBlockQueryBase(),
	inputConverter: convertBlockToInput,
}

func insertBlockQueryBase() string {
	query := `
	INSERT INTO %s (
		network,
		timestamp,
		number,
		hash,
		key,
		das_rows,
		das_columns,
		sample_until)
	VALUES`
	return query
}

func convertBlockToInput(blocks []*models.Block) proto.Input {
	var (
		networks    proto.ColStr
		timestamps  proto.ColDateTime
		numbers     proto.ColUInt64
		hashs       proto.ColStr
		keys        proto.ColStr
		rows        proto.ColUInt32
		columns     proto.ColUInt32
		sampleUntil proto.ColDateTime
	)

	for _, b := range blocks {
		networks.Append(b.Network)
		timestamps.Append(b.Timestamp)
		numbers.Append(b.Number)
		hashs.Append(b.Hash)
		keys.Append(b.Key)
		rows.Append(b.DASRows)
		columns.Append(b.DASColumns)
		sampleUntil.Append(b.SampleUntil)
	}

	return proto.Input{
		{Name: "network", Data: networks},
		{Name: "timestamp", Data: timestamps},
		{Name: "number", Data: numbers},
		{Name: "hash", Data: hashs},
		{Name: "key", Data: keys},
		{Name: "das_rows", Data: rows},
		{Name: "das_columns", Data: columns},
		{Name: "sample_until", Data: sampleUntil},
	}
}

func requestBlockWithCondition(ctx context.Context, highLevelConn driver.Conn, condition string) ([]*models.Block, error) {
	query := fmt.Sprintf(`
		SELECT
			network,
			timestamp,
			number,
			hash,
			key,
			das_rows,
			das_columns,
			sample_until,
		FROM %s
		%s
		ORDER BY block_number;
		`,
		blockTableDriver.tableName,
		condition,
	)

	// lock the connection
	var response []*models.Block
	err := highLevelConn.Select(ctx, &response, query)
	return response, err
}

func requestLastBlock(ctx context.Context, highLevelConn driver.Conn) (*models.Block, error) {
	log.WithFields(log.Fields{
		"table":      blockTableDriver.tableName,
		"query_type": "selecting latest block",
	}).Debugf("requesting from the clickhouse db")

	query := fmt.Sprintf(`
		SELECT
			network,
			timestamp,
			number,
			hash,
			key,
			das_rows,
			das_columns,
			sample_until,
		FROM %s
		ORDER BY number DESC
		LIMIT 1;
		`,
		blockTableDriver.tableName,
	)

	// lock the connection
	var blocks []*models.Block
	err := highLevelConn.Select(ctx, &blocks, query)
	if err != nil {
		return &models.Block{}, err
	}
	if len(blocks) == 0 {
		return &models.Block{}, nil
	}
	return blocks[0], nil
}

func requestAllBlocks(ctx context.Context, highLevelConn driver.Conn) ([]*models.Block, error) {
	log.WithFields(log.Fields{
		"table":      blockTableDriver.tableName,
		"query_type": "selecting all content",
	}).Debugf("requesting from the clickhouse db")
	return requestBlockWithCondition(ctx, highLevelConn, "")
}

func requestBlocksOnTTL(ctx context.Context, highLevelConn driver.Conn) ([]*models.Block, error) {
	log.WithFields(log.Fields{
		"table":      blockTableDriver.tableName,
		"query_type": "selecting items on TTL",
	}).Debugf("requesting from the clickhouse db")
	return requestBlockWithCondition(ctx, highLevelConn, "WHERE sample_until > now()")
}

func dropAllBlocksTable(ctx context.Context, highLevelConn driver.Conn) error {
	log.WithFields(log.Fields{
		"table":      blockTableDriver.tableName,
		"query_type": "deleting all content",
	}).Debugf("deleting blocks from the clickhouse db")

	query := fmt.Sprintf(`DELETE FROM %s WHERE network_id >= 0;`, blockTableDriver.tableName)
	return highLevelConn.Exec(ctx, query)
}
