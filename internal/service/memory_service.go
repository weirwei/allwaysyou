package service

import (
	"context"
	"fmt"
	"time"

	"github.com/allwaysyou/llm-agent/internal/config"
	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/pkg/constants"
	"github.com/allwaysyou/llm-agent/internal/pkg/embedding"
	"github.com/allwaysyou/llm-agent/internal/pkg/vector"
	"github.com/allwaysyou/llm-agent/internal/repository"
	"github.com/google/uuid"
)

// MemoryService handles memory storage and retrieval with semantic search
type MemoryService struct {
	memoryRepo    *repository.MemoryRepository
	knowledgeRepo *repository.KnowledgeRepository
	sessionRepo   *repository.SessionRepository
	vectorStore   *vector.VectorStore
	embedProvider embedding.Provider
	config        config.MemoryConfig
}

// NewMemoryService creates a new memory service
func NewMemoryService(
	memoryRepo *repository.MemoryRepository,
	knowledgeRepo *repository.KnowledgeRepository,
	sessionRepo *repository.SessionRepository,
	vectorStore *vector.VectorStore,
	embedProvider embedding.Provider,
	cfg config.MemoryConfig,
) *MemoryService {
	return &MemoryService{
		memoryRepo:    memoryRepo,
		knowledgeRepo: knowledgeRepo,
		sessionRepo:   sessionRepo,
		vectorStore:   vectorStore,
		embedProvider: embedProvider,
		config:        cfg,
	}
}

// SaveMemory saves a conversation memory
func (s *MemoryService) SaveMemory(ctx context.Context, sessionID string, role model.MessageRole, content string) (*model.Memory, error) {
	memory := &model.Memory{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := s.memoryRepo.Create(memory); err != nil {
		return nil, fmt.Errorf("failed to save memory: %w", err)
	}

	return memory, nil
}

// SearchMemories searches for relevant knowledge using semantic similarity
func (s *MemoryService) SearchMemories(ctx context.Context, query string, sessionID string, limit int) ([]model.KnowledgeSearchResult, error) {
	if s.embedProvider == nil {
		return nil, fmt.Errorf("embedding provider not configured")
	}

	if limit <= 0 {
		limit = s.config.DefaultSearchLimit
	}

	// Get query embedding
	queryEmb, err := s.embedProvider.GetEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get query embedding: %w", err)
	}

	// Search in vector store (knowledge only)
	filter := &vector.SearchFilter{
		ActiveOnly: true,
	}
	results := s.vectorStore.SearchWithFilter(queryEmb, limit, filter)

	// Convert to knowledge search results
	searchResults := make([]model.KnowledgeSearchResult, 0, len(results))
	for _, r := range results {
		// Only include knowledge documents
		if r.Document.MetaData == nil || r.Document.MetaData.Role != constants.RoleKnowledge {
			continue
		}

		knowledge, err := s.knowledgeRepo.GetByID(r.Document.ID)
		if err != nil || knowledge == nil {
			continue
		}

		searchResults = append(searchResults, model.KnowledgeSearchResult{
			Knowledge: *knowledge,
			Score:     r.Score,
			Distance:  1 - r.Score,
		})
	}

	return searchResults, nil
}
