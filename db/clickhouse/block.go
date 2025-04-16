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
		keys        = new(proto.ColStr).Nullable()
		rows        = new(proto.ColUInt32).Nullable()
		columns     = new(proto.ColUInt32).Nullable()
		sampleUntil proto.ColDateTime
	)

	for _, b := range blocks {
		networks.Append(b.Network)
		timestamps.Append(b.Timestamp)
		numbers.Append(b.Number)
		hashs.Append(b.Hash)

		key := proto.Null[string]()
		if b.Key != "" {
			key = proto.NewNullable[string](b.Key)
		}
		keys.Append(key)

		rs := proto.Null[uint32]()
		if b.DASRows > 0 {
			rs = proto.NewNullable[uint32](b.DASRows)
		}
		rows.Append(rs)

		cls := proto.Null[uint32]()
		if b.DASColumns > 0 {
			cls = proto.NewNullable[uint32](b.DASColumns)
		}
		columns.Append(cls)
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

func requestBlockWithCondition(ctx context.Context, highLevelConn driver.Conn, condition string) ([]models.Block, error) {
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
		ORDER BY number;
		`,
		blockTableDriver.tableName,
		condition,
	)

	// lock the connection
	var response []models.Block
	err := highLevelConn.Select(ctx, &response, query)
	return response, err
}

func requestLastBlock(ctx context.Context, highLevelConn driver.Conn) (models.Block, error) {
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
	var blocks []models.Block
	err := highLevelConn.Select(ctx, &blocks, query)
	if err != nil {
		return models.Block{}, err
	}
	if len(blocks) == 0 {
		return models.Block{}, nil
	}
	return blocks[0], nil
}

func requestAllBlocks(ctx context.Context, highLevelConn driver.Conn, network string) ([]models.Block, error) {
	log.WithFields(log.Fields{
		"table":      blockTableDriver.tableName,
		"query_type": "selecting all content",
	}).Debugf("requesting from the clickhouse db")
	return requestBlockWithCondition(
		ctx,
		highLevelConn,
		fmt.Sprintf("WHERE network = '%s'", network),
	)
}

func requestBlocksOnTTL(ctx context.Context, highLevelConn driver.Conn, network string) ([]models.Block, error) {
	log.WithFields(log.Fields{
		"table":      blockTableDriver.tableName,
		"query_type": "selecting items on TTL",
	}).Debugf("requesting from the clickhouse db")
	return requestBlockWithCondition(
		ctx,
		highLevelConn,
		fmt.Sprintf("WHERE sample_until > now() and network = '%s'", network),
	)
}

func dropAllBlocksTable(ctx context.Context, highLevelConn driver.Conn, network string) error {
	log.WithFields(log.Fields{
		"table":      blockTableDriver.tableName,
		"query_type": "deleting all content",
	}).Debugf("deleting blocks from the clickhouse db")

	query := fmt.Sprintf(`DELETE FROM %s WHERE network = '%s';`, blockTableDriver.tableName, network)
	return highLevelConn.Exec(ctx, query)
}
