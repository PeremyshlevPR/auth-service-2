package acceptance

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prperemyshlev/auth-service-2/internal/config"
	"github.com/prperemyshlev/auth-service-2/internal/handler"
	"github.com/prperemyshlev/auth-service-2/internal/repository"
	"github.com/prperemyshlev/auth-service-2/internal/service"
	"github.com/prperemyshlev/auth-service-2/internal/utils"
	"github.com/prperemyshlev/auth-service-2/pkg/database"
	"github.com/prperemyshlev/auth-service-2/pkg/observability"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
)

// TODO: вынести объект App в internal/app и билдить его там. В тесте просто вызывать конструктор для него и прикидывать все зависимости.
// тогда тесты будут работать с тем же экземпляром приложения, что и main.go

// TestApp represents a test application instance
type TestApp struct {
	Config       *config.Config
	Router       *gin.Engine
	Server       *http.Server
	Listener     net.Listener
	BaseURL      string
	AuthService  service.AuthService
	AuthHandler  *handler.AuthHandler
	Repositories *repository.Repositories
	JWTManager   *utils.JWTManager
	Blacklist    *service.TokenBlacklistService
	RateLimiter  *service.RateLimiter
	Logger       *zap.Logger
	Postgres     *database.Postgres
	Redis        *database.Redis
}

// NewTestApp creates a new test application instance
func NewTestApp(postgres *database.Postgres, redis *database.Redis) (*TestApp, error) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "localhost",
			Port:         "0", // Use 0 to get a random available port
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
			BCryptCost:        12,
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

	gin.SetMode(gin.TestMode)

	logger, err := observability.InitLogger("test")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	_, metricsHandler, err := observability.InitTelemetry("auth-service-test")
	if err != nil {
		logger.Sync()
		return nil, fmt.Errorf("failed to initialize telemetry: %w", err)
	}

	repos := repository.NewRepositories(postgres)
	jwtManager := utils.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry.Duration,
		cfg.JWT.RefreshTokenExpiry.Duration,
	)

	blacklistService := service.NewTokenBlacklistService(redis)
	rateLimiter := service.NewRateLimiter(redis)
	authService := service.NewAuthService(
		repos.User,
		repos.Token,
		jwtManager,
		blacklistService,
		cfg.Security.BCryptCost,
		cfg.JWT.RefreshTokenExpiry.Duration,
	)

	authHandler := handler.NewAuthHandler(authService)

	router := gin.New()
	router.Use(otelgin.Middleware("auth-service-test"))
	router.Use(handler.LoggerMiddleware(logger))
	router.Use(handler.CORSMiddleware(cfg.CORS.AllowedOrigins, cfg.CORS.AllowedMethods, cfg.CORS.AllowedHeaders))
	setupRoutes(router, authHandler, authService, rateLimiter, cfg, metricsHandler)

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		logger.Sync()
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	addr := listener.Addr().(*net.TCPAddr)
	baseURL := fmt.Sprintf("http://localhost:%d", addr.Port)

	srv := &http.Server{
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout.Duration,
		WriteTimeout: cfg.Server.WriteTimeout.Duration,
	}

	app := &TestApp{
		Config:       cfg,
		Router:       router,
		Server:       srv,
		Listener:     listener,
		BaseURL:      baseURL,
		AuthService:  authService,
		AuthHandler:  authHandler,
		Repositories: repos,
		JWTManager:   jwtManager,
		Blacklist:    blacklistService,
		RateLimiter:  rateLimiter,
		Logger:       logger,
		Postgres:     postgres,
		Redis:        redis,
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start test server", zap.Error(err))
		}
	}()
	time.Sleep(100 * time.Millisecond)

	return app, nil
}

func (app *TestApp) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	if app.Listener != nil {
		if err := app.Listener.Close(); err != nil {
			return fmt.Errorf("failed to close listener: %w", err)
		}
	}

	if app.Logger != nil {
		app.Logger.Sync()
	}

	return nil
}

func setupRoutes(router *gin.Engine, authHandler *handler.AuthHandler, authService service.AuthService, rateLimiter *service.RateLimiter, cfg *config.Config, metricsHandler http.Handler) {
	// Metrics endpoint
	router.GET("/metrics", observability.PrometheusHandler(metricsHandler))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "auth-service",
		})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register",
				handler.RateLimitMiddleware(rateLimiter, cfg.Security.RateLimitRequests, cfg.Security.RateLimitWindow.Duration, handler.IPBasedKey),
				authHandler.Register,
			)
			auth.POST("/login",
				handler.RateLimitMiddleware(rateLimiter, cfg.Security.RateLimitRequests, cfg.Security.RateLimitWindow.Duration, handler.IPBasedKey),
				authHandler.Login,
			)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", handler.AuthMiddleware(authService), authHandler.Logout)
			auth.GET("/me", handler.AuthMiddleware(authService), authHandler.GetMe)
		}
	}
}
