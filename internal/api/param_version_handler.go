package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/db"
)

// ParamVersionHandler provides HTTP endpoints for parameter version management.
type ParamVersionHandler struct {
	repo *db.Repository
}

func NewParamVersionHandler(repo *db.Repository) *ParamVersionHandler {
	return &ParamVersionHandler{repo: repo}
}

// ListParamVersions godoc
// @Summary List parameter versions for a strategy
// @Tags params
// @Produce json
// @Param strategy_id path string true "Strategy ID"
// @Success 200 {array} db.ParamVersion
// @Router /api/v1/strategies/{strategy_id}/params [get]
func (h *ParamVersionHandler) List(c *gin.Context) {
	strategyID := c.Param("id")
	if strategyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "strategy_id required"})
		return
	}

	versions, err := h.repo.ListParamVersions(c.Request.Context(), strategyID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if versions == nil {
		versions = []db.ParamVersion{}
	}
	c.JSON(http.StatusOK, versions)
}

// GetActiveParams godoc
// @Summary Get active parameters for a strategy
// @Tags params
// @Produce json
// @Param strategy_id path string true "Strategy ID"
// @Success 200 {object} db.ParamVersion
// @Router /api/v1/strategies/{strategy_id}/params/active [get]
func (h *ParamVersionHandler) GetActive(c *gin.Context) {
	strategyID := c.Param("id")
	if strategyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "strategy_id required"})
		return
	}

	pv, err := h.repo.GetActiveParams(c.Request.Context(), strategyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if pv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active params found"})
		return
	}
	c.JSON(http.StatusOK, pv)
}

// ActivateParams godoc
// @Summary Activate a parameter version
// @Tags params
// @Accept json
// @Produce json
// @Param strategy_id path string true "Strategy ID"
// @Param body body object true "version_tag"
// @Success 200 {object} object
// @Router /api/v1/strategies/{strategy_id}/params/activate [post]
func (h *ParamVersionHandler) Activate(c *gin.Context) {
	strategyID := c.Param("id")
	var req struct {
		VersionTag string `json:"version_tag" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.ActivateParams(c.Request.Context(), strategyID, req.VersionTag); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "strategy_id": strategyID, "active_version": req.VersionTag})
}

// DeactivateParams godoc
// @Summary Deactivate all params (revert to registry defaults)
// @Tags params
// @Param strategy_id path string true "Strategy ID"
// @Success 200 {object} object
// @Router /api/v1/strategies/{strategy_id}/params/deactivate [post]
func (h *ParamVersionHandler) Deactivate(c *gin.Context) {
	strategyID := c.Param("id")
	if err := h.repo.DeactivateAllParams(c.Request.Context(), strategyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "strategy_id": strategyID, "message": "reverted to registry defaults"})
}

func (h *ParamVersionHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/strategies/:id/params", h.List)
	r.GET("/strategies/:id/params/active", h.GetActive)
	r.POST("/strategies/:id/params/activate", h.Activate)
	r.POST("/strategies/:id/params/deactivate", h.Deactivate)
}
