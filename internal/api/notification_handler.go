package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	mw "github.com/lee-econ/orca-core/internal/api/middleware"
	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/notify"
)

type NotificationHandler struct {
	repo    *db.Repository
	manager *notify.Manager
}

func NewNotificationHandler(repo *db.Repository, mgr *notify.Manager) *NotificationHandler {
	return &NotificationHandler{repo: repo, manager: mgr}
}

func (h *NotificationHandler) GetSettings(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"settings": defaultNotificationSettings()})
		return
	}

	settings, err := h.repo.GetNotificationSettings(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"settings": defaultNotificationSettings()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"settings": settings.Settings})
}

func (h *NotificationHandler) UpdateSettings(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req struct {
		Settings map[string]interface{} `json:"settings" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"updated": true})
		return
	}

	if err := h.repo.UpsertNotificationSettings(c.Request.Context(), userID, req.Settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"updated": true, "settings": req.Settings})
}

func (h *NotificationHandler) TestNotification(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	if h.manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Notification manager not configured"})
		return
	}

	var req struct {
		Channel string `json:"channel" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	event := notify.NewEvent(
		notify.EventType("test_"+req.Channel),
		notify.LevelInfo,
		"Test Notification",
		"This is a test notification from Orca Algo via "+req.Channel,
	)

	h.manager.Publish(event)

	c.JSON(http.StatusOK, gin.H{"sent": true, "channel": req.Channel})
}

func (h *NotificationHandler) RegisterRoutes(router *gin.RouterGroup) {
	settings := router.Group("/settings/notifications")
	settings.Use(mw.AuthMiddleware())
	{
		settings.GET("", h.GetSettings)
		settings.PUT("", h.UpdateSettings)
		settings.POST("/test", h.TestNotification)
	}
}

func defaultNotificationSettings() map[string]interface{} {
	return map[string]interface{}{
		"telegram": map[string]interface{}{
			"enabled": false,
			"params":  map[string]interface{}{"chat_ids": []string{}},
			"filters": map[string]interface{}{
				"kill_switch_triggered": true,
				"drawdown_warning":     true,
				"daily_loss_warning":   true,
				"regime_changed":       false,
				"order_filled":         false,
			},
		},
		"email": map[string]interface{}{
			"enabled": false,
			"params":  map[string]interface{}{"to_emails": []string{}},
			"filters": map[string]interface{}{
				"backtest_complete":    true,
				"kill_switch_triggered": true,
				"credential_expiry":     true,
			},
		},
		"push": map[string]interface{}{
			"enabled": true,
			"params":  map[string]interface{}{},
			"filters": map[string]interface{}{
				"order_filled":    true,
				"position_closed": true,
			},
		},
	}
}
