package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/universe"
)

type UniverseHandler struct {
	repo    *db.Repository
	manager *universe.UniverseManager
}

func NewUniverseHandler(repo *db.Repository, manager *universe.UniverseManager) *UniverseHandler {
	return &UniverseHandler{repo: repo, manager: manager}
}

func (h *UniverseHandler) GetCurrent(c *gin.Context) {
	if h.manager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "universe manager not initialized"})
		return
	}
	symbols, err := h.manager.GetCurrentUniverse(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"symbols": symbols, "total": len(symbols)})
}

func (h *UniverseHandler) ListHistory(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"snapshots": []gin.H{}})
		return
	}

	startStr := c.Query("start")
	endStr := c.Query("end")
	now := time.Now()

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		start = now.AddDate(0, -1, 0)
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		end = now
	}

	snaps, err := h.repo.ListUniverseSnapshots(c.Request.Context(), "00000000-0000-0000-0000-000000000001", start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type snapshotSummary struct {
		ID           string    `json:"id"`
		SnapshotDate string    `json:"snapshot_date"`
		SymbolCount  int       `json:"symbol_count"`
		CreatedAt    time.Time `json:"created_at"`
	}
	results := make([]snapshotSummary, len(snaps))
	for i, s := range snaps {
		results[i] = snapshotSummary{
			ID:           s.ID,
			SnapshotDate: s.SnapshotDate.Format("2006-01-02"),
			SymbolCount:  len(s.SymbolIDs),
			CreatedAt:    s.CreatedAt,
		}
	}
	c.JSON(http.StatusOK, gin.H{"snapshots": results})
}

func (h *UniverseHandler) Refresh(c *gin.Context) {
	if h.manager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "universe manager not initialized"})
		return
	}
	if err := h.manager.Refresh(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	symbols, _ := h.manager.GetCurrentUniverse(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"refreshed": true, "total": len(symbols)})
}

func (h *UniverseHandler) Override(c *gin.Context) {
	var req struct {
		Ticker string `json:"ticker" binding:"required"`
		Action string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.manager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "universe manager not initialized"})
		return
	}

	switch req.Action {
	case "add":
		if err := h.manager.AddManualOverride(c.Request.Context(), req.Ticker); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	case "remove":
		if err := h.manager.RemoveManualOverride(c.Request.Context(), req.Ticker); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be 'add' or 'remove'"})
		return
	}

	symbols, _ := h.manager.GetCurrentUniverse(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"overridden": true, "action": req.Action, "ticker": req.Ticker, "total": len(symbols)})
}

func (h *UniverseHandler) ListConfigs(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"configs": []gin.H{}})
		return
	}
	configs, err := h.repo.ListUniverseConfigs(c.Request.Context(), "00000000-0000-0000-0000-000000000001")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"configs": configs})
}

func (h *UniverseHandler) CreateConfig(c *gin.Context) {
	var req db.UniverseConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.repo == nil {
		c.JSON(http.StatusCreated, gin.H{"created": true})
		return
	}
	if err := h.repo.InsertUniverseConfig(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"created": true, "id": req.ID})
}

func (h *UniverseHandler) UpdateConfig(c *gin.Context) {
	id := c.Param("id")
	var req db.UniverseConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = id
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"updated": true})
		return
	}
	if err := h.repo.InsertUniverseConfig(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": true})
}

func (h *UniverseHandler) ActivateConfig(c *gin.Context) {
	id := c.Param("id")
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"activated": true})
		return
	}
	if err := h.repo.SetActiveUniverseConfig(c.Request.Context(), id, "00000000-0000-0000-0000-000000000001"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"activated": true})
}

func (h *UniverseHandler) RegisterRoutes(rg *gin.RouterGroup) {
	universe := rg.Group("/universe")
	{
		universe.GET("/current", h.GetCurrent)
		universe.GET("/history", h.ListHistory)
		universe.POST("/refresh", h.Refresh)
		universe.POST("/override", h.Override)
		universe.GET("/configs", h.ListConfigs)
		universe.POST("/configs", h.CreateConfig)
		universe.PUT("/configs/:id", h.UpdateConfig)
		universe.POST("/configs/:id/activate", h.ActivateConfig)
	}
}
