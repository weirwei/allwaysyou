package handler

import (
	"net/http"

	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/service"
	"github.com/gin-gonic/gin"
)

// SystemConfigHandler handles system configuration HTTP requests
type SystemConfigHandler struct {
	service *service.SystemConfigService
}

// NewSystemConfigHandler creates a new system config handler
func NewSystemConfigHandler(service *service.SystemConfigService) *SystemConfigHandler {
	return &SystemConfigHandler{service: service}
}

// GetAll retrieves all system configs
func (h *SystemConfigHandler) GetAll(c *gin.Context) {
	configs, err := h.service.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responses := make([]model.SystemConfigResponse, len(configs))
	for i, config := range configs {
		responses[i] = config.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
}

// GetByCategory retrieves system configs by category
func (h *SystemConfigHandler) GetByCategory(c *gin.Context) {
	category := c.Query("category")
	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category is required"})
		return
	}

	configs, err := h.service.GetByCategory(category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responses := make([]model.SystemConfigResponse, len(configs))
	for i, config := range configs {
		responses[i] = config.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
}

// Update updates a system config value
func (h *SystemConfigHandler) Update(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	var req model.SystemConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.Update(key, req.Value); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Return updated config
	config, _ := h.service.Get(key)
	c.JSON(http.StatusOK, config.ToResponse())
}
