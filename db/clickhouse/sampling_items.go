package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
)

var samplingItemTableDriver = tableDriver[models.SamplingItem]{
	tableName:      models.SamplingItemTableName,
	tag:            "insert_new_item",
	baseQuery:      insertItemsQueryBase(),
	inputConverter: convertItemsToInput,
}

func insertItemsQueryBase() string {
	query := `
	INSERT INTO %s (
		timestamp,
		network,
		item_type,
		sample_type,
		block_link,
		key,
		hash,
		das_row,
		das_column,
		metadata,
		traceable,
		sample_until)
	VALUES`
	return query
}

func convertItemsToInput(items []*models.SamplingItem) proto.Input {
	var (
		timestamps  proto.ColDateTime
		networks    proto.ColStr
		itemTypes   proto.ColStr
		sampleTypes proto.ColStr
		blockLinks  proto.ColUInt64
		keys        proto.ColStr
		hashes      proto.ColStr
		rows        = new(proto.ColUInt32).Nullable()
		columns     = new(proto.ColUInt32).Nullable()
		metadatas   = new(proto.ColStr).Nullable()
		traceables  proto.ColBool
		sampleUntil proto.ColDateTime
	)

	for _, item := range items {
		networks.Append(item.Network)
		timestamps.Append(item.Timestamp)
		itemTypes.Append(item.ItemType)
		sampleTypes.Append(item.SampleType)
		blockLinks.Append(item.BlockLink)
		keys.Append(item.Key)
		hashes.Append(item.Hash)

		dr := proto.Null[uint32]()
		if item.DASRow > 0 {
			dr = proto.NewNullable[uint32](item.DASRow)
		}
		rows.Append(dr)

		dc := proto.Null[uint32]()
		if item.DASColumn > 0 {
			dc = proto.NewNullable[uint32](item.DASColumn)
		}
		columns.Append(dc)

		mtdt := proto.Null[string]()
		if item.Metadata != "" {
			mtdt = proto.NewNullable[string](item.Metadata)
		}
		metadatas.Append(mtdt)
		traceables.Append(item.Traceable)
		sampleUntil.Append(item.SampleUntil)
	}

	return proto.Input{
		{Name: "timestamp", Data: timestamps},
		{Name: "network", Data: networks},
		{Name: "item_type", Data: itemTypes},
		{Name: "sample_type", Data: sampleTypes},
		{Name: "block_link", Data: blockLinks},
		{Name: "key", Data: keys},
		{Name: "hash", Data: keys},
		{Name: "das_row", Data: rows},
		{Name: "das_column", Data: columns},
		{Name: "metadata", Data: metadatas},
		{Name: "traceable", Data: traceables},
		{Name: "sample_until", Data: sampleUntil},
	}
}

func requestItemsWithCondition(ctx context.Context, highLevelConn driver.Conn, conditions string) ([]models.SamplingItem, error) {
	query := fmt.Sprintf(`
		SELECT
			timestamp,
			network,
			item_type,
			sample_type,
			block_link,
			key,
			hash,
			das_row,
			das_column,
			metadata,
			traceable,
			sample_until,
		FROM %s
		%s;`,
		samplingItemTableDriver.tableName,
		conditions,
	)

	// lock the connection
	var response []models.SamplingItem
	err := highLevelConn.Select(ctx, &response, query)
	return response, err
}

func requestAllSAmplingItems(ctx context.Context, highLevelConn driver.Conn, network string) ([]models.SamplingItem, error) {
	log.WithFields(log.Fields{
		"table":      samplingItemTableDriver.tableName,
		"query_type": "selecting all content",
	}).Debugf("requesting from the clickhouse db")
	return requestItemsWithCondition(
		ctx,
		highLevelConn,
		fmt.Sprintf("WHERE network = '%s'", network),
	)
}

func requestItemsOnTTL(ctx context.Context, highLevelConn driver.Conn, network string) ([]models.SamplingItem, error) {
	log.WithFields(log.Fields{
		"table":      samplingItemTableDriver.tableName,
		"query_type": "selecting items on TTL",
	}).Debugf("requesting from the clickhouse db")
	return requestItemsWithCondition(
		ctx,
		highLevelConn,
		fmt.Sprintf("WHERE sample_until > now() and network = '%s'", network),
	)
}

func dropAllItemsTable(ctx context.Context, highLevelConn driver.Conn, network string) error {
	log.WithFields(log.Fields{
		"table":      samplingItemTableDriver.tableName,
		"query_type": "deleting all content",
	}).Debugf("deleting items from the clickhouse db")

	query := fmt.Sprintf(`DELETE FROM %s WHERE network = '%s';`, samplingItemTableDriver.tableName, network)
	return highLevelConn.Exec(ctx, query)
}
