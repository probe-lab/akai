package config

import (
	"time"

	"go.opentelemetry.io/otel/metric"
)

type SamplingType int

const (
	SampleUnknown SamplingType = iota
	SampleClosest
	SampleProviders
	SampleValue
	SamplePeers
	SamplePeer
)

type AkaiDataSamplerConfig struct {
	Network         string
	Workers         int64
	SamplingTimeout time.Duration
	DBsyncInterval  time.Duration
	AkaiSamplingDetails

	// metrics for the service
	Meter metric.Meter
}

type AkaiSamplingDetails struct {
	BlobsSetCacheSize    int
	SegmentsSetCacheSize int
	DelayBase            time.Duration
	DelayMultiplier      int
}

func UpdateSamplingDetailFromNetworkConfig(samplCfg *AkaiSamplingDetails, netCfg *NetworkConfiguration) {
	samplCfg.BlobsSetCacheSize = netCfg.BlobsSetCache
	samplCfg.SegmentsSetCacheSize = netCfg.SegmentsSetCacheSize
	samplCfg.DelayBase = netCfg.DelayBase
	samplCfg.DelayMultiplier = netCfg.DelayMultiplier
}

func ParseSamplingType(t SamplingType) string {
	switch t {
	case SampleClosest:
		return "FIND_CLOSEST"
	case SampleProviders:
		return "FIND_PROVIDERS"
	case SamplePeer:
		return "FIND_PEERS"
	case SamplePeers:
		return "FIND_PEER"
	case SampleValue:
		return "FIND_VALUE"
	case SampleUnknown:
		return "UNKNOWN_OPERATION"
	default:
		return "UNKNOWN_OPERATION"
	}
}
