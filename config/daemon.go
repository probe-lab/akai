package config

import (
	"fmt"
	"time"

	"github.com/probe-lab/akai/db/models"
)

var (
	// block every 20 seconds that should last a day (extra of 1 blob more to make sure we hit the cache)
	AvailBlobsSetCacheSize int = 2 * (3 + 1) * 60 * 24
	// number of blocks keeping a max of 1024 segments each
	AvailSegmentsSetCacheSize int = AvailBlobsSetCacheSize * 1024

	// TODO: random values
	IPFSBlobsSetCacheSize    int = 1024
	IPFSSegmentsSetCacheSize int = 1024
)

type AkaiDaemonConfig struct {
	Network string

	APIconfig         AkaiAPIServiceConfig
	DBconfig          DatabaseDetails
	DataSamplerConfig AkaiDataSamplerConfig
	DHTHostConfig     CommonDHTOpts
}

type AkaiDataSamplerConfig struct {
	Network         string
	Workers         int
	SamplingTimeout time.Duration
	DBsyncInterval  time.Duration
	AkaiSamplingDetails
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
		return AkaiSamplingDetails{
			BlobsSetCacheSize:    IPFSBlobsSetCacheSize,
			SegmentsSetCacheSize: IPFSSegmentsSetCacheSize,
			DelayBase:            IPFSDelayBase,
			DelayMultiplier:      IPFSDelayMultiplier,
		}, nil

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
