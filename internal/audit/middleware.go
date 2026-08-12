package audit

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

type HTTPAuditMiddleware struct {
	logger *Logger
}

func NewHTTPAuditMiddleware(logger *Logger) *HTTPAuditMiddleware {
	return &HTTPAuditMiddleware{logger: logger}
}

func (m *HTTPAuditMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		if m.logger == nil {
			return
		}

		userID := c.GetString("user_id")
		if userID == "" {
			userID = "anonymous"
		}

		details := map[string]interface{}{
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"status_code": c.Writer.Status(),
			"latency_ms":  time.Since(start).Milliseconds(),
			"query":       c.Request.URL.RawQuery,
		}

		if c.Writer.Status() >= 400 {
			details["error"] = c.Errors.String()
		}

		entry := Entry{
			Timestamp:    start,
			UserID:       userID,
			Action:       ActionAdminAction,
			ResourceType: "http_request",
			ResourceID:   c.Request.URL.Path,
			Details:      details,
			SourceIP:     c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
		}

		if err := m.logger.Log(c.Request.Context(), entry); err != nil {
			slog.Error("failed to log audit request", "error", err, "component", "audit")
		}
	}
}
