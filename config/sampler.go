package config

import (
	"time"

	"go.opentelemetry.io/otel/metric"
)

type SamplingType int

func (t SamplingType) String() string {
	switch t {
	case SampleClosest:
		return "FIND_CLOSEST"
	case SampleProviders:
		return "FIND_PROVIDERS"
	case SamplePeerInfo:
		return "FIND_PEER_INFO"
	case SamplePeers:
		return "FIND_PEERS"
	case SampleValue:
		return "FIND_VALUE"
	case SampleIPNSname:
		return "FIND_IPNS_VALUE"
	case SampleUnknown:
		return "UNKNOWN_OPERATION"
	default:
		return "UNKNOWN_OPERATION"
	}
}

func SamplingTypeFromStr(s string) SamplingType {
	switch s {
	case "FIND_CLOSEST":
		return SampleClosest
	case "FIND_PROVIDERS":
		return SampleProviders
	case "FIND_PEER_INFO":
		return SamplePeerInfo
	case "FIND_PEERS":
		return SamplePeers
	case "FIND_VALUE":
		return SampleValue
	case "FIND_IPNS_VALUE":
		return SampleIPNSname
	case "UNKNOWN_OPERATION":
		return SampleUnknown
	default:
		return SampleUnknown
	}
}

const (
	SampleUnknown SamplingType = iota
	SampleClosest
	SampleProviders
	SampleValue
	SamplePeers
	SamplePeerInfo
	SampleIPNSname
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
