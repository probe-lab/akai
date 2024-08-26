package api

import (
	"time"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/models"
)

// DHT specs
const (
	KeyDelimiter      string = ":"
	CoordinatesPerKey int    = 3
)

// Chain specs
const (
	BlockIntervalTarget = 20 * time.Second
	BlockTTL            = 24 * time.Hour
)

var (
	// from first block https://explorer.avail.so/#/explorer/query/1
	TuringGenesisTime  = time.Unix(int64(1711605040), int64(0))
	MainnetGenesisTime = time.Unix(int64(1720075080), int64(0))
)

func GenesisTimeFromNetwork(network models.Network) time.Time {
	var genTime time.Time
	switch network.NetworkName {
	case config.NetworkNameAvailTuring:
		genTime = TuringGenesisTime

	case config.NetworkNameAvailMainnet:
		genTime = MainnetGenesisTime

	default:
	}
	return genTime
}
