package observability

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PrometheusHandler returns a Gin handler for Prometheus metrics
func PrometheusHandler(handler http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if handler != nil {
			handler.ServeHTTP(c.Writer, c.Request)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "metrics handler not initialized",
			})
		}
	}
}
