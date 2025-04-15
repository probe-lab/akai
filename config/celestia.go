package config

import (
	"fmt"
	"time"

	record "github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const (
	CelestiaMainnet = "celestia"
	CelestiaMocha4  = "mocha-4"
)

var (
	CelestiaDelayBase       = 3 * time.Minute
	CelestiaDelayMultiplier = 1

	// TODO: random values
	CelestiaBlobsSetCacheSize    int = 1024
	CelestiaSegmentsSetCacheSize int = 1024
)

// DefaulIPFSNetworkConfig defines the default configuration for the IPFS network for Akai
var DefaultCelestiaNetworkConfig = &NetworkConfiguration{
	Network: Network{
		Protocol:    ProtocolCelestia,
		NetworkName: NetworkNameMainnet,
	},
	// network parameters
	BootstrapPeers: BootstrappersToMaddr(BootstrapNodesCelestiaMainnet),
	AgentVersion:   ComposeAkaiAgentVersion(),

	// dht paramets
	HostType:       AminoLibp2pHost,
	DHTHostMode:    DHTClient,
	V1Protocol:     ComposeCestiaDHTProtocolID(GetCelestiaDHTProtocolPrefix(NetworkNameMainnet)),
	ProtocolPrefix: nil,

	// sampling specifics
	SamplingType:         SamplePeers,
	BlobsSetCache:        CelestiaBlobsSetCacheSize,
	SegmentsSetCacheSize: CelestiaSegmentsSetCacheSize,
	DelayBase:            CelestiaDelayBase,
	DelayMultiplier:      CelestiaDelayMultiplier,
}

func GetCelestiaDHTProtocolPrefix(networkName NetworkName) string {
	net := CelestiaMainnet
	switch networkName {
	case NetworkNameMainnet:
		net = CelestiaMainnet

	case NetworkNameMocha4:
		net = CelestiaMocha4

	default:
	}

	return fmt.Sprintf("/celestia/%s", net)
}

func ComposeCestiaDHTProtocolID(protocolBase string) protocol.ID {
	return protocol.ID(fmt.Sprintf("%s/kad/1.0.0", protocolBase))
}

type CelestiaKeyValidator struct{}

var _ record.Validator = (*CelestiaKeyValidator)(nil)

func (v *CelestiaKeyValidator) Validate(key string, value []byte) error {
	// TODO: check if the given size is bigger than allowed cell lenght?
	// not sure which is the format of the value
	// - Cell bytes ?
	// - Cell proof ?
	// - Cell bytes + Cell proof ?
	return nil
}

func (v *CelestiaKeyValidator) Select(key string, values [][]byte) (int, error) {
	if len(values) <= 0 {
		return 0, ErrorNoValueForKey
	}
	return 1, nil
}

// Celestia DHT Namespace configuration
type CelestiaNetworkScrapperConfig struct {
	Network              string
	NotChannelBufferSize int
	SamplerNotifyTimeout time.Duration
	AkaiAPIServiceConfig *AkaiAPIServiceConfig
}
