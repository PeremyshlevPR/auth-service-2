package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/prperemyshlev/auth-service-2/internal/app"
	"github.com/prperemyshlev/auth-service-2/internal/config"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	infra, err := app.NewInfrastructure(ctx, *cfg)
	if err != nil {
		log.Fatalf("Failed to initialize infrastructure: %v", err)
	}

	application := app.NewApp(infra, cfg)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		infra.Logger().Info("Received shutdown signal")
		cancel()
	}()

	if err := application.Run(ctx); err != nil {
		infra.Logger().Fatal("Application failed", zap.Error(err))
	}
}
