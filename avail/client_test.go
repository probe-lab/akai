package avail

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	localAvailTestIP   = "localhost"
	localAvailTestPort = 5000
	ConnectionTimeout  = 10 * time.Second
)

// API connection
func Test_AvailHttpApiClient(t *testing.T) {
	httpCli, testMainCtx, cancel := genTestApiCli(t)
	defer cancel()

	err := httpCli.CheckConnection(testMainCtx)
	require.NoError(t, err)
}

// API endpoints
func Test_AvailHttpV2Version(t *testing.T) {
	httpCli, testMainCtx, cancel := genTestApiCli(t)
	defer cancel()

	_, err := httpCli.GetV2Version(testMainCtx)
	require.NoError(t, err)
}

func Test_AvailHttpV2Status(t *testing.T) {
	httpCli, testMainCtx, cancel := genTestApiCli(t)
	defer cancel()

	_, err := httpCli.GetV2Status(testMainCtx)
	require.NoError(t, err)
}

func Test_AvailHttpV2BlockStatus(t *testing.T) {
	httpCli, testMainCtx, cancel := genTestApiCli(t)
	defer cancel()

	status, err := httpCli.GetV2Status(testMainCtx)
	require.NoError(t, err)

	// make sure we request the last one of the available ones
	_, err = httpCli.GetV2BlockStatus(testMainCtx, status.Blocks.AvailableRange.Last)
	require.NoError(t, err)
}

func Test_AvailHttpV2BlockHeader(t *testing.T) {
	httpCli, testMainCtx, cancel := genTestApiCli(t)
	defer cancel()

	status, err := httpCli.GetV2Status(testMainCtx)
	require.NoError(t, err)

	// make sure we request the last one of the available ones
	_, err = httpCli.GetV2BlockHeader(testMainCtx, status.Blocks.AvailableRange.Last)
	require.NoError(t, err)
}

// generics
func genTestApiCli(t *testing.T) (*HttpClient, context.Context, context.CancelFunc) {
	testMainCtx, cancel := context.WithCancel(context.Background())

	opts := HttpClientConfig{
		IP:      localAvailTestIP,
		Port:    localAvailTestPort,
		Timeout: ConnectionTimeout,
	}

	httpCli, err := NewHttpCli(testMainCtx, opts)
	require.NoError(t, err)
	return httpCli, testMainCtx, cancel
}
