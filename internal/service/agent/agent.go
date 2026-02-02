package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/allwaysyou/llm-agent/internal/config"
	"github.com/allwaysyou/llm-agent/internal/domain/entity"
	"github.com/allwaysyou/llm-agent/internal/domain/repository"
	"github.com/allwaysyou/llm-agent/internal/service/memory"
	"github.com/google/uuid"
)

type Service struct {
	sessionRepo repository.SessionRepository
	providers   map[string]entity.LLMProvider
	memoryService *memory.Service
	config      config.Config
}

func NewService(
	sessionRepo repository.SessionRepository,
	providers map[string]entity.LLMProvider,
	memoryService *memory.Service,
	cfg config.Config,
) *Service {
	return &Service{
		sessionRepo:   sessionRepo,
		providers:     providers,
		memoryService: memoryService,
		config:        cfg,
	}
}

// ChatRequest represents a chat request
type ChatRequest struct {
	SessionID     uuid.UUID
	UserID        uuid.UUID
	Message       string
	Provider      string
	Model         string
	SystemPrompt  string
	MaxTokens     int
	Temperature   float64
}

// ChatResponse represents a chat response
type ChatResponse struct {
	SessionID uuid.UUID          `json:"session_id"`
	Message   *entity.Message    `json:"message"`
	Usage     *entity.TokenUsage `json:"usage,omitempty"`
}

// Chat handles a chat request
func (s *Service) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Get or create session
	session, err := s.getOrCreateSession(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get or create session: %w", err)
	}

	// Get the LLM provider
	provider, ok := s.providers[session.Provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", session.Provider)
	}

	// Get working memory (recent messages)
	workingMemory, err := s.memoryService.GetWorkingMemory(ctx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("get working memory: %w", err)
	}

	// Search semantic memory for relevant context
	relevantMemories, err := s.memoryService.SearchSemanticMemory(ctx, req.UserID, req.Message)
	if err != nil {
		// Log but don't fail
		relevantMemories = nil
	}

	// Build the prompt
	messages := s.buildPrompt(session, workingMemory, relevantMemories, req.Message)

	// Save user message
	userMsg := entity.NewMessage(session.ID, entity.RoleUser, req.Message)
	if err := s.memoryService.SaveMessage(ctx, session.ID, userMsg); err != nil {
		return nil, fmt.Errorf("save user message: %w", err)
	}

	// Call LLM
	chatReq := &entity.ChatRequest{
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	llmResp, err := provider.Chat(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("LLM chat: %w", err)
	}

	// Save assistant message
	assistantMsg := entity.NewMessage(session.ID, entity.RoleAssistant, llmResp.Content)
	if llmResp.Usage != nil {
		assistantMsg.TokensUsed = llmResp.Usage.TotalTokens
	}
	if err := s.memoryService.SaveMessage(ctx, session.ID, assistantMsg); err != nil {
		return nil, fmt.Errorf("save assistant message: %w", err)
	}

	// Async: store to semantic memory
	go func() {
		content := fmt.Sprintf("User: %s\nAssistant: %s", req.Message, llmResp.Content)
		_ = s.memoryService.StoreSemanticMemory(context.Background(), req.UserID, content, map[string]interface{}{
			"session_id": session.ID.String(),
		})
	}()

	return &ChatResponse{
		SessionID: session.ID,
		Message:   assistantMsg,
		Usage:     llmResp.Usage,
	}, nil
}

// ChatStream handles a streaming chat request
func (s *Service) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *entity.ChatChunk, *entity.Message, error) {
	// Get or create session
	session, err := s.getOrCreateSession(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("get or create session: %w", err)
	}

	// Get the LLM provider
	provider, ok := s.providers[session.Provider]
	if !ok {
		return nil, nil, fmt.Errorf("unknown provider: %s", session.Provider)
	}

	// Get working memory
	workingMemory, err := s.memoryService.GetWorkingMemory(ctx, session.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("get working memory: %w", err)
	}

	// Search semantic memory
	relevantMemories, _ := s.memoryService.SearchSemanticMemory(ctx, req.UserID, req.Message)

	// Build prompt
	messages := s.buildPrompt(session, workingMemory, relevantMemories, req.Message)

	// Save user message
	userMsg := entity.NewMessage(session.ID, entity.RoleUser, req.Message)
	if err := s.memoryService.SaveMessage(ctx, session.ID, userMsg); err != nil {
		return nil, nil, fmt.Errorf("save user message: %w", err)
	}

	// Call LLM stream
	chatReq := &entity.ChatRequest{
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}

	stream, err := provider.ChatStream(ctx, chatReq)
	if err != nil {
		return nil, nil, fmt.Errorf("LLM chat stream: %w", err)
	}

	// Wrap stream to collect full response
	outCh := make(chan *entity.ChatChunk)
	assistantMsg := entity.NewMessage(session.ID, entity.RoleAssistant, "")

	go func() {
		defer close(outCh)
		var fullContent strings.Builder

		for chunk := range stream {
			if chunk.Error != nil {
				outCh <- chunk
				return
			}

			fullContent.WriteString(chunk.Content)
			outCh <- chunk
		}

		// Save completed message
		assistantMsg.Content = fullContent.String()
		_ = s.memoryService.SaveMessage(context.Background(), session.ID, assistantMsg)

		// Store to semantic memory
		content := fmt.Sprintf("User: %s\nAssistant: %s", req.Message, fullContent.String())
		_ = s.memoryService.StoreSemanticMemory(context.Background(), req.UserID, content, map[string]interface{}{
			"session_id": session.ID.String(),
		})
	}()

	return outCh, assistantMsg, nil
}

func (s *Service) getOrCreateSession(ctx context.Context, req *ChatRequest) (*entity.Session, error) {
	if req.SessionID != uuid.Nil {
		return s.sessionRepo.GetByID(ctx, req.SessionID)
	}

	// Create new session
	providerName := req.Provider
	if providerName == "" {
		providerName = "openai"
	}

	modelName := req.Model
	if modelName == "" {
		if providerCfg, ok := s.config.Providers[providerName]; ok {
			modelName = providerCfg.Model
		}
	}

	session := entity.NewSession(req.UserID, providerName, modelName)
	session.SystemPrompt = req.SystemPrompt

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Service) buildPrompt(session *entity.Session, workingMemory []*entity.Message, relevantMemories []*entity.MemoryFragment, userMessage string) []entity.ChatMessage {
	var messages []entity.ChatMessage

	// Add system prompt
	systemPrompt := session.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "You are a helpful AI assistant."
	}

	// Add relevant memories to system prompt
	if len(relevantMemories) > 0 {
		var memoryContext strings.Builder
		memoryContext.WriteString("\n\nRelevant context from previous conversations:\n")
		for _, mem := range relevantMemories {
			memoryContext.WriteString(fmt.Sprintf("- %s\n", mem.Content))
		}
		systemPrompt += memoryContext.String()
	}

	messages = append(messages, entity.ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	// Add working memory
	for _, msg := range workingMemory {
		messages = append(messages, entity.ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	// Add current user message
	messages = append(messages, entity.ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	return messages
}

// GetSession retrieves a session by ID
func (s *Service) GetSession(ctx context.Context, sessionID uuid.UUID) (*entity.Session, error) {
	return s.sessionRepo.GetByID(ctx, sessionID)
}

// ListSessions lists sessions for a user
func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Session, error) {
	return s.sessionRepo.GetByUserID(ctx, userID, limit, offset)
}

// DeleteSession deletes a session and its messages
func (s *Service) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	// Clear working memory
	_ = s.memoryService.ClearSessionMemory(ctx, sessionID)

	// Delete from database
	return s.sessionRepo.Delete(ctx, sessionID)
}

// GetProviders returns the list of available providers
func (s *Service) GetProviders() []string {
	providers := make([]string, 0, len(s.providers))
	for name := range s.providers {
		providers = append(providers, name)
	}
	return providers
}
