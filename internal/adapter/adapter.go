package adapter

import (
	"context"

	"github.com/allwaysyou/llm-agent/internal/model"
)

// LLMAdapter defines the interface for LLM providers
type LLMAdapter interface {
	// Chat sends messages and returns a complete response
	Chat(ctx context.Context, messages []model.Message) (*model.ChatResponse, error)

	// ChatStream sends messages and returns a channel for streaming response
	ChatStream(ctx context.Context, messages []model.Message) (<-chan model.StreamChunk, error)

	// GetEmbedding returns the embedding vector for the given text
	GetEmbedding(ctx context.Context, text string) ([]float32, error)

	// CountTokens returns the estimated token count for the given text
	CountTokens(text string) int

	// Name returns the adapter name
	Name() string

	// Provider returns the provider type
	Provider() model.LLMProvider
}

// AdapterConfig holds configuration for creating an adapter
type AdapterConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	MaxTokens   int
	Temperature float64
}

// AdapterFactory creates adapters based on provider type
type AdapterFactory struct {
	creators map[model.LLMProvider]func(cfg AdapterConfig) (LLMAdapter, error)
}

// NewAdapterFactory creates a new AdapterFactory
func NewAdapterFactory() *AdapterFactory {
	return &AdapterFactory{
		creators: make(map[model.LLMProvider]func(cfg AdapterConfig) (LLMAdapter, error)),
	}
}

// Register registers a creator function for a provider
func (f *AdapterFactory) Register(provider model.LLMProvider, creator func(cfg AdapterConfig) (LLMAdapter, error)) {
	f.creators[provider] = creator
}

// Create creates an adapter for the given provider and config
func (f *AdapterFactory) Create(provider model.LLMProvider, cfg AdapterConfig) (LLMAdapter, error) {
	creator, ok := f.creators[provider]
	if !ok {
		return nil, &ErrUnsupportedProvider{Provider: provider}
	}
	return creator(cfg)
}

// ErrUnsupportedProvider is returned when a provider is not supported
type ErrUnsupportedProvider struct {
	Provider model.LLMProvider
}

func (e *ErrUnsupportedProvider) Error() string {
	return "unsupported LLM provider: " + string(e.Provider)
}
