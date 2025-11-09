package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware creates a CORS middleware
func CORSMiddleware(allowedOrigins, allowedMethods, allowedHeaders []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
		c.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
