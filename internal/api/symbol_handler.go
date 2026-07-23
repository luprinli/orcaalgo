package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/db"
)

type SymbolHandler struct {
	repo *db.Repository
}

func NewSymbolHandler(repo *db.Repository) *SymbolHandler { return &SymbolHandler{repo: repo} }

func (h *SymbolHandler) ListSymbols(c *gin.Context) {
	if h.repo == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not connected"}); return }
	symbols, err := h.repo.ListSymbols(c.Request.Context())
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"symbols": symbols})
}

func (h *SymbolHandler) CreateSymbol(c *gin.Context) {
	var req struct {
		Ticker    string  `json:"ticker" binding:"required"`
		Exchange  string  `json:"exchange" binding:"required"`
		AssetType string  `json:"asset_type" binding:"required"`
		TickSize  float64 `json:"tick_size" binding:"required"`
		LotSize   float64 `json:"lot_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if req.LotSize == 0 { req.LotSize = 1 }
	if h.repo == nil { c.JSON(http.StatusCreated, gin.H{"ticker": req.Ticker, "id": 0}); return }
	id, err := h.repo.InsertSymbol(c.Request.Context(), &db.Symbol{
		Ticker: req.Ticker, Exchange: req.Exchange, AssetType: req.AssetType,
		TickSize: req.TickSize, LotSize: req.LotSize, IsActive: true,
	})
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusCreated, gin.H{"id": id, "ticker": req.Ticker, "exchange": req.Exchange})
}

func (h *SymbolHandler) DeleteSymbol(c *gin.Context) {
	id := c.Param("id")
	if h.repo == nil { c.JSON(http.StatusOK, gin.H{"deleted": true}); return }
	var intID int32
	if _, err := fmt.Sscanf(id, "%d", &intID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.repo.DeleteSymbol(c.Request.Context(), intID); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *SymbolHandler) ListSymbolFeeds(c *gin.Context) {
	symbolID := c.Param("symbol")
	if h.repo == nil { c.JSON(http.StatusOK, gin.H{"feeds": []gin.H{}}); return }
	var intID int32
	fmt.Sscanf(symbolID, "%d", &intID)
	feeds, _ := h.repo.ListProviderSymbols(c.Request.Context(), intID)
	c.JSON(http.StatusOK, gin.H{"feeds": feeds})
}

func (h *SymbolHandler) AssignSymbolFeed(c *gin.Context) {
	symbolID := c.Param("symbol")
	var req struct {
		ProviderID string `json:"provider_id" binding:"required"`
		FeedType   string `json:"feed_type"`
		Priority   int16  `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if req.FeedType == "" { req.FeedType = "market_data" }
	if req.Priority == 0 { req.Priority = 100 }
	if h.repo == nil { c.JSON(http.StatusCreated, gin.H{"assigned": true}); return }
	var intID int32
	fmt.Sscanf(symbolID, "%d", &intID)
	err := h.repo.InsertProviderSymbol(c.Request.Context(), &db.ProviderSymbol{
		ProviderID: req.ProviderID, SymbolID: intID,
		FeedType: req.FeedType, Priority: req.Priority, IsEnabled: true,
	})
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusCreated, gin.H{"assigned": true})
}

func (h *SymbolHandler) RemoveSymbolFeed(c *gin.Context) {
	symbolID := c.Param("symbol")
	feedID := c.Param("feed_id")
	if h.repo == nil { c.JSON(http.StatusOK, gin.H{"removed": true}); return }
	var intSymID int32
	fmt.Sscanf(symbolID, "%d", &intSymID)
	h.repo.DeleteProviderSymbol(c.Request.Context(), feedID, intSymID, "market_data")
	c.JSON(http.StatusOK, gin.H{"removed": true})
}

func (h *SymbolHandler) RegisterRoutes(router *gin.RouterGroup) {
	symbols := router.Group("/symbols")
	{
		symbols.GET("", h.ListSymbols)
		symbols.POST("", h.CreateSymbol)
		symbols.DELETE("/:id", h.DeleteSymbol)
	}
	feeds := router.Group("/symbol-feeds")
	{
		feeds.GET("/:symbol", h.ListSymbolFeeds)
		feeds.POST("/:symbol", h.AssignSymbolFeed)
		feeds.DELETE("/:symbol/:feed_id", h.RemoveSymbolFeed)
	}
}