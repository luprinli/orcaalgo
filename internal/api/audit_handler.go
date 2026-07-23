package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	mw "github.com/lee-econ/orca-core/internal/api/middleware"
	"github.com/lee-econ/orca-core/internal/audit"
	errormgr "github.com/lee-econ/orca-core/internal/error"
	"github.com/lee-econ/orca-core/internal/monitor"
)

type AuditHandler struct {
	logger        *audit.Logger
	errorMgr      *errormgr.Manager
	healthMonitor *monitor.HealthMonitor
}

func NewAuditHandler(logger *audit.Logger, em *errormgr.Manager, hm *monitor.HealthMonitor) *AuditHandler {
	return &AuditHandler{logger: logger, errorMgr: em, healthMonitor: hm}
}

func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	userID := c.Query("user_id")
	action := c.Query("action")
	resourceType := c.Query("resource_type")

	filter := audit.Filter{
		UserID: userID, Action: audit.AuditAction(action), ResourceType: resourceType, Limit: 200,
	}

	if startStr := c.Query("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.Start = t
		}
	}
	if endStr := c.Query("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.End = t
		}
	}

	if h.logger == nil {
		c.JSON(http.StatusOK, gin.H{"audit_logs": []interface{}{}})
		return
	}

	entries, err := h.logger.Query(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if entries == nil {
		entries = []audit.Entry{}
	}
	c.JSON(http.StatusOK, gin.H{"audit_logs": entries})
}

func (h *AuditHandler) ListErrorLogs(c *gin.Context) {
	component := c.Query("component")

	if h.errorMgr == nil {
		c.JSON(http.StatusOK, gin.H{"error_logs": []interface{}{}})
		return
	}

	logs, err := h.errorMgr.Query(c.Request.Context(), component, 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if logs == nil {
		logs = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{"error_logs": logs})
}

func (h *AuditHandler) GetHealth(c *gin.Context) {
	if h.healthMonitor == nil {
		c.JSON(http.StatusOK, gin.H{"healthy": true, "components": []gin.H{}, "timestamp": time.Now().Format(time.RFC3339)})
		return
	}
	status := h.healthMonitor.Status()
	c.JSON(http.StatusOK, status)
}

func (h *AuditHandler) RegisterRoutes(router *gin.RouterGroup) {
	auditGroup := router.Group("/audit")
	auditGroup.Use(mw.AuthMiddleware())
	{
		auditGroup.GET("", h.ListAuditLogs)
		auditGroup.GET("/errors", h.ListErrorLogs)
	}

	router.GET("/health", h.GetHealth)
}
