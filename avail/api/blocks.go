package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

var (
	V2BlockStatusEndpoint = "/v2/blocks/%d"
	V2BlockHeaderEndpoint = "/v2/blocks/%d/header"
)

// API docs: https://docs.availproject.org/docs/operate-a-node/run-a-light-client/light-client-api-reference#v2blocksblock_number

type V2BlockStatus struct {
	Status     string  `json:"status"`
	Confidence float64 `json:"confidence"`
}

func (c *HTTPClient) GetV2BlockStatus(ctx context.Context, block uint64) (V2BlockStatus, error) {
	resp, err := c.get(ctx, fmt.Sprintf(V2BlockStatusEndpoint, block), "")
	if err != nil {
		return V2BlockStatus{}, errors.Wrap(err, "requesting v2-block-status")
	}

	var blockStatus V2BlockStatus
	err = json.Unmarshal(resp, &blockStatus)
	if err != nil {
		return V2BlockStatus{}, errors.Wrap(err, "unmarshaling v2-block-status from http request")
	}

	return blockStatus, nil
}

// API docs: https://docs.availproject.org/docs/operate-a-node/run-a-light-client/light-client-api-reference#v2blocksblock_numberheader

type V2BlockHeader struct {
	Hash           string `json:"hash"`
	ParentHash     string `json:"parent_hash"`
	Number         uint64 `json:"number"`
	StateRoot      string `json:"state_root"`
	ExtrinsicsRoot string `json:"extrinsics_root"`
	Extension      struct {
		Rows        uint64   `json:"rows"`
		Columns     uint64   `json:"cols"`
		DataRoot    string   `json:"data_root"`
		Commitments []string `json:"commitment"`
		AppLookup   struct {
			Size  uint64
			Index []struct {
				AppID uint64 `json:"app_id"`
				Start uint64 `json:"start"`
			} `json:"index"`
		} `json:"app_lookup"`
	} `json:"extension"`
	Digest struct {
		Logs []string `json:"logs"`
	} `json:"digest"`
}

func (c *HTTPClient) GetV2BlockHeader(ctx context.Context, block uint64) (V2BlockHeader, error) {
	resp, err := c.get(ctx, fmt.Sprintf(V2BlockHeaderEndpoint, block), "")
	if err != nil {
		return V2BlockHeader{}, errors.Wrap(err, "requesting v2-block-header")
	}

	var blockHeader V2BlockHeader
	err = json.Unmarshal(resp, &blockHeader)
	if err != nil {
		errStr := string(resp)
		return V2BlockHeader{}, fmt.Errorf("v2-block-header from http request reported: %s (block %d)", errStr, block)
	}

	return blockHeader, nil
}
