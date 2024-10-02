package api

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
)

const (
	V2VersionEndpoint = "/v2/version"
)

type V2Version struct {
	Version        string `json:"version"`
	NetworkVersion string `json:"network_version"`
}

func (c *HTTPClient) GetV2Version(ctx context.Context) (V2Version, error) {
	// make Http request
	resp, err := c.get(ctx, V2VersionEndpoint, "")
	if err != nil {
		return V2Version{}, errors.Wrap(err, "requesting v2-version")
	}

	// parse response into V2Version
	var version V2Version
	err = json.Unmarshal(resp, &version)
	if err != nil {
		return V2Version{}, errors.Wrap(err, "unmarshaling v2-version from http request")
	}

	return version, nil
}
