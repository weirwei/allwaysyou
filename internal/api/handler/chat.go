package handler

import (
	"io"
	"net/http"

	"github.com/allwaysyou/llm-agent/internal/service/agent"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChatHandler struct {
	agentService *agent.Service
}

func NewChatHandler(agentService *agent.Service) *ChatHandler {
	return &ChatHandler{agentService: agentService}
}

type ChatMessageRequest struct {
	SessionID    string  `json:"session_id,omitempty"`
	Message      string  `json:"message" binding:"required"`
	Provider     string  `json:"provider,omitempty"`
	Model        string  `json:"model,omitempty"`
	SystemPrompt string  `json:"system_prompt,omitempty"`
	MaxTokens    int     `json:"max_tokens,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
}

// Chat handles non-streaming chat requests
// @Summary Send a chat message
// @Tags chat
// @Accept json
// @Produce json
// @Param request body ChatMessageRequest true "Chat message"
// @Success 200 {object} agent.ChatResponse
// @Router /api/v1/chat [post]
func (h *ChatHandler) Chat(c *gin.Context) {
	var req ChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, _ := c.Get("user_id")
	uid, ok := userID.(uuid.UUID)
	if !ok {
		uid = uuid.New() // Default user for testing
	}

	var sessionID uuid.UUID
	if req.SessionID != "" {
		var err error
		sessionID, err = uuid.Parse(req.SessionID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id"})
			return
		}
	}

	chatReq := &agent.ChatRequest{
		SessionID:    sessionID,
		UserID:       uid,
		Message:      req.Message,
		Provider:     req.Provider,
		Model:        req.Model,
		SystemPrompt: req.SystemPrompt,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
	}

	resp, err := h.agentService.Chat(c.Request.Context(), chatReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ChatStream handles streaming chat requests using SSE
// @Summary Send a streaming chat message
// @Tags chat
// @Accept json
// @Produce text/event-stream
// @Param request body ChatMessageRequest true "Chat message"
// @Router /api/v1/chat/stream [post]
func (h *ChatHandler) ChatStream(c *gin.Context) {
	var req ChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	uid, ok := userID.(uuid.UUID)
	if !ok {
		uid = uuid.New()
	}

	var sessionID uuid.UUID
	if req.SessionID != "" {
		var err error
		sessionID, err = uuid.Parse(req.SessionID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id"})
			return
		}
	}

	chatReq := &agent.ChatRequest{
		SessionID:    sessionID,
		UserID:       uid,
		Message:      req.Message,
		Provider:     req.Provider,
		Model:        req.Model,
		SystemPrompt: req.SystemPrompt,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
	}

	stream, _, err := h.agentService.ChatStream(c.Request.Context(), chatReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	c.Stream(func(w io.Writer) bool {
		chunk, ok := <-stream
		if !ok {
			c.SSEvent("done", "")
			return false
		}

		if chunk.Error != nil {
			c.SSEvent("error", chunk.Error.Error())
			return false
		}

		c.SSEvent("message", chunk.Content)
		return true
	})
}
