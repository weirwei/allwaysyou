package entity

import "context"

// LLMProvider defines the unified interface for LLM providers
type LLMProvider interface {
	// Chat sends a chat completion request
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	// ChatStream sends a streaming chat completion request
	ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error)
	// Embedding generates embeddings for the given text
	Embedding(ctx context.Context, text string) ([]float32, error)
	// GetModelInfo returns information about the current model
	GetModelInfo() ModelInfo
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	TopP        float64       `json:"top_p,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// ChatMessage represents a single message in the conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID           string      `json:"id"`
	Content      string      `json:"content"`
	Role         string      `json:"role"`
	FinishReason string      `json:"finish_reason,omitempty"`
	Usage        *TokenUsage `json:"usage,omitempty"`
}

// ChatChunk represents a streaming response chunk
type ChatChunk struct {
	ID           string `json:"id"`
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason,omitempty"`
	Error        error  `json:"-"`
}

// TokenUsage represents token usage information
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModelInfo contains information about an LLM model
type ModelInfo struct {
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	MaxTokens   int    `json:"max_tokens"`
	ContextSize int    `json:"context_size"`
}
