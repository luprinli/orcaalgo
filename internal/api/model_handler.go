package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/ml"
)

type ModelHandler struct {
	repo *db.Repository
}

func NewModelHandler(repo *db.Repository) *ModelHandler {
	return &ModelHandler{repo: repo}
}

func (h *ModelHandler) RegisterRoutes(router *gin.RouterGroup) {
	models := router.Group("/models")
	{
		models.GET("", h.ListModels)
		models.POST("/register", h.RegisterModel)
		models.GET("/compare", h.CompareModel)
		models.GET("/latest/:type", h.GetLatestModel)
	}
}

// ListModels returns all registered models, newest first.
func (h *ModelHandler) ListModels(c *gin.Context) {
	ctx := c.Request.Context()
	records, err := ml.ListModels(ctx, h.repo.Pool())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"models": records})
}

type registerModelReq struct {
	ModelHash  string  `json:"model_hash" binding:"required"`
	ModelType  string  `json:"model_type" binding:"required"`
	ModelName  string  `json:"model_name" binding:"required"`
	BrierScore float64 `json:"brier_score"`
	ROCAUC     float64 `json:"roc_auc"`
	Metadata   string  `json:"metadata,omitempty"`
}

func (h *ModelHandler) RegisterModel(c *gin.Context) {
	var req registerModelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	err := ml.RegisterModel(ctx, h.repo.Pool(), ml.ModelRecord{
		ModelHash:  req.ModelHash,
		ModelType:  req.ModelType,
		ModelName:  req.ModelName,
		BrierScore: req.BrierScore,
		ROCAUC:     req.ROCAUC,
		Metadata:   req.Metadata,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"registered": req.ModelHash})
}

type compareReq struct {
	ModelHash string `json:"model_hash" binding:"required"`
}

func (h *ModelHandler) CompareModel(c *gin.Context) {
	hash := c.Query("model_hash")
	if hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model_hash query parameter required"})
		return
	}

	ctx := c.Request.Context()
	found, _ := ml.VerifyModelHash(ctx, h.repo.Pool(), hash)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"matches": false, "hash": hash, "reason": "not in registry"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"matches": true, "hash": hash})
}

func (h *ModelHandler) GetLatestModel(c *gin.Context) {
	modelType := c.Param("type")
	if modelType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model type required"})
		return
	}

	ctx := c.Request.Context()
	record, err := ml.GetLatestModel(ctx, h.repo.Pool(), modelType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, record)
}
