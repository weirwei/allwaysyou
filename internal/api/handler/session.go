package handler

import (
	"net/http"
	"strconv"

	"github.com/allwaysyou/llm-agent/internal/service/agent"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SessionHandler struct {
	agentService *agent.Service
}

func NewSessionHandler(agentService *agent.Service) *SessionHandler {
	return &SessionHandler{agentService: agentService}
}

// ListSessions returns a list of sessions for the user
// @Summary List sessions
// @Tags sessions
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} entity.Session
// @Router /api/v1/sessions [get]
func (h *SessionHandler) ListSessions(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(uuid.UUID)
	if !ok {
		uid = uuid.New()
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	sessions, err := h.agentService.ListSessions(c.Request.Context(), uid, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// GetSession returns a session by ID
// @Summary Get session
// @Tags sessions
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} entity.Session
// @Router /api/v1/sessions/{id} [get]
func (h *SessionHandler) GetSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	session, err := h.agentService.GetSession(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// DeleteSession deletes a session
// @Summary Delete session
// @Tags sessions
// @Param id path string true "Session ID"
// @Success 204
// @Router /api/v1/sessions/{id} [delete]
func (h *SessionHandler) DeleteSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	if err := h.agentService.DeleteSession(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetProviders returns the list of available LLM providers
// @Summary Get available providers
// @Tags providers
// @Produce json
// @Success 200 {array} string
// @Router /api/v1/providers [get]
func (h *SessionHandler) GetProviders(c *gin.Context) {
	providers := h.agentService.GetProviders()
	c.JSON(http.StatusOK, providers)
}
