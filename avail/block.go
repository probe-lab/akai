package avail

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	mc "github.com/multiformats/go-multicodec"
	mh "github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
	akai_api "github.com/probe-lab/akai/api"
	"github.com/probe-lab/akai/avail/api"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/models"
)

type BlockOption func(*Block) error

type Block struct {
	ReceivedAt     int64
	Hash           mh.Multihash
	ParentHash     mh.Multihash
	Number         uint64
	StateRoot      mh.Multihash
	ExtrinsicsRoot mh.Multihash
	Extension      BlockExtension
}

type BlockExtension struct {
	Rows        uint64
	Columns     uint64
	DataRoot    mh.Multihash
	Commitments []string // TODO: parse them correctly?
	Size        uint64
	Start       uint64
	AppLookup   BlockAppLookup
}

type BlockAppLookup struct {
	Size  uint64
	Index []BlockAppLookupIndex
}

type BlockAppLookupIndex struct {
	AppID uint64
	Start uint64
}

func NewBlock(opts ...BlockOption) (*Block, error) {
	block := &Block{
		Extension: BlockExtension{
			Commitments: make([]string, 0),
			AppLookup: BlockAppLookup{
				Index: make([]BlockAppLookupIndex, 0),
			},
		},
	}

	for _, opt := range opts {
		err := opt(block)
		if err != nil {
			return nil, err
		}
	}

	return block, nil
}

func (b *Block) Cid() cid.Cid {
	return cid.NewCidV1(uint64(mc.Raw), b.Hash)
}

func (b *Block) ToAkaiAPIBlob(network models.Network, fillSegments bool) akai_api.Blob {
	timestamp := time.Unix(int64(b.ReceivedAt), 0).UTC()
	blob := akai_api.Blob{
		Timestamp:   timestamp,
		Network:     network,
		Number:      b.Number,
		Hash:        b.Hash.HexString(),
		ParentHash:  b.ParentHash.HexString(),
		Rows:        b.Extension.Rows,
		Columns:     b.Extension.Columns,
		Segments:    make([]akai_api.BlobSegment, 0),
		Metadata:    make(map[string]any, 0),
		SampleUntil: timestamp.Add(config.BlockTTL + 3*time.Hour), // add 3 hours extra to ensure that we sample also after the 24 hour mark
	}
	// if needed, add all the inner segments into the blob struct for the API (make 1 single API call)
	if fillSegments {
		for row := 0; row < int(blob.Rows); row++ {
			for col := 0; col < int(blob.Columns); col++ {
				segmentKey := config.AvailKey{
					Block:  blob.Number,
					Row:    uint64(row),
					Column: uint64(col),
				}
				segment := akai_api.BlobSegment{
					Timestamp:   blob.Timestamp,
					BlobNumber:  segmentKey.Block,
					Row:         segmentKey.Row,
					Column:      segmentKey.Column,
					Key:         segmentKey.String(),
					Bytes:       make([]byte, 0),
					SampleUntil: blob.SampleUntil,
				}
				blob.Segments = append(blob.Segments, segment)
			}
		}
	}
	return blob
}

func FromAPIBlockHeader(blockHeader api.V2BlockHeader) BlockOption {
	return func(block *Block) (err error) {
		// parse block-number
		block.Number = blockHeader.Number

		// parse hashes
		block.Hash, err = MultihashFromHexString(blockHeader.Hash)
		if err != nil {
			return errors.Wrap(err, "parsing block-hash as multihash")
		}
		block.ParentHash, err = MultihashFromHexString(blockHeader.ParentHash)
		if err != nil {
			return errors.Wrap(err, "parsing block parent-hash as multihash")
		}
		block.StateRoot, err = MultihashFromHexString(blockHeader.StateRoot)
		if err != nil {
			return errors.Wrap(err, "parsing block state-root as multihash")
		}
		block.ExtrinsicsRoot, err = MultihashFromHexString(blockHeader.ExtrinsicsRoot)
		if err != nil {
			return errors.Wrap(err, "parsing block extrinsic-root as multihash")
		}

		// Parse extension
		block.Extension.Rows = blockHeader.Extension.Rows
		block.Extension.Columns = blockHeader.Extension.Columns
		copy(block.Extension.Commitments, blockHeader.Extension.Commitments)
		block.Extension.DataRoot, err = MultihashFromHexString(blockHeader.Extension.DataRoot)
		if err != nil {
			return errors.Wrap(err, "parsing block data-root as multihash")
		}

		// Parse app lookup
		block.Extension.AppLookup.Size = blockHeader.Extension.AppLookup.Size
		indexes := make([]BlockAppLookupIndex, len(blockHeader.Extension.AppLookup.Index))
		for i, index := range blockHeader.Extension.AppLookup.Index {
			indexes[i].AppID = index.AppID
			indexes[i].Start = index.Start
		}

		return err
	}
}

func MultihashFromHexString(s string) (mh.Multihash, error) {
	s = strings.TrimPrefix(s, "0x")

	buf, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return mh.EncodeName(buf, "sha2")
}
