package core

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/probe-lab/akai/amino"
	"github.com/probe-lab/akai/avail"
	"github.com/probe-lab/akai/config"
)

var (
	ErrorNotValidNetwork error = fmt.Errorf("no defined key for that network")
)

func ParseDHTKeyType(network config.Network, key string) (any, error) {
	switch network {
	case config.NetworkAvailTurin:
		return avail.KeyFromString(key)
	case config.NetworkIPFS:
		return amino.CidFromString(key)
	case config.NetworkLocalCustom:
		return avail.KeyFromString(key)

	default:
		return nil, errors.Wrap(ErrorNotValidNetwork, network.String())
	}
}
