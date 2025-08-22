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

var findIpnsRecord = struct {
	KeyType string
	Quorum  int64
}{
	KeyType: "raw-string",
	Quorum:  4,
}

var cmdFindIPNS = &cli.Command{
	Name:  "ipns",
	Usage: "IPNS record that will be looked for",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "key-type",
			Usage:       "Type of key that was provided [dns, cid]",
			Value:       findIpnsRecord.KeyType,
			Destination: &findIpnsRecord.KeyType,
		},
		&cli.IntFlag{
			Name:        "quorum",
			Usage:       "Number of confirmations that we want to have",
			Value:       findIpnsRecord.Quorum,
			Destination: &findIpnsRecord.Quorum,
		},
	},
	Action: cmdFindIPNSAction,
}

func cmdFindIPNSAction(ctx context.Context, cmd *cli.Command) error {
	log.WithFields(log.Fields{
		"operation": config.SampleProviders.String(),
		"key":       findOP.Key,
		"type":      findIpnsRecord.KeyType,
		"quorum":    findIpnsRecord.Quorum,
		"network":   findOP.Network,
		"timeout":   findOP.Timeout,
		"retries":   findOP.Retries,
	}).Info("requesting key from IPNS DHT...")

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

		item := &models.SamplingItem{
			Network: findOP.Network,
			Key:     findOP.Key,
		}
		task := core.SamplingTask{
			VisitRound: 1,
			Item:       item,
			Quorum:     int(findIpnsRecord.Quorum),
		}

		var err error
		var res models.GeneralVisit
		switch findIpnsRecord.KeyType {
		case "cid":
			res, err = core.ResolveIPNSRecord(ctx, dhtHost, task, findOP.Timeout)
		case "dns":
			res, err = core.ResolveDNS(ctx, dhtHost, task, findOP.Timeout)
		default:
			res, err = core.ResolveIPNSRecord(ctx, dhtHost, task, findOP.Timeout)
		}
		switch err {
		case nil:
			if res.GenericIPNSRecordVisit == nil {
				return fmt.Errorf("empty response from the IPNS record visit")
			}
			r := res.GenericIPNSRecordVisit[0]
			log.WithFields(log.Fields{
				"operation":   config.SampleValue.String(),
				"timestamp":   t,
				"key":         findOP.Key,
				"duration_ms": r.Duration,
				"ttl_s":       r.TTL.Seconds(),
				"is-valid":    r.IsValid,
				"path":        r.Result,
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
	return fmt.Errorf("the %s operation couldn't report any successful result", config.SampleProviders.String())
}
