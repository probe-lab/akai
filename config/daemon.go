package config

import (
	"fmt"
)

var (
	ClientName    = "akai"
	ClientVersion = "v0.1.1"
	Maintainer    = "probelab"
)

func ComposeAkaiAgentVersion() string {
	return fmt.Sprintf("%s/%s", ClientName, ClientVersion)
}

type AkaiDaemonConfig struct {
	Network string

	DBconfig          *DatabaseDetails
	DHTHostConfig     *CommonDHTHostOpts
	DataSamplerConfig *AkaiDataSamplerConfig
}
