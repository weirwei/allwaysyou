package memory

import (
	"context"

	"github.com/allwaysyou/llm-agent/internal/adapter"
	"github.com/allwaysyou/llm-agent/internal/model"
)

// Manager defines the interface for memory management
type Manager interface {
	// SaveConversationMemory saves a conversation message
	SaveConversationMemory(ctx context.Context, sessionID string, role model.MessageRole, content string) (*model.Memory, error)

	// SearchKnowledge searches for relevant knowledge
	SearchKnowledge(ctx context.Context, opts SearchOptions) ([]model.KnowledgeSearchResult, error)

	// BuildContext builds context messages for LLM requests
	BuildContext(ctx context.Context, sessionID, query string) ([]model.Message, error)

	// ProcessConversation extracts and stores knowledge from a conversation
	ProcessConversation(ctx context.Context, sessionID, userMsg, assistantResp string, llm adapter.LLMAdapter) error

	// SupersedeKnowledge marks old knowledge as superseded by new one
	SupersedeKnowledge(ctx context.Context, oldID, newID string) error
}

// AddKnowledgeOptions represents options for adding knowledge
type AddKnowledgeOptions struct {
	Content    string
	Category   model.KnowledgeCategory
	Source     model.KnowledgeSource
	Importance float32
	Tier       model.KnowledgeTier // Memory tier (mid-term or long-term)
}

// SearchOptions represents options for searching knowledge
type SearchOptions struct {
	Query      string
	Categories []model.KnowledgeCategory // Optional: filter by categories
	ActiveOnly bool                      // Only return active (not superseded) knowledge
	MinScore   float32                   // Minimum similarity score
	Limit      int
}

// ExtractedFact represents a fact extracted from conversation
type ExtractedFact struct {
	Content    string
	Category   model.KnowledgeCategory
	Importance float32
}

// ConflictResult represents the result of conflict detection
type ConflictResult struct {
	HasConflict   bool
	ConflictingID string         // ID of the conflicting knowledge
	OldContent    string         // Content of the old knowledge
	Action        ConflictAction // Recommended action
}

// ConflictAction represents what to do with a conflict
type ConflictAction string

const (
	ActionCreate ConflictAction = "create" // Create new knowledge
	ActionUpdate ConflictAction = "update" // Update old knowledge (supersede)
	ActionSkip   ConflictAction = "skip"   // Skip, duplicate
)
