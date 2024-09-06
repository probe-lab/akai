package config

import (
	"fmt"
	"time"

	"github.com/probe-lab/akai/db/models"
	"go.opentelemetry.io/otel/metric"
)

type AkaiPing struct {
	Network string
	Key     string
	Timeout time.Duration
}

type AkaiAPIClientConfig struct {
	Host       string
	Port       int64
	PrefixPath string
	Timeout    time.Duration
}

type AkaiAPIServiceConfig struct {
	// api host parameters
	Host       string
	Port       int64
	PrefixPath string
	Timeout    time.Duration
	Mode       string

	// configuration
	Network string

	// metrics for the service
	Meter metric.Meter
}

var (
	ClientName    = "akai"
	ClientVersion = "v0.1.0"
	Maintainer    = "probelab"
)

func ComposeAkaiUserAgent(network models.Network) string {
	switch network.Protocol {
	case ProtocolAvail:
		return "avail-light-client/rust-client"
	default:
		return fmt.Sprintf("%s/%s", Maintainer, ClientName)
	}
}
