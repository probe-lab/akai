package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	record "github.com/libp2p/go-libp2p-record"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/metric"
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

// Key constants
const (
	// https://github.com/availproject/avail-light/blob/eb1f40dcb9e71741e2f26786029ceec41a9755ca/core/src/types.rs#L31
	CallSize  int = 32 // bytes
	ProofSize int = 48 // bytes
)

var (
	DefaultAvailNetwork = Network{Protocol: ProtocolAvail, NetworkName: NetworkNameMainnet}

	// sampling delay parameters
	AvailDelayBase       = 3 * time.Minute
	AvailDelayMultiplier = 2

	// block every 20 seconds that should last a day (extra of 1 blob more to make sure we hit the cache)
	AvailBlobsSetCacheSize int = 2 * (3 + 1) * 60 * 24
	// number of blocks keeping a max of 1024 segments each
	AvailSegmentsSetCacheSize int = AvailBlobsSetCacheSize * 1024

	// Key errors
	ErrorNonValidKey   = fmt.Errorf("key doens't match avail standards")
	ErrorNoValueForKey = fmt.Errorf("no value found for given key")
)

// Block Tracker configuration
type AvailBlockTrackerConfig struct {
	Network string

	// for the API interaction
	AvailAPIconfig *AvailAPIConfig

	// metrics for the service
	Meter metric.Meter
}

// Block Tracker configuration
type AvailNetworkScrapperConfig struct {
	Network              string
	TrackBlocksOnDB      bool
	NotChannelBufferSize int
	SamplerNotifyTimeout time.Duration

	// defines for how long will the block tracker submit segments to the akai-API
	TrackDuration time.Duration
	// Defines every how ofter the block tracker will start to track again the segments
	TrackInterval time.Duration

	// for the API interaction
	BlockTrackerCfg *AvailBlockTrackerConfig
}

type AvailAPIConfig struct {
	Host    string
	Port    int64
	Timeout time.Duration
}

// DefaulIPFSNetworkConfig defines the default configuration for the IPFS network for Akai
var DefaultAvailNetworkConfig = &NetworkConfiguration{
	Network: Network{
		Protocol:    ProtocolAvail,
		NetworkName: NetworkNameMainnet,
	},

	// network parameters
	BootstrapPeers: BootstrappersToMaddr(BootstrapNodesAvailMainnet),
	AgentVersion:   "avail-light-client/light-client/1.12.11/rust-client", // TODO: update or automate with github?

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

type AvailKey struct {
	Block  uint64
	Row    uint64
	Column uint64
}

func AvailKeyFromString(str string) (AvailKey, error) {
	block, row, column, err := getCoordinatesFromStr(str)
	if err != nil {
		return AvailKey{}, fmt.Errorf("string %s doesn't have right format or coordinates", str)
	}
	return AvailKey{
		Block:  block,
		Row:    row,
		Column: column,
	}, nil
}

func (k AvailKey) String() string {
	return fmt.Sprintf("%d:%d:%d", k.Block, k.Row, k.Column)
}

func (k *AvailKey) Bytes() []byte {
	return []byte(k.String())
}

// DHTKey returns the string representation of the of the DHT Key
// TODO: This is likely to be deprecated as k.String() should be the right dht key
func (k *AvailKey) DHTKey() string {
	bytes32 := [32]byte{}
	for i, char := range k.String() {
		bytes32[i] = byte(char)
	}
	return string(bytes32[:32])
}

func (k *AvailKey) Hash() multihash.Multihash {
	mh, err := multihash.Sum([]byte(k.String()), multihash.SHA2_256, len(k.String()))
	if err != nil {
		log.Panic(errors.Wrap(err, "composing sha256 key from avail key"))
	}
	return mh
}

func (k AvailKey) IsZero() bool {
	return k.Block == 0 &&
		k.Row == 0 &&
		k.Column == 0
}

func getCoordinatesFromStr(str string) (block uint64, row uint64, column uint64, err error) {
	KeyValues := strings.Split(str, KeyDelimiter)
	if len(KeyValues) != CoordinatesPerKey {
		return block, row, column, fmt.Errorf("string %s doesn't have right format or coordinates", str)
	}
	for idx, val := range KeyValues {
		switch idx {
		case 0:
			block, err = strToUint64(val)
		case 1:
			row, err = strToUint64(val)
		case 2:
			column, err = strToUint64(val)
		default:
			err = fmt.Errorf("unexpected idx at avail key %d", idx)
		}
		if err != nil {
			return block, row, column, fmt.Errorf("unable to convert avail key's coordinate to uint64 (%s)", val)
		}
	}
	return block, row, column, nil
}

func strToUint64(s string) (uint64, error) {
	coordinate, err := strconv.Atoi(s)
	if err != nil {
		return uint64(0), err
	}
	return uint64(coordinate), nil
}
