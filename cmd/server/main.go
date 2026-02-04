package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/allwaysyou/llm-agent/internal/adapter"
	"github.com/allwaysyou/llm-agent/internal/config"
	"github.com/allwaysyou/llm-agent/internal/handler"
	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/pkg/crypto"
	"github.com/allwaysyou/llm-agent/internal/pkg/embedding"
	"github.com/allwaysyou/llm-agent/internal/pkg/memory"
	"github.com/allwaysyou/llm-agent/internal/pkg/vector"
	"github.com/allwaysyou/llm-agent/internal/repository"
	"github.com/allwaysyou/llm-agent/internal/service"
	"github.com/gin-gonic/gin"
)

var (
	configPath = flag.String("config", "", "path to config file")
)

func main() {
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := repository.NewDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize encryptor
	encryptionKey := cfg.Encryption.Key
	if encryptionKey == "" {
		log.Println("WARNING: Using auto-generated encryption key. Set LLM_AGENT_ENCRYPTION_KEY in production.")
		encryptionKey = "01234567890123456789012345678901" // 32 bytes default
	}

	encryptor, err := crypto.NewEncryptor(encryptionKey)
	if err != nil {
		log.Fatalf("Failed to initialize encryptor: %v", err)
	}

	// Initialize vector store
	vectorPath := filepath.Join(cfg.Vector.Path, "vectors.json")
	vectorStore, err := vector.NewVectorStore(vectorPath)
	if err != nil {
		log.Fatalf("Failed to initialize vector store: %v", err)
	}

	// Initialize repositories
	configRepo := repository.NewConfigRepository(db)
	providerRepo := repository.NewProviderRepository(db)
	modelConfigRepo := repository.NewModelConfigRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	memoryRepo := repository.NewMemoryRepository(db)
	knowledgeRepo := repository.NewKnowledgeRepository(db)

	// Initialize adapter factory with all providers
	adapterFactory := adapter.NewAdapterFactory()
	adapterFactory.Register(model.ProviderOpenAI, adapter.NewOpenAIAdapter)
	adapterFactory.Register(model.ProviderClaude, adapter.NewClaudeAdapter)
	adapterFactory.Register(model.ProviderAzure, adapter.NewAzureAdapter)

	// Initialize services
	configService := service.NewConfigService(configRepo, encryptor)
	providerService := service.NewProviderService(providerRepo, modelConfigRepo, encryptor)
	modelConfigService := service.NewModelConfigService(modelConfigRepo, providerRepo)

	// Initialize embedding provider based on config
	var embedProvider embedding.Provider
	switch cfg.Embedding.Provider {
	case "ollama":
		embedProvider = embedding.NewOllamaProvider(cfg.Embedding.BaseURL, cfg.Embedding.Model)
		log.Printf("Using Ollama embedding provider (model: %s, url: %s)", cfg.Embedding.Model, cfg.Embedding.BaseURL)
	case "openai":
		defaultConfig, _ := configService.GetDefault()
		if defaultConfig != nil {
			apiKey, err := configService.DecryptAPIKey(defaultConfig.APIKey)
			if err == nil {
				embedProvider = embedding.NewOpenAIProvider(apiKey, defaultConfig.BaseURL, cfg.Embedding.Model)
				log.Printf("Using OpenAI embedding provider (model: %s)", cfg.Embedding.Model)
			}
		}
	default:
		log.Printf("No embedding provider configured (provider: %s)", cfg.Embedding.Provider)
	}

	// Initialize memory manager (new architecture)
	memoryManager := memory.NewManager(memoryRepo, knowledgeRepo, vectorStore, embedProvider, cfg.Memory)

	// Initialize services
	memoryService := service.NewMemoryService(memoryRepo, knowledgeRepo, sessionRepo, vectorStore, embedProvider, cfg.Memory)
	chatService := service.NewChatService(configService, modelConfigService, providerService, sessionRepo, memoryManager, adapterFactory, cfg.LLM)
	summarizeService := service.NewSummarizeService(sessionRepo, memoryRepo, configService, adapterFactory)

	// Initialize handlers
	configHandler := handler.NewConfigHandler(configService, adapterFactory)
	providerHandler := handler.NewProviderHandler(providerService, adapterFactory)
	modelConfigHandler := handler.NewModelConfigHandler(modelConfigService, providerService, adapterFactory)
	chatHandler := handler.NewChatHandler(chatService)
	sessionHandler := handler.NewSessionHandler(sessionRepo, memoryRepo)
	memoryHandler := handler.NewMemoryHandler(memoryService, summarizeService)

	// Setup Gin
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Serve static files for frontend
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/", "./web/dist/index.html")
	r.StaticFile("/favicon.ico", "./web/dist/favicon.ico")

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// Legacy Config routes (kept for backward compatibility)
		configs := api.Group("/configs")
		{
			configs.POST("", configHandler.Create)
			configs.GET("", configHandler.GetAll)
			configs.GET("/:id", configHandler.GetByID)
			configs.PUT("/:id", configHandler.Update)
			configs.DELETE("/:id", configHandler.Delete)
			configs.POST("/:id/test", configHandler.Test)
		}

		// Provider routes (new)
		providers := api.Group("/providers")
		{
			providers.POST("", providerHandler.Create)
			providers.GET("", providerHandler.GetAll)
			providers.GET("/:id", providerHandler.GetByID)
			providers.PUT("/:id", providerHandler.Update)
			providers.DELETE("/:id", providerHandler.Delete)
			providers.POST("/:id/test", providerHandler.Test)
		}

		// Model Config routes (new)
		models := api.Group("/models")
		{
			models.POST("", modelConfigHandler.Create)
			models.GET("", modelConfigHandler.GetAll)
			models.GET("/:id", modelConfigHandler.GetByID)
			models.PUT("/:id", modelConfigHandler.Update)
			models.DELETE("/:id", modelConfigHandler.Delete)
			models.POST("/:id/test", modelConfigHandler.Test)
			models.POST("/:id/default", modelConfigHandler.SetDefault)
		}

		// Chat routes
		api.POST("/chat", chatHandler.Chat)

		// Session routes
		sessions := api.Group("/sessions")
		{
			sessions.GET("", sessionHandler.GetAll)
			sessions.GET("/:id", sessionHandler.GetByID)
			sessions.DELETE("/:id", sessionHandler.Delete)
			sessions.DELETE("/:id/messages/:messageId", sessionHandler.DeleteMessage)
			sessions.POST("/:id/summarize", memoryHandler.Summarize)
		}

		// Memory routes
		memories := api.Group("/memories")
		{
			memories.GET("/search", memoryHandler.Search)
			memories.POST("", memoryHandler.Create)
		}
	}

	// Start server
	addr := cfg.Address()
	fmt.Printf("Starting LLM Agent server on %s\n", addr)
	fmt.Printf("API: http://%s/api/v1\n", addr)
	fmt.Printf("Web UI: http://%s/\n", addr)

	go func() {
		if err := r.Run(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down server...")
}
