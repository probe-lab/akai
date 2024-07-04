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
	NetworkUnknown    Network = "UNKNOWN"
	NetworkIPFS       Network = "IPFS"
	NetworkAvailTurin Network = "AVAIL_TURING"
)

func Networks() []Network {
	return []Network{
		NetworkIPFS,
		NetworkAvailTurin,
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

func ConfigureNetwork(network string) ([]peer.AddrInfo, protocol.ID, error) {
	var (
		bootstrapPeers []peer.AddrInfo
		v1protocol     protocol.ID
	)
	switch Network(network) {
	case NetworkIPFS:
		bootstrapPeers = kaddht.GetDefaultBootstrapPeerAddrInfos()
		v1protocol = kaddht.ProtocolDHT
	case NetworkAvailTurin:
		bootstrapPeers = BootstrappersToMaddr(BootstrapNodesAvailTurin)
		v1protocol = protocol.ID("/Avail/kad")
	default:
		return bootstrapPeers, v1protocol, fmt.Errorf("unknown network identifier: %s", network)
	}

	return bootstrapPeers, v1protocol, nil
}

func BootstrappersToMaddr(strs []string) []peer.AddrInfo {
	bootnodeInfos := make([]peer.AddrInfo, len(strs), 0)

	for _, addrStr := range strs {
		bInfo, err := peer.AddrInfoFromString(addrStr)
		if err != nil {
			log.Panic("couldn't retrieve peer-info from bootnode maddr string", err)
		}
		bootnodeInfos = append(bootnodeInfos, *bInfo)
	}

	return bootnodeInfos
}
