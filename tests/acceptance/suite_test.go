package acceptance

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/prperemyshlev/auth-service-2/internal/app"
	"github.com/prperemyshlev/auth-service-2/internal/config"
	"github.com/prperemyshlev/auth-service-2/pkg/database"
	"github.com/prperemyshlev/auth-service-2/pkg/observability"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

const (
	postgresDSN = "postgres://auth_service:auth_service_password@localhost:5432/auth_service_db?sslmode=disable"
	redisDSN    = "localhost:6379"
)

type Suite struct {
	suite.Suite
	Postgres *database.Postgres
	Redis    *database.Redis
	BaseURL  string
	ctx      context.Context
	cancel   context.CancelFunc
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) SetupSuite() {
	pg, err := database.NewPostgres(postgresDSN)
	if err != nil {
		s.T().Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	redis, err := database.NewRedis(redisDSN, "", 0)
	if err != nil {
		pg.Close()
		s.T().Fatalf("Failed to connect to Redis: %v", err)
	}

	if err := s.setupDatabase(pg.DB); err != nil {
		pg.Close()
		redis.Close()
		s.T().Fatalf("Failed to run migrations: %v", err)
	}

	s.Postgres = pg
	s.Redis = redis

	baseURL, ctx, cancel, err := s.startApp(pg, redis)
	if err != nil {
		_ = pg.Close()
		_ = redis.Close()
		s.T().Fatalf("Failed to start app: %v", err)
	}

	s.BaseURL = baseURL
	s.ctx = ctx
	s.cancel = cancel
}

func (s *Suite) TearDownSuite() {
	if s.cancel != nil {
		s.cancel()
		time.Sleep(100 * time.Millisecond)
	}
	if s.Postgres != nil {
		_ = s.Postgres.Close()
	}
	if s.Redis != nil {
		_ = s.Redis.Close()
	}
}

func (s *Suite) SetupTest() {
	if err := s.cleanupDatabase(); err != nil {
		s.T().Fatalf("Failed to cleanup database: %v", err)
	}

	ctx := context.Background()
	if err := s.Redis.Client.FlushDB(ctx).Err(); err != nil {
		s.T().Fatalf("Failed to flush Redis: %v", err)
	}
}

func (s *Suite) startApp(postgres *database.Postgres, redis *database.Redis) (string, context.Context, context.CancelFunc, error) {
	cfg := s.createTestConfig()

	gin.SetMode(gin.TestMode)

	infra, err := s.createTestInfrastructure(postgres, redis, cfg)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to initialize test infrastructure: %w", err)
	}

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create listener: %w", err)
	}

	addr := listener.Addr().(*net.TCPAddr)
	baseURL := fmt.Sprintf("http://localhost:%d", addr.Port)

	cfg.Server.Port = fmt.Sprintf("%d", addr.Port)
	listener.Close()

	application := app.NewApp(infra, cfg)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := application.Run(ctx); err != nil {
			infra.Logger().Error("Application failed to run", zap.Error(err))
		}
	}()

	time.Sleep(100 * time.Millisecond)

	return baseURL, ctx, cancel, nil
}

func (s *Suite) createTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:         "localhost",
			Port:         "0",
			ReadTimeout:  config.Duration{Duration: 15 * time.Second},
			WriteTimeout: config.Duration{Duration: 15 * time.Second},
		},
		Postgres: config.PostgresConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "auth_service",
			Password: "auth_service_password",
			DBName:   "auth_service_db",
			SSLMode:  "disable",
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		JWT: config.JWTConfig{
			Secret:             "test-secret-key-that-is-at-least-32-characters-long",
			AccessTokenExpiry:  config.Duration{Duration: 15 * time.Minute},
			RefreshTokenExpiry: config.Duration{Duration: 7 * 24 * time.Hour},
		},
		Security: config.SecurityConfig{
			BCryptCost:        4,
			RateLimitRequests: 10,
			RateLimitWindow:   config.Duration{Duration: 1 * time.Minute},
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"http://localhost:3000"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Content-Type", "Authorization"},
		},
		Env: "test",
	}
}

func (s *Suite) createTestInfrastructure(postgres *database.Postgres, redis *database.Redis, cfg *config.Config) (*testInfrastructure, error) {
	logger, err := observability.InitLogger("test")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	meterProvider, metricsHandler, err := observability.InitTelemetry("auth-service-test")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize telemetry: %w", err)
	}

	return &testInfrastructure{
		postgres:       postgres,
		redis:          redis,
		logger:         logger,
		metricsHandler: metricsHandler,
		meterProvider:  meterProvider,
		cfg:            cfg,
	}, nil
}

func (s *Suite) cleanupDatabase() error {
	return s.executeSQLFile(s.Postgres.DB, filepath.Join("testdata", "cleanup.sql"))
}

func (s *Suite) setupDatabase(db *sql.DB) error {
	return s.executeSQLFile(db, filepath.Join("testdata", "setup.sql"))
}

func (s *Suite) executeSQLFile(db *sql.DB, filePath string) error {
	sqlBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	if _, err := db.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("failed to execute %s: %w", filePath, err)
	}

	return nil
}

type testInfrastructure struct {
	postgres       *database.Postgres
	redis          *database.Redis
	logger         *zap.Logger
	metricsHandler http.Handler
	meterProvider  *metric.MeterProvider
	cfg            *config.Config
}

func (i *testInfrastructure) Postgres() *database.Postgres {
	return i.postgres
}

func (i *testInfrastructure) Redis() *database.Redis {
	return i.redis
}

func (i *testInfrastructure) Logger() *zap.Logger {
	return i.logger
}

func (i *testInfrastructure) MetricsHandler() http.Handler {
	return i.metricsHandler
}

func (i *testInfrastructure) MeterProvider() *metric.MeterProvider {
	return i.meterProvider
}

func (i *testInfrastructure) Shutdown(ctx context.Context) error {
	if i.logger != nil {
		_ = i.logger.Sync()
	}
	if i.meterProvider != nil {
		_ = observability.Shutdown(ctx, i.meterProvider, i.logger)
	}
	return nil
}
