package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
)

var peerInfoVisitsTableDriver = tableDriver[models.PeerInfoVisit]{
	tableName:      models.PeerInfoVisitTableName,
	tag:            "insert_new_peer_info_visit",
	baseQuery:      insertPeerInfoVisitQueryBase(),
	inputConverter: convertPeerInfoVisitToInput,
}

func insertPeerInfoVisitQueryBase() string {
	query := `
	INSERT INTO %s (
		visit_round,
		timestamp,
		network,
		peer_id,
		duration_ms,
		agent_version,
		protocols,
		protocol_version,
		multi_addresses,
		error)
	VALUES`
	return query
}

func convertPeerInfoVisitToInput(visits []*models.PeerInfoVisit) proto.Input {
	var (
		visitRounds      proto.ColUInt64
		timestamps       proto.ColDateTime
		networks         proto.ColStr
		peerIDs          proto.ColStr
		durations        proto.ColInt64
		agentVersions    proto.ColStr
		protocols        = new(proto.ColStr).Array()
		protocolVersions proto.ColStr
		multiAddresses   = new(proto.ColStr).Array()
		errors           proto.ColStr
	)

	for _, visit := range visits {
		visitRounds.Append(visit.VisitRound)
		timestamps.Append(visit.Timestamp)
		networks.Append(visit.Network)
		peerIDs.Append(visit.PeerID)
		durations.Append(visit.Duration.Milliseconds())
		agentVersions.Append(visit.AgentVersion)

		protocolStrs := make([]string, len(visit.Protocols))
		for i, p := range visit.Protocols {
			protocolStrs[i] = string(p)
		}
		protocols.Append(protocolStrs)

		protocolVersions.Append(visit.ProtocolVersion)

		multiAddrStrs := make([]string, len(visit.MultiAddresses))
		for i, multiAddress := range visit.MultiAddresses {
			multiAddrStrs[i] = multiAddress.String()
		}
		multiAddresses.Append(multiAddrStrs)
		errors.Append(visit.Error)
	}

	return proto.Input{
		{Name: "visit_round", Data: visitRounds},
		{Name: "timestamp", Data: timestamps},
		{Name: "network", Data: networks},
		{Name: "peer_id", Data: peerIDs},
		{Name: "duration_ms", Data: durations},
		{Name: "agent_version", Data: agentVersions},
		{Name: "protocols", Data: protocols},
		{Name: "protocol_version", Data: protocolVersions},
		{Name: "multi_addresses", Data: multiAddresses},
		{Name: "error", Data: errors},
	}
}

func dropAllPeerInfoVisitsTable(ctx context.Context, client driver.Conn, network string) error {
	query := `DELETE FROM peer_info_visits WHERE network = ?`

	log.WithFields(log.Fields{
		"query":   query,
		"network": network,
	}).Debug("dropping all peer info visits")

	return client.Exec(ctx, query, network)
}

func requestAllPeerInfoVisits(ctx context.Context, client driver.Conn, network string) ([]*models.PeerInfoVisit, error) {
	query := `SELECT visit_round, timestamp, network, peer_id, duration_ms, agent_version, protocols, protocol_version, multi_addresses, error FROM peer_info_visits WHERE network = ? ORDER BY timestamp DESC`

	log.WithFields(log.Fields{
		"query":   query,
		"network": network,
	}).Debug("requesting all peer info visits")

	rows, err := client.Query(ctx, query, network)
	if err != nil {
		return nil, fmt.Errorf("querying peer info visits: %w", err)
	}
	defer rows.Close()

	var visits []*models.PeerInfoVisit
	for rows.Next() {
		visit := &models.PeerInfoVisit{}
		err := rows.Scan(
			&visit.VisitRound,
			&visit.Timestamp,
			&visit.Network,
			&visit.PeerID,
			&visit.Duration,
			&visit.AgentVersion,
			&visit.Protocols,
			&visit.ProtocolVersion,
			&visit.MultiAddresses,
			&visit.Error,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning peer info visit row: %w", err)
		}
		visits = append(visits, visit)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating peer info visit rows: %w", err)
	}

	return visits, nil
}
