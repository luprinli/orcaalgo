package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/broker"
	"github.com/lee-econ/orca-core/internal/types"
)

type WebhookHandler struct {
	adapter broker.Adapter
}

func NewWebhookHandler() *WebhookHandler { return &WebhookHandler{} }

func NewWebhookHandlerWithAdapter(adapter broker.Adapter) *WebhookHandler {
	return &WebhookHandler{adapter: adapter}
}

func (h *WebhookHandler) HandleTradingView(c *gin.Context) {
	var req struct {
		Ticker   string  `json:"ticker" binding:"required"`
		Action   string  `json:"action" binding:"required"`
		Price    float64 `json:"price"`
		Quantity float64 `json:"quantity"`
		Strategy string  `json:"strategy"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Quantity == 0 {
		req.Quantity = 100
	}
	if h.adapter != nil {
		side := broker.Buy
		if req.Action == "SELL" || req.Action == "sell" || req.Action == "short" {
			side = broker.Sell
		}
		orderReq := &broker.OrderRequest{
			Symbol:     req.Ticker,
			Side:       side,
			Type:       broker.Market,
			Quantity:   req.Quantity,
			StrategyID: req.Strategy,
		}
		if req.Price > 0 {
			orderReq.Type = broker.Limit
			orderReq.LimitPrice = types.FromFloat64(req.Price)
		}
		resp, err := h.adapter.PlaceOrder(c.Request.Context(), orderReq)
		if err != nil {
			c.JSON(http.StatusAccepted, gin.H{
				"source": "tradingview", "ticker": req.Ticker,
				"action": req.Action, "status": "queued",
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{
			"source": "tradingview", "ticker": req.Ticker,
			"action": req.Action, "status": "filled",
			"order_id": resp.BrokerOrderID,
		})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{
		"source": "tradingview", "ticker": req.Ticker,
		"action": req.Action, "status": "queued",
	})
}

func (h *WebhookHandler) HandleChartInk(c *gin.Context) {
	var req struct {
		Symbol string `json:"symbol" binding:"required"`
		Signal string `json:"signal" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"source": "chartink", "symbol": req.Symbol, "signal": req.Signal, "status": "queued"})
}

func (h *WebhookHandler) TestWebhook(c *gin.Context) {
	var req struct {
		Type    string                 `json:"type" binding:"required"`
		Payload map[string]interface{} `json:"payload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"type": req.Type, "parsed": true, "action": "dry_run", "status": "ok",
		"message": "Webhook payload validated successfully",
	})
}

func (h *WebhookHandler) RegisterRoutes(router *gin.RouterGroup) {
	wh := router.Group("/webhooks")
	{
		wh.POST("/tradingview", h.HandleTradingView)
		wh.POST("/chartink", h.HandleChartInk)
		wh.POST("/test", h.TestWebhook)
	}
}