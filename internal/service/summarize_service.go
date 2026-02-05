package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/allwaysyou/llm-agent/internal/adapter"
	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/repository"
)

// SummarizeService handles conversation summarization
type SummarizeService struct {
	sessionRepo        *repository.SessionRepository
	memoryRepo         *repository.MemoryRepository
	modelConfigService *ModelConfigService
	providerService    *ProviderService
	adapterFactory     *adapter.AdapterFactory
}

// NewSummarizeService creates a new summarize service
func NewSummarizeService(
	sessionRepo *repository.SessionRepository,
	memoryRepo *repository.MemoryRepository,
	modelConfigService *ModelConfigService,
	providerService *ProviderService,
	adapterFactory *adapter.AdapterFactory,
) *SummarizeService {
	return &SummarizeService{
		sessionRepo:        sessionRepo,
		memoryRepo:         memoryRepo,
		modelConfigService: modelConfigService,
		providerService:    providerService,
		adapterFactory:     adapterFactory,
	}
}

// SummarizeSession generates a summary for a session's conversation
func (s *SummarizeService) SummarizeSession(ctx context.Context, sessionID string) (string, error) {
	// Get session
	session, err := s.sessionRepo.GetByID(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return "", fmt.Errorf("session not found")
	}

	// Get all memories for this session
	memories, err := s.memoryRepo.GetBySessionID(sessionID, 0)
	if err != nil {
		return "", fmt.Errorf("failed to get memories: %w", err)
	}

	if len(memories) == 0 {
		return "", fmt.Errorf("no messages to summarize")
	}

	// Get model config (use summarize type config)
	modelConfig, err := s.modelConfigService.GetDefaultByType(model.ConfigTypeSummarize)
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}
	if modelConfig == nil || modelConfig.Provider == nil {
		// Fallback to chat config if no summarize config exists
		modelConfig, err = s.modelConfigService.GetDefaultByType(model.ConfigTypeChat)
		if err != nil {
			return "", fmt.Errorf("failed to get config: %w", err)
		}
	}
	if modelConfig == nil || modelConfig.Provider == nil {
		return "", fmt.Errorf("no LLM config available")
	}

	// Create adapter
	apiKey, err := s.providerService.GetDecryptedAPIKey(modelConfig.ProviderID)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	adapterCfg := adapter.AdapterConfig{
		APIKey:      apiKey,
		BaseURL:     modelConfig.Provider.BaseURL,
		Model:       modelConfig.Model,
		MaxTokens:   1024, // Limit summary length
		Temperature: 0.3,  // Lower temperature for more focused summary
	}

	llmAdapter, err := s.adapterFactory.Create(modelConfig.Provider.Type, adapterCfg)
	if err != nil {
		return "", fmt.Errorf("failed to create adapter: %w", err)
	}

	// Build conversation text
	var conversationParts []string
	for _, mem := range memories {
		conversationParts = append(conversationParts, fmt.Sprintf("%s: %s", mem.Role, mem.Content))
	}
	conversation := strings.Join(conversationParts, "\n\n")

	// Create summarization prompt
	messages := []model.Message{
		{
			Role: model.RoleSystem,
			Content: `You are a helpful assistant that summarizes conversations.
Create a concise summary that captures:
1. Main topics discussed
2. Key decisions or conclusions
3. Important information shared
4. Any action items or follow-ups

Keep the summary brief but comprehensive. Write in a neutral, factual tone.`,
		},
		{
			Role:    model.RoleUser,
			Content: fmt.Sprintf("Please summarize the following conversation:\n\n%s", conversation),
		},
	}

	// Generate summary
	resp, err := llmAdapter.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	summary := resp.Message.Content

	// Update session with summary
	session.Summary = summary
	if err := s.sessionRepo.Update(session); err != nil {
		return "", fmt.Errorf("failed to update session: %w", err)
	}

	return summary, nil
}

