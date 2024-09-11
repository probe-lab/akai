package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
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
