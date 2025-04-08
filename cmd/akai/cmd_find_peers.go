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
	"github.com/probe-lab/akai/db/models"
)

var cmdFindPeers = &cli.Command{
	Name:   "peers",
	Usage:  "Finds the existing peers advertised for the given Key at the network's DHT",
	Flags:  []cli.Flag{},
	Action: cmdFindPeersAction,
}

func cmdFindPeersAction(ctx context.Context, cmd *cli.Command) error {
	log.WithFields(log.Fields{
		"operation": config.ParseSamplingType(config.SamplePeers),
		"key":       findOP.Key,
		"network":   findOP.Network,
		"timeout":   findOP.Timeout,
		"retries":   findOP.Retries,
	}).Info("requesting key from given DHT...")

	network := models.NetworkFromStr(findOP.Network)
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
		sampleCtx, cancel := context.WithTimeout(ctx, findOP.Timeout)
		defer cancel()

		t := time.Now()
		duration, peerInfos, err := dhtHost.FindPeers(sampleCtx, findOP.Key, findOP.Timeout)
		switch err {
		case nil:
			for idx, pInfo := range peerInfos {
				log.WithFields(log.Fields{
					"peer_id":    pInfo.ID.String(),
					"maddresses": pInfo.Addrs,
				}).Info(fmt.Sprintf("%d", idx))
			}
			log.WithFields(log.Fields{
				"operation":   config.ParseSamplingType(config.SamplePeers),
				"timestamp":   t,
				"key":         findOP.Key,
				"duration_ms": duration,
				"num_peers":   len(peerInfos),
				"error":       "no-error",
			}).Infof("find providers done: (retry: %d)", retry)
			return nil

		default:
			log.WithFields(log.Fields{
				"retry": retry,
				"error": err,
			}).Warnf("key not found")
			continue
		}
	}
	return nil
}
