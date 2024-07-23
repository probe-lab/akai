package avail

import (
	"fmt"
	"strconv"
	"strings"

	record "github.com/libp2p/go-libp2p-record"
	"github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
	"github.com/probe-lab/akai/avail/api"
	log "github.com/sirupsen/logrus"
)

var (
	ErrorNonValidKey   = fmt.Errorf("key doens't match avail standards")
	ErrorNoValueForKey = fmt.Errorf("no value found for given key")
)

const (
	// https://github.com/availproject/avail-light/blob/eb1f40dcb9e71741e2f26786029ceec41a9755ca/core/src/types.rs#L31
	CallSize  int = 32 // bytes
	ProofSize int = 48 // bytes
)

type Key struct {
	Block  uint64
	Row    uint64
	Column uint64
}

func KeyFromString(str string) (Key, error) {
	block, row, column, err := getCoordinatesFromStr(str)
	if err != nil {
		return Key{}, fmt.Errorf("string %s doesn't have right format or coordinates", str)
	}
	return Key{
		Block:  block,
		Row:    row,
		Column: column,
	}, nil
}

func (k Key) String() string {
	return fmt.Sprintf("%d:%d:%d", k.Block, k.Row, k.Column)
}

func (k *Key) Bytes() []byte {
	return []byte(k.String())
}

// TODO: This is likely to be deprecated as k.String() should be the right dht key
func (k *Key) DHTKey() string {
	bytes32 := [32]byte{}
	for i, char := range k.String() {
		bytes32[i] = byte(char)
	}
	return string(bytes32[:32])
}

func (k *Key) Hash() multihash.Multihash {
	mh, err := multihash.Sum([]byte(k.String()), multihash.SHA2_256, len(k.String()))
	if err != nil {
		log.Panic(errors.Wrap(err, "composing sha256 key from avail key"))
	}
	return mh
}

func (k Key) IsZero() bool {
	return k.Block == 0 &&
		k.Row == 0 &&
		k.Column == 0
}

func getCoordinatesFromStr(str string) (block uint64, row uint64, column uint64, err error) {
	KeyValues := strings.Split(str, api.KeyDelimiter)
	if len(KeyValues) != api.CoordinatesPerKey {
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

type KeyValidator struct{}

var _ record.Validator = (*KeyValidator)(nil)

func (v *KeyValidator) Validate(key string, value []byte) error {
	_, err := KeyFromString(key)
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

func (v *KeyValidator) Select(key string, values [][]byte) (int, error) {
	if len(values) <= 0 {
		return 0, ErrorNoValueForKey
	}
	err := v.Validate(key, values[0])
	// there should be only one value for a given key, so no need to make extra stuff?
	return 0, err
}
