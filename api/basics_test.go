package api

import (
	"context"
	"testing"
	"time"

	"github.com/probe-lab/akai/db/models"
	"github.com/stretchr/testify/require"
)

func Test_Service(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(1 * time.Second)
	}()
	_, cli := basicServiceAndClient(t, ctx)
	ensureClientServerConnection(t, ctx, cli)
}

func Test_SupportedNetworks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(1 * time.Second)
	}()
	_, cli := basicServiceAndClient(t, ctx)
	ensureClientServerConnection(t, ctx, cli)

	serverNet := DefaulServiceConfig.Network
	network, err := cli.GetSupportedNetworks(ctx)
	require.NoError(t, err)
	require.Equal(t, serverNet, network.Network.String())
}

func Test_PostNewBlob(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(1 * time.Second)
	}()
	_, cli := basicServiceAndClient(t, ctx)
	ensureClientServerConnection(t, ctx, cli)

	// send a valid item
	blob := Blob{
		Network:    models.NetworkFromStr(DefaulServiceConfig.Network),
		Number:     1,
		Hash:       "0xHASH",
		ParentHash: "OxPARENTHASH",
		Rows:       1,
		Columns:    1,
		Segments:   make([]BlobSegment, 0),
		Metadata:   make(map[string]any),
	}
	err := cli.PostNewBlob(ctx, blob)
	require.NoError(t, err)
}

func Test_PostNewSegment(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(1 * time.Second)
	}()
	_, cli := basicServiceAndClient(t, ctx)
	ensureClientServerConnection(t, ctx, cli)

	segment := BlobSegment{
		BlobNumber: 1,
		Row:        1,
		Column:     1,
		Key:        "0xSEGMENT",
		Bytes:      make([]byte, 0),
	}
	err := cli.PostNewSegment(ctx, segment)
	require.NoError(t, err)
}

func Test_PostNewSegments(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(1 * time.Second)
	}()
	_, cli := basicServiceAndClient(t, ctx)
	ensureClientServerConnection(t, ctx, cli)

	segments := []BlobSegment{
		{
			BlobNumber: 1,
			Row:        1,
			Column:     1,
			Key:        "0xSEGMENT",
			Bytes:      make([]byte, 0),
		},
		{
			BlobNumber: 2,
			Row:        1,
			Column:     1,
			Key:        "0xSEGMENT_2",
			Bytes:      make([]byte, 0),
		},
	}

	err := cli.PostNewSegments(ctx, segments)
	require.NoError(t, err)
}

func ensureClientServerConnection(t *testing.T, ctx context.Context, cli *Client) {
	err := cli.Ping(ctx)
	require.NoError(t, err)
}

func basicServiceAndClient(t *testing.T, ctx context.Context) (*Service, *Client) {
	// create and init the API server
	serCfg := DefaulServiceConfig
	ser, err := NewService(serCfg)
	require.NoError(t, err)
	go ser.Serve(ctx)
	time.Sleep(1 * time.Second) // make sure we give enough time to init the host

	// create and init the API client
	cliCfg := DefaultClientConfig
	cli, err := NewClient(ctx, cliCfg)
	require.NoError(t, err)
	return ser, cli
}
