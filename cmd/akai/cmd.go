package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/probe-lab/akai/config"
	"github.com/probe-lab/akai/metrics"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/otel"
)

const (
	flagCategoryLogging   = "Logging Configuration:"
	flagCategoryTelemetry = "Telemetry Configuration:"
)

var rootConfig = &config.Root{
	Verbose:             false,
	LogLevel:            "info",
	LogFormat:           "text",
	LogSource:           false,
	LogNoColor:          false,
	MetricsAddr:         "127.0.0.1",
	MetricsPort:         9080,
	MetricsShutdownFunc: nil,
}

var app = &cli.Command{
	Name:                  "akai",
	Usage:                 "A libp2p generic DA sampler",
	EnableShellCompletion: true,
	Flags:                 rootFlags,
	Before:                rootBefore,
	Commands: []*cli.Command{
		cmdService,
		cmdPing,
		cmdAvailBlockTracker,
	},
	After: rootAfter,
}

var rootFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:    "verbose",
		Aliases: []string{"v"},
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_VERBOSE")},
		},
		Usage:       "Set logging level more verbose to include debug level logs",
		Value:       rootConfig.Verbose,
		Destination: &rootConfig.Verbose,
		Category:    flagCategoryLogging,
	},
	&cli.StringFlag{
		Name: "log.level",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_LOG_LEVEL")},
		},
		Usage:       "Sets an explicity logging level: debug, info, warn, error. Takes precedence over the verbose flag.",
		Destination: &rootConfig.LogLevel,
		Value:       rootConfig.LogLevel,
		Category:    flagCategoryLogging,
	},
	&cli.StringFlag{
		Name: "log.format",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_LOG_FORMAT")},
		},
		Usage:       "Sets the format to output the log statements in: text, json",
		Destination: &rootConfig.LogFormat,
		Value:       rootConfig.LogFormat,
		Category:    flagCategoryLogging,
	},
	&cli.BoolFlag{
		Name: "log.source",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_LOG_SOURCE")},
		},
		Usage:       "Compute the source code position of a log statement and add a SourceKey attribute to the output. Only text and json formats.",
		Destination: &rootConfig.LogSource,
		Value:       rootConfig.LogSource,
		Category:    flagCategoryLogging,
	},
	&cli.BoolFlag{
		Name: "log.nocolor",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_LOG_NO_COLOR")},
		},
		Usage:       "Whether to prevent the logger from outputting colored log statements",
		Destination: &rootConfig.LogNoColor,
		Value:       rootConfig.LogNoColor,
		Category:    flagCategoryLogging,
	},
	&cli.StringFlag{
		Name: "metrics.addr",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_METRICS_ADDR")},
		},
		Usage:       "Which network interface should the metrics endpoint bind to.",
		Value:       rootConfig.MetricsAddr,
		Destination: &rootConfig.MetricsAddr,
		Category:    flagCategoryTelemetry,
	},
	&cli.IntFlag{
		Name: "metrics.port",
		Sources: cli.ValueSourceChain{
			Chain: []cli.ValueSource{cli.EnvVar("AKAI_METRICS_PORT")},
		},
		Usage:       "On which port should the metrics endpoint listen",
		Value:       rootConfig.MetricsPort,
		Destination: &rootConfig.MetricsPort,
		Category:    flagCategoryTelemetry,
	},
}

func main() {
	sigs := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	go func() {
		defer cancel()
		defer signal.Stop(sigs)

		select {
		case <-ctx.Done():
		case sig := <-sigs:
			log.WithField("signal", sig.String()).Info("Received termination signal - Stopping...")
		}
	}()

	if err := app.Run(ctx, os.Args); err != nil && !errors.Is(err, context.Canceled) {
		log.Errorf("akai terminated abnormally %s", err.Error())
		os.Exit(1)
	}
}

func rootBefore(c context.Context, cmd *cli.Command) error {
	// don't set up anything if akai is run without arguments
	if cmd.NArg() == 0 {
		return nil
	}

	// read CLI args and configure the global logger
	if err := configureLogger(c, cmd); err != nil {
		return err
	}

	// read CLI args and configure the global meter provider
	if err := configureMetrics(c, cmd); err != nil {
		return err
	}

	return nil
}

func rootAfter(c context.Context, cmd *cli.Command) error {
	log.Info("Akai successfully shutted down")
	return nil
}

// configureLogger configures the global logger based on the provided CLI
// context. It sets the log level based on the "--log-level" flag or the
// "--verbose" flag. The log format is determined by the "--log.format" flag.
// The function returns an error if the log level or log format is not supported.
// Possible log formats include "tint", "hlog", "text", and "json". The default
// logger is overwritten with the configured logger.
func configureLogger(_ context.Context, cmd *cli.Command) error {
	// log level
	logLevel := log.InfoLevel
	if cmd.IsSet("log.level") {
		switch strings.ToLower(rootConfig.LogLevel) {
		case "debug":
			logLevel = log.DebugLevel
		case "info":
			logLevel = log.InfoLevel
		case "warn":
			logLevel = log.WarnLevel
		case "error":
			logLevel = log.ErrorLevel
		default:
			return fmt.Errorf("unknown log level: %s", rootConfig.LogLevel)
		}
	} else if rootConfig.Verbose {
		logLevel = log.DebugLevel
	}
	log.SetLevel(log.Level(logLevel))

	// log format
	switch strings.ToLower(rootConfig.LogFormat) {
	case "text":
		log.SetFormatter(&log.TextFormatter{
			DisableColors: rootConfig.LogNoColor,
		})
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	default:
		return fmt.Errorf("unknown log format: %q", rootConfig.LogFormat)
	}

	return nil
}

// configureMetrics configures the prometheus metrics export based on the provided CLI context.
// If metrics are not enabled, it uses a no-op meter provider
// ([tele.NoopMeterProvider]) and does not serve an endpoint. If metrics are
// enabled, it sets up the Prometheus meter provider ([tele.PromMeterProvider]).
// The function returns an error if there is an issue with creating the meter
// provider.
func configureMetrics(ctx context.Context, _ *cli.Command) error {
	// user wants to have metrics, use the prometheus meter provider
	provider, err := metrics.PromMeterProvider(ctx)
	if err != nil {
		return fmt.Errorf("new prometheus meter provider: %w", err)
	}

	otel.SetMeterProvider(provider)

	// expose the /metrics endpoint. Use new context, so that the metrics server
	// won't stop when an interrupt is received. If the shutdown procedure hangs
	// this will give us a chance to still query pprof or the metrics endpoints.
	shutdownFunc := metrics.ServeMetrics(context.Background(), rootConfig.MetricsAddr, rootConfig.MetricsPort)

	rootConfig.MetricsShutdownFunc = shutdownFunc

	return nil
}
