package config

import (
	"fmt"
)

var (
	ClientName = "akai"
	Maintainer = "probelab"
)

func ComposeAkaiUserAgent(network Network) string {
	switch network.Protocol {
	case ProtocolAvail:
		return "avail-light-client/rust-client"
	default:
		return fmt.Sprintf("%s/%s", Maintainer, ClientName)
	}
}
