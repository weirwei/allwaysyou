package handler

import (
	"net/http"
	"strconv"

	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/repository"
	"github.com/gin-gonic/gin"
)

// SessionHandler handles session HTTP requests
type SessionHandler struct {
	sessionRepo *repository.SessionRepository
	memoryRepo  *repository.MemoryRepository
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(sessionRepo *repository.SessionRepository, memoryRepo *repository.MemoryRepository) *SessionHandler {
	return &SessionHandler{
		sessionRepo: sessionRepo,
		memoryRepo:  memoryRepo,
	}
}

// GetAll retrieves all sessions
// GET /api/v1/sessions
func (h *SessionHandler) GetAll(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	sessions, err := h.sessionRepo.GetAll(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responses := make([]model.SessionResponse, len(sessions))
	for i, s := range sessions {
		responses[i] = s.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
}

// GetByID retrieves a session by ID with its messages
// GET /api/v1/sessions/:id
func (h *SessionHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	session, err := h.sessionRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if session == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// Get messages for this session
	memories, err := h.memoryRepo.GetBySessionID(id, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	messages := make([]model.MessageWithID, len(memories))
	for i, m := range memories {
		messages[i] = model.MessageWithID{
			ID:      m.ID,
			Role:    m.Role,
			Content: m.Content,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"session":  session.ToResponse(),
		"messages": messages,
	})
}

// Delete deletes a session and its messages
// DELETE /api/v1/sessions/:id
func (h *SessionHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	// Delete memories first
	if err := h.memoryRepo.DeleteBySessionID(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Delete session
	if err := h.sessionRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// DeleteMessage deletes a single message from a session
// DELETE /api/v1/sessions/:id/messages/:messageId
func (h *SessionHandler) DeleteMessage(c *gin.Context) {
	sessionID := c.Param("id")
	messageID := c.Param("messageId")

	// Verify the message belongs to this session
	memory, err := h.memoryRepo.GetByID(messageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if memory == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}
	if memory.SessionID != sessionID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message does not belong to this session"})
		return
	}

	// Delete the message
	if err := h.memoryRepo.Delete(messageID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
