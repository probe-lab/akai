package config

import (
	"time"

	"go.opentelemetry.io/otel/metric"
)

type HostType int8
type DHTHostType int8

const (
	// Host Type
	UnknownHostType HostType = iota
	AminoLibp2pHost

	// DHT Host Type
	DHTUknownType DHTHostType = iota
	DHTClient
	DHTServer
)

type CommonDHTHostOpts struct {
	ID          int
	IP          string
	Port        int64
	DialTimeout time.Duration

	HostType     HostType
	DHTMode      DHTHostType
	AgentVersion string

	// metrics for the service
	Meter metric.Meter
}
