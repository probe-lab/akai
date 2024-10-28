package main

import (
	"context"

	akai_api "github.com/probe-lab/akai/api"
	"github.com/probe-lab/akai/avail"
	"github.com/probe-lab/akai/avail/api"
	"github.com/probe-lab/akai/config"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/otel"
)

var availBlockTrackerConf = &config.AvailBlockTracker{
	TextConsumer:    true,
	AkaiAPIconsumer: false,
	Network:         config.DefaultNetwork.String(),
	AvailAPIconfig:  api.DefaultClientConfig,
	AkaiAPIconfig:   akai_api.DefaultClientConfig,
}

var cmdAvailBlockTracker = &cli.Command{
	Name:   "avail-block-tracker",
	Usage:  "Pings any Key from the given network's DHT",
	Flags:  cmdAvailBlockTrackerFlags,
	Action: cmdAvailBlockTrackerAction,
}

var cmdAvailBlockTrackerFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "network",
		Aliases: []string{"n"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_NETWORK")},
		},
		Usage:       "The network where the Akai will be launched.",
		DefaultText: config.ListNetworksForProtocol(config.ProtocolAvail),
		Value:       availBlockTrackerConf.Network,
		Destination: &availBlockTrackerConf.Network,
		Action:      validateNetworkFlag,
	},
	&cli.BoolFlag{
		Name:    "text-consumer",
		Aliases: []string{"tc"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_BLOCK_TRACKER_TEXT_CONSUMER")},
		},
		Usage:       "Text-log processing when a new Avail block is tracked",
		DefaultText: "true",
		Value:       availBlockTrackerConf.TextConsumer,
		Destination: &availBlockTrackerConf.TextConsumer,
	},
	&cli.BoolFlag{
		Name:    "api-consumer",
		Aliases: []string{"ac"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_BLOCK_TRACKER_API_CONSUMER")},
		},
		Usage:       "Process new blocks sending them to Akai's API service after a new Avail block is tracked",
		DefaultText: "false",
		Value:       availBlockTrackerConf.AkaiAPIconsumer,
		Destination: &availBlockTrackerConf.AkaiAPIconsumer,
	},
	&cli.StringFlag{
		Name:    "avail-http-host",
		Aliases: []string{"avh"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_AVAIL_HTTP_HOST")},
		},
		Usage:       "Host IP of the avail-light client's HTTP API",
		Value:       availBlockTrackerConf.AvailAPIconfig.Host,
		Destination: &availBlockTrackerConf.AvailAPIconfig.Host,
	},
	&cli.IntFlag{
		Name:    "avail-http-port",
		Aliases: []string{"avp"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_AVAIL_HTTP_PORT")},
		},
		Usage:       "Port of the avail-light client's HTTP API",
		Value:       availBlockTrackerConf.AvailAPIconfig.Port,
		Destination: &availBlockTrackerConf.AvailAPIconfig.Port,
	},
	&cli.DurationFlag{
		Name:    "avail-http-timeout",
		Aliases: []string{"t"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_AVAIL_HTTP_TIMEOUT")},
		},
		Usage:       "Duration for the HTTP API operations (20s, 1min)",
		DefaultText: availBlockTrackerConf.AvailAPIconfig.Timeout.String(),
		Value:       availBlockTrackerConf.AvailAPIconfig.Timeout,
		Destination: &availBlockTrackerConf.AvailAPIconfig.Timeout,
	},
	&cli.StringFlag{
		Name:    "akai-http-host",
		Aliases: []string{"akh"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_HTTP_HOST")},
		},
		Usage:       "Port for the Akai's HTTP API server",
		Value:       availBlockTrackerConf.AkaiAPIconfig.Host,
		Destination: &availBlockTrackerConf.AkaiAPIconfig.Host,
	},
	&cli.IntFlag{
		Name:    "akai-http-port",
		Aliases: []string{"akp"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_HTTP_PORT")},
		},
		Usage:       "Port of Akai Daemon's HTTP API",
		Value:       availBlockTrackerConf.AkaiAPIconfig.Port,
		Destination: &availBlockTrackerConf.AkaiAPIconfig.Port,
	},
	&cli.DurationFlag{
		Name:    "akai-http-timeout",
		Aliases: []string{"akt"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_HTTP_TIMEOUT")},
		},
		Usage:       "Port of Akai Daemon's HTTP API",
		Value:       availBlockTrackerConf.AkaiAPIconfig.Timeout,
		Destination: &availBlockTrackerConf.AkaiAPIconfig.Timeout,
	},
}

func cmdAvailBlockTrackerAction(ctx context.Context, cmd *cli.Command) error {
	log.WithFields(log.Fields{
		"network":         availBlockTrackerConf.Network,
		"avail-http-host": availBlockTrackerConf.AvailAPIconfig.Host,
		"avail-http-port": availBlockTrackerConf.AvailAPIconfig.Port,
		"timeout":         availBlockTrackerConf.AvailAPIconfig.Timeout,
		"text-consumer":   availBlockTrackerConf.TextConsumer,
		"api-consumer":    availBlockTrackerConf.AkaiAPIconsumer,
	}).Info("starting avail-block-tracker...")
	defer log.Infof("stopped akai-block-tracker for %s", availBlockTrackerConf.Network)

	blockTrackerConfig := &config.AvailBlockTracker{
		TextConsumer:    availBlockTrackerConf.TextConsumer,
		AkaiAPIconsumer: availBlockTrackerConf.AkaiAPIconsumer,
		Network:         availBlockTrackerConf.Network,
		AvailAPIconfig:  availBlockTrackerConf.AvailAPIconfig,
		AkaiAPIconfig:   availBlockTrackerConf.AkaiAPIconfig,
		Meter:           otel.GetMeterProvider().Meter("akai_avail_block_tracker"),
	}

	blockTracker, err := avail.NewBlockTracker(blockTrackerConfig)
	if err != nil {
		return err
	}

	return blockTracker.Start(ctx)
}
