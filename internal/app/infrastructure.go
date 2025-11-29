package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/prperemyshlev/auth-service-2/internal/config"
	"github.com/prperemyshlev/auth-service-2/pkg/database"
	"github.com/prperemyshlev/auth-service-2/pkg/observability"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

type Infrastructure interface {
	Postgres() *database.Postgres
	Redis() *database.Redis
	Logger() *zap.Logger
	MetricsHandler() http.Handler
	MeterProvider() *metric.MeterProvider

	Shutdown(ctx context.Context) error
}

type infrastructure struct {
	postgres       *database.Postgres
	redis          *database.Redis
	logger         *zap.Logger
	metricsHandler http.Handler
	meterProvider  *metric.MeterProvider
}

var _ Infrastructure = &infrastructure{}

func NewInfrastructure(ctx context.Context, cfg config.Config) (*infrastructure, error) {
	i := &infrastructure{}

	logger, err := observability.InitLogger(cfg.Env)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	i.logger = logger

	postgres, err := database.NewPostgres(cfg.Postgres.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	i.postgres = postgres

	redis, err := database.NewRedis(cfg.Redis.Address(), cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		_ = i.postgres.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	i.redis = redis

	meterProvider, metricsHandler, err := observability.InitTelemetry("auth-service")
	if err != nil {
		_ = i.postgres.Close()
		_ = i.redis.Close()
		return nil, fmt.Errorf("failed to initialize telemetry: %w", err)
	}
	i.meterProvider = meterProvider
	i.metricsHandler = metricsHandler

	return i, nil
}

func (i *infrastructure) Postgres() *database.Postgres {
	return i.postgres
}

func (i *infrastructure) Redis() *database.Redis {
	return i.redis
}

func (i *infrastructure) Logger() *zap.Logger {
	return i.logger
}

func (i *infrastructure) MetricsHandler() http.Handler {
	return i.metricsHandler
}

func (i *infrastructure) MeterProvider() *metric.MeterProvider {
	return i.meterProvider
}

func (i *infrastructure) Shutdown(ctx context.Context) error {
	errs := make(chan error, 4)

	go func() { errs <- i.postgres.Close() }()
	go func() { errs <- i.redis.Close() }()
	go func() { errs <- i.logger.Sync() }()
	go func() { errs <- observability.Shutdown(ctx, i.meterProvider, i.logger) }()

	return errors.Join(<-errs, <-errs, <-errs, <-errs)
}
