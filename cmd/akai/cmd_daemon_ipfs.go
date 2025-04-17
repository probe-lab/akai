package main

import (
	"context"
	"fmt"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/core"
	"github.com/probe-lab/akai/db"
	"github.com/probe-lab/akai/ipfs"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

var cmdDaemonIPFSConf = ipfs.DefaultNetworkScrapperConfig

var cmdDaemonIPFS = &cli.Command{
	Name:   "ipfs",
	Usage:  "Tracks IPFS CIDs over the API and checks their availability in the DHT",
	Flags:  cmdDaemonIPFSFlags,
	Action: cmdDaemonIPFSAction,
}

var cmdDaemonIPFSFlags = []cli.Flag{
	&cli.StringFlag{
		Name: "api-http-host",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_API_HTTP_HOST")},
		},
		Usage:       "Port for the Akai's HTTP API server",
		Value:       cmdDaemonIPFSConf.AkaiAPIServiceConfig.Host,
		Destination: &cmdDaemonIPFSConf.AkaiAPIServiceConfig.Host,
	},
	&cli.IntFlag{
		Name: "api-http-port",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_API_HTTP_PORT")},
		},
		Usage:       "Port of Akai Daemon's HTTP API",
		Value:       cmdDaemonIPFSConf.AkaiAPIServiceConfig.Port,
		Destination: &cmdDaemonIPFSConf.AkaiAPIServiceConfig.Port,
	},
}

func cmdDaemonIPFSAction(ctx context.Context, cmd *cli.Command) error {
	log.WithFields(log.Fields{
		"akai-http-host": cmdDaemonIPFSConf.AkaiAPIServiceConfig.Host,
		"akai-http-port": cmdDaemonIPFSConf.AkaiAPIServiceConfig.Port,
	}).Info("starting IPFS CID tracker...")
	defer log.Infof("stopped ipfs-namespace-tracker for %s", cmdDaemonIPFSConf.Network)

	// set all network to be on the same one as the given one
	network := config.NetworkFromStr(daemonConfig.Network)
	if network.Protocol != config.ProtocolIPFS {
		return fmt.Errorf("the given network doesn't belong to the ipfs protocol %s", daemonConfig.Network)
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

	netScrapper, err := ipfs.NewNetworkScrapper(cmdDaemonIPFSConf, dbSer)
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
