package service

import (
	"context"
	"fmt"
	"time"

	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/pkg/embedding"
	"github.com/allwaysyou/llm-agent/internal/pkg/vector"
	"github.com/allwaysyou/llm-agent/internal/repository"
	"github.com/google/uuid"
)

// MemoryService handles memory storage and retrieval with semantic search
type MemoryService struct {
	memoryRepo    *repository.MemoryRepository
	sessionRepo   *repository.SessionRepository
	vectorStore   *vector.VectorStore
	embedProvider embedding.Provider
}

// NewMemoryService creates a new memory service
func NewMemoryService(
	memoryRepo *repository.MemoryRepository,
	sessionRepo *repository.SessionRepository,
	vectorStore *vector.VectorStore,
	embedProvider embedding.Provider,
) *MemoryService {
	return &MemoryService{
		memoryRepo:    memoryRepo,
		sessionRepo:   sessionRepo,
		vectorStore:   vectorStore,
		embedProvider: embedProvider,
	}
}

// SaveMemory saves a memory with its embedding
func (s *MemoryService) SaveMemory(ctx context.Context, sessionID string, role model.MessageRole, content string) (*model.Memory, error) {
	memory := &model.Memory{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		Tokens:    len(content) / 4, // Rough estimation
		CreatedAt: time.Now(),
	}

	// Save to relational database
	if err := s.memoryRepo.Create(memory); err != nil {
		return nil, fmt.Errorf("failed to save memory: %w", err)
	}

	// Generate and save embedding for semantic search
	if s.embedProvider != nil && content != "" {
		go func() {
			emb, err := s.embedProvider.GetEmbedding(context.Background(), content)
			if err != nil {
				fmt.Printf("failed to get embedding: %v\n", err)
				return
			}

			doc := vector.Document{
				ID:        memory.ID,
				Content:   content,
				Embedding: emb,
				Metadata: map[string]string{
					"session_id": sessionID,
					"role":       string(role),
				},
			}

			if err := s.vectorStore.Add(doc); err != nil {
				fmt.Printf("failed to save embedding: %v\n", err)
			}
		}()
	}

	return memory, nil
}

// SearchMemories searches for relevant memories using semantic similarity
func (s *MemoryService) SearchMemories(ctx context.Context, query string, sessionID string, limit int) ([]model.MemorySearchResult, error) {
	if s.embedProvider == nil {
		return nil, fmt.Errorf("embedding provider not configured")
	}

	if limit <= 0 {
		limit = 10
	}

	// Get query embedding
	queryEmb, err := s.embedProvider.GetEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get query embedding: %w", err)
	}

	// Build filter
	var filter map[string]string
	if sessionID != "" {
		filter = map[string]string{"session_id": sessionID}
	}

	// Search in vector store
	results := s.vectorStore.Search(queryEmb, limit, filter)

	// Convert to memory search results
	searchResults := make([]model.MemorySearchResult, 0, len(results))
	for _, r := range results {
		memory, err := s.memoryRepo.GetByID(r.Document.ID)
		if err != nil || memory == nil {
			continue
		}

		searchResults = append(searchResults, model.MemorySearchResult{
			Memory:   *memory,
			Score:    r.Score,
			Distance: 1 - r.Score, // Convert similarity to distance
		})
	}

	return searchResults, nil
}

// SearchAcrossSessions searches memories across all sessions
func (s *MemoryService) SearchAcrossSessions(ctx context.Context, query string, limit int) ([]model.MemorySearchResult, error) {
	return s.SearchMemories(ctx, query, "", limit)
}

// GetSessionMemories retrieves all memories for a session
func (s *MemoryService) GetSessionMemories(ctx context.Context, sessionID string, limit int) ([]model.Memory, error) {
	return s.memoryRepo.GetBySessionID(sessionID, limit)
}

// GetRecentMemories retrieves recent memories for a session
func (s *MemoryService) GetRecentMemories(ctx context.Context, sessionID string, limit int) ([]model.Memory, error) {
	return s.memoryRepo.GetRecentBySessionID(sessionID, limit)
}

// DeleteSessionMemories deletes all memories for a session
func (s *MemoryService) DeleteSessionMemories(ctx context.Context, sessionID string) error {
	// Delete from vector store
	if err := s.vectorStore.DeleteByMetadata("session_id", sessionID); err != nil {
		return fmt.Errorf("failed to delete from vector store: %w", err)
	}

	// Delete from relational database
	if err := s.memoryRepo.DeleteBySessionID(sessionID); err != nil {
		return fmt.Errorf("failed to delete memories: %w", err)
	}

	return nil
}

// BuildContextMessages builds context messages for a chat request
// It combines recent session history with semantically relevant memories
func (s *MemoryService) BuildContextMessages(ctx context.Context, sessionID, query string, maxContextTokens int) ([]model.Message, error) {
	var messages []model.Message

	// 1. Get recent conversation history from current session
	recentMemories, err := s.memoryRepo.GetRecentBySessionID(sessionID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent memories: %w", err)
	}

	// 2. Search for relevant memories across all sessions
	var relevantMemories []model.MemorySearchResult
	if s.embedProvider != nil && query != "" {
		relevantMemories, _ = s.SearchAcrossSessions(ctx, query, 5)
	}

	// 3. Build system context with relevant memories
	if len(relevantMemories) > 0 {
		var contextParts []string
		for _, rm := range relevantMemories {
			// Skip if already in recent memories
			isRecent := false
			for _, recent := range recentMemories {
				if recent.ID == rm.Memory.ID {
					isRecent = true
					break
				}
			}
			if !isRecent && rm.Score > 0.7 { // Only include highly relevant memories
				contextParts = append(contextParts, fmt.Sprintf("[%s]: %s", rm.Memory.Role, rm.Memory.Content))
			}
		}

		if len(contextParts) > 0 {
			contextContent := "Relevant context from previous conversations:\n"
			for _, part := range contextParts {
				contextContent += "- " + part + "\n"
			}
			messages = append(messages, model.Message{
				Role:    model.RoleSystem,
				Content: contextContent,
			})
		}
	}

	// 4. Add recent conversation history
	for _, mem := range recentMemories {
		messages = append(messages, model.Message{
			Role:    mem.Role,
			Content: mem.Content,
		})
	}

	return messages, nil
}
