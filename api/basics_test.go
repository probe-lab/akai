package api

import (
	"context"
	"testing"
	"time"

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

func Test_PostNewBlock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(1 * time.Second)
	}()
	_, cli := basicServiceAndClient(t, ctx)
	ensureClientServerConnection(t, ctx, cli)

	// send a valid item
	block := Block{
		Network:    DefaulServiceConfig.Network,
		Number:     1,
		Hash:       "0xHASH",
		ParentHash: "OxPARENTHASH",
		Rows:       1,
		Columns:    1,
		Items:      make([]DASItem, 0),
		Metadata:   make(map[string]any),
	}
	err := cli.PostNewBlock(ctx, block)
	require.NoError(t, err)
}

func Test_PostNewitem(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(1 * time.Second)
	}()
	_, cli := basicServiceAndClient(t, ctx)
	ensureClientServerConnection(t, ctx, cli)

	item := DASItem{
		Timestamp: time.Now(),
		Network:   DefaulServiceConfig.Network,
		BlockLink: 1,
		Key:       "0xitem",
		Hash:      "0xhash",
		Row:       1,
		Column:    1,
	}
	err := cli.PostNewItem(ctx, item)
	require.NoError(t, err)
}

func Test_PostNewitems(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(1 * time.Second)
	}()
	_, cli := basicServiceAndClient(t, ctx)
	ensureClientServerConnection(t, ctx, cli)

	items := []DASItem{
		{
			BlockLink: 1,
			Row:       1,
			Column:    1,
			Key:       "0xitem",
		},
		{
			BlockLink: 2,
			Row:       1,
			Column:    1,
			Key:       "0xitem_2",
		},
	}

	err := cli.PostNewItems(ctx, items)
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
