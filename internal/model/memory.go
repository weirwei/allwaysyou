package model

import "time"

// MessageRole represents the role of a message sender
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

// Memory represents a conversation message (short-term, session-scoped)
type Memory struct {
	ID        string      `json:"id" gorm:"primaryKey"`
	SessionID string      `json:"session_id" gorm:"index;not null"`
	Role      MessageRole `json:"role" gorm:"not null"`
	Content   string      `json:"content" gorm:"not null"`
	CreatedAt time.Time   `json:"created_at"`
}

// Message represents a chat message (used for API requests/responses)
type Message struct {
	Role    MessageRole `json:"role"`
	Content string      `json:"content"`
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	SessionID string    `json:"session_id"` // Optional: continue existing session
	ConfigID  string    `json:"config_id"`  // Optional: use specific config
	Messages  []Message `json:"messages"`   // Current conversation messages
	Stream    bool      `json:"stream"`     // Enable streaming response
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID        string  `json:"id"`
	SessionID string  `json:"session_id"`
	Message   Message `json:"message"`
	Usage     *Usage  `json:"usage,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk represents a chunk in streaming response
type StreamChunk struct {
	ID    string `json:"id"`
	Delta string `json:"delta"`
	Done  bool   `json:"done"`
	Usage *Usage `json:"usage,omitempty"`
}

// KnowledgeSearchRequest represents a request to search knowledge
type KnowledgeSearchRequest struct {
	Query string `json:"query" binding:"required"`
	Limit int    `json:"limit"`
}

// KnowledgeSearchResult represents a knowledge search result
type KnowledgeSearchResult struct {
	Knowledge Knowledge `json:"knowledge"`
	Score     float32   `json:"score"`
	Distance  float32   `json:"distance"`
}
