package memory

import (
	"context"

	"github.com/allwaysyou/llm-agent/internal/config"
	"github.com/allwaysyou/llm-agent/internal/domain/entity"
	"github.com/allwaysyou/llm-agent/internal/domain/repository"
	"github.com/allwaysyou/llm-agent/internal/infrastructure/cache"
	"github.com/google/uuid"
)

type Service struct {
	memoryRepo repository.MemoryRepository
	msgRepo    repository.MessageRepository
	cache      *cache.RedisCache
	llm        entity.LLMProvider
	config     config.MemoryConfig
}

func NewService(
	memoryRepo repository.MemoryRepository,
	msgRepo repository.MessageRepository,
	cache *cache.RedisCache,
	llm entity.LLMProvider,
	cfg config.MemoryConfig,
) *Service {
	return &Service{
		memoryRepo: memoryRepo,
		msgRepo:    msgRepo,
		cache:      cache,
		llm:        llm,
		config:     cfg,
	}
}

// GetWorkingMemory retrieves the working memory (recent messages) for a session
func (s *Service) GetWorkingMemory(ctx context.Context, sessionID uuid.UUID) ([]*entity.Message, error) {
	// Try cache first
	messages, err := s.cache.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if messages != nil {
		return messages, nil
	}

	// Fall back to database
	messages, err = s.msgRepo.GetBySessionID(ctx, sessionID, s.config.Working.MaxMessages, 0)
	if err != nil {
		return nil, err
	}

	// Populate cache
	if len(messages) > 0 {
		if err := s.cache.SaveMessages(ctx, sessionID, messages); err != nil {
			// Log but don't fail
		}
	}

	return messages, nil
}

// SaveMessage saves a message to both working and episodic memory
func (s *Service) SaveMessage(ctx context.Context, sessionID uuid.UUID, msg *entity.Message) error {
	// Save to episodic memory (database)
	if err := s.msgRepo.Create(ctx, msg); err != nil {
		return err
	}

	// Update working memory (cache)
	return s.cache.AppendMessage(ctx, sessionID, msg, s.config.Working.MaxMessages)
}

// SearchSemanticMemory searches for relevant memories based on semantic similarity
func (s *Service) SearchSemanticMemory(ctx context.Context, userID uuid.UUID, query string) ([]*entity.MemoryFragment, error) {
	// Generate embedding for the query
	embedding, err := s.llm.Embedding(ctx, query)
	if err != nil {
		return nil, err
	}

	// Search for similar memories
	return s.memoryRepo.SearchSimilar(
		ctx,
		userID,
		embedding,
		s.config.Semantic.MaxResults,
		s.config.Semantic.SimilarityThreshold,
	)
}

// StoreSemanticMemory stores a memory with its embedding
func (s *Service) StoreSemanticMemory(ctx context.Context, userID uuid.UUID, content string, metadata map[string]interface{}) error {
	// Generate embedding
	embedding, err := s.llm.Embedding(ctx, content)
	if err != nil {
		return err
	}

	// Create memory vector
	memory := entity.NewMemoryVector(userID, content, embedding)
	memory.Metadata = metadata

	return s.memoryRepo.Create(ctx, memory)
}

// DeleteMemory deletes a specific memory
func (s *Service) DeleteMemory(ctx context.Context, memoryID uuid.UUID) error {
	return s.memoryRepo.Delete(ctx, memoryID)
}

// ClearSessionMemory clears the working memory for a session
func (s *Service) ClearSessionMemory(ctx context.Context, sessionID uuid.UUID) error {
	return s.cache.ClearSession(ctx, sessionID)
}
