package config

import (
	"fmt"
	"time"

	"github.com/probe-lab/akai/db/models"
	"go.opentelemetry.io/otel/metric"
)

type SamplingType int

const (
	SampleUnknown SamplingType = iota
	SampleProviders
	SampleValue
)

type AkaiDataSamplerConfig struct {
	Network         string
	Workers         int
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

func SamplingConfigForNetwork(network models.Network) (AkaiSamplingDetails, error) {
	switch network.Protocol {
	case ProtocolIPFS:
		return AkaiSamplingDetails{}, nil

	case ProtocolAvail:
		return AkaiSamplingDetails{
			BlobsSetCacheSize:    AvailBlobsSetCacheSize,
			SegmentsSetCacheSize: AvailSegmentsSetCacheSize,
			DelayBase:            AvailDelayBase,
			DelayMultiplier:      AvailDelayMultiplier,
		}, nil

	case ProtocolLocalCustom:
		return AkaiSamplingDetails{
			BlobsSetCacheSize:    IPFSBlobsSetCacheSize,
			SegmentsSetCacheSize: IPFSSegmentsSetCacheSize,
			DelayBase:            IPFSDelayBase,
			DelayMultiplier:      IPFSDelayMultiplier,
		}, nil
	default:
		err := fmt.Errorf("protocol not supported")
		return AkaiSamplingDetails{}, err
	}
}
