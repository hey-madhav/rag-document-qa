package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Errors is a centralized error-response middleware.
// Handlers in this codebase typically write responses directly, but this ensures panics
// and unhandled errors become consistent JSON responses.
func Errors() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered: method=%s path=%s panic=%v", c.Request.Method, c.Request.URL.Path, rec)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}()

		c.Next()

		if c.Writer.Written() {
			return
		}
		if len(c.Errors) == 0 {
			return
		}

		// If a handler attached errors without writing a response, return the first one.
		first := c.Errors[0]
		status := http.StatusInternalServerError
		if first.Meta != nil {
			if v, ok := first.Meta.(int); ok && v != 0 {
				status = v
			}
		}

		if status >= http.StatusInternalServerError {
			log.Printf(
				"critical request error: method=%s path=%s status=%d error=%q",
				c.Request.Method,
				c.Request.URL.Path,
				status,
				first.Error(),
			)
		}

		c.AbortWithStatusJSON(status, gin.H{"error": first.Error()})
	}
}
