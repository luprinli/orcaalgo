package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/indicator"
	"github.com/lee-econ/orca-core/internal/monitor"
)

type IndicatorHandler struct {
	liveStream *indicator.LiveStreamService
	hub        *monitor.WSHub
}

func NewIndicatorHandler(hub *monitor.WSHub) *IndicatorHandler {
	h := &IndicatorHandler{hub: hub}
	if hub != nil {
		h.liveStream = indicator.NewLiveStreamService(hub, nil)
	}
	return h
}

func (h *IndicatorHandler) SetLiveStream(ls *indicator.LiveStreamService) {
	h.liveStream = ls
}

func (h *IndicatorHandler) SetCandleProvider(provider func(symbol, timeframe string, limit int) ([]indicator.Candle, error)) {
	if h.liveStream != nil {
		h.liveStream.SetCandleProvider(provider)
	}
}

func (h *IndicatorHandler) listIndicators(c *gin.Context) {
	specs := indicator.List()
	c.JSON(http.StatusOK, gin.H{"indicators": specs})
}

func (h *IndicatorHandler) computeIndicator(c *gin.Context) {
	indicatorID := c.Query("indicator")
	symbol := c.Query("symbol")
	timeframe := c.DefaultQuery("timeframe", "M1")
	limitStr := c.DefaultQuery("limit", "500")

	if indicatorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "indicator query param is required"})
		return
	}
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol query param is required"})
		return
	}

	spec, ok := indicator.Get(indicatorID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown indicator: " + indicatorID})
		return
	}

	var body struct {
		Parameters map[string]interface{} `json:"parameters"`
		Candles    []indicator.Candle     `json:"candles"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(body.Candles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "candles array is required in request body"})
		return
	}

	params := body.Parameters
	if params == nil {
		params = make(map[string]interface{})
	}

	_ = limitStr
	_ = timeframe

	result, err := indicator.Compute(indicatorID, body.Candles, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"indicator": result,
		"metadata":  spec,
	})
}

func (h *IndicatorHandler) startLiveStream(c *gin.Context) {
	var req indicator.LiveStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Symbol == "" || req.Indicator == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol and indicator are required"})
		return
	}
	if req.Timeframe == "" {
		req.Timeframe = "M1"
	}
	if h.liveStream == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "live stream service not available"})
		return
	}
	h.liveStream.Start(req)
	c.JSON(http.StatusOK, gin.H{
		"streaming": true,
		"symbol":    req.Symbol,
		"indicator": req.Indicator,
		"timeframe": req.Timeframe,
	})
}

func (h *IndicatorHandler) stopLiveStream(c *gin.Context) {
	var req indicator.LiveStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Symbol == "" || req.Indicator == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol and indicator are required"})
		return
	}
	if req.Timeframe == "" {
		req.Timeframe = "M1"
	}
	if h.liveStream == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "live stream service not available"})
		return
	}
	h.liveStream.Stop(req)
	c.JSON(http.StatusOK, gin.H{"streaming": false})
}

func (h *IndicatorHandler) liveStreamStatus(c *gin.Context) {
	if h.liveStream == nil {
		c.JSON(http.StatusOK, gin.H{"active": 0})
		return
	}
	c.JSON(http.StatusOK, gin.H{"active": h.liveStream.ActiveCount()})
}

func (h *IndicatorHandler) RegisterRoutes(router *gin.RouterGroup) {
	indicators := router.Group("/indicators")
	{
		indicators.GET("", h.listIndicators)
		indicators.POST("/compute", h.computeIndicator)
		indicators.POST("/stream/start", h.startLiveStream)
		indicators.POST("/stream/stop", h.stopLiveStream)
		indicators.GET("/stream/status", h.liveStreamStatus)
	}
}
