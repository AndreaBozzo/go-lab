/*
internal/middleware/recovery.go
Package middleware provides panic recovery middleware.
*/

package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// RecoveryMiddleware creates a middleware that recovers from panics
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				stack := debug.Stack()
				log.Printf("PANIC recovered: %v\n%s", err, stack)

				// Return error response
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"message": fmt.Sprintf("Panic: %v", err),
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}
