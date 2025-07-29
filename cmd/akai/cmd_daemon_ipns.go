package main

import (
	"context"
	"fmt"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/core"
	"github.com/probe-lab/akai/db"
	"github.com/probe-lab/akai/ipns"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

var cmdDaemonIPNSConf = ipns.DefaultNetworkScrapperConfig

var cmdDaemonIPNS = &cli.Command{
	Name:   "ipns",
	Usage:  "Tracks IPNS Records and DNS links over the API and checks their availability in the DHT",
	Flags:  cmdDaemonIPNSFlags,
	Action: cmdDaemonIPNSAction,
}

var cmdDaemonIPNSFlags = []cli.Flag{
	&cli.StringFlag{
		Name: "api-http-host",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_API_HTTP_HOST")},
		},
		Usage:       "Port for the Akai's HTTP API server",
		Value:       cmdDaemonIPNSConf.AkaiAPIServiceConfig.Host,
		Destination: &cmdDaemonIPNSConf.AkaiAPIServiceConfig.Host,
	},
	&cli.IntFlag{
		Name: "api-http-port",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_API_HTTP_PORT")},
		},
		Usage:       "Port of Akai Daemon's HTTP API",
		Value:       cmdDaemonIPNSConf.AkaiAPIServiceConfig.Port,
		Destination: &cmdDaemonIPNSConf.AkaiAPIServiceConfig.Port,
	},
	&cli.IntFlag{
		Name: "ipns-quorum",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_IPNS_QUORUM")},
		},
		Usage:       "Quorum target that will be use to fetch IPNS records from the DHT",
		Value:       cmdDaemonIPNSConf.Quorum,
		Destination: &cmdDaemonIPNSConf.Quorum,
	},
}

func cmdDaemonIPNSAction(ctx context.Context, cmd *cli.Command) error {
	log.WithFields(log.Fields{
		"akai-http-host": cmdDaemonIPNSConf.AkaiAPIServiceConfig.Host,
		"akai-http-port": cmdDaemonIPNSConf.AkaiAPIServiceConfig.Port,
		"ipns-quorum":    cmdDaemonIPNSConf.Quorum,
	}).Info("starting IPNS CID tracker...")
	defer log.Infof("stopped ipns-namespace-tracker for %s", cmdDaemonIPNSConf.Network)

	// set all network to be on the same one as the given one
	network := config.NetworkFromStr(daemonConfig.Network)
	if network.Protocol != config.ProtocolIPNS {
		return fmt.Errorf("the given network doesn't belong to the ipns protocol %s", daemonConfig.Network)
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

	netScrapper, err := ipns.NewNetworkScrapper(cmdDaemonIPNSConf, dbSer)
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
