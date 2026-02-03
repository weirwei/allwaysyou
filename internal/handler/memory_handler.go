package handler

import (
	"net/http"
	"strconv"

	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/service"
	"github.com/gin-gonic/gin"
)

// MemoryHandler handles memory-related HTTP requests
type MemoryHandler struct {
	memoryService    *service.MemoryService
	summarizeService *service.SummarizeService
}

// NewMemoryHandler creates a new memory handler
func NewMemoryHandler(memoryService *service.MemoryService, summarizeService *service.SummarizeService) *MemoryHandler {
	return &MemoryHandler{
		memoryService:    memoryService,
		summarizeService: summarizeService,
	}
}

// Search searches for relevant memories
// GET /api/v1/memories/search?query=xxx&session_id=xxx&limit=10
func (h *MemoryHandler) Search(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	sessionID := c.Query("session_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	results, err := h.memoryService.SearchMemories(c.Request.Context(), query, sessionID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}

// Summarize generates a summary for a session
// POST /api/v1/sessions/:id/summarize
func (h *MemoryHandler) Summarize(c *gin.Context) {
	sessionID := c.Param("id")

	summary, err := h.summarizeService.SummarizeSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"summary":    summary,
	})
}

// Create manually creates a memory
// POST /api/v1/memories
func (h *MemoryHandler) Create(c *gin.Context) {
	var req struct {
		SessionID string            `json:"session_id" binding:"required"`
		Role      model.MessageRole `json:"role" binding:"required"`
		Content   string            `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	memory, err := h.memoryService.SaveMemory(c.Request.Context(), req.SessionID, req.Role, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, memory)
}

// GetAllKnowledge returns all knowledge entries
// GET /api/v1/knowledge?active_only=true&limit=100
func (h *MemoryHandler) GetAllKnowledge(c *gin.Context) {
	activeOnly := c.DefaultQuery("active_only", "true") == "true"
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	knowledge, err := h.memoryService.GetAllKnowledge(c.Request.Context(), activeOnly, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, knowledge)
}

// GetKnowledge returns a single knowledge entry
// GET /api/v1/knowledge/:id
func (h *MemoryHandler) GetKnowledge(c *gin.Context) {
	id := c.Param("id")

	knowledge, err := h.memoryService.GetKnowledge(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if knowledge == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge not found"})
		return
	}

	c.JSON(http.StatusOK, knowledge)
}

// CreateKnowledge creates a new knowledge entry
// POST /api/v1/knowledge
func (h *MemoryHandler) CreateKnowledge(c *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	knowledge, err := h.memoryService.CreateKnowledge(c.Request.Context(), req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, knowledge)
}

// UpdateKnowledge updates a knowledge entry
// PUT /api/v1/knowledge/:id
func (h *MemoryHandler) UpdateKnowledge(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	knowledge, err := h.memoryService.UpdateKnowledge(c.Request.Context(), id, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, knowledge)
}

// DeleteKnowledge deletes a knowledge entry
// DELETE /api/v1/knowledge/:id
func (h *MemoryHandler) DeleteKnowledge(c *gin.Context) {
	id := c.Param("id")

	if err := h.memoryService.DeleteKnowledge(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "knowledge deleted"})
}
