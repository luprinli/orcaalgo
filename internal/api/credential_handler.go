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
	if h.repo == nil {
		c.JSON(http.StatusOK, gin.H{"credentials": []interface{}{}})
		return
	}
	providers, err := h.repo.ListProviders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(providers))
	for _, p := range providers {
		out = append(out, gin.H{
			"provider_id": p.ID,
			"name":        p.Name,
			"type":        p.Type,
			"driver":      p.Driver,
		})
	}
	c.JSON(http.StatusOK, gin.H{"credentials": out})
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
		"id":         vaultPath,
		"vault_path": vaultPath,
		"stored":     true,
	})
}

func (h *CredentialHandler) RotateCredential(c *gin.Context) {
	id := c.Param("id") // vault-path suffix (providerID/keyLabel)
	path := "providers/" + id
	existing, err := h.vault.Load(path)
	if err != nil || existing["api_key"] == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
		return
	}
	newPath := path + "_rotated"
	if err := h.vault.Store(newPath, existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rotate credential"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":       id,
		"rotated":  true,
		"new_path": newPath,
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
