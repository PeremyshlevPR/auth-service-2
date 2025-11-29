package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prperemyshlev/auth-service-2/internal/config"
	"github.com/prperemyshlev/auth-service-2/internal/handler"
	"github.com/prperemyshlev/auth-service-2/internal/repository"
	"github.com/prperemyshlev/auth-service-2/internal/service"
	"github.com/prperemyshlev/auth-service-2/internal/utils"
	"github.com/prperemyshlev/auth-service-2/pkg/observability"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
)

const shutdownTimeout = 5 * time.Second

type App struct {
	infra  Infrastructure
	config *config.Config
	router *gin.Engine
	server *http.Server
}

func NewApp(infra Infrastructure, cfg *config.Config) *App {
	repos := repository.NewRepositories(infra.Postgres())

	jwtManager := utils.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry.Duration,
		cfg.JWT.RefreshTokenExpiry.Duration,
	)

	blacklistService := service.NewTokenBlacklistService(infra.Redis())
	rateLimiter := service.NewRateLimiter(infra.Redis())
	healthChecker := NewHealthChecker(infra)

	authService := service.NewAuthService(
		repos.User,
		repos.Token,
		jwtManager,
		blacklistService,
		cfg.Security.BCryptCost,
		cfg.JWT.RefreshTokenExpiry.Duration,
	)

	authHandler := handler.NewAuthHandler(authService)

	router := gin.Default()
	router.Use(otelgin.Middleware("auth-service"))
	router.Use(handler.LoggerMiddleware(infra.Logger()))
	router.Use(handler.CORSMiddleware(cfg.CORS.AllowedOrigins, cfg.CORS.AllowedMethods, cfg.CORS.AllowedHeaders))

	setupRoutes(router, cfg, authHandler, authService, rateLimiter, healthChecker, infra.MetricsHandler())

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout.Duration,
		WriteTimeout: cfg.Server.WriteTimeout.Duration,
	}

	return &App{
		infra:  infra,
		config: cfg,
		router: router,
		server: srv,
	}
}

func (a *App) Router() *gin.Engine {
	return a.router
}

func setupRoutes(
	router *gin.Engine,
	cfg *config.Config,
	authHandler *handler.AuthHandler,
	authService service.AuthService,
	rateLimiter *service.RateLimiter,
	healthChecker *HealthChecker,
	metricsHandler http.Handler,
) {
	router.GET("/metrics", observability.PrometheusHandler(metricsHandler))
	router.GET("/health", healthChecker.Handler)

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

func (a *App) Run(ctx context.Context) error {
	errChan := make(chan error, 1)

	go func() {
		a.infra.Logger().Info("Application starting",
			zap.String("host", a.config.Server.Host),
			zap.String("port", a.config.Server.Port),
		)

		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.infra.Logger().Error("Server error", zap.Error(err))
			errChan <- err
		}
	}()

	var serverErr error
	select {
	case err := <-errChan:
		a.infra.Logger().Error("Application failed to start", zap.Error(err))
		serverErr = err
	case <-ctx.Done():
		a.infra.Logger().Info("Application stopped by context")
	}

	if err := a.Shutdown(); err != nil {
		a.infra.Logger().Error("Shutdown error", zap.Error(err))
		if serverErr != nil {
			return errors.Join(serverErr, err)
		}
		return err
	}

	return serverErr
}

func (a *App) Shutdown() error {
	a.infra.Logger().Info("Application shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	errs := make(chan error, 2)

	go func() {
		errs <- a.server.Shutdown(ctx)
	}()

	go func() {
		errs <- a.infra.Shutdown(ctx)
	}()

	err := errors.Join(<-errs, <-errs)
	if err != nil {
		a.infra.Logger().Error("Shutdown failed", zap.Error(err))
		return err
	}

	a.infra.Logger().Info("Application exited successfully")
	return nil
}
