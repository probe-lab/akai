package api

import (
	"time"

	"github.com/probe-lab/akai/config"
)

// DHT specs
const (
	KeyDelimiter      string = ":"
	CoordinatesPerKey int    = 3
)

// Chain specs
const (
	BlockIntervalTarget = 20 * time.Second
)

var (
	// from first block https://explorer.avail.so/#/explorer/query/1
	TuringTestnetGenesisTime = time.Unix(int64(1711605040), int64(0))
)

func GenesisTimeFromNetwork(network config.Network) time.Time {
	var genTime time.Time
	switch network.NetworkName {
	case config.NetworkNameAvailTuring:
		genTime = TuringTestnetGenesisTime
	default:
	}
	return genTime
}
