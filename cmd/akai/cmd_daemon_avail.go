package main

import (
	"context"
	"fmt"

	"github.com/probe-lab/akai/api"
	"github.com/probe-lab/akai/avail"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/core"
	"github.com/probe-lab/akai/db"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/otel"
)

var cmdDaemonAvailDAStrackerConf = &config.AvailNetworkScrapperConfig{
	Network:              config.DefaultAvailNetwork.String(),
	TrackBlocksOnDB:      avail.DefaultNetorkScrapperConfig.TrackBlocksOnDB,
	NotChannelBufferSize: avail.DefaultNetorkScrapperConfig.NotChannelBufferSize,
	SamplerNotifyTimeout: avail.DefaultNetorkScrapperConfig.SamplerNotifyTimeout,
	TrackDuration:        avail.DefaultNetorkScrapperConfig.TrackDuration,
	TrackInterval:        avail.DefaultNetorkScrapperConfig.TrackInterval,
	BlockTrackerCfg: &config.AvailBlockTrackerConfig{
		Network: config.DefaultAvailNetwork.String(),
		Meter:   otel.GetMeterProvider().Meter("akai_api_server"),
		AvailAPIconfig: &config.AvailAPIConfig{
			Host:    api.DefaultClientConfig.Host,
			Port:    api.DefaultClientConfig.Port,
			Timeout: api.DefaultClientConfig.Timeout,
		},
	},
}

var cmdDaemonAvailDAStracker = &cli.Command{
	Name:   "avail",
	Usage:  "Tracks DAS cells from the Avail network and checks their availability in the DHT",
	Flags:  cmdDaemonAvailDAStrackerFlags,
	Action: cmdDaemonAvailDAStrackerAction,
}

var cmdDaemonAvailDAStrackerFlags = []cli.Flag{
	&cli.BoolFlag{
		Name: "track-blocks",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_AVAIL_TRACK_BLOCKS")},
		},
		Usage:       "",
		Value:       cmdDaemonAvailDAStrackerConf.TrackBlocksOnDB,
		Destination: &cmdDaemonAvailDAStrackerConf.TrackBlocksOnDB,
	},
	&cli.DurationFlag{
		Name: "track-duration",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_AVAIL_TRACK_DURATION")},
		},
		Usage:       "",
		Value:       cmdDaemonAvailDAStrackerConf.TrackDuration,
		Destination: &cmdDaemonAvailDAStrackerConf.TrackDuration,
	},
	&cli.DurationFlag{
		Name: "track-interval",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_AVAIL_TRACK_INTERVAL")},
		},
		Usage:       "",
		Value:       cmdDaemonAvailDAStrackerConf.TrackInterval,
		Destination: &cmdDaemonAvailDAStrackerConf.TrackInterval,
	},
	&cli.StringFlag{
		Name: "http-host",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_AVAIL_HTTP_HOST")},
		},
		Usage:       "Host IP of the avail-light client's HTTP API",
		Value:       cmdDaemonAvailDAStrackerConf.BlockTrackerCfg.AvailAPIconfig.Host,
		Destination: &cmdDaemonAvailDAStrackerConf.BlockTrackerCfg.AvailAPIconfig.Host,
	},
	&cli.IntFlag{
		Name: "http-port",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_AVAIL_HTTP_PORT")},
		},
		Usage:       "Port of the avail-light client's HTTP API",
		Value:       cmdDaemonAvailDAStrackerConf.BlockTrackerCfg.AvailAPIconfig.Port,
		Destination: &cmdDaemonAvailDAStrackerConf.BlockTrackerCfg.AvailAPIconfig.Port,
	},
	&cli.DurationFlag{
		Name: "http-timeout",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_AVAIL_HTTP_TIMEOUT")},
		},
		Usage:       "Duration for the HTTP API operations (20s, 1min)",
		DefaultText: cmdDaemonAvailDAStrackerConf.BlockTrackerCfg.AvailAPIconfig.Timeout.String(),
		Value:       cmdDaemonAvailDAStrackerConf.BlockTrackerCfg.AvailAPIconfig.Timeout,
		Destination: &cmdDaemonAvailDAStrackerConf.BlockTrackerCfg.AvailAPIconfig.Timeout,
	},
}

func cmdDaemonAvailDAStrackerAction(ctx context.Context, cmd *cli.Command) error {
	log.WithFields(log.Fields{
		"network":           cmdDaemonAvailDAStrackerConf.Network,
		"avail-http-host":   cmdDaemonAvailDAStrackerConf.BlockTrackerCfg.AvailAPIconfig.Host,
		"avail-http-port":   cmdDaemonAvailDAStrackerConf.BlockTrackerCfg.AvailAPIconfig.Port,
		"avail-api-timeout": cmdDaemonAvailDAStrackerConf.BlockTrackerCfg.AvailAPIconfig.Timeout,
	}).Info("starting avail-das-daemon...")
	defer log.Infof("stopped avail-das-daemon for %s", daemonConfig.Network)

	// set all network to be on the same one as the given one
	network := config.NetworkFromStr(daemonConfig.Network)
	if network.Protocol != config.ProtocolAvail {
		return fmt.Errorf("the given network doesn't belong to the avail protocol %s", daemonConfig.Network)
	}
	daemonConfig.DataSamplerConfig.Network = daemonConfig.Network
	cmdDaemonAvailDAStrackerConf.BlockTrackerCfg.Network = daemonConfig.Network

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

	netScrapper, err := avail.NewNetworkScrapper(cmdDaemonAvailDAStrackerConf, dbSer)
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
