package service

import (
	"context"
	"fmt"
	"time"

	"github.com/allwaysyou/llm-agent/internal/adapter"
	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/repository"
	"github.com/google/uuid"
)

// ChatService handles chat interactions
type ChatService struct {
	configService   *ConfigService
	sessionRepo     *repository.SessionRepository
	memoryRepo      *repository.MemoryRepository
	adapterFactory  *adapter.AdapterFactory
}

// NewChatService creates a new chat service
func NewChatService(
	configService *ConfigService,
	sessionRepo *repository.SessionRepository,
	memoryRepo *repository.MemoryRepository,
	adapterFactory *adapter.AdapterFactory,
) *ChatService {
	return &ChatService{
		configService:  configService,
		sessionRepo:    sessionRepo,
		memoryRepo:     memoryRepo,
		adapterFactory: adapterFactory,
	}
}

// Chat processes a chat request and returns a response
func (s *ChatService) Chat(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
	// Get LLM config
	var llmConfig *model.LLMConfig
	var err error

	if req.ConfigID != "" {
		llmConfig, err = s.configService.GetByID(req.ConfigID)
	} else {
		llmConfig, err = s.configService.GetDefault()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	if llmConfig == nil {
		return nil, fmt.Errorf("no LLM config available")
	}

	// Decrypt API key
	apiKey, err := s.configService.DecryptAPIKey(llmConfig.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt API key: %w", err)
	}

	// Create adapter
	adapterCfg := adapter.AdapterConfig{
		APIKey:      apiKey,
		BaseURL:     llmConfig.BaseURL,
		Model:       llmConfig.Model,
		MaxTokens:   llmConfig.MaxTokens,
		Temperature: llmConfig.Temperature,
	}

	llmAdapter, err := s.adapterFactory.Create(llmConfig.Provider, adapterCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create adapter: %w", err)
	}

	// Get or create session
	var session *model.Session
	if req.SessionID != "" {
		session, err = s.sessionRepo.GetByID(req.SessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get session: %w", err)
		}
	}

	if session == nil {
		session = &model.Session{
			ID:        uuid.New().String(),
			Title:     generateTitle(req.Messages),
			ConfigID:  llmConfig.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.sessionRepo.Create(session); err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	}

	// Build messages with context
	messages := s.buildMessagesWithContext(ctx, session.ID, req.Messages)

	// Call LLM
	resp, err := llmAdapter.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM chat failed: %w", err)
	}

	resp.SessionID = session.ID

	// Save messages to memory
	now := time.Now()
	memories := make([]model.Memory, 0, len(req.Messages)+1)

	// Save user messages
	for _, msg := range req.Messages {
		memories = append(memories, model.Memory{
			ID:        uuid.New().String(),
			SessionID: session.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			Tokens:    llmAdapter.CountTokens(msg.Content),
			CreatedAt: now,
		})
	}

	// Save assistant response
	memories = append(memories, model.Memory{
		ID:        uuid.New().String(),
		SessionID: session.ID,
		Role:      model.RoleAssistant,
		Content:   resp.Message.Content,
		Tokens:    llmAdapter.CountTokens(resp.Message.Content),
		CreatedAt: now,
	})

	if err := s.memoryRepo.CreateBatch(memories); err != nil {
		// Log error but don't fail the response
		fmt.Printf("failed to save memories: %v\n", err)
	}

	// Update session timestamp
	session.UpdatedAt = time.Now()
	_ = s.sessionRepo.Update(session)

	return resp, nil
}

// ChatStream processes a chat request and returns a streaming response
func (s *ChatService) ChatStream(ctx context.Context, req *model.ChatRequest) (<-chan model.StreamChunk, string, error) {
	// Get LLM config
	var llmConfig *model.LLMConfig
	var err error

	if req.ConfigID != "" {
		llmConfig, err = s.configService.GetByID(req.ConfigID)
	} else {
		llmConfig, err = s.configService.GetDefault()
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to get config: %w", err)
	}
	if llmConfig == nil {
		return nil, "", fmt.Errorf("no LLM config available")
	}

	// Decrypt API key
	apiKey, err := s.configService.DecryptAPIKey(llmConfig.APIKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	// Create adapter
	adapterCfg := adapter.AdapterConfig{
		APIKey:      apiKey,
		BaseURL:     llmConfig.BaseURL,
		Model:       llmConfig.Model,
		MaxTokens:   llmConfig.MaxTokens,
		Temperature: llmConfig.Temperature,
	}

	llmAdapter, err := s.adapterFactory.Create(llmConfig.Provider, adapterCfg)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create adapter: %w", err)
	}

	// Get or create session
	var session *model.Session
	if req.SessionID != "" {
		session, err = s.sessionRepo.GetByID(req.SessionID)
		if err != nil {
			return nil, "", fmt.Errorf("failed to get session: %w", err)
		}
	}

	if session == nil {
		session = &model.Session{
			ID:        uuid.New().String(),
			Title:     generateTitle(req.Messages),
			ConfigID:  llmConfig.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.sessionRepo.Create(session); err != nil {
			return nil, "", fmt.Errorf("failed to create session: %w", err)
		}
	}

	// Build messages with context
	messages := s.buildMessagesWithContext(ctx, session.ID, req.Messages)

	// Save user messages to memory
	now := time.Now()
	for _, msg := range req.Messages {
		memory := model.Memory{
			ID:        uuid.New().String(),
			SessionID: session.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			Tokens:    llmAdapter.CountTokens(msg.Content),
			CreatedAt: now,
		}
		_ = s.memoryRepo.Create(&memory)
	}

	// Call LLM with streaming
	stream, err := llmAdapter.ChatStream(ctx, messages)
	if err != nil {
		return nil, "", fmt.Errorf("LLM chat stream failed: %w", err)
	}

	// Wrap stream to save response
	outCh := make(chan model.StreamChunk, 100)
	go func() {
		defer close(outCh)

		var fullContent string
		for chunk := range stream {
			outCh <- chunk
			fullContent += chunk.Delta

			if chunk.Done {
				// Save assistant response
				memory := model.Memory{
					ID:        uuid.New().String(),
					SessionID: session.ID,
					Role:      model.RoleAssistant,
					Content:   fullContent,
					Tokens:    llmAdapter.CountTokens(fullContent),
					CreatedAt: time.Now(),
				}
				_ = s.memoryRepo.Create(&memory)

				// Update session
				session.UpdatedAt = time.Now()
				_ = s.sessionRepo.Update(session)
			}
		}
	}()

	return outCh, session.ID, nil
}

// buildMessagesWithContext builds messages with historical context
func (s *ChatService) buildMessagesWithContext(ctx context.Context, sessionID string, currentMessages []model.Message) []model.Message {
	// Get recent memories from this session
	recentMemories, err := s.memoryRepo.GetRecentBySessionID(sessionID, 20)
	if err != nil || len(recentMemories) == 0 {
		return currentMessages
	}

	// Build message list: history + current
	messages := make([]model.Message, 0, len(recentMemories)+len(currentMessages))

	for _, mem := range recentMemories {
		messages = append(messages, model.Message{
			Role:    mem.Role,
			Content: mem.Content,
		})
	}

	messages = append(messages, currentMessages...)
	return messages
}

// generateTitle generates a title from the first message
func generateTitle(messages []model.Message) string {
	for _, msg := range messages {
		if msg.Role == model.RoleUser && msg.Content != "" {
			title := msg.Content
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			return title
		}
	}
	return "New Chat"
}
