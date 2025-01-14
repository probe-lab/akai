package core

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"go.opentelemetry.io/otel"

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
	key, err := config.AvailKeyFromString(fmt.Sprintf("%d:%d:%d", block, row, column))
	require.NoError(t, err)

	value := []byte("avail test cell")

	time.Sleep(2 * time.Second)
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
	network := models.Network{Protocol: config.ProtocolLocal, NetworkName: config.NetworkNameCustom}
	networkConfig, err := config.ConfigureNetwork(network)
	require.NoError(t, err)

	// base parameters
	port := 9020
	baseAgentVersion := networkConfig.AgentVersion
	networkConfig.AgentVersion = fmt.Sprintf("%s_%d", baseAgentVersion, port)
	networkConfig.DHTHostMode = config.DHTServer

	// get the bootstrapper
	hosts := make([]DHTHost, nodeNumbers+1)
	bootstraperNode := composeDHTHost(t, ctx, 9020, networkConfig)
	hosts[0] = bootstraperNode

	// bootstraper info
	bAi := peer.AddrInfo{
		ID:    bootstraperNode.Host().ID(),
		Addrs: make([]multiaddr.Multiaddr, 0),
	}
	bAi.Addrs = bootstraperNode.Host().Addrs()
	networkConfig.BootstrapPeers = []peer.AddrInfo{bAi}

	// compose nodes and connect them to the bootstrapper
	for i := int64(1); i <= nodeNumbers; i++ {
		// dht_host specifics
		port := 9020 + i
		networkConfig.AgentVersion = fmt.Sprintf("%s_%d", baseAgentVersion, port)

		// create dht_host
		host := composeDHTHost(t, ctx, port, networkConfig)
		connecteHostToPeer(t, ctx, host, bAi)
		testDirectConnection(t, host, bAi)
		hosts[i] = host
	}
	return hosts
}

func composeDHTHost(t *testing.T, ctx context.Context, port int64, netCfg *config.NetworkConfiguration) DHTHost {
	dhtHostOpts := &config.CommonDHTHostOpts{
		ID:          0,
		IP:          "127.0.0.1",      // default?
		Port:        port,             // default?
		DialTimeout: 10 * time.Second, // this is the DialTimeout, not the timeout for the operation
		Meter:       otel.GetMeterProvider().Meter("akai_host"),
	}
	dhtHost, err := NewDHTHost(ctx, netCfg, dhtHostOpts)
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
