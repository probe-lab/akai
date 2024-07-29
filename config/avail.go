package config

import "time"

type AvailBlockTracker struct {
	// type of consumers
	TextConsumer bool
	Network      string

	// for the API interaction
	AvailHttpAPIClient
}

type AvailHttpAPIClient struct {
	IP      string
	Port    int64
	Timeout time.Duration
}
