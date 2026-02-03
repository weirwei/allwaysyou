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

// ConfigHandler handles LLM config HTTP requests
type ConfigHandler struct {
	service        *service.ConfigService
	adapterFactory *adapter.AdapterFactory
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(service *service.ConfigService, adapterFactory *adapter.AdapterFactory) *ConfigHandler {
	return &ConfigHandler{
		service:        service,
		adapterFactory: adapterFactory,
	}
}

// Create creates a new LLM config
// POST /api/v1/configs
func (h *ConfigHandler) Create(c *gin.Context) {
	var req model.CreateLLMConfigRequest
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

// GetAll retrieves all LLM configs
// GET /api/v1/configs
func (h *ConfigHandler) GetAll(c *gin.Context) {
	configs, err := h.service.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responses := make([]model.LLMConfigResponse, len(configs))
	for i, cfg := range configs {
		responses[i] = cfg.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
}

// GetByID retrieves an LLM config by ID
// GET /api/v1/configs/:id
func (h *ConfigHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	config, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if config == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	c.JSON(http.StatusOK, config.ToResponse())
}

// Update updates an LLM config
// PUT /api/v1/configs/:id
func (h *ConfigHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req model.UpdateLLMConfigRequest
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

// Delete deletes an LLM config
// DELETE /api/v1/configs/:id
func (h *ConfigHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Test tests an LLM config connection
// POST /api/v1/configs/:id/test
func (h *ConfigHandler) Test(c *gin.Context) {
	id := c.Param("id")

	config, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if config == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	// Decrypt API key
	apiKey, err := h.service.DecryptAPIKey(config.APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decrypt API key"})
		return
	}

	// Create adapter
	adapterCfg := adapter.AdapterConfig{
		APIKey:      apiKey,
		BaseURL:     config.BaseURL,
		Model:       config.Model,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
	}

	llmAdapter, err := h.adapterFactory.Create(config.Provider, adapterCfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider: " + string(config.Provider)})
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
