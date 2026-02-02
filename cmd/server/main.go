package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/allwaysyou/llm-agent/internal/api"
	"github.com/allwaysyou/llm-agent/internal/config"
	"github.com/allwaysyou/llm-agent/internal/domain/entity"
	"github.com/allwaysyou/llm-agent/internal/infrastructure/cache"
	"github.com/allwaysyou/llm-agent/internal/infrastructure/database"
	"github.com/allwaysyou/llm-agent/internal/infrastructure/llm/openai"
	"github.com/allwaysyou/llm-agent/internal/service/agent"
	"github.com/allwaysyou/llm-agent/internal/service/memory"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	// Load configuration
	var cfg *config.Config
	var err error

	if *configPath != "" {
		cfg, err = config.Load(*configPath)
	} else {
		cfg, err = config.Load("")
	}
	if err != nil {
		log.Printf("Warning: using default config: %v", err)
		cfg = config.Default()
	}

	// Initialize database
	db, err := database.NewPostgres(cfg.Database.Postgres)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize cache
	redisCache, err := cache.NewRedis(cfg.Database.Redis, cfg.Memory.Working.TTL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisCache.Close()

	// Initialize LLM providers
	providers := make(map[string]entity.LLMProvider)

	if providerCfg, ok := cfg.Providers["openai"]; ok && providerCfg.APIKey != "" {
		providers["openai"] = openai.New(providerCfg)
	}

	if len(providers) == 0 {
		log.Fatal("No LLM providers configured. Please set OPENAI_API_KEY or configure providers in config file.")
	}

	// Initialize repositories
	sessionRepo := database.NewSessionRepository(db)
	messageRepo := database.NewMessageRepository(db)
	memoryRepo := database.NewMemoryRepository(db)

	// Get default provider for embeddings
	var defaultProvider entity.LLMProvider
	for _, p := range providers {
		defaultProvider = p
		break
	}

	// Initialize services
	memoryService := memory.NewService(memoryRepo, messageRepo, redisCache, defaultProvider, cfg.Memory)
	agentService := agent.NewService(sessionRepo, providers, memoryService, *cfg)

	// Initialize router
	router := api.NewRouter(cfg, agentService)

	// Create server
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	// Start server
	go func() {
		log.Printf("Starting server on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
