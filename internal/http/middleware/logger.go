package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/internal/config"
	"github.com/lejianwen/rustdesk-api/v2/internal/lib/logger"
)

// responseBodyWriter wraps gin.ResponseWriter to capture the response body.
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Logger returns a gin middleware that emits a structured log record for
// every HTTP request. It logs method, path, status, latency, client IP,
// response size, user-agent, request/response headers, request body, and
// response body.
//
// The heartbeat endpoint (/api/heartbeat) can be silenced via config.
//
// The level of each record reflects the response status: 5xx -> Error,
// 4xx -> Warn, anything else -> Info.
func Logger(log *logger.Logger, loggerCfg config.Logger) gin.HandlerFunc {
	sl := log.Slog()
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Skip heartbeat logging if disabled
		if !loggerCfg.LogHeartbeat && strings.HasSuffix(path, "/heartbeat") {
			c.Next()
			return
		}

		// Skip logging for file download endpoints (binary stream, not useful in logs)
		if strings.Contains(path, "/custom-client/download/") {
			c.Next()
			return
		}

		start := time.Now()

		// Read request body
		var reqBody string
		if c.Request.Body != nil && c.Request.ContentLength > 0 {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				reqBody = string(bodyBytes)
				// Restore the body so downstream handlers can read it
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// Capture request headers
		reqHeaders := formatHeaders(c.Request.Header)

		// Wrap response writer to capture body
		rbw := &responseBodyWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = rbw

		c.Next()

		if raw != "" {
			path = path + "?" + raw
		}

		status := c.Writer.Status()
		latency := time.Since(start)

		// Capture response headers
		respHeaders := formatHeaders(http.Header(c.Writer.Header()))

		attrs := []any{
			slog.Int("status", status),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("ip", c.ClientIP()),
			slog.Duration("latency", latency),
			slog.Int("size", c.Writer.Size()),
			slog.String("ua", c.Request.UserAgent()),
			slog.String("req_headers", reqHeaders),
			slog.String("resp_headers", respHeaders),
		}
		if reqBody != "" {
			attrs = append(attrs, slog.String("req_body", reqBody))
		}
		if respBody := rbw.body.String(); respBody != "" {
			attrs = append(attrs, slog.String("resp_body", respBody))
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

// formatHeaders converts http.Header to a compact string representation.
func formatHeaders(h http.Header) string {
	if len(h) == 0 {
		return ""
	}
	var b strings.Builder
	for k, v := range h {
		if b.Len() > 0 {
			b.WriteString("; ")
		}
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(strings.Join(v, ","))
	}
	return b.String()
}
