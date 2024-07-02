package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

const (
	flagCategoryLogging = "Logging Configuration:"
)

var rootConfig = struct {
	Verbose    bool
	LogLevel   string
	LogFormat  string
	LogSource  bool
	LogNoColor bool

	// Functions that shut down the telemetry providers.
	// Both block until they're done
	metricsShutdownFunc func(ctx context.Context) error
	tracerShutdownFunc  func(ctx context.Context) error
}{
	Verbose:    false,
	LogLevel:   "info",
	LogFormat:  "text",
	LogSource:  false,
	LogNoColor: false,

	// unexported fields are derived or initialized during startup
	tracerShutdownFunc:  nil,
	metricsShutdownFunc: nil,
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
		log.Error("terminated abnormally", err)
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
func configureLogger(c context.Context, cmd *cli.Command) error {
	// log level
	logLevel := log.InfoLevel
	if cmd.IsSet("log-level") {
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
