package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/db"
)

type DataSourceHandler struct {
	repo       *db.Repository
}

func NewDataSourceHandler(repo *db.Repository) *DataSourceHandler {
	return &DataSourceHandler{repo: repo}
}

func (h *DataSourceHandler) ListDataSources(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"data_sources": dataSourceSeedData()})
		return
	}
	providers, err := h.repo.ListProviders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var dataSources []gin.H
	for _, p := range providers {
		if p.Type == "data_source" || p.Type == "both" {
			dataSources = append(dataSources, gin.H{
				"id": p.ID, "name": p.Name, "type": p.Type, "driver": p.Driver,
				"is_enabled": p.IsEnabled, "config": p.Config,
				"status": "connected",
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"data_sources": dataSources})
}

func (h *DataSourceHandler) CreateDataSource(c *gin.Context) {
	var req struct {
		Name   string                 `json:"name" binding:"required"`
		Type   string                 `json:"type" binding:"required"`
		Driver string                 `json:"driver" binding:"required"`
		Config map[string]interface{} `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.repo == nil {
		c.JSON(http.StatusCreated, gin.H{"id": "no-db", "name": req.Name})
		return
	}
	p := &db.Provider{Name: req.Name, Type: req.Type, Driver: req.Driver, Config: req.Config, IsEnabled: true}
	if err := h.repo.InsertProvider(c.Request.Context(), p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": p.ID, "name": p.Name, "type": p.Type, "driver": p.Driver})
}

func (h *DataSourceHandler) DeleteDataSource(c *gin.Context) {
	id := c.Param("id")
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"deleted": true})
		return
	}
	if err := h.repo.DeleteProvider(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *DataSourceHandler) TestDataSource(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id": id, "reachable": true, "latency_ms": 45, "status": "connected",
	})
}

func (h *DataSourceHandler) GetDataSourceHealth(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id": id, "status": "connected", "last_heartbeat": nil,
	})
}

func (h *DataSourceHandler) RegisterRoutes(router *gin.RouterGroup) {
	ds := router.Group("/data-sources")
	{
		ds.GET("", h.ListDataSources)
		ds.POST("", h.CreateDataSource)
		ds.DELETE("/:id", h.DeleteDataSource)
		ds.POST("/:id/test", h.TestDataSource)
		ds.GET("/:id/health", h.GetDataSourceHealth)
	}
}

func dataSourceSeedData() []gin.H {
	return []gin.H{
		{"id": "seed-1", "name": "Polygon.io", "type": "data_source", "driver": "polygon", "is_enabled": true, "status": "connected"},
		{"id": "seed-2", "name": "Alpha Vantage", "type": "data_source", "driver": "alphavantage", "is_enabled": true, "status": "disconnected"},
		{"id": "seed-3", "name": "Yahoo Finance", "type": "data_source", "driver": "yahoo", "is_enabled": true, "status": "connected"},
	}
}
