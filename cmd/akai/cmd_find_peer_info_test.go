package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/core"
)

func Test_cmdFindPeerInfoAction_Success(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := composeDemoTestNetwork(ctx, t, 3)
	defer stopTestNetwork(t, hosts)

	time.Sleep(2 * time.Second)

	targetHost := hosts[1]
	targetPeerID := targetHost.Host().ID().String()

	oldFindOP := findOP
	findOP = &config.AkaiFindOP{
		Network: config.DefaultNetwork.String(),
		Key:     targetPeerID,
		Timeout: 10 * time.Second,
		Retries: 3,
	}
	defer func() { findOP = oldFindOP }()

	cmd := &cli.Command{}
	err := cmdFindPeerInfoAction(ctx, cmd)
	require.NoError(t, err)
}

func Test_cmdFindPeerInfoAction_InvalidPeerID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := composeDemoTestNetwork(ctx, t, 2)
	defer stopTestNetwork(t, hosts)

	oldFindOP := findOP
	findOP = &config.AkaiFindOP{
		Network: config.DefaultNetwork.String(),
		Key:     "invalid-peer-id",
		Timeout: 5 * time.Second,
		Retries: 1,
	}
	defer func() { findOP = oldFindOP }()

	cmd := &cli.Command{}
	err := cmdFindPeerInfoAction(ctx, cmd)
	require.Error(t, err)
}

func Test_cmdFindPeerInfoAction_PeerNotFound(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := composeDemoTestNetwork(ctx, t, 2)
	defer stopTestNetwork(t, hosts)

	oldFindOP := findOP
	findOP = &config.AkaiFindOP{
		Network: config.DefaultNetwork.String(),
		Key:     "12D3KooWNonExistentPeerID123456789abcdef",
		Timeout: 2 * time.Second,
		Retries: 1,
	}
	defer func() { findOP = oldFindOP }()

	cmd := &cli.Command{}
	err := cmdFindPeerInfoAction(ctx, cmd)
	require.Error(t, err)
	require.Contains(t, err.Error(), "operation couldn't report any successful result")
}

func Test_cmdFindPeerInfoAction_InvalidNetwork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := composeDemoTestNetwork(ctx, t, 2)
	defer stopTestNetwork(t, hosts)

	targetHost := hosts[1]
	targetPeerID := targetHost.Host().ID().String()

	oldFindOP := findOP
	findOP = &config.AkaiFindOP{
		Network: "invalid-network",
		Key:     targetPeerID,
		Timeout: 5 * time.Second,
		Retries: 1,
	}
	defer func() { findOP = oldFindOP }()

	cmd := &cli.Command{}
	err := cmdFindPeerInfoAction(ctx, cmd)
	require.Error(t, err)
}

func Test_cmdFindPeerInfoAction_TimeoutExceeded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := composeDemoTestNetwork(ctx, t, 3)
	defer stopTestNetwork(t, hosts)

	targetHost := hosts[1]
	targetPeerID := targetHost.Host().ID().String()

	oldFindOP := findOP
	findOP = &config.AkaiFindOP{
		Network: config.DefaultNetwork.String(),
		Key:     targetPeerID,
		Timeout: 1 * time.Nanosecond,
		Retries: 1,
	}
	defer func() { findOP = oldFindOP }()

	cmd := &cli.Command{}
	err := cmdFindPeerInfoAction(ctx, cmd)
	require.Error(t, err)
	require.Contains(t, err.Error(), "operation couldn't report any successful result")
}

func Test_cmdFindPeerInfoAction_MultipleRetries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := composeDemoTestNetwork(ctx, t, 3)
	defer stopTestNetwork(t, hosts)

	time.Sleep(2 * time.Second)

	targetHost := hosts[1]
	targetPeerID := targetHost.Host().ID().String()

	oldFindOP := findOP
	findOP = &config.AkaiFindOP{
		Network: config.DefaultNetwork.String(),
		Key:     targetPeerID,
		Timeout: 10 * time.Second,
		Retries: 3,
	}
	defer func() { findOP = oldFindOP }()

	cmd := &cli.Command{}
	err := cmdFindPeerInfoAction(ctx, cmd)
	require.NoError(t, err)
}

func composeDemoTestNetwork(ctx context.Context, t *testing.T, nodeNumbers int64) []core.DHTHost {
	network := config.Network{Protocol: config.ProtocolLocal, NetworkName: config.NetworkNameCustom}
	networkConfig, err := config.ConfigureNetwork(network)
	require.NoError(t, err)

	port := 9020
	baseAgentVersion := networkConfig.AgentVersion
	networkConfig.AgentVersion = fmt.Sprintf("%s_%d", baseAgentVersion, port)
	networkConfig.DHTHostMode = config.DHTServer

	hosts := make([]core.DHTHost, nodeNumbers+1)
	bootstraperNode := composeDHTTestHost(t, ctx, 9020, networkConfig)
	hosts[0] = bootstraperNode

	bootstraperAddrInfo := bootstraperNode.Host().Peerstore().PeerInfo(bootstraperNode.Host().ID())
	networkConfig.BootstrapPeers = []peer.AddrInfo{bootstraperAddrInfo}

	for i := int64(1); i <= nodeNumbers; i++ {
		port := 9020 + i
		networkConfig.AgentVersion = fmt.Sprintf("%s_%d", baseAgentVersion, port)

		host := composeDHTTestHost(t, ctx, port, networkConfig)
		err := host.Host().Connect(ctx, bootstraperAddrInfo)
		require.NoError(t, err)

		connections := host.Host().Network().ConnsToPeer(bootstraperAddrInfo.ID)
		require.Greater(t, len(connections), 0)

		hosts[i] = host
	}
	return hosts
}

func composeDHTTestHost(t *testing.T, ctx context.Context, port int64, netCfg *config.NetworkConfiguration) core.DHTHost {
	dhtHostOpts := &config.CommonDHTHostOpts{
		ID:          0,
		IP:          "127.0.0.1",
		Port:        port,
		DialTimeout: 10 * time.Second,
		Meter:       nil,
	}
	dhtHost, err := core.NewDHTHost(ctx, netCfg, dhtHostOpts)
	require.NoError(t, err)
	return dhtHost
}

func stopTestNetwork(t *testing.T, hosts []core.DHTHost) {
	for _, h := range hosts {
		err := h.Host().Close()
		require.NoError(t, err)
	}
}
