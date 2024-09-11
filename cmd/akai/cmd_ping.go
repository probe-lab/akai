package main

import (
	"context"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/core"
	"github.com/probe-lab/akai/db/models"
)

var pingConfig = &config.AkaiPing{
	Network: models.Network{Protocol: config.ProtocolIPFS, NetworkName: config.NetworkNameIPFSAmino}.String(),
	Key:     "",
	Timeout: 60 * time.Second,
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
}

func cmdPingAction(ctx context.Context, cmd *cli.Command) error {
	log.WithFields(log.Fields{
		"key":     pingConfig.Key,
		"network": pingConfig.Network,
		"timeout": pingConfig.Timeout,
	}).Info("requesting key from given DHT...")

	network := models.NetworkFromStr(pingConfig.Network)
	networkConfig, err := config.ConfigureNetwork(network)
	if err != nil {
		return err
	}

	dhtHostOpts := &config.CommonDHTHostOpts{
		IP:           "0.0.0.0",        // default?
		Port:         9020,             // default?
		DialTimeout:  10 * time.Second, // this is the DialTimeout, not the timeout for the operation
		DHTMode:      config.DHTClient,
		AgentVersion: networkConfig.AgentVersion,
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

	visit, err := samplingFn(sampleCtx, dhtHost, sampleSegment)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"timestamp":      time.Now(),
		"key":            sampleSegment.Key,
		"duration_ms":    visit.DurationMs,
		"is_retriebable": visit.IsRetrievable,
		"n_providers":    visit.Providers,
		"bytes":          visit.Bytes,
		"error":          visit.Error,
	}).Info("find providers operation done")

	return nil

}

// fetchCidProviders was a dedicated ping function for fetching CID's providers in the Amino DHT (IPFS)
// DEPRECATED in favor of core.GetSamplingFnFromType()
// func fetchCidProviders(ctx context.Context, h core.DHTHost, key amino.Cid) error {
// 	// request the key from the network
// 	findCtx, cancel := context.WithTimeout(ctx, pingConfig.Timeout)
// 	defer cancel()

// 	opDuration, providers, err := h.FindProviders(findCtx, key.Cid())
// 	if err != nil {
// 		return errors.Wrap(err, "finding providers for key")
// 	}

// 	log.WithFields(log.Fields{
// 		"duration":    opDuration,
// 		"n_providers": len(providers),
// 		"peer_ids":    core.ListPeerIDsFromAddrsInfos(providers),
// 	}).Info("find providers operation done")

// 	return nil
// }

// fetchCidProviders was a dedicated ping function for fetching CIDs content in the Amino DHT (Avail)
// DEPRECATED in favor of core.GetSamplingFnFromType()
// func fetchAvailKey(ctx context.Context, h core.DHTHost, key config.AvailKey) error {

// 	log.WithFields(log.Fields{
// 		"block":  key.Block,
// 		"row":    key.Row,
// 		"column": key.Column,
// 	}).Info("finding providers for cell...")

// 	// request the key from the network
// 	findCtx, cancel := context.WithTimeout(ctx, pingConfig.Timeout)
// 	defer cancel()

// 	dhtKey := key.String()

// 	findPeersDuration, closesPs, err := h.FindClosestPeers(findCtx, dhtKey)
// 	if err != nil {
// 		return err
// 	}
// 	log.WithFields(log.Fields{
// 		"duration": findPeersDuration,
// 		"peers":    len(closesPs),
// 		"peer_ids": closesPs,
// 	}).Info("found closest peers to cell...")

// 	findDuration, bytes, err := h.FindValue(findCtx, dhtKey)
// 	if err != nil {
// 		log.WithFields(log.Fields{
// 			"duration": findDuration,
// 		}).Warn("no cell found for block...")
// 	} else {
// 		log.WithFields(log.Fields{
// 			"duration": findDuration,
// 			"bytes":    string(bytes),
// 		}).Info("found block cell...")
// 	}
// 	return nil
// }
