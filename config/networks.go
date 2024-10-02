package config

import (
	"fmt"
	"log"
	"time"

	record "github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/probe-lab/akai/db/models"
)

var DefaultNetwork = models.Network{
	Protocol:    ProtocolAvail,
	NetworkName: NetworkNameAvailMainnet,
	NetworkID:   0,
}

// Protocols
const (
	ProtocolUnknown     string = "UNKNOWN"
	ProtocolLocalCustom string = "LOCAL"
	ProtocolIPFS        string = "IPFS"
	ProtocolAvail       string = "AVAIL"
)

// Networks
const (
	// IPFS
	NetworkNameIPFSAmino string = "AMINO"
	// AVAIL
	NetworkNameAvailTuring  string = "TURING"
	NetworkNameAvailMainnet string = "MAINNET"
	NetworkNameAvailHex     string = "HEX"
	// LOCAL
	NetworkNameLocalCustom string = "CUSTOM"
)

var AvailableProtocols map[string][]string = map[string][]string{
	ProtocolIPFS: {
		NetworkNameIPFSAmino,
	},
	ProtocolAvail: {
		NetworkNameAvailTuring,
		NetworkNameAvailMainnet,
		NetworkNameAvailHex,
	},
	ProtocolLocalCustom: {
		NetworkNameLocalCustom,
	},
}

func ListAllNetworkCombinations() string {
	var networks string
	for protocol := range AvailableProtocols {
		if networks == "" {
			networks = ListNetworksForProtocol(protocol)
		} else {
			networks = networks + ListNetworksForProtocol(protocol)
		}
	}
	return networks
}

func ListNetworksForProtocol(protocol string) string {
	networks := make([]string, 0)
	for _, networkName := range AvailableProtocols[protocol] {
		net := models.Network{Protocol: protocol, NetworkName: networkName}
		networks = append(networks, net.String())
	}
	return "[" + NetworkListToText(networks) + "]"
}

func NetworkListToText(networks []string) string {
	text := ""
	for idx, str := range networks {
		if idx == 0 {
			text = str
		} else {
			text = fmt.Sprintf("%s, %s", text, str)
		}
	}
	return text
}

// NetworkConfiguration describes the entire entire list of parameters and configurations that akai needs
// based on each network's specifics
// TODO: extend this to be readable from a JSON / YAML / TOML
type NetworkConfiguration struct {
	// Network specifics
	Network        models.Network
	BootstrapPeers []peer.AddrInfo
	AgentVersion   string

	// DHT parameters
	HostType        HostType
	DHTHostMode     DHTHostType
	V1Protocol      protocol.ID
	ProtocolPrefix  *string
	CustomValidator record.Validator

	// Sampling specifics
	SamplingType         SamplingType
	BlobsSetCache        int
	SegmentsSetCacheSize int
	DelayBase            time.Duration
	DelayMultiplier      int

	// Chain parameters
	GenesisTime time.Time
}

func ConfigureNetwork(network models.Network) (*NetworkConfiguration, error) {
	switch network.Protocol {
	case ProtocolIPFS:
		// currently we only support the AMINO DHT
		dafultIPFSconfig := DefaultIPFSNetworkConfig
		switch network.NetworkName {
		case NetworkNameIPFSAmino:
			dafultIPFSconfig.Network = network
			return &dafultIPFSconfig, nil

		default:
			return &NetworkConfiguration{}, fmt.Errorf("unknown network identifier %s for protocol %s", network.NetworkName, network.Protocol)
		}

	case ProtocolAvail:
		protocolPrefix := ""
		defaultAvailConfig := DefaultAvailNetworkConfig
		defaultAvailConfig.Network = network

		switch network.NetworkName {
		case NetworkNameAvailMainnet:
			defaultAvailConfig.BootstrapPeers = BootstrappersToMaddr(BootstrapNodesAvailMainnet)
			defaultAvailConfig.V1Protocol = protocol.ID("/avail_kad/id/1.0.0-b91746")
			defaultAvailConfig.GenesisTime = AvailMainnetGenesisTime
			defaultAvailConfig.ProtocolPrefix = &protocolPrefix
			return defaultAvailConfig, nil

		case NetworkNameAvailTuring:
			defaultAvailConfig.BootstrapPeers = BootstrappersToMaddr(BootstrapNodesAvailTurin)
			defaultAvailConfig.V1Protocol = protocol.ID("/avail_kad/id/1.0.0-6f0996")
			defaultAvailConfig.GenesisTime = AvailTuringGenesisTime
			defaultAvailConfig.ProtocolPrefix = &protocolPrefix
			return defaultAvailConfig, nil

		case NetworkNameAvailHex:
			defaultAvailConfig.BootstrapPeers = BootstrappersToMaddr(BootstrapNodesAvailHex)
			defaultAvailConfig.V1Protocol = protocol.ID("/avail_kad/id/1.0.0-9d5ea6")
			defaultAvailConfig.GenesisTime = AvailTuringGenesisTime // TODO: update this to latest calculus
			defaultAvailConfig.ProtocolPrefix = &protocolPrefix
			return defaultAvailConfig, nil

		default:
			return &NetworkConfiguration{}, fmt.Errorf("unknown network identifier %s for protocol %s", network.NetworkName, network.Protocol)
		}

	case ProtocolLocalCustom:
		// mimic of the Avail Mainnet config, but without bootstrappers
		protocolPrefix := ""
		defaultAvailConfig := DefaultAvailNetworkConfig
		defaultAvailConfig.Network = network

		switch network.NetworkName {
		case NetworkNameLocalCustom:
			defaultAvailConfig.BootstrapPeers = BootstrappersToMaddr([]string{})
			defaultAvailConfig.V1Protocol = protocol.ID("/local_custom/kad")
			defaultAvailConfig.GenesisTime = time.Time{}
			defaultAvailConfig.ProtocolPrefix = &protocolPrefix
			return defaultAvailConfig, nil

		default:
			return &NetworkConfiguration{}, fmt.Errorf("unknown network identifier %s for protocol %s", network.NetworkName, network.Protocol)
		}

	default:
		return &NetworkConfiguration{}, fmt.Errorf("unknown protocol identifier: %s", network.Protocol)
	}
}

func BootstrappersToMaddr(strs []string) []peer.AddrInfo {
	bootnodeInfos := make([]peer.AddrInfo, len(strs))

	for idx, addrStr := range strs {
		bInfo, err := peer.AddrInfoFromString(addrStr)
		if err != nil {
			log.Panic("couldn't retrieve peer-info from bootnode maddr string", err)
		}
		bootnodeInfos[idx] = *bInfo
	}

	return bootnodeInfos
}
