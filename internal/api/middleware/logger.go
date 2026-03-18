package middleware

import (
	"time"

	"github.com/example/youtube-dialogue-crawler/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		if query != "" {
			path = path + "?" + query
		}

		logger.Infof("%s %s %d %v",
			c.Request.Method,
			path,
			status,
			latency,
		)
	}
}
