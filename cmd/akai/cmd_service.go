package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/probe-lab/akai/config"
	"github.com/urfave/cli/v3"
)

var serviceConfig = &config.Service{
	Network: string(config.NetworkAvailTurin),
}

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
		DefaultText: config.ListAllNetworks(),
		Value:       serviceConfig.Network,
		Destination: &serviceConfig.Network,
		Action:      validateNetworkFlag,
	},
}

func validateNetworkFlag(ctx context.Context, cli *cli.Command, s string) error {
	networks := config.Networks()
	for _, value := range networks {
		if strings.ToUpper(s) == string(value) {
			return nil
		}
	}
	return fmt.Errorf(" given network %s not valid for supported ones", s)
}

func cmdServiceAction(ctx context.Context, cmd *cli.Command) error {
	network := config.NetworkFromStr(serviceConfig.Network)
	slog.Info(fmt.Sprintf("running Akai on %s network", network))

	return nil
}
