package config

import (
	"time"
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
