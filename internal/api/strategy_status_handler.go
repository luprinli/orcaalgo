package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/db"
)

type StrategyStatusHandler struct {
	repo *db.Repository
}

func NewStrategyStatusHandler(repo *db.Repository) *StrategyStatusHandler {
	return &StrategyStatusHandler{repo: repo}
}

func (h *StrategyStatusHandler) GetStatus(c *gin.Context) {
	strategyID := c.Param("id")
	if strategyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	status, err := h.repo.GetStrategyStatus(c.Request.Context(), strategyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "strategy status not found"})
		return
	}

	c.JSON(http.StatusOK, status)
}

func (h *StrategyStatusHandler) ListStatuses(c *gin.Context) {
	statuses, err := h.repo.ListStrategyStatuses(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statuses == nil {
		statuses = []db.StrategyStatus{}
	}
	c.JSON(http.StatusOK, statuses)
}

type promoteRequest struct {
	Reason              string  `json:"reason" binding:"required"`
	Weight              float64 `json:"allocation_pct"`
	OrchestrationRunID  *string `json:"orchestration_run_id,omitempty"`
}

func (h *StrategyStatusHandler) Promote(c *gin.Context) {
	strategyID := c.Param("id")
	if strategyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	var req promoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	allocationPct := req.Weight
	if allocationPct <= 0 {
		allocationPct = 0.5
	}

	status := &db.StrategyStatus{
		StrategyID:         strategyID,
		Status:             "active",
		AllocationPct:      allocationPct,
		DemotionReason:      nil,
		OrchestrationRunID: req.OrchestrationRunID,
	}

	if err := h.repo.UpsertStrategyStatus(c.Request.Context(), status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"strategy_id": strategyID,
		"status":      "active",
		"reason":      req.Reason,
	})
}

type demoteRequest struct {
	Reason        string  `json:"reason" binding:"required"`
	AllocationPct float64 `json:"allocation_pct"`
}

func (h *StrategyStatusHandler) Demote(c *gin.Context) {
	strategyID := c.Param("id")
	if strategyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	var req demoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reason := req.Reason

	status := &db.StrategyStatus{
		StrategyID:     strategyID,
		Status:         "inactive",
		AllocationPct:  req.AllocationPct,
		DemotionReason: &reason,
	}

	if err := h.repo.UpsertStrategyStatus(c.Request.Context(), status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"strategy_id":    strategyID,
		"status":         "inactive",
		"allocation_pct": req.AllocationPct,
		"reason":         req.Reason,
	})
}

func (h *StrategyStatusHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/strategies/:id/status", h.GetStatus)
	r.GET("/strategies/statuses", h.ListStatuses)
	r.POST("/strategies/:id/promote", h.Promote)
	r.POST("/strategies/:id/demote", h.Demote)
}
