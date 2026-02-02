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
	sessionRepo    *repository.SessionRepository
	memoryRepo     *repository.MemoryRepository
	configService  *ConfigService
	adapterFactory *adapter.AdapterFactory
}

// NewSummarizeService creates a new summarize service
func NewSummarizeService(
	sessionRepo *repository.SessionRepository,
	memoryRepo *repository.MemoryRepository,
	configService *ConfigService,
	adapterFactory *adapter.AdapterFactory,
) *SummarizeService {
	return &SummarizeService{
		sessionRepo:    sessionRepo,
		memoryRepo:     memoryRepo,
		configService:  configService,
		adapterFactory: adapterFactory,
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

	// Get LLM config
	var llmConfig *model.LLMConfig
	if session.ConfigID != "" {
		llmConfig, err = s.configService.GetByID(session.ConfigID)
	} else {
		llmConfig, err = s.configService.GetDefault()
	}
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}
	if llmConfig == nil {
		return "", fmt.Errorf("no LLM config available")
	}

	// Create adapter
	apiKey, err := s.configService.DecryptAPIKey(llmConfig.APIKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	adapterCfg := adapter.AdapterConfig{
		APIKey:      apiKey,
		BaseURL:     llmConfig.BaseURL,
		Model:       llmConfig.Model,
		MaxTokens:   1024, // Limit summary length
		Temperature: 0.3,  // Lower temperature for more focused summary
	}

	llmAdapter, err := s.adapterFactory.Create(llmConfig.Provider, adapterCfg)
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

// AutoSummarizeIfNeeded checks if a session needs summarization and does it
func (s *SummarizeService) AutoSummarizeIfNeeded(ctx context.Context, sessionID string, threshold int) error {
	if threshold <= 0 {
		threshold = 20 // Default: summarize after 20 messages
	}

	// Count messages
	count, err := s.memoryRepo.CountBySessionID(sessionID)
	if err != nil {
		return fmt.Errorf("failed to count memories: %w", err)
	}

	// Check if we need to summarize
	if count < int64(threshold) {
		return nil
	}

	// Check if already summarized recently
	session, err := s.sessionRepo.GetByID(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return nil
	}

	// If already has a summary, skip (could add timestamp check for re-summarization)
	if session.Summary != "" {
		return nil
	}

	// Generate summary
	_, err = s.SummarizeSession(ctx, sessionID)
	return err
}

// GenerateTitle generates a title for a session based on its first messages
func (s *SummarizeService) GenerateTitle(ctx context.Context, sessionID string) (string, error) {
	// Get first few messages
	memories, err := s.memoryRepo.GetBySessionID(sessionID, 5)
	if err != nil {
		return "", fmt.Errorf("failed to get memories: %w", err)
	}

	if len(memories) == 0 {
		return "New Chat", nil
	}

	// Get LLM config
	session, err := s.sessionRepo.GetByID(sessionID)
	if err != nil || session == nil {
		return "New Chat", nil
	}

	var llmConfig *model.LLMConfig
	if session.ConfigID != "" {
		llmConfig, _ = s.configService.GetByID(session.ConfigID)
	} else {
		llmConfig, _ = s.configService.GetDefault()
	}

	if llmConfig == nil {
		// Fallback: use first user message as title
		for _, mem := range memories {
			if mem.Role == model.RoleUser {
				title := mem.Content
				if len(title) > 50 {
					title = title[:47] + "..."
				}
				return title, nil
			}
		}
		return "New Chat", nil
	}

	// Create adapter
	apiKey, err := s.configService.DecryptAPIKey(llmConfig.APIKey)
	if err != nil {
		return "New Chat", nil
	}

	adapterCfg := adapter.AdapterConfig{
		APIKey:      apiKey,
		BaseURL:     llmConfig.BaseURL,
		Model:       llmConfig.Model,
		MaxTokens:   50,
		Temperature: 0.3,
	}

	llmAdapter, err := s.adapterFactory.Create(llmConfig.Provider, adapterCfg)
	if err != nil {
		return "New Chat", nil
	}

	// Build conversation text
	var conversationParts []string
	for _, mem := range memories {
		conversationParts = append(conversationParts, fmt.Sprintf("%s: %s", mem.Role, mem.Content))
	}

	messages := []model.Message{
		{
			Role:    model.RoleSystem,
			Content: "Generate a short, descriptive title (max 50 characters) for this conversation. Return only the title, nothing else.",
		},
		{
			Role:    model.RoleUser,
			Content: strings.Join(conversationParts, "\n"),
		},
	}

	resp, err := llmAdapter.Chat(ctx, messages)
	if err != nil {
		return "New Chat", nil
	}

	title := strings.TrimSpace(resp.Message.Content)
	title = strings.Trim(title, "\"'")
	if len(title) > 50 {
		title = title[:47] + "..."
	}

	// Update session title
	session.Title = title
	_ = s.sessionRepo.Update(session)

	return title, nil
}
