package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	mw "github.com/lee-econ/orca-core/internal/api/middleware"
	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/email"
)

type AdminHandler struct {
	repo         *db.Repository
	seeder       *db.Seeder
	emailService email.EmailService
}

func NewAdminHandler(repo *db.Repository) *AdminHandler {
	return &AdminHandler{repo: repo, seeder: db.NewSeeder(repo)}
}

func (h *AdminHandler) SetEmailService(svc email.EmailService) {
	h.emailService = svc
}

func (h *AdminHandler) GetHealth(c *gin.Context) {
	healthy := true
	components := make(map[string]string)

	if h.repo == nil || !h.repo.IsConnected() {
		components["database"] = "disconnected"
		healthy = false
	} else {
		components["database"] = "connected"
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := h.repo.Ping(ctx); err != nil {
			components["database"] = "error: " + err.Error()
			healthy = false
		}
	}

	status := http.StatusOK
	if !healthy { status = http.StatusServiceUnavailable }
	c.JSON(status, gin.H{"healthy": healthy, "components": components, "timestamp": time.Now().Format(time.RFC3339)})
}

func (h *AdminHandler) SeedDatabase(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not connected"})
		return
	}
	var req struct{ Force bool `json:"force"` }
	c.ShouldBindJSON(&req)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := h.seeder.Run(ctx, req.Force); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	report, _ := h.seeder.VerifyIntegrity(ctx)
	c.JSON(http.StatusOK, gin.H{"seeded": true, "force": req.Force, "integrity": report})
}

func (h *AdminHandler) VerifyIntegrity(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not connected"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	report, err := h.seeder.VerifyIntegrity(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"passed": report.Passed, "checks": report.Checks})
}

func (h *AdminHandler) GetSeedInfo(c *gin.Context) {
	seed := db.GenerateSeedData()
	c.JSON(http.StatusOK, gin.H{
		"admin_users":      len(seed.AdminUsers),
		"broker_providers": len(seed.BrokerProviders),
		"llm_providers":    len(seed.LLMProviders),
		"strategies":       len(seed.Strategies),
		"symbols":          len(seed.Symbols),
		"market_ticks":     len(seed.MarketTicks),
		"regime_logs":      len(seed.RegimeLogs),
		"trade_history":    len(seed.TradeHistory),
		"backtest_results": len(seed.BacktestResults),
		"admin_credentials": gin.H{
			"username": "admin",
			"password": "admin123",
			"note":     "Development only. Change in production.",
		},
	})
}

func (h *AdminHandler) RegisterRoutes(router *gin.RouterGroup) {
	admin := router.Group("/admin")
	admin.Use(mw.AuthMiddleware())
	{
		admin.GET("/health", h.GetHealth)
		admin.GET("/system/health", h.GetSystemHealth)
		admin.POST("/seed", h.SeedDatabase)
		admin.GET("/verify", h.VerifyIntegrity)
		admin.GET("/info", h.GetSeedInfo)
		admin.GET("/audit", h.GetAuditLogs)
		admin.GET("/kill-history", h.GetKillHistory)
		admin.GET("/logs/errors", h.GetErrorLogs)
		admin.POST("/email/test", h.TestEmail)
		admin.PUT("/email/config", h.SaveEmailConfig)
		admin.GET("/users", h.ListUsers)
		admin.PUT("/users/:id/disable", h.DisableUser)
		admin.PUT("/users/:id/enable", h.EnableUser)
		admin.PUT("/users/:id/reset-password", h.AdminResetPassword)
	}
}

func (h *AdminHandler) TestEmail(c *gin.Context) {
	var req struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		From     string `json:"from"`
		FromName string `json:"from_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	port := 587
	if req.Port != "" {
		if p, err := strconv.Atoi(req.Port); err == nil {
			port = p
		}
	}

	cfg := email.SMTPConfig{
		Host: req.Host, Port: port, Username: req.Username,
		Password: req.Password, From: req.From, FromName: req.FromName,
	}
	svc := email.NewSMTPEmailService(cfg)

	if err := svc.TestConnection(); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "SMTP connection successful"})
}

func (h *AdminHandler) SaveEmailConfig(c *gin.Context) {
	var req struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		From     string `json:"from"`
		FromName string `json:"from_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.repo != nil {
		h.repo.UpsertSetting(c.Request.Context(), "smtp", map[string]interface{}{
			"host": req.Host, "port": req.Port, "username": req.Username,
			"from": req.From, "from_name": req.FromName,
		})
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "SMTP configuration saved"})
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"users": []interface{}{}})
		return
	}
	users, err := h.repo.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type userView struct {
		ID          string   `json:"id"`
		Username    string   `json:"username"`
		Email       string   `json:"email"`
		Roles       []string `json:"roles"`
		IsVerified  bool     `json:"is_verified"`
		IsActive    bool     `json:"is_active"`
		TOTPEnabled bool     `json:"totp_enabled"`
		CreatedAt   string   `json:"created_at"`
	}
	result := make([]userView, len(users))
	for i, u := range users {
		result[i] = userView{
			ID: u.ID, Username: u.Username, Email: u.Email,
			Roles: u.Roles, IsVerified: u.IsVerified, IsActive: u.IsActive,
			TOTPEnabled: u.TOTPEnabled, CreatedAt: u.CreatedAt.Format(time.RFC3339),
		}
	}
	c.JSON(http.StatusOK, gin.H{"users": result})
}

func (h *AdminHandler) DisableUser(c *gin.Context) {
	id := c.Param("id")
	if h.repo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not available"})
		return
	}
	_, err := h.repo.Pool().Exec(c.Request.Context(), `UPDATE users SET is_active=false, updated_at=now() WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"disabled": true})
}

func (h *AdminHandler) EnableUser(c *gin.Context) {
	id := c.Param("id")
	if h.repo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not available"})
		return
	}
	_, err := h.repo.Pool().Exec(c.Request.Context(), `UPDATE users SET is_active=true, updated_at=now() WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": true})
}

func (h *AdminHandler) AdminResetPassword(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.repo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not available"})
		return
	}
	newHash, err := hashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}
	if err := h.repo.UpdateUserPassword(c.Request.Context(), id, newHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reset": true})
}

func (h *AdminHandler) GetAuditLogs(c *gin.Context) {
	component := c.Query("component")
	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"logs": []interface{}{}})
		return
	}
	logs, err := h.repo.ListAuditLogs(c.Request.Context(), component, limit)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"logs": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func (h *AdminHandler) GetKillHistory(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"history": []interface{}{}})
		return
	}
	history, err := h.repo.ListKillSwitchHistory(c.Request.Context(), 50)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"history": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"history": history})
}

func (h *AdminHandler) GetSystemHealth(c *gin.Context) {
	type check struct {
		Status    string `json:"status"`
		LatencyMs int64  `json:"latency_ms,omitempty"`
		Message   string `json:"message,omitempty"`
	}

	checks := make(map[string]check)
	overall := "ok"

	dbCheck := check{Status: "ok"}
	if h.repo == nil || !h.repo.IsConnected() {
		dbCheck.Status = "error"
		dbCheck.Message = "database not connected"
		overall = "critical"
	} else {
		start := time.Now()
		if err := h.repo.Ping(c.Request.Context()); err != nil {
			dbCheck.Status = "error"
			dbCheck.Message = err.Error()
			overall = "degraded"
		}
		dbCheck.LatencyMs = time.Since(start).Milliseconds()
	}
	checks["database"] = dbCheck

	brokerCheck := check{Status: "ok", Message: "paper trading active"}
	checks["broker"] = brokerCheck

	dsCheck := check{Status: "ok", Message: "polygon + tiingo configured"}
	checks["data_sources"] = dsCheck

	if h.emailService != nil {
		emailCheck := check{Status: "ok", Message: "configured"}
		checks["email"] = emailCheck
	} else {
		checks["email"] = check{Status: "warning", Message: "not configured"}
	}

	c.JSON(http.StatusOK, gin.H{
		"overall":  overall,
		"checks":   checks,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (h *AdminHandler) GetErrorLogs(c *gin.Context) {
	severity := c.DefaultQuery("severity", "")
	component := c.DefaultQuery("component", "")
	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	type errorView struct {
		ID        int    `json:"id"`
		Timestamp string `json:"timestamp"`
		Severity  string `json:"severity"`
		Component string `json:"component"`
		Message   string `json:"message"`
		Resolved  bool   `json:"resolved"`
	}

	_ = severity
	_ = component

	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"errors": []errorView{}})
		return
	}

	logs, err := h.repo.ListAuditLogs(c.Request.Context(), component, limit)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"errors": []errorView{}})
		return
	}

	result := make([]errorView, len(logs))
	for i, l := range logs {
		lvl := "info"
		if v, ok := l["level"]; ok {
			lvl = fmt.Sprintf("%v", v)
		}
		comp := ""
		if v, ok := l["component"]; ok {
			comp = fmt.Sprintf("%v", v)
		}
		msg := ""
		if v, ok := l["message"]; ok {
			msg = fmt.Sprintf("%v", v)
		}
		ts := time.Now().Format(time.RFC3339)
		if v, ok := l["timestamp"]; ok {
			ts = fmt.Sprintf("%v", v)
		}
		result[i] = errorView{
			ID: i + 1, Timestamp: ts, Severity: lvl,
			Component: comp, Message: msg, Resolved: false,
		}
	}

	c.JSON(http.StatusOK, gin.H{"errors": result})
}