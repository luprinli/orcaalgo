package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/monitor"
	"github.com/lee-econ/orca-core/internal/risk"
)

type ProviderHandler struct {
	repo      *db.Repository
	vault     risk.VaultProvider
	hub       *monitor.WSHub
	brokerReg *broker.BrokerDriverRegistry
}

func NewProviderHandler(repo *db.Repository, vault risk.VaultProvider, hub *monitor.WSHub, brokerReg *broker.BrokerDriverRegistry) *ProviderHandler {
	return &ProviderHandler{repo: repo, vault: vault, hub: hub, brokerReg: brokerReg}
}

func (h *ProviderHandler) ListProviders(c *gin.Context) {
	if h.repo == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not connected"}); return }
	providers, err := h.repo.ListProviders(c.Request.Context())
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

func (h *ProviderHandler) CreateProvider(c *gin.Context) {
	var req struct {
		Name   string                 `json:"name" binding:"required"`
		Type   string                 `json:"type" binding:"required"`
		Driver string                 `json:"driver" binding:"required"`
		Config map[string]interface{} `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if h.repo == nil { c.JSON(http.StatusCreated, gin.H{"id": "no-db", "name": req.Name}); return }
	p := &db.Provider{Name: req.Name, Type: req.Type, Driver: req.Driver, Config: req.Config}
	if err := h.repo.InsertProvider(c.Request.Context(), p); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusCreated, gin.H{"id": p.ID, "name": p.Name, "type": p.Type, "driver": p.Driver, "config": p.Config})
}

func (h *ProviderHandler) DeleteProvider(c *gin.Context) {
	id := c.Param("id")
	if h.repo == nil { c.JSON(http.StatusOK, gin.H{"deleted": true}); return }
	if err := h.repo.DeleteProvider(c.Request.Context(), id); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *ProviderHandler) TestProvider(c *gin.Context) {
	id := c.Param("id")
	if h.brokerReg == nil {
		c.JSON(http.StatusNotFound, gin.H{"id": id, "reachable": false, "error": "broker registry unavailable"})
		return
	}
	adapter, ok := h.brokerReg.Get(id)
	if !ok || adapter == nil {
		c.JSON(http.StatusNotFound, gin.H{"id": id, "reachable": false, "error": "provider not found"})
		return
	}
	start := time.Now()
	acct, err := adapter.GetAccount(c.Request.Context())
	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"id": id, "reachable": false, "latency_ms": latencyMs, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id": id, "reachable": true, "latency_ms": latencyMs,
		"account": gin.H{
			"balance": acct.Balance, "equity": acct.Equity,
			"daily_pnl": acct.DailyPL, "buying_power": acct.BuyingPower,
		},
	})
}

func (h *ProviderHandler) GetAccount(c *gin.Context) {
	id := c.Param("id")
	if h.brokerReg != nil {
		if adapter, ok := h.brokerReg.Get(id); ok && adapter != nil {
			if acct, err := adapter.GetAccount(c.Request.Context()); err == nil && acct != nil {
				c.JSON(http.StatusOK, gin.H{
					"balance": acct.Balance, "equity": acct.Equity,
					"daily_pnl": acct.DailyPL, "buying_power": acct.BuyingPower,
				})
				return
			}
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
}

func (h *ProviderHandler) RegisterRoutes(router *gin.RouterGroup) {
	providers := router.Group("/providers")
	{
		providers.GET("", h.ListProviders)
		providers.POST("", h.CreateProvider)
		providers.DELETE("/:id", h.DeleteProvider)
		providers.POST("/:id/test", h.TestProvider)
		providers.GET("/:id/account", h.GetAccount)
	}
}