package metrics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
	prom "github.com/prometheus/client_golang/prometheus/promhttp"
	promexp "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	mnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetrics "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const (
	MeterName = "github.com/probe-lab/akai"
)

func PromMeterProvider(ctx context.Context) (metric.MeterProvider, error) {
	// initializing a new registery, otherwise we would export all the
	// automatically registered prysm metrics.
	registry := prometheus.NewRegistry()

	prometheus.DefaultRegisterer = registry
	prometheus.DefaultGatherer = registry

	opts := []promexp.Option{
		promexp.WithRegisterer(registry), // actually unnecessary, as we overwrite the default values above
		promexp.WithNamespace("akai"),
	}

	exporter, err := promexp.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("new prometheus exporter: %w", err)
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName("akai"),
	))
	if err != nil {
		return nil, fmt.Errorf("new metrics resource: %w", err)
	}

	options := []sdkmetrics.Option{
		sdkmetrics.WithReader(exporter),
		sdkmetrics.WithResource(res),
	}

	return sdkmetrics.NewMeterProvider(options...), nil
}

func NoopMeterProvider() metric.MeterProvider {
	return mnoop.NewMeterProvider()
}

func ServeMetrics(ctx context.Context, host string, port int64) func(context.Context) error {
	mux := http.NewServeMux()

	mux.Handle("/metrics", prom.Handler())
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	addr := fmt.Sprintf("%s:%d", host, port)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	done := make(chan struct{})

	go func() {
		defer close(done)
		log.WithFields(log.Fields{
			"address": fmt.Sprintf("http://%s/metrics", addr),
		}).Info("Starting metrics server")

		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("metrics server failed", err)
		}
	}()

	cancelCtx, cancel := context.WithCancel(ctx)

	go func() {
		<-cancelCtx.Done()

		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		slog.Info("Shutting down metrics server")
		if err := srv.Shutdown(timeoutCtx); err != nil {
			log.Error("Failed to shut down metrics server", err)
		}
	}()

	shutdownFunc := func(ctx context.Context) error {
		cancel()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			return nil
		}
	}

	return shutdownFunc
}
