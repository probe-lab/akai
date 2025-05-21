package config

import (
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

type DatabaseDetails struct {
	Driver      string
	Address     string
	User        string
	Password    string
	Database    string
	TLSrequired bool

	// metrics for the service
	Meter metric.Meter
}

func (d DatabaseDetails) LocalMigrationDSN() string {
	return fmt.Sprintf(
		"%s://%s:%s@%s/%s?secure=%t&x-multi-statement=true",
		d.getDriver(),
		d.User,
		d.Password,
		d.Address,
		d.Database,
		d.TLSrequired,
	)
}

func (d DatabaseDetails) ReplicatedMigrationDSN() string {
	return fmt.Sprintf(
		"%s://%s:%s@%s/%s?secure=%t&x-multi-statement=true&x-migrations-table-engine=ReplicatedMergeTree",
		d.getDriver(),
		d.User,
		d.Password,
		d.Address,
		d.Database,
		d.TLSrequired,
	)
}

func (d DatabaseDetails) getDriver() string {
	switch d.Driver {
	case "clickhouse-local", "clickhouse-replicated":
		return "clickhouse"
	default:
		return "clickhouse"
	}
}
