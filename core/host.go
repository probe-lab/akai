package core

import (
	"context"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.opentelemetry.io/otel"

	"github.com/probe-lab/akai/amino"
	"github.com/probe-lab/akai/config"
)

var DefaultDHTHostOpts = &config.CommonDHTHostOpts{
	ID:           0,
	IP:           "0.0.0.0",        // default?
	Port:         9020,             // default?
	DialTimeout:  10 * time.Second, // this is the DialTimeout, not the timeout for the operation
	HostType:     config.UnknownHostType,
	DHTMode:      config.DHTUknownType,
	AgentVersion: config.ComposeAkaiAgentVersion(),
	Meter:        otel.GetMeterProvider().Meter("akai_host"),
}

type DHTHost interface {
	IntenalID() int
	Host() host.Host
	FindClosestPeers(context.Context, string) (time.Duration, []peer.ID, error)
	FindProviders(context.Context, cid.Cid) (time.Duration, []peer.AddrInfo, error)
	FindValue(context.Context, string) (time.Duration, []byte, error)
	FindPeers(context.Context, string, time.Duration) (time.Duration, []peer.AddrInfo, error)
	PutValue(context.Context, string, []byte) (time.Duration, error)
}

var _ DHTHost = (*amino.DHTHost)(nil)

func NewDHTHost(ctx context.Context, networkConfig *config.NetworkConfiguration, commonCfg *config.CommonDHTHostOpts) (DHTHost, error) {
	return composeHostForNetwork(ctx, networkConfig, commonCfg)
}

func composeHostForNetwork(ctx context.Context, networkConfig *config.NetworkConfiguration, commonCfg *config.CommonDHTHostOpts) (DHTHost, error) {
	// override host config from network one
	DHTHostConfigFromNetworkConfig(commonCfg, networkConfig)

	switch networkConfig.HostType {
	case config.AminoLibp2pHost:
		aminoDHTHostConfig := &amino.DHTHostConfig{
			HostID:               commonCfg.ID,
			IP:                   commonCfg.IP,
			Port:                 commonCfg.Port,
			DialTimeout:          commonCfg.DialTimeout,
			AgentVersion:         commonCfg.AgentVersion,
			DHTMode:              ParseAminoDHTHostMode(commonCfg.DHTMode),
			Bootstrapers:         networkConfig.BootstrapPeers,
			V1Protocol:           networkConfig.V1Protocol,
			CustomProtocolPrefix: networkConfig.ProtocolPrefix,
			CustomValidator:      networkConfig.CustomValidator,
			Meter:                commonCfg.Meter,
		}
		return amino.NewDHTHost(ctx, aminoDHTHostConfig, networkConfig)

	default:
		return nil, fmt.Errorf("no dht host available for network %s", &networkConfig.Network)
	}
}

func ParseAminoDHTHostMode(mode config.DHTHostType) kaddht.ModeOpt {
	switch mode {
	case config.DHTClient:
		return kaddht.ModeClient
	case config.DHTServer:
		return kaddht.ModeServer
	default:
		return kaddht.ModeClient
	}
}

func DHTHostConfigFromNetworkConfig(hostCfg *config.CommonDHTHostOpts, netCfg *config.NetworkConfiguration) {
	hostCfg.HostType = netCfg.HostType
	hostCfg.AgentVersion = netCfg.AgentVersion
	hostCfg.DHTMode = netCfg.DHTHostMode
}
