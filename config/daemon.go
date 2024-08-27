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

	BlobsSetCacheSize    int
	SegmentsSetCacheSize int
}

type AkaiDataSamplerConfig struct {
	Network         string
	Workers         int
	SamplingTimeout time.Duration
	DBsyncInterval  time.Duration
	SampleTTL       time.Duration
}

func DaemonConfigForNetwork(network models.Network) (blobSetSize int, segmentSetSize int, err error) {
	switch network.Protocol {
	case ProtocolIPFS:
		blobSetSize = AvailBlobsSetCacheSize
		segmentSetSize = AvailSegmentsSetCacheSize
		return

	case ProtocolAvail:
		blobSetSize = IPFSBlobsSetCacheSize
		segmentSetSize = IPFSSegmentsSetCacheSize
		return

	case ProtocolLocalCustom:
		blobSetSize = IPFSBlobsSetCacheSize
		segmentSetSize = IPFSSegmentsSetCacheSize
		return
	default:
		err = fmt.Errorf("")
		return
	}
}
