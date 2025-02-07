package config

import (
	"time"

	record "github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/pkg/errors"
	"github.com/probe-lab/akai/db/models"
)

// DHT specs
const (
	KeyDelimiter      string = ":"
	CoordinatesPerKey int    = 3
)

// Chain specs
const (
	BlockIntervalTarget = 20 * time.Second
	BlockTTL            = 24 * time.Hour
)

var (
	// sampling delay parameters
	AvailDelayBase       = 3 * time.Minute
	AvailDelayMultiplier = 2

	// block every 20 seconds that should last a day (extra of 1 blob more to make sure we hit the cache)
	AvailBlobsSetCacheSize int = 2 * (3 + 1) * 60 * 24
	// number of blocks keeping a max of 1024 segments each
	AvailSegmentsSetCacheSize int = AvailBlobsSetCacheSize * 1024
)

// DefaulIPFSNetworkConfig defines the default configuration for the IPFS network for Akai
var DefaultAvailNetworkConfig = &NetworkConfiguration{
	Network: models.Network{
		Protocol:    ProtocolAvail,
		NetworkName: NetworkNameMainnet,
	},

	// network parameters
	BootstrapPeers: BootstrappersToMaddr(BootstrapNodesAvailMainnet),
	AgentVersion:   "avail-light-client/light-client/1.12.3/rust-client", // TODO: update or automate with github?

	// dht paramets
	HostType:        AminoLibp2pHost,
	DHTHostMode:     DHTClient,
	V1Protocol:      protocol.ID("/Avail/kad"),
	ProtocolPrefix:  nil,
	CustomValidator: &AvailKeyValidator{},

	// sampling specifics
	SamplingType:         SampleValue,
	BlobsSetCache:        AvailBlobsSetCacheSize,
	SegmentsSetCacheSize: AvailSegmentsSetCacheSize,
	DelayBase:            AvailDelayBase,
	DelayMultiplier:      AvailDelayMultiplier,
}

type AvailKeyValidator struct{}

var _ record.Validator = (*AvailKeyValidator)(nil)

func (v *AvailKeyValidator) Validate(key string, value []byte) error {
	_, err := AvailKeyFromString(key)
	if err != nil {
		return errors.Wrap(ErrorNonValidKey, key)
	}
	// TODO: check if the given size is bigger than allowed cell lenght?
	// not sure which is the format of the value
	// - Cell bytes ?
	// - Cell proof ?
	// - Cell bytes + Cell proof ?
	return nil
}

func (v *AvailKeyValidator) Select(key string, values [][]byte) (int, error) {
	if len(values) <= 0 {
		return 0, ErrorNoValueForKey
	}
	err := v.Validate(key, values[0])
	// there should be only one value for a given key, so no need to make extra stuff?
	return 1, err
}
