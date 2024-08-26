package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
)

var networkTableDriver = tableDriver[models.Network]{
	tableName:      models.NetworkTableName,
	tag:            "insert_new_network",
	baseQuery:      insertNetworkQueryBase(),
	inputConverter: convertNetworkToInput,
}

func insertNetworkQueryBase() string {
	query := `
	INSERT INTO %s (
		network_id,
		protocol, 
		network_name)
		VALUES`
	return query
}

func convertNetworkToInput(networks []models.Network) proto.Input {
	// one item per column, which can ingests an entire array
	var (
		networkIDs   proto.ColUInt16
		protocols    proto.ColStr
		networkNames proto.ColStr
	)

	for _, network := range networks {
		networkIDs.Append((network.NetworkID))
		protocols.Append(network.Protocol)
		networkNames.Append(network.NetworkName)
	}

	return proto.Input{
		{Name: "network_id", Data: networkIDs},
		{Name: "protocol", Data: protocols},
		{Name: "network_name", Data: networkNames},
	}
}

func requestNetworks(ctx context.Context, highLevelConn driver.Conn) ([]models.Network, error) {
	log.WithFields(log.Fields{
		"table":      models.NetworkTableName,
		"query_type": "selecting all content",
	}).Debugf("requesting from the clickhouse db")

	query := fmt.Sprintf(`
		SELECT 
			network_id,
			protocol,
			network_name
		FROM %s
		ORDER BY network_id;
		`,
		models.NetworkTableName)

	// lock the connection
	var response []models.Network
	err := highLevelConn.Select(ctx, &response, query)
	return response, err
}

func dropValuesNetworksTable(ctx context.Context, highLevelConn driver.Conn) error {
	log.WithFields(log.Fields{
		"table":      models.NetworkTableName,
		"query_type": "deleting all content",
	}).Debugf("deleting network from the clickhouse db")

	query := fmt.Sprintf(`DELETE FROM %s WHERE network_id >= 0;`, networkTableDriver.tableName)
	return highLevelConn.Exec(ctx, query)
}
