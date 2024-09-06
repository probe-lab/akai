package config

import (
	"time"

	"go.opentelemetry.io/otel/metric"
)

var (
	AvailDelayBase       = 3 * time.Minute
	AvailDelayMultiplier = 2
)

type AvailBlockTracker struct {
	// type of consumers
	TextConsumer bool

	AkaiAPIconsumer bool
	AkaiAPIconfig   AkaiAPIClientConfig

	Network string

	// for the API interaction
	AvailAPIconfig AvailAPIConfig

	// metrics for the service
	Meter metric.Meter
}

type AvailAPIConfig struct {
	Host    string
	Port    int64
	Timeout time.Duration
}
