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
	Params      string
	TLSrequired bool

	// metrics for the service
	Meter metric.Meter
}

func (d DatabaseDetails) String() string {
	return fmt.Sprintf(
		"%s://%s:%s@%s/%s%s",
		d.Driver,
		d.User,
		d.Password,
		d.Address,
		d.Database,
		d.Params,
	)
}
