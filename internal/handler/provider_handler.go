package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/allwaysyou/llm-agent/internal/adapter"
	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/service"
	"github.com/gin-gonic/gin"
)

// ProviderHandler handles Provider HTTP requests
type ProviderHandler struct {
	service        *service.ProviderService
	adapterFactory *adapter.AdapterFactory
}

// NewProviderHandler creates a new provider handler
func NewProviderHandler(service *service.ProviderService, adapterFactory *adapter.AdapterFactory) *ProviderHandler {
	return &ProviderHandler{
		service:        service,
		adapterFactory: adapterFactory,
	}
}

// Create creates a new provider
// POST /api/v1/providers
func (h *ProviderHandler) Create(c *gin.Context) {
	var req model.CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider, err := h.service.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, provider.ToResponse())
}

// GetAll retrieves all providers
// GET /api/v1/providers
func (h *ProviderHandler) GetAll(c *gin.Context) {
	providers, err := h.service.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responses := make([]model.ProviderResponse, len(providers))
	for i, p := range providers {
		responses[i] = p.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
}

// GetByID retrieves a provider by ID with its models
// GET /api/v1/providers/:id
func (h *ProviderHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	resp, err := h.service.GetByIDWithModels(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if resp == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Update updates a provider
// PUT /api/v1/providers/:id
func (h *ProviderHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req model.UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider, err := h.service.Update(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, provider.ToResponse())
}

// Delete deletes a provider
// DELETE /api/v1/providers/:id
func (h *ProviderHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Test tests a provider's API key
// POST /api/v1/providers/:id/test
func (h *ProviderHandler) Test(c *gin.Context) {
	id := c.Param("id")

	provider, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if provider == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	// Decrypt API key
	apiKey, err := h.service.DecryptAPIKey(provider.APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decrypt API key"})
		return
	}

	// Create adapter with a test model
	testModel := "gpt-3.5-turbo"
	switch provider.Type {
	case model.ProviderTypeClaude:
		testModel = "claude-3-haiku-20240307"
	case model.ProviderTypeOllama:
		testModel = "llama2"
	}

	adapterCfg := adapter.AdapterConfig{
		APIKey:      apiKey,
		BaseURL:     provider.BaseURL,
		Model:       testModel,
		MaxTokens:   100,
		Temperature: 0.7,
	}

	llmAdapter, err := h.adapterFactory.Create(provider.Type, adapterCfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider: " + string(provider.Type)})
		return
	}

	// Test with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Test chat
	messages := []model.Message{
		{Role: model.RoleUser, Content: "Hi, just testing. Reply with 'OK' only."},
	}
	_, err = llmAdapter.Chat(ctx, messages)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection successful",
	})
}
