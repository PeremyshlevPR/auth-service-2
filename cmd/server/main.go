package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

func main() {
	// Load configuration
	ctx := context.Background()
	cfg, err := config.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connections
	postgres, err := database.NewPostgres(cfg.Postgres.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer postgres.Close()

	redis, err := database.NewRedis(cfg.Redis.Address(), cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()

	// Set Gin mode
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize logger
	logger, err := observability.InitLogger(cfg.Env)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Initialize OpenTelemetry metrics
	meterProvider, metricsHandler, err := observability.InitTelemetry("auth-service")
	if err != nil {
		logger.Fatal("Failed to initialize telemetry", zap.Error(err))
	}
	defer observability.Shutdown(context.Background(), meterProvider, logger)

	// Initialize repositories
	repos := repository.NewRepositories(postgres)

	// Initialize JWT manager
	jwtManager := utils.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry.Duration,
		cfg.JWT.RefreshTokenExpiry.Duration,
	)

	// Initialize token blacklist service
	blacklistService := service.NewTokenBlacklistService(redis)

	// Initialize rate limiter
	rateLimiter := service.NewRateLimiter(redis)

	// Initialize auth service
	authService := service.NewAuthService(
		repos.User,
		repos.Token,
		jwtManager,
		blacklistService,
		cfg.Security.BCryptCost,
		cfg.JWT.RefreshTokenExpiry.Duration,
	)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)

	// Initialize router
	router := gin.Default()

	// Setup middleware
	// OpenTelemetry instrumentation (automatically collects metrics and traces)
	router.Use(otelgin.Middleware("auth-service"))

	// Structured logging
	router.Use(handler.LoggerMiddleware(logger))

	// CORS
	router.Use(handler.CORSMiddleware(cfg.CORS.AllowedOrigins, cfg.CORS.AllowedMethods, cfg.CORS.AllowedHeaders))

	// Setup routes
	setupRoutes(router, authHandler, authService, rateLimiter, cfg, metricsHandler)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout.Duration,
		WriteTimeout: cfg.Server.WriteTimeout.Duration,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting",
			zap.String("host", cfg.Server.Host),
			zap.String("port", cfg.Server.Port),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// setupRoutes sets up all routes
func setupRoutes(router *gin.Engine, authHandler *handler.AuthHandler, authService service.AuthService, rateLimiter *service.RateLimiter, cfg *config.Config, metricsHandler http.Handler) {
	// Metrics endpoint (Prometheus format)
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
			// Apply rate limiting to register and login endpoints
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
