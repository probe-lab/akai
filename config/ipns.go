package config

import (
	"time"

	"github.com/ipfs/boxo/ipns"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
)

var (
	DefaultIPNSNetwork = Network{Protocol: ProtocolIPNS, NetworkName: NetworkNameAmino}

	IPNSDelayBase       = 2 * time.Minute
	IPNSDelayMultiplier = 1

	// TODO: random values
	IPNSBlobsSetCacheSize    int = 1024
	IPNSSegmentsSetCacheSize int = 1024
)

// DefaulIPNSNetworkConfig defines the default configuration for the IPNS network for Akai
var DefaultIPNSNetworkConfig = NetworkConfiguration{
	Network: DefaultIPNSNetwork,
	// network parameters
	BootstrapPeers: kaddht.GetDefaultBootstrapPeerAddrInfos(),
	AgentVersion:   ComposeAkaiAgentVersion(),

	// dht paramets
	HostType:       AminoLibp2pHost,
	DHTHostMode:    DHTClient,
	V1Protocol:     kaddht.ProtocolDHT,
	ProtocolPrefix: nil,

	// sampling specifics
	SamplingType:         SampleIPNSname,
	BlobsSetCache:        IPNSBlobsSetCacheSize,
	SegmentsSetCacheSize: IPNSSegmentsSetCacheSize,
	DelayBase:            IPNSDelayBase,
	DelayMultiplier:      IPNSDelayMultiplier,
}

// IPNS DHT Namespace configuration
type IPNSNetworkScrapperConfig struct {
	Network              string
	NotChannelBufferSize int
	SamplerNotifyTimeout time.Duration
	Quorum               int64
	AkaiAPIServiceConfig *AkaiAPIServiceConfig
}

func ComposeIpnsKey(k string) ([]byte, error) {
	key, err := ipns.NameFromString(k)
	if err != nil {
		return []byte{}, err
	}
	return key.RoutingKey(), nil
}
