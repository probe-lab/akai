package core

import (
	"context"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"

	"github.com/probe-lab/akai/amino"
	"github.com/probe-lab/akai/avail"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/models"
)

type DHTHostType int8

const (
	DHTClient DHTHostType = iota
	DHTServer
)

type CommonDHTOpts struct {
	ID          int
	IP          string
	Port        int64
	DialTimeout time.Duration
	DHTMode     DHTHostType
	UserAgent   string
}

type DHTHost interface {
	IntenalID() int
	Host() host.Host
	FindClosestPeers(context.Context, string) (time.Duration, []peer.ID, error)
	FindProviders(context.Context, cid.Cid) (time.Duration, []peer.AddrInfo, error)
	FindValue(context.Context, string) (time.Duration, []byte, error)
	PutValue(context.Context, string, []byte) (time.Duration, error)
}

var _ DHTHost = (*amino.DHTHost)(nil)

func NewDHTHost(ctx context.Context, network models.Network, cfg CommonDHTOpts) (DHTHost, error) {
	return composeHostForNetwork(ctx, network, cfg)
}

func composeHostForNetwork(ctx context.Context, network models.Network, commonCfg CommonDHTOpts) (DHTHost, error) {
	switch network.Protocol {
	case config.ProtocolIPFS:
		// configure amino DHT
		bootstapers, v1protocol, _, err := config.ConfigureNetwork(network)
		if err != nil {
			return nil, errors.Wrap(err, "extracting network info from given network")
		}
		aminoDHTHostConfig := amino.DHTHostConfig{
			HostID:               commonCfg.ID,
			IP:                   commonCfg.IP,
			Port:                 commonCfg.Port,
			DialTimeout:          commonCfg.DialTimeout,
			UserAgent:            commonCfg.UserAgent,
			DHTMode:              ParseAminoDHTHostMode(commonCfg.DHTMode),
			Bootstrapers:         bootstapers,
			V1Protocol:           v1protocol,
			CustomProtocolPrefix: nil, // better not no change it, as it is the default at go-libp2p-kad-dht
		}
		return amino.NewDHTHost(ctx, aminoDHTHostConfig)

	case config.ProtocolAvail, config.ProtocolLocalCustom:
		// configure amino DHT
		bootstapers, v1protocol, protoPrefix, err := config.ConfigureNetwork(network)
		if err != nil {
			return nil, errors.Wrap(err, "extracting network info from given network")
		}
		aminoDHTHostConfig := amino.DHTHostConfig{
			HostID:               commonCfg.ID,
			IP:                   commonCfg.IP,
			Port:                 commonCfg.Port,
			DialTimeout:          commonCfg.DialTimeout,
			UserAgent:            commonCfg.UserAgent,
			DHTMode:              ParseAminoDHTHostMode(commonCfg.DHTMode),
			Bootstrapers:         bootstapers,
			V1Protocol:           v1protocol,
			CustomValidator:      &avail.KeyValidator{},
			CustomProtocolPrefix: &protoPrefix,
		}
		return amino.NewDHTHost(ctx, aminoDHTHostConfig)

	default:
		return nil, fmt.Errorf("no dht host available for network %s", network)
	}
}

func ParseAminoDHTHostMode(mode DHTHostType) kaddht.ModeOpt {
	switch mode {
	case DHTClient:
		return kaddht.ModeClient
	case DHTServer:
		return kaddht.ModeServer
	default:
		return kaddht.ModeClient
	}
}
