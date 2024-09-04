package config

import "context"

// Root
type Root struct {
	Verbose    bool
	LogLevel   string
	LogFormat  string
	LogSource  bool
	LogNoColor bool

	MetricsAddr string
	MetricsPort int64

	// Functions that shut down the telemetry providers.
	// Both block until they're done
	MetricsShutdownFunc func(ctx context.Context) error
}
