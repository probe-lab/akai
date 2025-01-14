package main

import (
	"context"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/otel"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/core"
	"github.com/probe-lab/akai/db/models"
)

var pingConfig = &config.AkaiPing{
	Network: models.Network{Protocol: config.ProtocolIPFS, NetworkName: config.NetworkNameAmino}.String(),
	Key:     "",
	Timeout: 60 * time.Second,
	Retries: 3, 
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
		DefaultText: config.ListAllNetworkCombinations(),
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
	},
	&cli.IntFlag{
		Name: "retries",
		Aliases: []string{"r"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_PING_RETRIES")},
		},
		Usage: "Number of attempts that akai will try to fetch the content",
		Value: pingConfig.Retries,
		Destination: &pingConfig.Retries,
	},
}

func cmdPingAction(ctx context.Context, cmd *cli.Command) error {
	log.WithFields(log.Fields{
		"key":     pingConfig.Key,
		"network": pingConfig.Network,
		"timeout": pingConfig.Timeout,
		"retries": pingConfig.Retries,
	}).Info("requesting key from given DHT...")

	network := models.NetworkFromStr(pingConfig.Network)
	networkConfig, err := config.ConfigureNetwork(network)
	if err != nil {
		return err
	}

	dhtHostOpts := &config.CommonDHTHostOpts{
		IP:           "0.0.0.0",        // default?
		Port:         9020,             // default?
		DialTimeout:  20 * time.Second, // this is the DialTimeout, not the timeout for the operation
		DHTMode:      config.DHTClient,
		AgentVersion: networkConfig.AgentVersion,
		Meter:        otel.GetMeterProvider().Meter("akai_host"),
	}

	dhtHost, err := core.NewDHTHost(ctx, networkConfig, dhtHostOpts)
	if err != nil {
		return errors.Wrap(err, "creating DHT host")
	}

	// ensure that the format is correct
	_, err = core.ParseDHTKeyType(models.Network(network), pingConfig.Key)
	if err != nil {
		return err
	}
	sampleSegment := models.AgnosticSegment{
		Timestamp: time.Now(),
		Key:       pingConfig.Key,
	}

	// get sampling for network
	samplingFn, err := core.GetSamplingFnFromType(networkConfig.SamplingType)
	if err != nil {
		return err
	}

	sampleCtx, cancel := context.WithTimeout(ctx, pingConfig.Timeout)
	defer cancel()

	for retry := int64(1); retry <= pingConfig.Retries; retry++ {
		visit, err := samplingFn(sampleCtx, dhtHost, sampleSegment)
		if err != nil {
			return err
		} else {
			switch visit.Error {
				case "":
					log.WithFields(log.Fields{
						"operation":      config.ParseSamplingType(networkConfig.SamplingType),
						"timestamp":      time.Now(),
						"key":            sampleSegment.Key,
						"duration_ms":    visit.DurationMs,
						"is_retriebable": visit.IsRetrievable,
						"n_providers":    visit.Providers,
						"bytes":          visit.Bytes,
						"error":          visit.Error,
					}).Infof("ping done: (retry: %d)", retry)
					return nil

				default:
					log.WithFields(log.Fields{
						"retry": retry,
						"error": visit.Error,
					}).Warnf("key not found")
					continue
			}
		}
	}
	return nil
}
