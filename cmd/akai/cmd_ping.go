package main

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/host"
	"github.com/probe-lab/akai/ipfs"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

var pingConfig = &config.Ping{
	Network: string(config.NetworkAvailTurin),
	Key:     "",
	Timeout: 60 * time.Second,
}

var cmdPing = &cli.Command{
	Name:   "ping",
	Usage:  "Pings any Key from the given network's DHT",
	Flags:  cmdPingFlags,
	Action: cmdPingAction,
}

var cmdPingFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "network",
		Aliases: []string{"n"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_PING_NETWORK")},
		},
		Usage:       "The network where the Akai will be launched.",
		DefaultText: config.ListAllNetworks(),
		Value:       pingConfig.Network,
		Destination: &pingConfig.Network,
		Action:      validateNetworkFlag,
	},
	&cli.StringFlag{
		Name:    "key",
		Aliases: []string{"k"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_PING_KEY")},
		},
		Usage:       "Key that the host will try to fetch from the DHT",
		Value:       pingConfig.Key,
		Destination: &pingConfig.Key,
	},
	&cli.DurationFlag{
		Name:    "timeout",
		Aliases: []string{"t"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_PING_TIMEOUT")},
		},
		Usage:       "Duration for the DHT find-providers operation (20s, 1min)",
		DefaultText: "20sec",
		Value:       pingConfig.Timeout,
		Destination: &pingConfig.Timeout,
	}}

func cmdPingAction(ctx context.Context, cmd *cli.Command) error {
	log.WithFields(log.Fields{
		"key":     pingConfig.Key,
		"network": pingConfig.Network,
		"timeout": pingConfig.Timeout,
	}).Info("requesting key from given DHT...")

	// parse key
	key, err := ipfs.CidFromString(pingConfig.Key)
	if err != nil {
		return errors.Wrap(err, "parsing given key")
	}

	// get network
	bootstapers, v1protocol, err := config.ConfigureNetwork(pingConfig.Network)
	if err != nil {
		return errors.Wrap(err, "extracting network info from given network")
	}

	// configure the host
	dhtHostConfig := host.DHTHostOpts{
		IP:           "0.0.0.0",        // default?
		Port:         9020,             // default?
		DialTimeout:  10 * time.Second, // this is the DialTimeout, not the timeout for the operation
		UserAgent:    config.ComposeAkaiUserAgent(),
		Bootstrapers: bootstapers,
		V1Protocol:   v1protocol,
	}

	// create a new libp2p host
	dhtHost, err := host.NewDHTHost(ctx, dhtHostConfig)
	if err != nil {
		return errors.Wrap(err, "generating dht host")
	}

	findCtx, cancel := context.WithTimeout(ctx, pingConfig.Timeout)
	defer cancel()

	opDuration, providers, err := dhtHost.FindProviders(findCtx, key)
	if err != nil {
		return errors.Wrap(err, "finding providers for key")
	}

	log.WithFields(log.Fields{
		"duration":    opDuration,
		"n_providers": len(providers),
		"peer_ids":    listPeerIDsFromAddrsInfos(providers),
	}).Info("find providers operation done")

	return nil
}

func listPeerIDsFromAddrsInfos(addrs []peer.AddrInfo) []peer.ID {
	peerIDs := make([]peer.ID, len(addrs), len(addrs))

	for i, addr := range addrs {
		peerIDs[i] = addr.ID
	}
	return peerIDs
}
