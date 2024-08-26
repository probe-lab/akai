package config

import (
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/probe-lab/akai/db/models"

	kaddht "github.com/libp2p/go-libp2p-kad-dht"
)

var DefaultNetwork = models.Network{
	Protocol:    ProtocolAvail,
	NetworkName: NetworkNameAvailTuring,
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
	NetworkNameIPFSAmino    string = "AMINO"
	NetworkNameAvailTuring  string = "TURING"
	NetworkNameAvailMainnet string = "MAINNET"
	NetworkNameLocalCustom  string = "CUSTOM"
)

var AvailableProtocols map[string][]string = map[string][]string{
	ProtocolIPFS: {
		NetworkNameIPFSAmino,
	},
	ProtocolAvail: {
		NetworkNameAvailTuring,
		NetworkNameAvailMainnet,
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

func ConfigureNetwork(network models.Network) ([]peer.AddrInfo, protocol.ID, string, error) {
	var (
		bootstrapPeers []peer.AddrInfo
		v1protocol     protocol.ID
		protocolPrefix string
	)
	switch network.Protocol {
	case ProtocolIPFS:
		switch network.NetworkName {
		case NetworkNameIPFSAmino:
			bootstrapPeers = kaddht.GetDefaultBootstrapPeerAddrInfos()
			v1protocol = kaddht.ProtocolDHT
			protocolPrefix = "/ipfs"
		default:
			return bootstrapPeers, v1protocol, protocolPrefix, fmt.Errorf("unknown network identifier %s for protocol %s", network.NetworkName, network.Protocol)
		}

	case ProtocolAvail:
		switch network.NetworkName {
		case NetworkNameAvailTuring:
			bootstrapPeers = BootstrappersToMaddr(BootstrapNodesAvailTurin)
			v1protocol = protocol.ID("/Avail/kad")
			protocolPrefix = ""
		case NetworkNameAvailMainnet:
			bootstrapPeers = BootstrappersToMaddr(BootstrapNodesAvailMainnet)
			v1protocol = protocol.ID("/Avail/kad")
			protocolPrefix = ""
		default:
			return bootstrapPeers, v1protocol, protocolPrefix, fmt.Errorf("unknown network identifier %s for protocol %s", network.NetworkName, network.Protocol)
		}

	case ProtocolLocalCustom:
		switch network.NetworkName {
		case NetworkNameLocalCustom:
			bootstrapPeers = BootstrappersToMaddr([]string{})
			v1protocol = protocol.ID("/local_custom/kad")
			protocolPrefix = ""
		default:
			return bootstrapPeers, v1protocol, protocolPrefix, fmt.Errorf("unknown network identifier %s for protocol %s", network.NetworkName, network.Protocol)
		}

	default:
		return bootstrapPeers, v1protocol, protocolPrefix, fmt.Errorf("unknown protocol identifier: %s", network.Protocol)
	}

	return bootstrapPeers, v1protocol, protocolPrefix, nil
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
