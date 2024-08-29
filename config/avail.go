package config

import (
	"time"
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
}

type AvailAPIConfig struct {
	Host    string
	Port    int64
	Timeout time.Duration
}
