package config

import (
	"time"

	"go.opentelemetry.io/otel/metric"
)

// Block Tracker configuration
type AvailBlockTracker struct {
	Network string

	// type of consumers
	TextConsumer    bool
	AkaiAPIconsumer bool
	AkaiAPIconfig   *AkaiAPIClientConfig

	// for the API interaction
	AvailAPIconfig *AvailAPIConfig

	// metrics for the service
	Meter metric.Meter
}

type AvailAPIConfig struct {
	Host    string
	Port    int64
	Timeout time.Duration
}
