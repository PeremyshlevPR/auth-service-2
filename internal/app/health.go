package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const healthCheckTimeout = 2 * time.Second

type HealthChecker struct {
	infra Infrastructure
}

func NewHealthChecker(infra Infrastructure) *HealthChecker {
	return &HealthChecker{
		infra: infra,
	}
}

func (h *HealthChecker) check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	errs := make(chan error, 2)

	go func() {
		errs <- h.infra.Postgres().Ping(ctx)
	}()

	go func() {
		errs <- h.infra.Redis().Ping(ctx)
	}()

	return errors.Join(<-errs, <-errs)
}

func (h *HealthChecker) Handler(c *gin.Context) {
	if err := h.check(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "fail",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "pass",
	})
}
