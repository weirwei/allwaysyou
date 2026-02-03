package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/allwaysyou/llm-agent/internal/adapter"
	"github.com/allwaysyou/llm-agent/internal/config"
	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/pkg/constants"
	"github.com/allwaysyou/llm-agent/internal/pkg/memory"
	"github.com/allwaysyou/llm-agent/internal/repository"
	"github.com/google/uuid"
)

// ChatService handles chat interactions
type ChatService struct {
	configService  *ConfigService
	sessionRepo    *repository.SessionRepository
	memoryManager  *memory.DefaultManager
	adapterFactory *adapter.AdapterFactory
	llmConfig      config.LLMDefaults
}

// NewChatService creates a new chat service
func NewChatService(
	configService *ConfigService,
	sessionRepo *repository.SessionRepository,
	memoryManager *memory.DefaultManager,
	adapterFactory *adapter.AdapterFactory,
	llmCfg config.LLMDefaults,
) *ChatService {
	return &ChatService{
		configService:  configService,
		sessionRepo:    sessionRepo,
		memoryManager:  memoryManager,
		adapterFactory: adapterFactory,
		llmConfig:      llmCfg,
	}
}

// Chat processes a chat request and returns a response
func (s *ChatService) Chat(ctx context.Context, req *model.ChatRequest) (*model.ChatResponse, error) {
	log.Printf("[ChatService:Chat] Starting - SessionID=%s, ConfigID=%s, MsgCount=%d",
		req.SessionID, req.ConfigID, len(req.Messages))

	// Get LLM config
	var llmConfig *model.LLMConfig
	var err error

	if req.ConfigID != "" {
		llmConfig, err = s.configService.GetByID(req.ConfigID)
	} else {
		llmConfig, err = s.configService.GetDefaultByType(model.ConfigTypeChat)
	}
	if err != nil {
		log.Printf("[ChatService:Chat] Error getting config: %v", err)
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	if llmConfig == nil {
		log.Printf("[ChatService:Chat] Error: no LLM config available")
		return nil, fmt.Errorf("no LLM config available")
	}
	log.Printf("[ChatService:Chat] Using config - ID=%s, Provider=%s, Model=%s",
		llmConfig.ID, llmConfig.Provider, llmConfig.Model)

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
			Title:     generateTitle(req.Messages, s.llmConfig.TitleMaxLength),
			ConfigID:  llmConfig.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.sessionRepo.Create(session); err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	}

	// Build messages with context (includes semantic search for long-term memory)
	query := ""
	for _, msg := range req.Messages {
		if msg.Role == model.RoleUser {
			query = msg.Content
			break
		}
	}
	log.Printf("[ChatService:Chat] Building context - SessionID=%s, Query='%.50s...'", session.ID, query)
	contextMessages, _ := s.memoryManager.BuildContext(ctx, session.ID, query)
	messages := append(contextMessages, req.Messages...)
	log.Printf("[ChatService:Chat] Context built - ContextMsgs=%d, TotalMsgs=%d", len(contextMessages), len(messages))

	// Call LLM
	log.Printf("[ChatService:Chat] Calling LLM...")
	resp, err := llmAdapter.Chat(ctx, messages)
	if err != nil {
		log.Printf("[ChatService:Chat] LLM call failed: %v", err)
		return nil, fmt.Errorf("LLM chat failed: %w", err)
	}
	log.Printf("[ChatService:Chat] LLM response received - ContentLen=%d", len(resp.Message.Content))

	resp.SessionID = session.ID

	// Save user messages via MemoryManager (generates embeddings)
	log.Printf("[ChatService:Chat] Saving user messages...")
	for _, msg := range req.Messages {
		if _, err := s.memoryManager.SaveConversationMemory(ctx, session.ID, msg.Role, msg.Content); err != nil {
			log.Printf("[ChatService:Chat] Failed to save user memory: %v", err)
		}
	}

	// Save assistant response via MemoryManager (generates embeddings)
	log.Printf("[ChatService:Chat] Saving assistant response...")
	if _, err := s.memoryManager.SaveConversationMemory(ctx, session.ID, model.RoleAssistant, resp.Message.Content); err != nil {
		log.Printf("[ChatService:Chat] Failed to save assistant memory: %v", err)
	}

	// Extract and save knowledge asynchronously
	log.Printf("[ChatService:Chat] Starting async knowledge extraction...")
	go func() {
		log.Printf("[ChatService:Chat:Async] ProcessConversation starting...")
		if err := s.memoryManager.ProcessConversation(context.Background(), session.ID, query, resp.Message.Content, llmAdapter); err != nil {
			log.Printf("[ChatService:Chat:Async] Failed to extract knowledge: %v", err)
		} else {
			log.Printf("[ChatService:Chat:Async] ProcessConversation completed")
		}
	}()

	// Update session timestamp
	session.UpdatedAt = time.Now()
	_ = s.sessionRepo.Update(session)

	log.Printf("[ChatService:Chat] Complete - SessionID=%s", session.ID)
	return resp, nil
}

// ChatStream processes a chat request and returns a streaming response
func (s *ChatService) ChatStream(ctx context.Context, req *model.ChatRequest) (<-chan model.StreamChunk, string, error) {
	log.Printf("[ChatService:ChatStream] Starting - SessionID=%s, ConfigID=%s, MsgCount=%d",
		req.SessionID, req.ConfigID, len(req.Messages))

	// Get LLM config
	var llmConfig *model.LLMConfig
	var err error

	if req.ConfigID != "" {
		llmConfig, err = s.configService.GetByID(req.ConfigID)
	} else {
		llmConfig, err = s.configService.GetDefaultByType(model.ConfigTypeChat)
	}
	if err != nil {
		log.Printf("[ChatService:ChatStream] Error getting config: %v", err)
		return nil, "", fmt.Errorf("failed to get config: %w", err)
	}
	if llmConfig == nil {
		log.Printf("[ChatService:ChatStream] Error: no LLM config available")
		return nil, "", fmt.Errorf("no LLM config available")
	}
	log.Printf("[ChatService:ChatStream] Using config - ID=%s, Provider=%s, Model=%s",
		llmConfig.ID, llmConfig.Provider, llmConfig.Model)

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
			Title:     generateTitle(req.Messages, s.llmConfig.TitleMaxLength),
			ConfigID:  llmConfig.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.sessionRepo.Create(session); err != nil {
			return nil, "", fmt.Errorf("failed to create session: %w", err)
		}
	}

	// Build messages with context (includes semantic search for long-term memory)
	query := ""
	for _, msg := range req.Messages {
		if msg.Role == model.RoleUser {
			query = msg.Content
			break
		}
	}
	log.Printf("[ChatService:ChatStream] Building context - SessionID=%s, Query='%.50s...'", session.ID, query)
	contextMessages, _ := s.memoryManager.BuildContext(ctx, session.ID, query)
	messages := append(contextMessages, req.Messages...)
	log.Printf("[ChatService:ChatStream] Context built - ContextMsgs=%d, TotalMsgs=%d", len(contextMessages), len(messages))

	// Save user messages via MemoryManager (generates embeddings)
	log.Printf("[ChatService:ChatStream] Saving user messages...")
	for _, msg := range req.Messages {
		if _, err := s.memoryManager.SaveConversationMemory(ctx, session.ID, msg.Role, msg.Content); err != nil {
			log.Printf("[ChatService:ChatStream] Failed to save user memory: %v", err)
		}
	}

	// Call LLM with streaming
	log.Printf("[ChatService:ChatStream] Starting LLM stream...")
	stream, err := llmAdapter.ChatStream(ctx, messages)
	if err != nil {
		log.Printf("[ChatService:ChatStream] LLM stream failed: %v", err)
		return nil, "", fmt.Errorf("LLM chat stream failed: %w", err)
	}

	// Wrap stream to save response
	outCh := make(chan model.StreamChunk, s.llmConfig.StreamBufferSize)
	go func() {
		defer close(outCh)

		var fullContent string
		var saved bool
		for chunk := range stream {
			outCh <- chunk
			fullContent += chunk.Delta

			if chunk.Done && !saved {
				saved = true
				log.Printf("[ChatService:ChatStream:Async] Stream done - ContentLen=%d", len(fullContent))

				// Save assistant response via MemoryManager (generates embeddings)
				log.Printf("[ChatService:ChatStream:Async] Saving assistant response...")
				if _, err := s.memoryManager.SaveConversationMemory(context.Background(), session.ID, model.RoleAssistant, fullContent); err != nil {
					log.Printf("[ChatService:ChatStream:Async] Failed to save assistant memory: %v", err)
				}

				// Extract and save knowledge asynchronously
				log.Printf("[ChatService:ChatStream:Async] Starting knowledge extraction...")
				go func(userQuery, assistantResp string) {
					log.Printf("[ChatService:ChatStream:Async:Knowledge] ProcessConversation starting...")
					if err := s.memoryManager.ProcessConversation(context.Background(), session.ID, userQuery, assistantResp, llmAdapter); err != nil {
						log.Printf("[ChatService:ChatStream:Async:Knowledge] Failed: %v", err)
					} else {
						log.Printf("[ChatService:ChatStream:Async:Knowledge] ProcessConversation completed")
					}
				}(query, fullContent)

				// Update session
				session.UpdatedAt = time.Now()
				_ = s.sessionRepo.Update(session)
			}
		}
	}()

	log.Printf("[ChatService:ChatStream] Stream started - SessionID=%s", session.ID)
	return outCh, session.ID, nil
}

// generateTitle generates a title from the first message
func generateTitle(messages []model.Message, maxLen int) string {
	for _, msg := range messages {
		if msg.Role == model.RoleUser && msg.Content != "" {
			title := msg.Content
			if len(title) > maxLen {
				title = title[:maxLen-3] + "..."
			}
			return title
		}
	}
	return constants.DefaultSessionTitle
}
