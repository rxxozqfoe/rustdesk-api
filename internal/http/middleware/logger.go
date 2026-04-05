package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/lib/logger"
)

// Logger returns a gin middleware that emits a structured log record for
// every HTTP request. It replaces gin's built-in text logger with slog
// output including method, path, status, latency, client IP, response
// size, user-agent, and any errors attached to the gin.Context.
//
// The level of each record reflects the response status: 5xx -> Error,
// 4xx -> Warn, anything else -> Info.
func Logger(log *logger.Logger) gin.HandlerFunc {
	sl := log.Slog()
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		if raw != "" {
			path = path + "?" + raw
		}

		status := c.Writer.Status()
		latency := time.Since(start)
		attrs := []any{
			slog.Int("status", status),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("ip", c.ClientIP()),
			slog.Duration("latency", latency),
			slog.Int("size", c.Writer.Size()),
			slog.String("ua", c.Request.UserAgent()),
		}
		if errMsg := c.Errors.ByType(gin.ErrorTypePrivate).String(); errMsg != "" {
			attrs = append(attrs, slog.String("error", errMsg))
		}

		const msg = "HTTP"
		switch {
		case status >= 500:
			sl.Error(msg, attrs...)
		case status >= 400:
			sl.Warn(msg, attrs...)
		default:
			sl.Info(msg, attrs...)
		}
	}
}
