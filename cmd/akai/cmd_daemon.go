package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/core"
	"github.com/probe-lab/akai/db"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

var daemonConfig = config.AkaiDaemonConfig{
	Network:           config.DefaultNetwork.String(),
	DBconfig:          db.DefaultConnectionDetails,
	DataSamplerConfig: core.DefaultDataSamplerConfig,
	DHTHostConfig:     core.DefaultDHTHostOpts,
}

var cmdDaemon = &cli.Command{
	Name:   "daemon",
	Usage:  "Runs the core of Akai's Data Sampler as a daemon",
	Flags:  cmdDaemonFlags,
	Before: daemonRunBefore,
	Commands: []*cli.Command{
		cmdDaemonAvailDAStracker,
		cmdDaemonCelestiaNamespace,
		cmdDaemonIPFS,
	},
}

var cmdDaemonFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "network",
		Aliases: []string{"n"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_NETWORK")},
		},
		Usage:       "The network where the Akai will be launched.",
		DefaultText: config.ListAllNetworkCombinations(),
		Value:       daemonConfig.Network,
		Destination: &daemonConfig.Network,
		Action:      validateNetworkFlag,
	},
	&cli.StringFlag{
		Name: "db-driver",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_DB_DRIVER")},
		},
		Usage:       "Driver of the Database that will keep all the raw data (clickhouse-local, clickhouse-replicated)",
		Value:       daemonConfig.DBconfig.Driver,
		Destination: &daemonConfig.DBconfig.Driver,
	},
	&cli.StringFlag{
		Name: "db-address",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_DB_ADDRESS")},
		},
		Usage:       "Address of the Database that will keep all the raw data",
		Value:       daemonConfig.DBconfig.Address,
		Destination: &daemonConfig.DBconfig.Address,
	},
	&cli.StringFlag{
		Name: "db-user",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_DB_USER")},
		},
		Usage:       "User of the Database that will keep all the raw data",
		Value:       daemonConfig.DBconfig.User,
		Destination: &daemonConfig.DBconfig.User,
	},
	&cli.StringFlag{
		Name: "db-password",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_DB_PASSWORD")},
		},
		Usage:       "Password for the user of the given Database",
		Value:       daemonConfig.DBconfig.Password,
		Destination: &daemonConfig.DBconfig.Password,
	},
	&cli.StringFlag{
		Name: "db-database",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_DB_DATABASE")},
		},
		Usage:       "Name of the Database that will keep all the raw data",
		Value:       daemonConfig.DBconfig.Database,
		Destination: &daemonConfig.DBconfig.Database,
	},
	&cli.BoolFlag{
		Name: "db-tls",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_DB_TLS")},
		},
		Usage:       "use or not use of TLS while connecting clickhouse",
		Value:       daemonConfig.DBconfig.TLSrequired,
		Destination: &daemonConfig.DBconfig.TLSrequired,
	},
	&cli.IntFlag{
		Name: "samplers",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_SAMPLERS")},
		},
		Usage:       "Number of workers the daemon will spawn to perform the sampling",
		Value:       daemonConfig.DataSamplerConfig.Workers,
		Destination: &daemonConfig.DataSamplerConfig.Workers,
	},
	&cli.DurationFlag{
		Name: "sampling-timeout",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_SAMPLING_TIMEOUT")},
		},
		Usage:       "Timeout for each sampling operation at the daemon after we deduce it failed",
		Value:       daemonConfig.DataSamplerConfig.SamplingTimeout,
		Destination: &daemonConfig.DataSamplerConfig.SamplingTimeout,
	},
}

func validateNetworkFlag(ctx context.Context, cli *cli.Command, s string) error {
	for protocol, networkNames := range config.AvailableProtocols {
		for _, networkName := range networkNames {
			network := config.Network{
				Protocol:    protocol,
				NetworkName: networkName,
			}
			if strings.ToUpper(s) == network.String() {
				return nil
			}
		}
	}
	return fmt.Errorf(" given network %s not valid for supported ones", s)
}

func daemonRunBefore(c context.Context, cmd *cli.Command) (context.Context, error) {
	log.WithFields(log.Fields{
		"network":           daemonConfig.Network,
		"database-driver":   daemonConfig.DBconfig.Driver,
		"database-address":  daemonConfig.DBconfig.Address,
		"database-user":     daemonConfig.DBconfig.User,
		"database-database": daemonConfig.DBconfig.Database,
		"database-tls":      daemonConfig.DBconfig.TLSrequired,
		"samplers":          daemonConfig.DataSamplerConfig.Workers,
		"sampling-timeout":  daemonConfig.DataSamplerConfig.SamplingTimeout,
	}).Info("starting akai-daemon...")
	return c, nil
}
