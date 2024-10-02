package config

import (
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

var (
	ClientName    = "akai"
	ClientVersion = "v0.1.0"
	Maintainer    = "probelab"
)

func ComposeAkaiAgentVersion() string {
	return fmt.Sprintf("%s/%s", ClientName, ClientVersion)
}

type AkaiDaemonConfig struct {
	Network string

	APIconfig         *AkaiAPIServiceConfig
	DBconfig          *DatabaseDetails
	DataSamplerConfig *AkaiDataSamplerConfig
	DHTHostConfig     *CommonDHTHostOpts

	// metrics for the service
	Meter metric.Meter
}
