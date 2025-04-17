package main

import (
	"context"
	"fmt"

	"github.com/probe-lab/akai/celestia"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/core"
	"github.com/probe-lab/akai/db"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

var cmdDaemonCelestiaNamespaceConf = celestia.DefaultNetworkScrapperConfig

var cmdDaemonCelestiaNamespace = &cli.Command{
	Name:   "celestia",
	Usage:  "Tracks Celestia's Namespaces and checks the number of nodes supporting them at the DHT",
	Flags:  cmdDaemonCelestiaNamespaceFlags,
	Action: cmdDaemonCelestiaNamespaceAction,
}

var cmdDaemonCelestiaNamespaceFlags = []cli.Flag{
	&cli.StringFlag{
		Name: "api-http-host",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_API_HTTP_HOST")},
		},
		Usage:       "Port for the Akai's HTTP API server",
		Value:       cmdDaemonCelestiaNamespaceConf.AkaiAPIServiceConfig.Host,
		Destination: &cmdDaemonCelestiaNamespaceConf.AkaiAPIServiceConfig.Host,
	},
	&cli.IntFlag{
		Name: "api-http-port",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_API_HTTP_PORT")},
		},
		Usage:       "Port of Akai Daemon's HTTP API",
		Value:       cmdDaemonCelestiaNamespaceConf.AkaiAPIServiceConfig.Port,
		Destination: &cmdDaemonCelestiaNamespaceConf.AkaiAPIServiceConfig.Port,
	},
}

func cmdDaemonCelestiaNamespaceAction(ctx context.Context, cmd *cli.Command) error {
	log.WithFields(log.Fields{
		"akai-http-host": cmdDaemonCelestiaNamespaceConf.AkaiAPIServiceConfig.Host,
		"akai-http-port": cmdDaemonCelestiaNamespaceConf.AkaiAPIServiceConfig.Port,
	}).Info("starting celestia namespace tracker...")
	defer log.Infof("stopped celestia-namespace-tracker for %s", cmdDaemonCelestiaNamespaceConf.Network)

	// set all network to be on the same one as the given one
	network := config.NetworkFromStr(daemonConfig.Network)
	if network.Protocol != config.ProtocolCelestia {
		return fmt.Errorf("the given network doesn't belong to the celestia protocol %s", daemonConfig.Network)
	}
	daemonConfig.DataSamplerConfig.Network = daemonConfig.Network

	networkConfig, err := config.ConfigureNetwork(network)
	if err != nil {
		return err
	}

	// start the database
	dbSer, err := db.NewDatabase(daemonConfig.DBconfig, networkConfig)
	if err != nil {
		return err
	}

	// get a DHThost for the given network
	dhtHost, err := core.NewDHTHost(ctx, networkConfig, daemonConfig.DHTHostConfig)
	if err != nil {
		return err
	}

	config.UpdateSamplingDetailFromNetworkConfig(&daemonConfig.DataSamplerConfig.AkaiSamplingDetails, networkConfig)
	if err != nil {
		return err
	}

	netScrapper, err := celestia.NewNetworkScrapper(cmdDaemonCelestiaNamespaceConf, dbSer)
	if err != nil {
		return err
	}

	dataSampler, err := core.NewDataSampler(daemonConfig.DataSamplerConfig, dbSer, netScrapper, dhtHost)
	if err != nil {
		return err
	}

	daemon, err := core.NewDaemon(daemonConfig, dhtHost, dbSer, dataSampler, netScrapper)
	if err != nil {
		return err
	}

	return daemon.Start(ctx)
}
