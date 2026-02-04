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

// ModelConfigHandler handles ModelConfig HTTP requests
type ModelConfigHandler struct {
	service         *service.ModelConfigService
	providerService *service.ProviderService
	adapterFactory  *adapter.AdapterFactory
}

// NewModelConfigHandler creates a new model config handler
func NewModelConfigHandler(service *service.ModelConfigService, providerService *service.ProviderService, adapterFactory *adapter.AdapterFactory) *ModelConfigHandler {
	return &ModelConfigHandler{
		service:         service,
		providerService: providerService,
		adapterFactory:  adapterFactory,
	}
}

// Create creates a new model config
// POST /api/v1/models
func (h *ModelConfigHandler) Create(c *gin.Context) {
	var req model.CreateModelConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config.ToResponse())
}

// GetAll retrieves all model configs
// GET /api/v1/models
func (h *ModelConfigHandler) GetAll(c *gin.Context) {
	// Check if filtering by provider
	providerID := c.Query("provider_id")

	var configs []model.ModelConfig
	var err error

	if providerID != "" {
		configs, err = h.service.GetByProvider(providerID)
	} else {
		configs, err = h.service.GetAll()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responses := make([]model.ModelConfigResponse, len(configs))
	for i, cfg := range configs {
		responses[i] = cfg.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
}

// GetByID retrieves a model config by ID
// GET /api/v1/models/:id
func (h *ModelConfigHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	config, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if config == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "model config not found"})
		return
	}

	c.JSON(http.StatusOK, config.ToResponse())
}

// Update updates a model config
// PUT /api/v1/models/:id
func (h *ModelConfigHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req model.UpdateModelConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.service.Update(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config.ToResponse())
}

// Delete deletes a model config
// DELETE /api/v1/models/:id
func (h *ModelConfigHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// SetDefault sets a model config as default for its type
// POST /api/v1/models/:id/default
func (h *ModelConfigHandler) SetDefault(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.SetDefault(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Model set as default"})
}

// Test tests a model config connection
// POST /api/v1/models/:id/test
func (h *ModelConfigHandler) Test(c *gin.Context) {
	id := c.Param("id")

	config, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if config == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "model config not found"})
		return
	}

	// Get provider
	provider, err := h.providerService.GetByID(config.ProviderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if provider == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	// Decrypt API key
	apiKey, err := h.providerService.DecryptAPIKey(provider.APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decrypt API key"})
		return
	}

	// Create adapter
	adapterCfg := adapter.AdapterConfig{
		APIKey:      apiKey,
		BaseURL:     provider.BaseURL,
		Model:       config.Model,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
	}

	llmAdapter, err := h.adapterFactory.Create(provider.Type, adapterCfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider: " + string(provider.Type)})
		return
	}

	// Test with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Test based on config type
	if config.ConfigType == model.ConfigTypeEmbedding {
		// Test embedding
		_, err = llmAdapter.GetEmbedding(ctx, "test")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
	} else {
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
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection successful",
	})
}
