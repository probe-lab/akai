package main

import (
	"time"

	"github.com/urfave/cli/v3"

	"github.com/probe-lab/akai/config"
)

var findOP = &config.AkaiFindOP{
	Network: config.Network{Protocol: config.ProtocolIPFS, NetworkName: config.NetworkNameAmino}.String(),
	Key:     "",
	Timeout: 60 * time.Second,
	Retries: 3,
}

var cmdFindOP = &cli.Command{
	Name:  "find",
	Usage: "Performs a single operation for any Key from the given network's DHT",
	Flags: cmdFindOPFlags,
	Commands: []*cli.Command{
		cmdFindClosests,
		cmdFindProviders,
		cmdFindValue,
		cmdFindPeers,
		cmdFindPeerInfo,
		cmdFindIPNS,
	},
}

var cmdFindOPFlags = []cli.Flag{
	&cli.StringFlag{
		Name:        "network",
		Aliases:     []string{"n"},
		Usage:       "The network where the Akai will be launched.",
		DefaultText: config.ListAllNetworkCombinations(),
		Value:       findOP.Network,
		Destination: &findOP.Network,
		Action:      validateNetworkFlag,
	},
	&cli.StringFlag{
		Name:        "key",
		Aliases:     []string{"k"},
		Usage:       "Key for the DHT operation that will be taken place (CID, peerID, key, etc.)",
		Value:       findOP.Key,
		Destination: &findOP.Key,
	},
	&cli.DurationFlag{
		Name:        "timeout",
		Aliases:     []string{"t"},
		Usage:       "Duration for the DHT find-providers operation (20s, 1min)",
		DefaultText: "20sec",
		Value:       findOP.Timeout,
		Destination: &findOP.Timeout,
	},
	&cli.IntFlag{
		Name:        "retries",
		Aliases:     []string{"r"},
		Usage:       "Number of attempts that akai will try to fetch the content",
		Value:       findOP.Retries,
		Destination: &findOP.Retries,
	},
}
