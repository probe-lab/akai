package config

import (
	"time"

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
