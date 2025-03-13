package config

import (
	"time"

	"go.opentelemetry.io/otel/metric"
)

// Block Tracker configuration
type AvailBlockTracker struct {
	Network string
	// defines for how long will the block tracker submit segments to the akai-API
	TrackDuration time.Duration
	// Defines every how ofter the block tracker will start to track again the segments
	TrackInterval time.Duration

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
