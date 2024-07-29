package main

import (
	"context"
	"time"

	"github.com/probe-lab/akai/avail"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/db"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

var availBlockTrackerConf = &config.AvailBlockTracker{
	TextConsumer: true,
	Network:      db.Network{Protocol: config.ProtocolAvail, NetworkName: config.NetworkNameAvailTuring}.String(),
	AvailHttpAPIClient: config.AvailHttpAPIClient{
		IP:      "localhost",
		Port:    5000,
		Timeout: 10 * time.Second,
	},
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
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_AVAIL_BLOCK_TRACKER_NETWORK")},
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
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_AVAIL__BLOCK_TRACKER_TEXT_CONSUMER")},
		},
		Usage:       "Text-log processing when a new Avail block is tracked",
		DefaultText: "true",
		Value:       availBlockTrackerConf.TextConsumer,
		Destination: &availBlockTrackerConf.TextConsumer,
	},
	&cli.StringFlag{
		Name:    "avail-http-host",
		Aliases: []string{"avh"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_AVAIL_BLOCK_TRACKER_AVAIL_HTTP_HOST")},
		},
		Usage:       "Host IP of the avail-light client's HTTP API",
		Value:       availBlockTrackerConf.IP,
		Destination: &availBlockTrackerConf.IP,
	},
	&cli.IntFlag{
		Name:    "avail-http-port",
		Aliases: []string{"avp"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_AVAIL_BLOCK_TRACKER_AVAIL_HTTP_HOST")},
		},
		Usage:       "Port of the avail-light client's HTTP API",
		Value:       availBlockTrackerConf.Port,
		Destination: &availBlockTrackerConf.Port,
	},
	&cli.DurationFlag{
		Name:    "http-timeout",
		Aliases: []string{"t"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_AVAIL_BLOCK_TRACKER_HTTP_TIMEOUT")},
		},
		Usage:       "Duration for the HTTP API operations (20s, 1min)",
		DefaultText: "20sec",
		Value:       availBlockTrackerConf.Timeout,
		Destination: &availBlockTrackerConf.Timeout,
	}}

func cmdAvailBlockTrackerAction(ctx context.Context, cmd *cli.Command) error {
	log.WithFields(log.Fields{
		"network":       availBlockTrackerConf.Network,
		"http-host":     availBlockTrackerConf.IP,
		"http-port":     availBlockTrackerConf.Port,
		"timeout":       availBlockTrackerConf.Timeout,
		"text-consumer": availBlockTrackerConf.TextConsumer,
	}).Info("starting avail-block-tracker...")
	defer log.Infof("stopped akai-block-tracker for %s", availBlockTrackerConf.Network)

	blockTrackerConfig := config.AvailBlockTracker{
		TextConsumer: availBlockTrackerConf.TextConsumer,
		Network:      availBlockTrackerConf.Network,
		AvailHttpAPIClient: config.AvailHttpAPIClient{
			IP:      availBlockTrackerConf.IP,
			Port:    availBlockTrackerConf.Port,
			Timeout: availBlockTrackerConf.Timeout,
		},
	}

	blockTracker, err := avail.NewBlockTracker(blockTrackerConfig)
	if err != nil {
		return err
	}

	return blockTracker.Start(ctx)

}
