package api

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
)

const (
	V2StatusEndpoint = "/v2/status"
)

type V2Status struct {
	Modes       []string `json:"modes"`
	GenesisHash string   `json:"genesis_hash"`
	Network     string   `json:"network"`
	Blocks      struct {
		Latest         uint64 `json:"latest"`
		AvailableRange struct {
			First uint64 `json:"first"`
			Last  uint64 `json:"last"`
		} `json:"available"`
	} `json:"blocks"`
}

func (c *HTTPClient) GetV2Status(ctx context.Context) (V2Status, error) {
	// make Http request
	resp, err := c.get(ctx, V2StatusEndpoint, "")
	if err != nil {
		return V2Status{}, errors.Wrap(err, "requesting v2-status")
	}

	// parse response into V2Version
	var status V2Status
	err = json.Unmarshal(resp, &status)
	if err != nil {
		return V2Status{}, errors.Wrap(err, "unmarshaling v2-status from http request")
	}

	return status, nil
}
