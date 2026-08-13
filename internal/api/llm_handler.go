package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lee-econ/orca-core/internal/api/middleware"
	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/llm"
	"github.com/lee-econ/orca-core/internal/risk"
)

// LLMHandler manages per-user BYOK (bring-your-own-key) LLM provider keys and
// connection testing. Secrets are stored in the encrypted vault under
// llm/{user_id}/{provider}; only masked metadata is persisted in the database.
type LLMHandler struct {
	repo  *db.Repository
	vault risk.VaultProvider
}

func NewLLMHandler(repo *db.Repository, vault risk.VaultProvider) *LLMHandler {
	return &LLMHandler{repo: repo, vault: vault}
}

// maskSuffix returns a non-reversible suffix mask (e.g. "••••abcd"). It is the
// single place masking happens, so list/add responses never leak a full key.
func maskSuffix(key string) string {
	if len(key) <= 4 {
		return "••••"
	}
	return "••••" + key[len(key)-4:]
}

func (h *LLMHandler) currentUserID(c *gin.Context) (string, bool) {
	userID := c.GetString("user_id")
	return userID, userID != ""
}

func (h *LLMHandler) ListKeys(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	keys, err := h.repo.ListLLMKeys(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

func (h *LLMHandler) AddKey(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		Provider string `json:"provider" binding:"required"`
		APIKey   string `json:"api_key" binding:"required"`
		BaseURL  string `json:"base_url"`
		Model    string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	vaultPath := "llm/" + userID + "/" + provider

	if err := h.vault.Store(vaultPath, map[string]string{"api_key": req.APIKey}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store credential"})
		return
	}
	if err := h.repo.UpsertLLMKey(c.Request.Context(), &db.LLMKey{
		UserID:       userID,
		Provider:     provider,
		VaultPath:    vaultPath,
		BaseURL:      req.BaseURL,
		Model:        req.Model,
		MaskedSuffix: maskSuffix(req.APIKey),
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"provider": provider, "masked_suffix": maskSuffix(req.APIKey)})
}

func (h *LLMHandler) DeleteKey(c *gin.Context) {
	userID, ok := h.currentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if err := h.repo.DeleteLLMKey(c.Request.Context(), userID, c.Param("provider")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

// Test validates an LLM provider connection. The passed key is honored (fixes
// the previous env-only behavior); when no key is supplied it falls back to the
// user's stored key for that provider.
func (h *LLMHandler) Test(c *gin.Context) {
	var req struct {
		Provider string `json:"provider" binding:"required"`
		APIKey   string `json:"api_key"`
		BaseURL  string `json:"base_url"`
		Model    string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	key := req.APIKey
	baseURL := req.BaseURL
	model := req.Model

	if key == "" {
		if userID, ok := h.currentUserID(c); ok {
			if rec, err := h.repo.GetLLMKey(c.Request.Context(), userID, provider); err == nil {
				if data, err := h.vault.Load(rec.VaultPath); err == nil {
					key = data["api_key"]
				}
				if baseURL == "" {
					baseURL = rec.BaseURL
				}
				if model == "" {
					model = rec.Model
				}
			}
		}
	}
	if key == "" {
		c.JSON(http.StatusOK, gin.H{"reachable": false, "error": "no API key provided"})
		return
	}

	client := llm.NewClientWithKey(llm.Provider(provider), key, baseURL)
	resp, err := client.Chat(&llm.ChatRequest{
		Model:     model,
		Messages:  []llm.Message{{Role: "user", Content: "ping"}},
		MaxTokens: 5,
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"reachable": false, "error": err.Error()})
		return
	}
	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}
	c.JSON(http.StatusOK, gin.H{"reachable": true, "model": model, "response": content})
}

func (h *LLMHandler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/llm")
	// LLM endpoints make external, billed calls — throttle per-IP to prevent
	// abuse/cost spikes.
	group.Use(middleware.RateLimitMiddleware(2))
	{
		group.GET("/keys", h.ListKeys)
		group.POST("/keys", h.AddKey)
		group.DELETE("/keys/:provider", h.DeleteKey)
		group.POST("/test", h.Test)
	}
}
