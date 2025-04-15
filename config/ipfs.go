package config

import (
	"time"

	kaddht "github.com/libp2p/go-libp2p-kad-dht"
)

var (
	DefaultIPFSNetwork = Network{Protocol: ProtocolIPFS, NetworkName: NetworkNameAmino}

	IPFSDelayBase       = 5 * time.Minute
	IPFSDelayMultiplier = 1

	// TODO: random values
	IPFSBlobsSetCacheSize    int = 1024
	IPFSSegmentsSetCacheSize int = 1024
)

// DefaulIPFSNetworkConfig defines the default configuration for the IPFS network for Akai
var DefaultIPFSNetworkConfig = NetworkConfiguration{
	Network: Network{
		Protocol:    ProtocolIPFS,
		NetworkName: NetworkNameAmino,
	},
	// network parameters
	BootstrapPeers: kaddht.GetDefaultBootstrapPeerAddrInfos(),
	AgentVersion:   ComposeAkaiAgentVersion(),

	// dht paramets
	HostType:       AminoLibp2pHost,
	DHTHostMode:    DHTClient,
	V1Protocol:     kaddht.ProtocolDHT,
	ProtocolPrefix: nil,

	// sampling specifics
	SamplingType:         SampleProviders,
	BlobsSetCache:        IPFSBlobsSetCacheSize,
	SegmentsSetCacheSize: IPFSSegmentsSetCacheSize,
	DelayBase:            IPFSDelayBase,
	DelayMultiplier:      IPFSDelayMultiplier,
}

// IPFS DHT Namespace configuration
type IPFSNetworkScrapperConfig struct {
	Network              string
	NotChannelBufferSize int
	SamplerNotifyTimeout time.Duration
	AkaiAPIServiceConfig *AkaiAPIServiceConfig
}
