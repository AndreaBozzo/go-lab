/*
internal/middleware/logging.go
Package middleware provides HTTP middleware for the API Gateway.
*/

package middleware

import (
	"log"
	"time"

	"github.com/AndreaBozzo/go-lab/internal/collector"
	"github.com/AndreaBozzo/go-lab/internal/storage"
	"github.com/gin-gonic/gin"
)

// LoggingMiddleware creates a middleware that logs all HTTP requests
func LoggingMiddleware(store storage.LogStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record start time
		startTime := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Get backend from context (set by proxy handler)
		backend, _ := c.Get("backend")
		backendStr := ""
		if backend != nil {
			backendStr = backend.(string)
		}

		// Create log entry
		entry := collector.LogEntry{
			Source:     "apigateway",
			Level:      getLogLevel(c.Writer.Status()),
			Message:    buildLogMessage(c, latency),
			Time:       startTime,
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			StatusCode: c.Writer.Status(),
			Latency:    latency,
			ClientIP:   c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			Backend:    backendStr,
		}

		// Save to storage asynchronously to avoid blocking
		go func() {
			if err := store.Save([]collector.LogEntry{entry}); err != nil {
				log.Printf("Failed to save log entry: %v", err)
			}
		}()

		// Also log to stdout for immediate visibility
		log.Printf("[%s] %s %s - %d (%v) - Backend: %s",
			entry.Level,
			entry.Method,
			entry.Path,
			entry.StatusCode,
			entry.Latency,
			backendStr)
	}
}

// getLogLevel determines log level based on HTTP status code
func getLogLevel(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "ERROR"
	case statusCode >= 400:
		return "WARN"
	case statusCode >= 300:
		return "INFO"
	default:
		return "INFO"
	}
}

// buildLogMessage creates a human-readable log message
func buildLogMessage(c *gin.Context, latency time.Duration) string {
	return c.Request.Method + " " + c.Request.URL.Path + " completed in " + latency.String()
}
