package config

import (
	"fmt"
	"time"

	"github.com/probe-lab/akai/db/models"
)

var (
	BlobsSetCacheSize int = 10_000 // TODO: define better values (or at least reason them)
)

type AkaiPing struct {
	Network string
	Key     string
	Timeout time.Duration
}

type AkaiAPIClientConfig struct {
	Host       string
	Port       int64
	PrefixPath string
	Timeout    time.Duration
}

type AkaiAPIServiceConfig struct {
	// api host parameters
	Host       string
	Port       int64
	PrefixPath string
	Timeout    time.Duration
	Mode       string

	// configuration
	Network string
}

type AkaiDaemonConfig struct {
	Network           string
	APIconfig         AkaiAPIServiceConfig
	DBconfig          DatabaseDetails
	DataSamplerConfig AkaiDataSamplerConfig
}

type AkaiDataSamplerConfig struct {
	Network         string
	Workers         int
	SamplingTimeout time.Duration
	DBsyncInterval  time.Duration
	SampleTTL       time.Duration
}

var (
	ClientName    = "akai"
	ClientVersion = "v0.1.0"
	Maintainer    = "probelab"
)

func ComposeAkaiUserAgent(network models.Network) string {
	switch network.Protocol {
	case ProtocolAvail:
		return "avail-light-client/rust-client"
	default:
		return fmt.Sprintf("%s/%s", Maintainer, ClientName)
	}
}
