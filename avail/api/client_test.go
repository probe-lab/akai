package api

import (
	"context"
	"testing"
	"time"

	"github.com/probe-lab/akai/config"
	"github.com/stretchr/testify/require"
)

var (
	localAvailTestIP   = "localhost"
	localAvailTestPort = int64(5000)
	ConnectionTimeout  = 10 * time.Second
)

// API connection
func Test_AvailHttpAPIClient(t *testing.T) {
	httpCli, testMainCtx, cancel := genTestAPICli(t)
	defer cancel()

	err := httpCli.CheckConnection(testMainCtx)
	require.NoError(t, err)
}

// API endpoints
func Test_AvailHttpV2Version(t *testing.T) {
	httpCli, testMainCtx, cancel := genTestAPICli(t)
	defer cancel()

	_, err := httpCli.GetV2Version(testMainCtx)
	require.NoError(t, err)
}

func Test_AvailHttpV2Status(t *testing.T) {
	httpCli, testMainCtx, cancel := genTestAPICli(t)
	defer cancel()

	_, err := httpCli.GetV2Status(testMainCtx)
	require.NoError(t, err)
}

func Test_AvailHttpV2BlockStatus(t *testing.T) {
	httpCli, testMainCtx, cancel := genTestAPICli(t)
	defer cancel()

	status, err := httpCli.GetV2Status(testMainCtx)
	require.NoError(t, err)

	// make sure we request the last one of the available ones
	_, err = httpCli.GetV2BlockStatus(testMainCtx, status.Blocks.AvailableRange.Last)
	require.NoError(t, err)
}

func Test_AvailHttpV2BlockHeader(t *testing.T) {
	httpCli, testMainCtx, cancel := genTestAPICli(t)
	defer cancel()

	status, err := httpCli.GetV2Status(testMainCtx)
	require.NoError(t, err)

	// make sure we request the last one of the available ones
	_, err = httpCli.GetV2BlockHeader(testMainCtx, status.Blocks.AvailableRange.Last)
	require.NoError(t, err)
}

// generics
func genTestAPICli(t *testing.T) (*HTTPClient, context.Context, context.CancelFunc) {
	testMainCtx, cancel := context.WithCancel(context.Background())

	opts := config.AvailHttpAPIClient{
		IP:      localAvailTestIP,
		Port:    localAvailTestPort,
		Timeout: ConnectionTimeout,
	}

	httpCli, err := NewHTTPCli(opts)
	require.NoError(t, err)
	return httpCli, testMainCtx, cancel
}
