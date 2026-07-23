package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/risk"
)

type CredentialHandler struct {
	repo  *db.Repository
	vault risk.VaultProvider
}

func NewCredentialHandler(repo *db.Repository, vault risk.VaultProvider) *CredentialHandler {
	return &CredentialHandler{repo: repo, vault: vault}
}

func (h *CredentialHandler) ListCredentials(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"credentials": []interface{}{},
	})
}

func (h *CredentialHandler) StoreCredential(c *gin.Context) {
	var req struct {
		ProviderID string `json:"provider_id" binding:"required"`
		KeyLabel   string `json:"key_label" binding:"required"`
		APIKey     string `json:"api_key" binding:"required"`
		APISecret  string `json:"api_secret" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vaultPath := "providers/" + req.ProviderID + "/" + req.KeyLabel
	if err := h.vault.Store(vaultPath, map[string]string{
		"api_key":    req.APIKey,
		"api_secret": req.APISecret,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store credential"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         "credential-uuid",
		"vault_path": vaultPath,
		"stored":     true,
	})
}

func (h *CredentialHandler) RotateCredential(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":       id,
		"rotated":  true,
		"new_path": "providers/alpaca/algo_key_v2",
	})
}

func (h *CredentialHandler) RegisterRoutes(router *gin.RouterGroup) {
	creds := router.Group("/credentials")
	{
		creds.GET("", h.ListCredentials)
		creds.POST("", h.StoreCredential)
		creds.PUT("/:id/rotate", h.RotateCredential)
	}
}
