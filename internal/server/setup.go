package server

import (
	"log"
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
)

// Handlers contains all HTTP handlers
type Handlers struct {
	Provider    *handler.ProviderHandler
	ModelConfig *handler.ModelConfigHandler
	Chat        *handler.ChatHandler
	Session     *handler.SessionHandler
	Memory      *handler.MemoryHandler
}

// Dependencies contains all initialized dependencies
type Dependencies struct {
	DB              *repository.DB
	Encryptor       *crypto.Encryptor
	VectorStore     *vector.VectorStore
	AdapterFactory  *adapter.AdapterFactory
	EmbedProvider   embedding.Provider

	// Services
	ProviderService    *service.ProviderService
	ModelConfigService *service.ModelConfigService
	ChatService        *service.ChatService
	MemoryService      *service.MemoryService
	SummarizeService   *service.SummarizeService
	MemoryManager      *memory.DefaultManager

	// Handlers
	Handlers *Handlers

	// Config
	Config *config.Config
}

// Initialize creates all dependencies from config
func Initialize(cfg *config.Config) (*Dependencies, error) {
	deps := &Dependencies{Config: cfg}

	// Initialize database
	db, err := repository.NewDB(cfg.Database.Path)
	if err != nil {
		return nil, err
	}
	deps.DB = db

	// Initialize encryptor
	encryptionKey := cfg.Encryption.Key
	if encryptionKey == "" {
		log.Println("WARNING: Using auto-generated encryption key. Set LLM_AGENT_ENCRYPTION_KEY in production.")
		encryptionKey = "01234567890123456789012345678901" // 32 bytes default
	}

	encryptor, err := crypto.NewEncryptor(encryptionKey)
	if err != nil {
		db.Close()
		return nil, err
	}
	deps.Encryptor = encryptor

	// Initialize vector store
	vectorPath := filepath.Join(cfg.Vector.Path, "vectors.json")
	os.MkdirAll(cfg.Vector.Path, 0755)
	vectorStore, err := vector.NewVectorStore(vectorPath)
	if err != nil {
		db.Close()
		return nil, err
	}
	deps.VectorStore = vectorStore

	// Initialize repositories
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
	adapterFactory.Register(model.ProviderOllama, adapter.NewOllamaAdapter)
	deps.AdapterFactory = adapterFactory

	// Initialize services
	providerService := service.NewProviderService(providerRepo, modelConfigRepo, encryptor)
	modelConfigService := service.NewModelConfigService(modelConfigRepo, providerRepo)
	deps.ProviderService = providerService
	deps.ModelConfigService = modelConfigService

	// Initialize embedding provider based on config
	var embedProvider embedding.Provider
	switch cfg.Embedding.Provider {
	case "ollama":
		embedProvider = embedding.NewOllamaProvider(cfg.Embedding.BaseURL, cfg.Embedding.Model)
		log.Printf("Using Ollama embedding provider (model: %s, url: %s)", cfg.Embedding.Model, cfg.Embedding.BaseURL)
	case "openai":
		// Try to get embedding config from database
		embeddingConfig, _ := modelConfigService.GetDefaultByType(model.ConfigTypeEmbedding)
		if embeddingConfig != nil && embeddingConfig.Provider != nil {
			apiKey, err := providerService.GetDecryptedAPIKey(embeddingConfig.ProviderID)
			if err == nil {
				embedProvider = embedding.NewOpenAIProvider(apiKey, embeddingConfig.Provider.BaseURL, embeddingConfig.Model)
				log.Printf("Using OpenAI embedding provider (model: %s)", embeddingConfig.Model)
			}
		}
	default:
		log.Printf("No embedding provider configured (provider: %s)", cfg.Embedding.Provider)
	}
	deps.EmbedProvider = embedProvider

	// Initialize memory manager
	memoryManager := memory.NewManager(memoryRepo, knowledgeRepo, vectorStore, embedProvider, cfg.Memory)
	deps.MemoryManager = memoryManager

	// Initialize services
	memoryService := service.NewMemoryService(memoryRepo, knowledgeRepo, sessionRepo, vectorStore, embedProvider, cfg.Memory)
	chatService := service.NewChatService(modelConfigService, providerService, sessionRepo, memoryManager, adapterFactory, cfg.LLM)
	summarizeService := service.NewSummarizeService(sessionRepo, memoryRepo, modelConfigService, providerService, adapterFactory)
	deps.MemoryService = memoryService
	deps.ChatService = chatService
	deps.SummarizeService = summarizeService

	// Initialize handlers
	deps.Handlers = &Handlers{
		Provider:    handler.NewProviderHandler(providerService, adapterFactory),
		ModelConfig: handler.NewModelConfigHandler(modelConfigService, providerService, adapterFactory),
		Chat:        handler.NewChatHandler(chatService),
		Session:     handler.NewSessionHandler(sessionRepo, memoryRepo),
		Memory:      handler.NewMemoryHandler(memoryService, summarizeService),
	}

	return deps, nil
}

// RegisterRoutes registers all API routes
func RegisterRoutes(api *gin.RouterGroup, deps *Dependencies) {
	h := deps.Handlers

	// Provider routes
	providers := api.Group("/providers")
	{
		providers.POST("", h.Provider.Create)
		providers.GET("", h.Provider.GetAll)
		providers.GET("/:id", h.Provider.GetByID)
		providers.PUT("/:id", h.Provider.Update)
		providers.DELETE("/:id", h.Provider.Delete)
		providers.POST("/:id/test", h.Provider.Test)
	}

	// Model Config routes
	models := api.Group("/models")
	{
		models.POST("", h.ModelConfig.Create)
		models.GET("", h.ModelConfig.GetAll)
		models.GET("/:id", h.ModelConfig.GetByID)
		models.PUT("/:id", h.ModelConfig.Update)
		models.DELETE("/:id", h.ModelConfig.Delete)
		models.POST("/:id/test", h.ModelConfig.Test)
		models.POST("/:id/default", h.ModelConfig.SetDefault)
	}

	// Chat routes
	api.POST("/chat", h.Chat.Chat)

	// Session routes
	sessions := api.Group("/sessions")
	{
		sessions.GET("", h.Session.GetAll)
		sessions.GET("/:id", h.Session.GetByID)
		sessions.DELETE("/:id", h.Session.Delete)
		sessions.DELETE("/:id/messages/:messageId", h.Session.DeleteMessage)
		sessions.POST("/:id/summarize", h.Memory.Summarize)
	}

	// Memory routes
	memories := api.Group("/memories")
	{
		memories.GET("/search", h.Memory.Search)
		memories.POST("", h.Memory.Create)
	}

	// Knowledge routes
	knowledge := api.Group("/knowledge")
	{
		knowledge.GET("", h.Memory.GetAllKnowledge)
		knowledge.GET("/:id", h.Memory.GetKnowledge)
		knowledge.POST("", h.Memory.CreateKnowledge)
		knowledge.PUT("/:id", h.Memory.UpdateKnowledge)
		knowledge.DELETE("/:id", h.Memory.DeleteKnowledge)
	}
}

// Close releases all resources
func (d *Dependencies) Close() {
	if d.DB != nil {
		d.DB.Close()
	}
}
