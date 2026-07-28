package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/db"
)

type SettingsHandler struct {
	repo *db.Repository
}

func NewSettingsHandler(repo *db.Repository) *SettingsHandler {
	return &SettingsHandler{repo: repo}
}

func (h *SettingsHandler) GetSettings(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"settings": gin.H{
			"general":     gin.H{"environment": "paper", "timezone": "America/New_York"},
			"risk":         gin.H{"daily_loss_limit_pct": 5.0, "max_drawdown_pct": 10.0, "consistency_threshold": 30.0},
			"notifications": []gin.H{},
			"trading_hours": gin.H{"start": "09:30", "end": "16:00"},
			"llm":          gin.H{"active_provider": "openai", "configs": gin.H{}},
			"grafana_url":  "http://localhost:3000",
		}})
		return
	}
	settings, err := h.repo.ListSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"settings": defaultSettings()})
		return
	}
	if len(settings) == 0 {
		c.JSON(http.StatusOK, gin.H{"settings": defaultSettings()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	var req map[string]map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"updated": true})
		return
	}
	for key, value := range req {
		if err := h.repo.UpsertSetting(c.Request.Context(), key, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"updated": true})
}

func (h *SettingsHandler) RegisterRoutes(router *gin.RouterGroup) {
	settings := router.Group("/settings")
	{
		settings.GET("", h.GetSettings)
		settings.PUT("", h.UpdateSettings)
	}
}

func defaultSettings() gin.H {
	return gin.H{
		"general":       gin.H{"environment": "paper", "timezone": "America/New_York"},
		"risk":          gin.H{"daily_loss_limit_pct": 5.0, "max_drawdown_pct": 10.0, "consistency_threshold": 30.0},
		"notifications": gin.H{"channels": []gin.H{}},
		"trading_hours": gin.H{"start": "09:30", "end": "16:00"},
		"llm":           gin.H{"active_provider": "", "configs": gin.H{}},
		"grafana_url":   "http://localhost:3000",
	}
}
