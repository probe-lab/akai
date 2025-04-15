package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
)

var sampleGenericVisistsTableDriver = tableDriver[models.SampleGenericVisit]{
	tableName:      models.SampleGenericVisitTableName,
	tag:            "insert_new_sample_closest_visit",
	baseQuery:      insertSampleGenericVisitQueryBase(),
	inputConverter: convertSampleGenericVisitToInput,
}

func insertSampleGenericVisitQueryBase() string {
	query := `
	INSERT INTO %s (
		visit_type,
		visit_round,
		network,
		timestamp,
		key,
		duration_ms,
		response_items,
		peer_ids,
		error)
	VALUES`
	return query
}

func convertSampleGenericVisitToInput(visits []*models.SampleGenericVisit) proto.Input {
	var (
		visitTypes    proto.ColStr
		visitRounds   proto.ColUInt64
		networks      proto.ColStr
		timestamps    proto.ColDateTime
		keys          proto.ColStr
		durations     proto.ColInt64
		responseItems proto.ColInt32
		peerIDs       proto.ColArr[proto.ColStr]
		errors        proto.ColStr
	)

	for _, visit := range visits {
		visitTypes.Append(visit.VisitType)
		visitRounds.Append(visit.VisitRound)
		networks.Append(visit.Network)
		timestamps.Append(visit.Timestamp)
		keys.Append(visit.Key)
		durations.Append(visit.DurationMs)
		responseItems.Append(int32(visit.ResponseItems))
		peers := make([]proto.ColStr, len(visit.Peers))
		for i, p := range visit.Peers {
			peers[i].Append(p)
		}
		peerIDs.Append(peers)
		errors.Append(visit.Error)
	}

	return proto.Input{
		{Name: "visit_type", Data: visitTypes},
		{Name: "visit_round", Data: visitRounds},
		{Name: "network", Data: networks},
		{Name: "timestamp", Data: timestamps},
		{Name: "key", Data: keys},
		{Name: "duration_ms", Data: durations},
		{Name: "response_items", Data: responseItems},
		{Name: "peer_ids", Data: peerIDs},
		{Name: "error", Data: errors},
	}
}

func requestSampleGenericVisitWithCondition(ctx context.Context, highLevelConn driver.Conn, condition string) ([]*models.SampleGenericVisit, error) {
	query := fmt.Sprintf(`
		SELECT
			visit_type,
			visit_round,
			timestamp,
			network,
			key,
			duration_ms,
			response_items,
			peer_ids,
			error
		FROM %s
		%s
		ORDER BY number, row, column;
		`,
		sampleGenericVisistsTableDriver.tableName,
		condition,
	)

	// lock the connection
	var response []*models.SampleGenericVisit
	err := highLevelConn.Select(ctx, &response, query)
	return response, err
}

func requestAllSampleGenericVisit(ctx context.Context, highLevelConn driver.Conn) ([]*models.SampleGenericVisit, error) {
	log.WithFields(log.Fields{
		"table":      sampleGenericVisistsTableDriver.tableName,
		"query_type": "selecting all content",
	}).Debugf("requesting from the clickhouse db")
	return requestSampleGenericVisitWithCondition(ctx, highLevelConn, "")
}

func dropAllSampleGenericVisitTable(ctx context.Context, highLevelConn driver.Conn) error {
	log.WithFields(log.Fields{
		"table":      sampleGenericVisistsTableDriver.tableName,
		"query_type": "deleting all content",
	}).Debugf("deleting visits from the clickhouse db")

	query := fmt.Sprintf(`DELETE * FROM %s;`, sampleGenericVisistsTableDriver.tableName)
	return highLevelConn.Exec(ctx, query)
}
