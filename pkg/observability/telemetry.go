package observability

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

// InitTelemetry initializes OpenTelemetry metrics
func InitTelemetry(serviceName string) (*metric.MeterProvider, http.Handler, error) {
	// Create a Prometheus registry
	registry := prometheus.NewRegistry()

	// Create Prometheus exporter with custom registry
	exporter, err := otelprom.New(
		otelprom.WithRegisterer(registry),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	// Create meter provider
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(exporter),
	)

	// Set global meter provider
	otel.SetMeterProvider(meterProvider)

	// Create HTTP handler for Prometheus metrics using the registry
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	return meterProvider, handler, nil
}

// InitLogger initializes structured logger
func InitLogger(env string) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	if env == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Replace global logger
	zap.ReplaceGlobals(logger)

	return logger, nil
}

// Shutdown gracefully shuts down telemetry
func Shutdown(ctx context.Context, meterProvider *metric.MeterProvider, logger *zap.Logger) error {
	if meterProvider != nil {
		if err := meterProvider.Shutdown(ctx); err != nil {
			logger.Error("failed to shutdown meter provider", zap.Error(err))
			return err
		}
	}

	if logger != nil {
		if err := logger.Sync(); err != nil {
			// Ignore sync errors in some environments
			_ = err
		}
	}

	return nil
}
