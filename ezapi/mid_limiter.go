package ezapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequestLimiter(maxConcurrent int) gin.HandlerFunc {
	if maxConcurrent <= 0 {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	sem := make(chan struct{}, maxConcurrent)
	return func(c *gin.Context) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			c.Next()
		default:
			c.Header("Retry-After", "5")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too Many Requests",
				"message": "The server is currently overloaded, please try again later.",
			})
			c.Abort()
		}
	}
}
