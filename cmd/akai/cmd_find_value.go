package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/otel"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/core"
)

var cmdFindValue = &cli.Command{
	Name:   "value",
	Usage:  "Finds the value for any Key at the given network's DHT",
	Flags:  []cli.Flag{},
	Action: cmdFindValueAction,
}

func cmdFindValueAction(ctx context.Context, cmd *cli.Command) error {
	log.WithFields(log.Fields{
		"operation": config.SampleValue.String(),
		"key":       findOP.Key,
		"network":   findOP.Network,
		"timeout":   findOP.Timeout,
		"retries":   findOP.Retries,
	}).Info("requesting key from given DHT...")

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
		duration, value, err := dhtHost.FindValue(ctx, findOP.Key, findOP.Timeout)

		if err != nil {
			return err
		} else {
			switch err {
			case nil:
				log.WithFields(log.Fields{
					"operation":   config.SampleValue.String(),
					"timestamp":   t,
					"key":         findOP.Key,
					"duration_ms": duration,
					"num_bytes":   len(value),
					"bytes":       hex.EncodeToString(value),
					"error":       "no-error",
				}).Infof("find value done: (retry: %d)", retry)
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
	return fmt.Errorf("the %s operation couldn't report any successful result", config.SampleValue.String())
}
