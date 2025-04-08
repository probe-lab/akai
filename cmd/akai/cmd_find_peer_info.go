package main

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/otel"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/core"
	"github.com/probe-lab/akai/db/models"
)

var cmdFindPeerInfo = &cli.Command{
	Name:   "peer-info",
	Usage:  "Finds all the existing information about a PeerID at the given network's DHT",
	Flags:  []cli.Flag{},
	Action: cmdFindPeerInfoAction,
}

func cmdFindPeerInfoAction(ctx context.Context, cmd *cli.Command) error {
	log.WithFields(log.Fields{
		"operation": config.ParseSamplingType(config.SamplePeerInfo),
		"peer_id":   findOP.Key,
		"network":   findOP.Network,
		"timeout":   findOP.Timeout,
		"retries":   findOP.Retries,
	}).Info("requesting info from given peer...")

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

	peerID, err := peer.Decode(findOP.Key)
	if err != nil {
		return err
	}

	for retry := int64(1); retry <= findOP.Retries; retry++ {
		t := time.Now()
		duration, peerInfo, err := dhtHost.FindPeer(ctx, peerID, findOP.Timeout)
		switch err {
		case nil:
			hostInfo, err := dhtHost.ConnectAndIdentifyPeer(ctx, peerInfo, int(findOP.Retries), findOP.Timeout)
			errStr := ""
			if err == nil {
				errStr = "no-error"
			} else {
				errStr = err.Error()
			}
			log.WithFields(log.Fields{
				"operation":   config.ParseSamplingType(config.SamplePeerInfo),
				"timestamp":   t,
				"peer_id":     peerID.String(),
				"maddres":     peerInfo.Addrs,
				"peer_info":   hostInfo,
				"duration_ms": duration,
				"error":       errStr,
			}).Infof("find peer info done: (retry: %d)", retry)
			return nil

		default:
			log.WithFields(log.Fields{
				"retry": retry,
				"error": err,
			}).Warnf("Peer or connection not possible... ")
			continue
		}
	}
	return fmt.Errorf("the %s operation couldn't report any successful result", config.ParseSamplingType(config.SamplePeerInfo))
}
