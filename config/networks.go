package config

import (
	"fmt"
	"log"
	"strings"
	"time"

	record "github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var DefaultNetwork = Network{
	Protocol:    ProtocolAvail,
	NetworkName: NetworkNameMainnet,
}

type ProtocolType string

func (p ProtocolType) String() string { return string(p) }

type NetworkName string

func (n NetworkName) String() string { return string(n) }

// Protocols
const (
	ProtocolUnknown  ProtocolType = "UNKNOWN"
	ProtocolLocal    ProtocolType = "LOCAL"
	ProtocolIPFS     ProtocolType = "IPFS"
	ProtocolAvail    ProtocolType = "AVAIL"
	ProtocolCelestia ProtocolType = "CELESTIA"
)

// Networks
const (
	// GENERIC
	NetworkNameMainnet NetworkName = "MAINNET"
	// IPFS
	NetworkNameAmino NetworkName = "AMINO"
	// AVAIL
	NetworkNameTuring NetworkName = "TURING"
	NetworkNameHex    NetworkName = "HEX"
	// CELESTIA
	NetworkNameMocha4 NetworkName = "MOCHA-4"
	// LOCAL
	NetworkNameCustom NetworkName = "CUSTOM"
)

var AvailableProtocols map[ProtocolType][]NetworkName = map[ProtocolType][]NetworkName{
	ProtocolIPFS: {
		NetworkNameAmino,
	},
	ProtocolAvail: {
		NetworkNameTuring,
		NetworkNameMainnet,
		NetworkNameHex,
	},
	ProtocolCelestia: {
		NetworkNameMainnet,
		NetworkNameMocha4,
	},
	ProtocolLocal: {
		NetworkNameCustom,
	},
}

func ProtocolTypeFromString(s string) ProtocolType {
	switch s {
	case ProtocolLocal.String():
		return ProtocolLocal
	case ProtocolIPFS.String():
		return ProtocolIPFS
	case ProtocolAvail.String():
		return ProtocolAvail
	case ProtocolCelestia.String():
		return ProtocolCelestia
	default:
		return ProtocolUnknown
	}
}

func NetworkNameFromString(s string) NetworkName {
	switch s {
	case NetworkNameMainnet.String():
		return NetworkNameMainnet
	case NetworkNameAmino.String():
		return NetworkNameAmino
	case NetworkNameTuring.String():
		return NetworkNameTuring
	case NetworkNameHex.String():
		return NetworkNameHex
	case NetworkNameMocha4.String():
		return NetworkNameMocha4
	case NetworkNameCustom.String():
		return NetworkNameCustom
	default:
		return NetworkNameMainnet
	}
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

func ListNetworksForProtocol(protocol ProtocolType) string {
	networks := make([]string, 0)
	for _, networkName := range AvailableProtocols[protocol] {
		net := Network{Protocol: protocol, NetworkName: networkName}
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

type Network struct {
	Protocol    ProtocolType `ch:"protocol" json:"protocol"`
	NetworkName NetworkName  `ch:"network_name" json:"network_name"`
}

func (n Network) String() string {
	return fmt.Sprintf("%s_%s", n.Protocol, n.NetworkName)
}

func (n Network) FromString(s string) Network {
	parts := strings.Split(s, "_")
	protocol := strings.ToUpper(parts[0]) // the protocol goes always first
	network := "MAINNET"
	if len(parts) >= 2 {
		network = strings.ToUpper(parts[1])
	}
	return Network{
		Protocol:    ProtocolTypeFromString(protocol),
		NetworkName: NetworkNameFromString(network),
	}
}

func (n Network) IsComplete() bool {
	return n.Protocol != "" && n.NetworkName != ""
}

func NetworkFromStr(s string) Network {
	return Network{}.FromString(s)
}

// NetworkConfiguration describes the entire entire list of parameters and configurations that akai needs
// based on each network's specifics
// TODO: extend this to be readable from a JSON / YAML / TOML
type NetworkConfiguration struct {
	// Network specifics
	Network        Network
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

func ConfigureNetwork(network Network) (*NetworkConfiguration, error) {
	switch network.Protocol {
	case ProtocolIPFS:
		// currently we only support the AMINO DHT
		dafultIPFSconfig := DefaultIPFSNetworkConfig
		switch network.NetworkName {
		case NetworkNameAmino:
			return &dafultIPFSconfig, nil
		default:
			return &NetworkConfiguration{}, fmt.Errorf("unknown network identifier %s for protocol %s", network.NetworkName, network.Protocol)
		}

	case ProtocolAvail:
		protocolPrefix := ""
		defaultAvailConfig := DefaultAvailNetworkConfig
		defaultAvailConfig.Network = network

		switch network.NetworkName {
		case NetworkNameMainnet:
			defaultAvailConfig.BootstrapPeers = BootstrappersToMaddr(BootstrapNodesAvailMainnet)
			defaultAvailConfig.V1Protocol = protocol.ID("/avail_kad/id/1.0.0-b91746")
			defaultAvailConfig.ProtocolPrefix = &protocolPrefix
			return defaultAvailConfig, nil

		case NetworkNameTuring:
			defaultAvailConfig.BootstrapPeers = BootstrappersToMaddr(BootstrapNodesAvailTurin)
			defaultAvailConfig.V1Protocol = protocol.ID("/avail_kad/id/1.0.0-6f0996")
			defaultAvailConfig.ProtocolPrefix = &protocolPrefix
			return defaultAvailConfig, nil

		case NetworkNameHex:
			defaultAvailConfig.BootstrapPeers = BootstrappersToMaddr(BootstrapNodesAvailHex)
			defaultAvailConfig.V1Protocol = protocol.ID("/avail_kad/id/1.0.0-9d5ea6")
			defaultAvailConfig.ProtocolPrefix = &protocolPrefix
			return defaultAvailConfig, nil

		default:
			return &NetworkConfiguration{}, fmt.Errorf("unknown network identifier %s for protocol %s", network.NetworkName, network.Protocol)
		}

	case ProtocolCelestia:
		protocolPrefix := GetCelestiaDHTProtocolPrefix(network.NetworkName)
		defaultCelestiaConfig := DefaultCelestiaNetworkConfig

		switch network.NetworkName {
		case NetworkNameMainnet:
			defaultCelestiaConfig.BootstrapPeers = BootstrappersToMaddr(BootstrapNodesCelestiaMainnet)
			defaultCelestiaConfig.V1Protocol = ComposeCestiaDHTProtocolID(protocolPrefix)
			defaultCelestiaConfig.GenesisTime = time.Time{}
			defaultCelestiaConfig.ProtocolPrefix = &protocolPrefix
			return defaultCelestiaConfig, nil

		case NetworkNameMocha4:
			defaultCelestiaConfig.BootstrapPeers = BootstrappersToMaddr(BootstrapNodesCelestiaMocha4)
			defaultCelestiaConfig.V1Protocol = ComposeCestiaDHTProtocolID(protocolPrefix)
			defaultCelestiaConfig.ProtocolPrefix = &protocolPrefix
			return defaultCelestiaConfig, nil

		default:
			return &NetworkConfiguration{}, fmt.Errorf("unknown network identifier %s for protocol %s", network.NetworkName, network.Protocol)
		}

	case ProtocolLocal:
		// mimic of the Avail Mainnet config, but without bootstrappers
		protocolPrefix := ""
		defaultAvailConfig := DefaultAvailNetworkConfig
		defaultAvailConfig.Network = network

		switch network.NetworkName {
		case NetworkNameCustom:
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
