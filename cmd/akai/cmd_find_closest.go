package main

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/otel"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/core"
)

var cmdFindClosests = &cli.Command{
	Name:   "closest",
	Usage:  "Finds the k closest peers for any Key at the given network's DHT",
	Flags:  []cli.Flag{},
	Action: cmdFindClosestsAction,
}

func cmdFindClosestsAction(ctx context.Context, cmd *cli.Command) error {
	log.WithFields(log.Fields{
		"operation": config.SampleClosest.String(),
		"key":       findOP.Key,
		"network":   findOP.Network,
		"timeout":   findOP.Timeout,
		"retries":   findOP.Retries,
	}).Info("requesting closest peers from given DHT Key...")

	network := config.NetworkFromStr(findOP.Network)
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

	for retry := int64(1); retry <= findOP.Retries; retry++ {
		t := time.Now()
		duration, closestPeers, err := dhtHost.FindClosestPeers(ctx, findOP.Key, findOP.Timeout)
		if err != nil {
			return err
		} else {
			switch err {
			case nil:
				log.WithFields(log.Fields{
					"operation":   config.SampleClosest.String(),
					"timestamp":   t,
					"key":         findOP.Key,
					"duration_ms": duration,
					"num_peers":   len(closestPeers),
					"peers":       closestPeers,
					"error":       "no-error",
				}).Infof("find closest peers done: (retry: %d)", retry)
				return nil

			default:
				log.WithFields(log.Fields{
					"retry": retry,
					"error": err,
				}).Warnf("key not found")
				continue
			}
		}
	}
	return fmt.Errorf("the %s operation couldn't report any successful result", config.SampleClosest.String())
}
