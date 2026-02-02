package api

import (
	"github.com/allwaysyou/llm-agent/internal/api/handler"
	"github.com/allwaysyou/llm-agent/internal/api/middleware"
	"github.com/allwaysyou/llm-agent/internal/config"
	"github.com/allwaysyou/llm-agent/internal/service/agent"
	"github.com/gin-gonic/gin"
)

func NewRouter(cfg *config.Config, agentService *agent.Service) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1
	v1 := r.Group("/api/v1")

	// Auth middleware (optional)
	if cfg.Auth.Enabled {
		v1.Use(middleware.Auth(cfg.Auth.JWTSecret))
	}

	// Chat endpoints
	chatHandler := handler.NewChatHandler(agentService)
	v1.POST("/chat", chatHandler.Chat)
	v1.POST("/chat/stream", chatHandler.ChatStream)

	// Session endpoints
	sessionHandler := handler.NewSessionHandler(agentService)
	v1.GET("/sessions", sessionHandler.ListSessions)
	v1.GET("/sessions/:id", sessionHandler.GetSession)
	v1.DELETE("/sessions/:id", sessionHandler.DeleteSession)

	// Provider endpoints
	v1.GET("/providers", sessionHandler.GetProviders)

	return r
}
