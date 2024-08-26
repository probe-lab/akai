package core

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/probe-lab/akai/avail"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db/models"

	"github.com/stretchr/testify/require"
)

func Test_NetworkConnectivity(t *testing.T) {
	textCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := composeDemoDHTNetwork(textCtx, t, 5)
	defer stopTestNetwork(t, hosts)
}

func Test_AvailKeyPing(t *testing.T) {
	textCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := composeDemoDHTNetwork(textCtx, t, 5)
	defer stopTestNetwork(t, hosts)

	// compose an avail Key-Value pair
	// I guess this is the right format of the keys https://github.com/availproject/avail-light/blob/eb1f40dcb9e71741e2f26786029ceec41a9755ca/core/src/network/p2p/event_loop.rs#L118
	block := uint64(123)
	row := uint64(12)
	column := uint64(54)
	key, err := avail.KeyFromString(fmt.Sprintf("%d:%d:%d", block, row, column))
	require.NoError(t, err)

	value := []byte("avail test cell")

	// seed the DHT with such key
	seeder := hosts[1]
	_, err = seeder.PutValue(textCtx, key.String(), value)
	require.NoError(t, err)

	// retrieve it
	retriever := hosts[2]
	_, dhtValue, err := retriever.FindValue(textCtx, key.String())
	require.NoError(t, err)
	for idx, char := range dhtValue {
		require.Equal(t, char, value[idx])
	}

}

func composeDemoDHTNetwork(ctx context.Context, t *testing.T, nodeNumbers int64) []DHTHost {
	// get the bootstrapper
	hosts := make([]DHTHost, nodeNumbers+1)
	bootstraperNode := composeDHTHost(t, ctx, 9020, DHTServer)
	hosts[0] = bootstraperNode

	// bootstraper info
	bAi := peer.AddrInfo{
		ID:    bootstraperNode.Host().ID(),
		Addrs: make([]multiaddr.Multiaddr, 0),
	}
	bAi.Addrs = bootstraperNode.Host().Addrs()

	// compose nodes and connecte them to the bootstrapper
	for i := int64(1); i <= nodeNumbers; i++ {
		host := composeDHTHost(t, ctx, 9020+i, DHTServer)
		connecteHostToPeer(t, ctx, host, bAi)
		testDirectConnection(t, host, bAi)
		hosts[i] = host
	}
	return hosts
}

func composeDHTHost(t *testing.T, ctx context.Context, port int64, mode DHTHostType) DHTHost {
	network := models.Network{Protocol: config.ProtocolLocalCustom, NetworkName: config.NetworkNameLocalCustom}
	dhtHostOpts := CommonDHTOpts{
		IP:          "127.0.0.1",      // default?
		Port:        port,             // default?
		DialTimeout: 10 * time.Second, // this is the DialTimeout, not the timeout for the operation
		DHTMode:     mode,
		UserAgent:   fmt.Sprintf("%s_%d", config.ComposeAkaiUserAgent(network), port),
	}
	dhtHost, err := NewDHTHost(ctx, models.Network(network), dhtHostOpts)
	require.NoError(t, err)

	return dhtHost
}

func connecteHostToPeer(t *testing.T, ctx context.Context, h DHTHost, ai peer.AddrInfo) {
	err := h.Host().Connect(ctx, ai)
	require.NoError(t, err)
}

func testDirectConnection(t *testing.T, h DHTHost, ai peer.AddrInfo) {
	connections := h.Host().Network().ConnsToPeer(ai.ID)
	require.Greater(t, len(connections), 0)
}

func stopTestNetwork(t *testing.T, hosts []DHTHost) {
	for _, h := range hosts {
		err := h.Host().Close()
		require.NoError(t, err)
	}
}
