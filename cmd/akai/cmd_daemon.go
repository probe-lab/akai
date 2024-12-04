package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/probe-lab/akai/api"
	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/core"
	"github.com/probe-lab/akai/db"
	"github.com/probe-lab/akai/db/models"
	"go.opentelemetry.io/otel"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

var daemonConfig = config.AkaiDaemonConfig{
	Network:           config.DefaultNetwork.String(),
	APIconfig:         api.DefaulServiceConfig,
	DBconfig:          db.DefaultConnectionDetails,
	DataSamplerConfig: core.DefaultDataSamplerConfig,
	DHTHostConfig:     core.DefaultDHTHostOpts,
	Meter:             otel.GetMeterProvider().Meter("akai_daemon"),
}

var cmdService = &cli.Command{
	Name:   "daemon",
	Usage:  "Runs the core of Akai's Data Sampler as a daemon",
	Flags:  cmdDaemonFlags,
	Action: cmdDaemonAction,
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
		Name:    "api-http-host",
		Aliases: []string{"ah"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_API_HTTP_HOST")},
		},
		Usage:       "Port for the Akai's HTTP API server",
		Value:       daemonConfig.APIconfig.Host,
		Destination: &daemonConfig.APIconfig.Host,
	},
	&cli.IntFlag{
		Name:    "api-http-port",
		Aliases: []string{"ap"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_API_HTTP_PORT")},
		},
		Usage:       "Port of Akai Daemon's HTTP API",
		Value:       daemonConfig.APIconfig.Port,
		Destination: &daemonConfig.APIconfig.Port,
	},
	&cli.StringFlag{
		Name: "db-driver",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_DB_DRIVER")},
		},
		Usage:       "Driver of the Database that will keep all the raw data",
		Value:       daemonConfig.DBconfig.Driver,
		Destination: &daemonConfig.DBconfig.Driver,
	},
	&cli.StringFlag{
		Name:    "db-address",
		Aliases: []string{"dba"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_DB_ADDRESS")},
		},
		Usage:       "Address of the Database that will keep all the raw data",
		Value:       daemonConfig.DBconfig.Address,
		Destination: &daemonConfig.DBconfig.Address,
	},
	&cli.StringFlag{
		Name:    "db-user",
		Aliases: []string{"dbu"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_DB_USER")},
		},
		Usage:       "User of the Database that will keep all the raw data",
		Value:       daemonConfig.DBconfig.User,
		Destination: &daemonConfig.DBconfig.User,
	},
	&cli.StringFlag{
		Name:    "db-password",
		Aliases: []string{"dbp"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_DAEMON_DB_PASSWORD")},
		},
		Usage:       "Password for the user of the given Database",
		Value:       daemonConfig.DBconfig.Password,
		Destination: &daemonConfig.DBconfig.Password,
	},
	&cli.StringFlag{
		Name:    "db-database",
		Aliases: []string{"dbd"},
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
			network := models.Network{Protocol: protocol, NetworkName: networkName}
			if strings.ToUpper(s) == network.String() {
				return nil
			}
		}
	}
	return fmt.Errorf(" given network %s not valid for supported ones", s)
}

func cmdDaemonAction(ctx context.Context, cmd *cli.Command) (err error) {
	log.WithFields(log.Fields{
		"network":           daemonConfig.Network,
		"database-driver":   daemonConfig.DBconfig.Driver,
		"database-address":  daemonConfig.DBconfig.Address,
		"database-user":     daemonConfig.DBconfig.User,
		"database-database": daemonConfig.DBconfig.Database,
		"database-tls":      daemonConfig.DBconfig.TLSrequired,
		"akai-api-host":     daemonConfig.APIconfig.Host,
		"akai-api-port":     daemonConfig.APIconfig.Port,
		"daemon-samplers":   daemonConfig.DataSamplerConfig.Workers,
		"sampling-timeout":  daemonConfig.DataSamplerConfig.SamplingTimeout,
	}).Info("starting akai-daemon...")
	defer log.Infof("stopped akai-daemon for %s", daemonConfig.Network)

	// set all network to be on the same one as the given one
	network := models.NetworkFromStr(daemonConfig.Network)
	daemonConfig.APIconfig.Network = daemonConfig.Network
	daemonConfig.DataSamplerConfig.Network = daemonConfig.Network

	networkConfig, err := config.ConfigureNetwork(network)
	if err != nil {
		return err
	}

	// start the database
	dbSer, err := db.NewDatabase(daemonConfig.DBconfig, networkConfig)
	if err != nil {
		return err
	}

	// get a DHThost for the given network
	dhtHost, err := core.NewDHTHost(ctx, networkConfig, daemonConfig.DHTHostConfig)
	if err != nil {
		return err
	}

	// start the DataSampler (requires DB)
	config.UpdateSamplingDetailFromNetworkConfig(&daemonConfig.DataSamplerConfig.AkaiSamplingDetails, networkConfig)
	if err != nil {
		return err
	}
	samplingFn, err := core.GetSamplingFnFromType(networkConfig.SamplingType)
	if err != nil {
		return err
	}
	dataSampler, err := core.NewDataSampler(daemonConfig.DataSamplerConfig, dbSer, dhtHost, samplingFn)
	if err != nil {
		return err
	}
	// start the API service (requires DataSampler)
	apiSer, err := api.NewService(daemonConfig.APIconfig)
	if err != nil {
		return err
	}

	// NOTE: this creates a basic appHandlers
	// -> They MUST be overwriten after the init of the daemon to plug the DB and the DataSampler
	// to hear from new samples and blocks
	daemon, err := core.NewDaemon(daemonConfig, dhtHost, dbSer, dataSampler, apiSer)
	if err != nil {
		return err
	}

	return daemon.Start(ctx)
}
