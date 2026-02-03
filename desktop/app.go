package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

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
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx    context.Context
	server *http.Server
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Start the embedded HTTP server in a goroutine
	go a.startServer()
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	if a.server != nil {
		a.server.Shutdown(ctx)
	}
}

// Window control methods for custom title bar
func (a *App) Minimize() {
	runtime.WindowMinimise(a.ctx)
}

func (a *App) Maximize() {
	runtime.WindowToggleMaximise(a.ctx)
}

func (a *App) Close() {
	runtime.Quit(a.ctx)
}

// getDataDir returns the application data directory
func getDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Println("Failed to get home directory, using current directory")
		return "."
	}
	dataDir := filepath.Join(homeDir, "Library", "Application Support", "AllWaysYou")
	os.MkdirAll(dataDir, 0755)
	return dataDir
}

// getConfigPath returns the config file path, creating default if needed
func getConfigPath() string {
	dataDir := getDataDir()
	configPath := filepath.Join(dataDir, "config.yaml")

	// If config doesn't exist, create default
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := `server:
  host: "127.0.0.1"
  port: 18080
  mode: "release"

database:
  path: "` + filepath.Join(dataDir, "llm.db") + `"

vector:
  path: "` + filepath.Join(dataDir, "vectors") + `"
  collection: "memories"

embedding:
  provider: "ollama"
  model: "nomic-embed-text"
  base_url: "http://localhost:11434"

memory:
  conflict_detection_threshold: 0.85
  similar_knowledge_threshold: 0.7
  context_relevance_threshold: 0.5

llm:
  max_tokens: 4096
  temperature: 0.7
  stream_buffer_size: 100
  title_max_length: 50
`
		os.WriteFile(configPath, []byte(defaultConfig), 0644)
	}
	return configPath
}

// startServer starts the embedded HTTP server
func (a *App) startServer() {
	configPath := getConfigPath()

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		return
	}

	// Initialize database
	db, err := repository.NewDB(cfg.Database.Path)
	if err != nil {
		log.Printf("Failed to initialize database: %v", err)
		return
	}

	// Initialize encryptor
	encryptionKey := cfg.Encryption.Key
	if encryptionKey == "" {
		encryptionKey = "01234567890123456789012345678901" // 32 bytes default
	}

	encryptor, err := crypto.NewEncryptor(encryptionKey)
	if err != nil {
		log.Printf("Failed to initialize encryptor: %v", err)
		return
	}

	// Initialize vector store
	vectorPath := filepath.Join(cfg.Vector.Path, "vectors.json")
	os.MkdirAll(cfg.Vector.Path, 0755)
	vectorStore, err := vector.NewVectorStore(vectorPath)
	if err != nil {
		log.Printf("Failed to initialize vector store: %v", err)
		return
	}

	// Initialize repositories
	configRepo := repository.NewConfigRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	memoryRepo := repository.NewMemoryRepository(db)
	knowledgeRepo := repository.NewKnowledgeRepository(db)

	// Initialize adapter factory
	adapterFactory := adapter.NewAdapterFactory()
	adapterFactory.Register(model.ProviderOpenAI, adapter.NewOpenAIAdapter)
	adapterFactory.Register(model.ProviderClaude, adapter.NewClaudeAdapter)
	adapterFactory.Register(model.ProviderAzure, adapter.NewAzureAdapter)
	adapterFactory.Register(model.ProviderOllama, adapter.NewOllamaAdapter)

	// Initialize config service
	configService := service.NewConfigService(configRepo, encryptor)

	// Initialize embedding provider
	var embedProvider embedding.Provider

	// First try to get embedding config from database
	embeddingConfig, _ := configService.GetDefaultByType(model.ConfigTypeEmbedding)
	if embeddingConfig != nil {
		apiKey, err := configService.DecryptAPIKey(embeddingConfig.APIKey)
		if err == nil {
			switch embeddingConfig.Provider {
			case model.ProviderOpenAI, model.ProviderCustom:
				embedProvider = embedding.NewOpenAIProvider(apiKey, embeddingConfig.BaseURL, embeddingConfig.Model)
			case model.ProviderAzure:
				embedProvider = embedding.NewOpenAIProvider(apiKey, embeddingConfig.BaseURL, embeddingConfig.Model)
			}
		}
	}

	// Fallback to YAML config if no database config
	if embedProvider == nil {
		switch cfg.Embedding.Provider {
		case "ollama":
			embedProvider = embedding.NewOllamaProvider(cfg.Embedding.BaseURL, cfg.Embedding.Model)
		case "openai":
			defaultConfig, _ := configService.GetDefaultByType(model.ConfigTypeChat)
			if defaultConfig != nil {
				apiKey, err := configService.DecryptAPIKey(defaultConfig.APIKey)
				if err == nil {
					embedProvider = embedding.NewOpenAIProvider(apiKey, defaultConfig.BaseURL, cfg.Embedding.Model)
				}
			}
		}
	}

	// Initialize memory manager
	memoryManager := memory.NewManager(memoryRepo, knowledgeRepo, vectorStore, embedProvider, cfg.Memory)

	// Initialize services
	memoryService := service.NewMemoryService(memoryRepo, knowledgeRepo, sessionRepo, vectorStore, embedProvider, cfg.Memory)
	chatService := service.NewChatService(configService, sessionRepo, memoryManager, adapterFactory, cfg.LLM)
	summarizeService := service.NewSummarizeService(sessionRepo, memoryRepo, configService, adapterFactory)

	// Initialize handlers
	configHandler := handler.NewConfigHandler(configService, adapterFactory)
	chatHandler := handler.NewChatHandler(chatService)
	sessionHandler := handler.NewSessionHandler(sessionRepo, memoryRepo)
	memoryHandler := handler.NewMemoryHandler(memoryService, summarizeService)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
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

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		configs := api.Group("/configs")
		{
			configs.POST("", configHandler.Create)
			configs.GET("", configHandler.GetAll)
			configs.GET("/:id", configHandler.GetByID)
			configs.PUT("/:id", configHandler.Update)
			configs.DELETE("/:id", configHandler.Delete)
			configs.POST("/:id/test", configHandler.Test)
		}

		api.POST("/chat", chatHandler.Chat)

		sessions := api.Group("/sessions")
		{
			sessions.GET("", sessionHandler.GetAll)
			sessions.GET("/:id", sessionHandler.GetByID)
			sessions.DELETE("/:id", sessionHandler.Delete)
			sessions.DELETE("/:id/messages/:messageId", sessionHandler.DeleteMessage)
			sessions.POST("/:id/summarize", memoryHandler.Summarize)
		}

		memories := api.Group("/memories")
		{
			memories.GET("/search", memoryHandler.Search)
			memories.POST("", memoryHandler.Create)
		}

		knowledge := api.Group("/knowledge")
		{
			knowledge.GET("", memoryHandler.GetAllKnowledge)
			knowledge.GET("/:id", memoryHandler.GetKnowledge)
			knowledge.POST("", memoryHandler.CreateKnowledge)
			knowledge.PUT("/:id", memoryHandler.UpdateKnowledge)
			knowledge.DELETE("/:id", memoryHandler.DeleteKnowledge)
		}
	}

	// Create HTTP server
	addr := "127.0.0.1:18080"
	a.server = &http.Server{
		Addr:    addr,
		Handler: r,
	}

	fmt.Printf("Starting embedded server on %s\n", addr)
	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("Server error: %v", err)
	}
}
