package core

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/probe-lab/akai/amino"
	"github.com/probe-lab/akai/avail"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/models"
)

var (
	ErrorNotValidNetwork error = fmt.Errorf("no defined key for that network")
)

func ParseDHTKeyType(network models.Network, key string) (any, error) {
	switch network.Protocol {
	case config.ProtocolAvail:
		return avail.KeyFromString(key)
	case config.ProtocolIPFS:
		return amino.CidFromString(key)
	case config.ProtocolLocalCustom:
		return avail.KeyFromString(key)

	default:
		return nil, errors.Wrap(ErrorNotValidNetwork, network.String())
	}
}
