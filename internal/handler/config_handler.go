package handler

import (
	"net/http"

	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/service"
	"github.com/gin-gonic/gin"
)

// ConfigHandler handles LLM config HTTP requests
type ConfigHandler struct {
	service *service.ConfigService
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(service *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{service: service}
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
