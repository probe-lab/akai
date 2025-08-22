package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/probe-lab/akai/db/models"
	log "github.com/sirupsen/logrus"
)

var ipnsRecordVisitsTableDriver = tableDriver[models.IPNSRecordVisit]{
	tableName:      models.IPNSRecordVisitTableName,
	tag:            "insert_new_ipns_record_visit",
	baseQuery:      insertIPNSRecordVisitQueryBase(),
	inputConverter: convertIPNSRecordVisitToInput,
}

func insertIPNSRecordVisitQueryBase() string {
	query := `
	INSERT INTO %s (
		visit_round,
		timestamp,
		record,
		record_type,
		quorum,
		seq_number,
		ttl_s,
		is_valid,
		is_retrievable,
		result,
		duration_ms,
		error)
	VALUES`
	return query
}

func convertIPNSRecordVisitToInput(visits []*models.IPNSRecordVisit) proto.Input {
	var (
		visitRounds    proto.ColUInt64
		timestamps     proto.ColDateTime
		records        proto.ColStr
		recordTypes    proto.ColStr
		quorums        proto.ColUInt8
		seqNumbers     proto.ColUInt32
		ttls           proto.ColFloat64
		isValids       proto.ColBool
		isRetrievables proto.ColBool
		results        proto.ColStr
		durations      proto.ColInt64
		errors         proto.ColStr
	)

	for _, visit := range visits {
		visitRounds.Append(visit.VisitRound)
		timestamps.Append(visit.Timestamp)
		records.Append(visit.Record)
		recordTypes.Append(visit.RecordType)
		quorums.Append(visit.Quorum)
		seqNumbers.Append(visit.SeqNumber)
		ttls.Append(visit.TTL.Seconds())
		isValids.Append(visit.IsValid)
		isRetrievables.Append(visit.IsRetrievable)
		results.Append(visit.Result)
		durations.Append(visit.Duration.Milliseconds())
		errors.Append(visit.Error)
	}

	return proto.Input{
		{Name: "visit_round", Data: visitRounds},
		{Name: "timestamp", Data: timestamps},
		{Name: "record", Data: records},
		{Name: "record_type", Data: recordTypes},
		{Name: "quorum", Data: quorums},
		{Name: "seq_number", Data: seqNumbers},
		{Name: "ttl_s", Data: ttls},
		{Name: "is_valid", Data: isValids},
		{Name: "is_retrievable", Data: isRetrievables},
		{Name: "result", Data: results},
		{Name: "duration_ms", Data: durations},
		{Name: "error", Data: errors},
	}
}

func dropAllIPNSRecordVisitsTable(ctx context.Context, client driver.Conn) error {
	query := `DELETE FROM ipns_record_visits WHERE true`

	log.WithFields(log.Fields{
		"query": query,
	}).Debug("dropping all ipns record visits")

	return client.Exec(ctx, query)
}

func requestAllIPNSRecordVisits(ctx context.Context, client driver.Conn) ([]*models.IPNSRecordVisit, error) {
	query := `SELECT visit_round, timestamp, record, record_type, quorum, seq_number, ttl_s, is_valid, is_retrievable, result, duration_ms, error FROM ipns_record_visits ORDER BY timestamp DESC`

	log.WithFields(log.Fields{
		"query": query,
	}).Debug("requesting all ipns record visits")

	rows, err := client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying ipns record visits: %w", err)
	}
	defer rows.Close()

	var visits []*models.IPNSRecordVisit
	var ttl_s float64
	var duration_ms int64
	for rows.Next() {
		visit := &models.IPNSRecordVisit{}
		err := rows.Scan(
			&visit.VisitRound,
			&visit.Timestamp,
			&visit.Record,
			&visit.RecordType,
			&visit.Quorum,
			&visit.SeqNumber,
			&ttl_s,
			&visit.IsValid,
			&visit.IsRetrievable,
			&visit.Result,
			&duration_ms,
			&visit.Error,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning ipns record visit row: %w", err)
		}
		visit.TTL = time.Duration(ttl_s) * time.Second
		visit.Duration = time.Duration(duration_ms) * time.Millisecond
		visits = append(visits, visit)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating ipns record visit rows: %w", err)
	}

	return visits, nil
}
