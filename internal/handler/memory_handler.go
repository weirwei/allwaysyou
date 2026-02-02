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
		SessionID string          `json:"session_id" binding:"required"`
		Role      model.MessageRole `json:"role" binding:"required"`
		Content   string          `json:"content" binding:"required"`
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
