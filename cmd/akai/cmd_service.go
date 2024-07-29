package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db"
	"github.com/urfave/cli/v3"
)

var serviceConfig = &config.Service{
	Network: db.Network{Protocol: config.ProtocolIPFS, NetworkName: config.NetworkNameIPFSAmino}.String(),
}

var _ = serviceConfig

var cmdService = &cli.Command{
	Name:   "service",
	Usage:  "Listen to gossipsub topics of the Ethereum network",
	Flags:  cmdServiceFlags,
	Action: cmdServiceAction,
}

var cmdServiceFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "network",
		Aliases: []string{"n"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_SERVICE_NETWORK")},
		},
		Usage:       "The network where the Akai will be launched.",
		DefaultText: config.ListAllNetworkCombinations(),
		Value:       pingConfig.Network,
		Destination: &pingConfig.Network,
		Action:      validateNetworkFlag,
	},
}

func validateNetworkFlag(ctx context.Context, cli *cli.Command, s string) error {
	for protocol, networkNames := range config.AvailableProtocols {
		for _, networkName := range networkNames {
			network := db.Network{Protocol: protocol, NetworkName: networkName}
			if strings.ToUpper(s) == network.String() {
				return nil
			}
		}
	}
	return fmt.Errorf(" given network %s not valid for supported ones", s)
}

func cmdServiceAction(ctx context.Context, cmd *cli.Command) error {
	network := config.NetworkFromStr(pingConfig.Network)
	slog.Info(fmt.Sprintf("running Akai on %s network", network))

	return nil
}
