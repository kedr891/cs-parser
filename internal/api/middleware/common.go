package middleware

import (
	"time"

	"github.com/cs-parser/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Logger - middleware для логирования запросов
func Logger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()

		log.Info("HTTP Request",
			"method", method,
			"path", path,
			"status", statusCode,
			"duration", duration.String(),
			"ip", c.ClientIP(),
		)
	}
}

// CORS - middleware для CORS
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RateLimit - простой rate limiter на базе IP
// В продакшене лучше использовать Redis-based rate limiter
func RateLimit(requestsPerMinute int) gin.HandlerFunc {
	// Упрощённая версия - в продакшене нужен Redis
	return func(c *gin.Context) {
		// TODO: Implement Redis-based rate limiting
		c.Next()
	}
}
