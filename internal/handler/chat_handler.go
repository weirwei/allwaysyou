package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/service"
	"github.com/gin-gonic/gin"
)

// ChatHandler handles chat HTTP requests
type ChatHandler struct {
	chatService *service.ChatService
}

// NewChatHandler creates a new chat handler
func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

// Chat handles chat requests
// POST /api/v1/chat
func (h *ChatHandler) Chat(c *gin.Context) {
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "messages cannot be empty"})
		return
	}

	if req.Stream {
		h.handleStream(c, &req)
		return
	}

	resp, err := h.chatService.Chat(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// handleStream handles streaming chat requests
func (h *ChatHandler) handleStream(c *gin.Context, req *model.ChatRequest) {
	stream, sessionID, err := h.chatService.ChatStream(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Session-ID", sessionID)

	c.Writer.Flush()

	for chunk := range stream {
		data, _ := json.Marshal(chunk)
		_, _ = fmt.Fprintf(c.Writer, "event: message\ndata: %s\n\n", string(data))
		c.Writer.Flush()

		if chunk.Done {
			break
		}
	}
}
