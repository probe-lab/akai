package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	kaddht "github.com/libp2p/go-libp2p-kad-dht"
)

type Network string

func (n Network) String() string {
	return string(n)
}

const (
	NetworkUnknown     Network = "UNKNOWN"
	NetworkLocalCustom Network = "Custom"
	NetworkIPFS        Network = "IPFS"
	NetworkAvailTurin  Network = "AVAIL_TURING"
)

func Networks() []Network {
	return []Network{
		NetworkIPFS,
		NetworkAvailTurin,
		NetworkLocalCustom,
	}
}

func AvailNetworks() []Network {
	return []Network{
		NetworkAvailTurin,
	}
}

func ListAllNetworks() string {
	return NetworkListToText(Networks())
}

func ListAvailNetworks() string {
	return NetworkListToText(AvailNetworks())
}

func NetworkListToText(networks []Network) string {
	text := ""
	for idx, str := range networks {
		if idx == 0 {
			text = fmt.Sprintf("[%s", str)
		} else if idx == len(networks)-1 {
			text = fmt.Sprintf("%s, %s]", text, str)
		} else {
			text = fmt.Sprintf("%s, %s", text, str)
		}
	}
	return text
}

func NetworkFromStr(s string) Network {
	networks := Networks()
	for _, value := range networks {
		if strings.ToUpper(s) == string(value) {
			return value
		}
	}
	return NetworkUnknown
}

func ConfigureNetwork(network Network) ([]peer.AddrInfo, protocol.ID, string, error) {
	var (
		bootstrapPeers []peer.AddrInfo
		v1protocol     protocol.ID
		protocolPrefix string
	)
	switch network {
	case NetworkIPFS:
		bootstrapPeers = kaddht.GetDefaultBootstrapPeerAddrInfos()
		v1protocol = kaddht.ProtocolDHT
		protocolPrefix = "/ipfs"
	case NetworkAvailTurin:
		bootstrapPeers = BootstrappersToMaddr(BootstrapNodesAvailTurin)
		v1protocol = protocol.ID("/Avail/kad") // ("/avail_kad/id/1.0.0-6f0996") //
		protocolPrefix = ""
	case NetworkLocalCustom:
		bootstrapPeers = BootstrappersToMaddr([]string{})
		v1protocol = protocol.ID("/local_custom/kad")
		protocolPrefix = ""
	default:
		return bootstrapPeers, v1protocol, protocolPrefix, fmt.Errorf("unknown network identifier: %s", network)
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
