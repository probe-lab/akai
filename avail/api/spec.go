package api

import (
	"time"

	"github.com/probe-lab/akai/config"
)

const (
	BlockIntervalTarget = 20 * time.Second
)

var (
	// from first block https://explorer.avail.so/#/explorer/query/1
	TuringTestnetGenesisTime = time.Unix(int64(1711605040), int64(0))
)

type Block uint64

func GenesisTimeFromNetwork(network config.Network) time.Time {
	var genTime time.Time
	switch network {
	case config.NetworkAvailTurin:
		genTime = TuringTestnetGenesisTime
	default:
	}
	return genTime
}
