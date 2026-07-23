package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"github.com/lee-econ/orca-core/internal/propfirm"
)

type PropFirmHandler struct {
	manager *propfirm.Manager
}

func NewPropFirmHandler(manager *propfirm.Manager) *PropFirmHandler {
	return &PropFirmHandler{manager: manager}
}

func (h *PropFirmHandler) CreateProfile(c *gin.Context) {
	var req struct {
		ID                     string    `json:"id" binding:"required"`
		Name                   string    `json:"name" binding:"required"`
		MaxDailyLossPct        float64   `json:"max_daily_loss_pct"`
		MaxDrawdownPct         float64   `json:"max_drawdown_pct"`
		DrawdownType           string    `json:"drawdown_type"`
		MaxPositionPct         float64   `json:"max_position_pct"`
		MaxOpenPositions       int       `json:"max_open_positions"`
		MaxTradesPerDay        int       `json:"max_trades_per_day"`
		ConsistencyEnabled     bool      `json:"consistency_enabled"`
		ConsistencyThresholdPct float64  `json:"consistency_threshold_pct"`
		ConsistencyPenalty     float64   `json:"consistency_penalty"`
		ProfitTargetPctPhase1  float64   `json:"profit_target_pct_phase1"`
		ProfitTargetPctPhase2  float64   `json:"profit_target_pct_phase2"`
		MinTradingDays         int       `json:"min_trading_days"`
		RegimeMultipliers      [4]float64 `json:"regime_multipliers"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	profile := &propfirm.Profile{
		ID: req.ID, Name: req.Name,
		MaxDailyLossPct: req.MaxDailyLossPct, MaxDrawdownPct: req.MaxDrawdownPct,
		DrawdownType: req.DrawdownType, MaxPositionPct: req.MaxPositionPct,
		MaxOpenPositions: req.MaxOpenPositions, MaxTradesPerDay: req.MaxTradesPerDay,
		ConsistencyEnabled: req.ConsistencyEnabled, ConsistencyThresholdPct: req.ConsistencyThresholdPct,
		ConsistencyPenalty: req.ConsistencyPenalty,
		ProfitTargetPctPhase1: req.ProfitTargetPctPhase1, ProfitTargetPctPhase2: req.ProfitTargetPctPhase2,
		MinTradingDays: req.MinTradingDays, RegimeMultipliers: req.RegimeMultipliers,
	}
	yamlPath := filepath.Join("configs/propfirms", req.ID+".yaml")
	data, err := yaml.Marshal(profile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal profile"})
		return
	}
	if err := os.WriteFile(yamlPath, data, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write profile"})
		return
	}
	c.JSON(http.StatusCreated, profileToH(profile))
}

func (h *PropFirmHandler) UpdateProfile(c *gin.Context) {
	id := c.Param("id")
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	yamlPath := filepath.Join("configs/propfirms", id+".yaml")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}
	if err := h.manager.LoadProfile(yamlPath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}
	if err := h.manager.ActivateProfile(id); err == nil {
		_ = err
	}
	c.JSON(http.StatusOK, gin.H{"updated": true})
}

func (h *PropFirmHandler) DeleteProfile(c *gin.Context) {
	id := c.Param("id")
	builtIn := map[string]bool{"ftmo": true, "tft": true, "e8": true, "topstep": true}
	if builtIn[id] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete built-in profile"})
		return
	}
	yamlPath := filepath.Join("configs/propfirms", id+".yaml")
	os.Remove(yamlPath)
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *PropFirmHandler) OverridePhase(c *gin.Context) {
	h.manager.AdvancePhase()
	c.JSON(http.StatusOK, gin.H{"phase": h.manager.State().CurrentPhase})
}

func (h *PropFirmHandler) RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/propfirm/profiles", h.ListProfiles)
	v1.POST("/propfirm/profiles", h.CreateProfile)
	v1.PUT("/propfirm/profiles/:id", h.UpdateProfile)
	v1.DELETE("/propfirm/profiles/:id", h.DeleteProfile)
	v1.GET("/propfirm/active", h.GetActiveProfile)
	v1.PUT("/propfirm/active", h.SetActiveProfile)
	v1.GET("/propfirm/status", h.GetPropStatus)
	v1.POST("/propfirm/phase", h.OverridePhase)
}

func (h *PropFirmHandler) ListProfiles(c *gin.Context) {
	configsDir := "configs/propfirms"
	entries, err := os.ReadDir(configsDir)
	if err != nil {
		profile := propfirm.DefaultFTMOProfile()
		c.JSON(http.StatusOK, gin.H{"profiles": []gin.H{profileToH(profile)}})
		return
	}

	var profiles []gin.H
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		path := filepath.Join(configsDir, entry.Name())
		if err := h.manager.LoadProfile(path); err != nil {
			continue
		}
	}
	for id, p := range h.manager.AllProfiles() {
		profiles = append(profiles, profileToH(p))
		_ = id
	}

	if len(profiles) == 0 {
		profiles = append(profiles, profileToH(propfirm.DefaultFTMOProfile()))
	}

	c.JSON(http.StatusOK, gin.H{"profiles": profiles})
}

func (h *PropFirmHandler) GetActiveProfile(c *gin.Context) {
	p := h.manager.ActiveProfile()
	if p == nil {
		p = propfirm.DefaultFTMOProfile()
	}
	c.JSON(http.StatusOK, profileToH(p))
}

func (h *PropFirmHandler) SetActiveProfile(c *gin.Context) {
	var req struct {
		ID string `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.manager.ActivateProfile(req.ID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"active": req.ID, "status": "ok"})
}

func (h *PropFirmHandler) GetPropStatus(c *gin.Context) {
	state := h.manager.State()
	p := h.manager.ActiveProfile()
	if p == nil {
		p = propfirm.DefaultFTMOProfile()
	}
	c.JSON(http.StatusOK, gin.H{
		"profile_id":       state.ProfileID,
		"daily_pnl_pct":    state.DailyPnLPct,
		"cumulative_pnl":   state.CumulativePnL,
		"drawdown_pct":     0.0,
		"trading_days":     state.TradingDays,
		"current_phase":    state.CurrentPhase,
		"phase_target_met": state.PhaseTargetMet,
		"violated":         state.Violated,
		"violation_reason": state.ViolationReason,
		"max_daily_loss":   p.MaxDailyLossPct,
		"max_drawdown":     p.MaxDrawdownPct,
		"profit_target":    p.ProfitTargetPctPhase1,
		"consistency_mult": state.ConsistencyMult,
	})
}

func profileToH(p *propfirm.Profile) gin.H {
	return gin.H{
		"id":                     p.ID,
		"name":                   p.Name,
		"max_daily_loss_pct":     p.MaxDailyLossPct,
		"max_drawdown_pct":       p.MaxDrawdownPct,
		"drawdown_type":          p.DrawdownType,
		"max_position_pct":       p.MaxPositionPct,
		"max_open_positions":     p.MaxOpenPositions,
		"max_trades_per_day":     p.MaxTradesPerDay,
		"consistency_enabled":    p.ConsistencyEnabled,
		"consistency_threshold":  p.ConsistencyThresholdPct,
		"profit_target_phase1":   p.ProfitTargetPctPhase1,
		"profit_target_phase2":   p.ProfitTargetPctPhase2,
		"min_trading_days":       p.MinTradingDays,
		"regime_multipliers":     p.RegimeMultipliers,
	}
}
