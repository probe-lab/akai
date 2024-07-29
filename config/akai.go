package config

import (
	"fmt"

	"github.com/probe-lab/akai/db"
)

var (
	ClientName    = "akai"
	ClientVersion = "v0.1.0"
	Maintainer    = "probelab"
)

func ComposeAkaiUserAgent(network db.Network) string {
	switch network.Protocol {
	case ProtocolAvail:
		return "avail-light-client/rust-client"
	default:
		return fmt.Sprintf("%s/%s", Maintainer, ClientName)
	}
}
