package memory

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/allwaysyou/llm-agent/internal/adapter"
	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/pkg/embedding"
	"github.com/allwaysyou/llm-agent/internal/pkg/vector"
	"github.com/allwaysyou/llm-agent/internal/repository"
	"github.com/google/uuid"
)

// DefaultManager implements the Manager interface
type DefaultManager struct {
	memoryRepo    *repository.MemoryRepository
	knowledgeRepo *repository.KnowledgeRepository
	vectorStore   *vector.VectorStore
	embedProvider embedding.Provider
	processor     *Processor
}

// NewManager creates a new memory manager
func NewManager(
	memoryRepo *repository.MemoryRepository,
	knowledgeRepo *repository.KnowledgeRepository,
	vectorStore *vector.VectorStore,
	embedProvider embedding.Provider,
) *DefaultManager {
	return &DefaultManager{
		memoryRepo:    memoryRepo,
		knowledgeRepo: knowledgeRepo,
		vectorStore:   vectorStore,
		embedProvider: embedProvider,
		processor:     NewProcessor(),
	}
}

// SaveConversationMemory saves a conversation message (short-term, session-scoped)
func (m *DefaultManager) SaveConversationMemory(ctx context.Context, sessionID string, role model.MessageRole, content string) (*model.Memory, error) {
	log.Printf("[Memory:SaveConversation] SessionID=%s, Role=%s, ContentLen=%d", sessionID, role, len(content))

	if content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	memory := &model.Memory{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := m.memoryRepo.Create(memory); err != nil {
		log.Printf("[Memory:SaveConversation] Error saving to DB: %v", err)
		return nil, fmt.Errorf("failed to save memory: %w", err)
	}
	log.Printf("[Memory:SaveConversation] Saved to DB - ID=%s", memory.ID)

	return memory, nil
}

// AddKnowledge adds extracted knowledge (long-term, global)
func (m *DefaultManager) AddKnowledge(ctx context.Context, opts AddKnowledgeOptions) (*model.Knowledge, error) {
	log.Printf("[Knowledge:Add] Starting - Category=%s, Source=%s, Importance=%.2f, ContentLen=%d",
		opts.Category, opts.Source, opts.Importance, len(opts.Content))

	if opts.Content == "" {
		log.Printf("[Knowledge:Add] Error: content cannot be empty")
		return nil, fmt.Errorf("content cannot be empty")
	}

	// Set defaults
	if opts.Importance <= 0 || opts.Importance > 1 {
		opts.Importance = 0.5
	}
	if opts.Source == "" {
		opts.Source = model.SourceExtracted
	}

	knowledge := &model.Knowledge{
		ID:        uuid.New().String(),
		Content:   opts.Content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to database
	if err := m.knowledgeRepo.Create(knowledge); err != nil {
		log.Printf("[Knowledge:Add] Error saving to DB: %v", err)
		return nil, fmt.Errorf("failed to save knowledge: %w", err)
	}
	log.Printf("[Knowledge:Add] Saved to DB - ID=%s", knowledge.ID)

	// Generate and save embedding asynchronously
	if m.embedProvider != nil {
		log.Printf("[Knowledge:Add] Scheduling embedding generation for ID=%s", knowledge.ID)
		go m.saveKnowledgeEmbedding(knowledge, opts.Category, opts.Source, opts.Importance)
	}

	return knowledge, nil
}

// saveKnowledgeEmbedding generates and saves the embedding for knowledge
func (m *DefaultManager) saveKnowledgeEmbedding(knowledge *model.Knowledge, category model.KnowledgeCategory, source model.KnowledgeSource, importance float32) {
	log.Printf("[Knowledge:Embedding] Generating embedding - ID=%s, ContentLen=%d", knowledge.ID, len(knowledge.Content))

	emb, err := m.embedProvider.GetEmbedding(context.Background(), knowledge.Content)
	if err != nil {
		log.Printf("[Knowledge:Embedding] Error getting embedding - ID=%s, Error=%v", knowledge.ID, err)
		return
	}
	log.Printf("[Knowledge:Embedding] Got embedding - ID=%s, Dimensions=%d", knowledge.ID, len(emb))

	doc := vector.Document{
		ID:        knowledge.ID,
		Content:   knowledge.Content,
		Embedding: emb,
		Metadata: map[string]string{
			"type":      "knowledge",
			"category":  string(category),
			"source":    string(source),
			"is_active": "true",
		},
		MetaData: &vector.DocumentMetadata{
			Role:       "knowledge",
			Category:   string(category),
			Source:     string(source),
			Importance: importance,
			IsActive:   true,
			CreatedAt:  knowledge.CreatedAt.Unix(),
		},
	}

	if err := m.vectorStore.Add(doc); err != nil {
		log.Printf("[Knowledge:Embedding] Error saving to vector store - ID=%s, Error=%v", knowledge.ID, err)
	} else {
		log.Printf("[Knowledge:Embedding] Saved to vector store - ID=%s", knowledge.ID)
	}
}

// SearchKnowledge searches for relevant knowledge
func (m *DefaultManager) SearchKnowledge(ctx context.Context, opts SearchOptions) ([]model.KnowledgeSearchResult, error) {
	log.Printf("[Knowledge:Search] Starting - Query='%s', Categories=%v, ActiveOnly=%v, MinScore=%.2f, Limit=%d",
		truncateStr(opts.Query, 50), opts.Categories, opts.ActiveOnly, opts.MinScore, opts.Limit)

	if m.embedProvider == nil {
		log.Printf("[Knowledge:Search] Error: embedding provider not configured")
		return nil, fmt.Errorf("embedding provider not configured")
	}

	if opts.Limit <= 0 {
		opts.Limit = 10
	}

	// Get query embedding
	queryEmb, err := m.embedProvider.GetEmbedding(ctx, opts.Query)
	if err != nil {
		log.Printf("[Knowledge:Search] Error getting query embedding: %v", err)
		return nil, fmt.Errorf("failed to get query embedding: %w", err)
	}
	log.Printf("[Knowledge:Search] Got query embedding - Dimensions=%d", len(queryEmb))

	// Build filter - only search knowledge (not conversation memories)
	filter := &vector.SearchFilter{
		ActiveOnly: opts.ActiveOnly,
		MinScore:   opts.MinScore,
	}

	// Convert categories
	if len(opts.Categories) > 0 {
		for _, c := range opts.Categories {
			filter.Categories = append(filter.Categories, string(c))
		}
	}

	// Search in vector store
	results := m.vectorStore.SearchWithFilter(queryEmb, opts.Limit, filter)
	log.Printf("[Knowledge:Search] Vector search returned %d results", len(results))

	// Convert to knowledge search results
	searchResults := make([]model.KnowledgeSearchResult, 0, len(results))
	for _, r := range results {
		// Only include knowledge documents
		if r.Document.MetaData == nil || r.Document.MetaData.Role != "knowledge" {
			continue
		}

		knowledge, err := m.knowledgeRepo.GetByID(r.Document.ID)
		if err != nil || knowledge == nil {
			log.Printf("[Knowledge:Search] Skip result - ID=%s, Error=%v", r.Document.ID, err)
			continue
		}

		searchResults = append(searchResults, model.KnowledgeSearchResult{
			Knowledge: *knowledge,
			Score:     r.Score,
			Distance:  1 - r.Score,
		})
		log.Printf("[Knowledge:Search] Result - ID=%s, Score=%.3f, Category=%s, Content='%s'",
			knowledge.ID, r.Score, r.Document.MetaData.Category, truncateStr(knowledge.Content, 50))
	}

	log.Printf("[Knowledge:Search] Returning %d results", len(searchResults))
	return searchResults, nil
}

// BuildContext builds context messages for LLM requests
func (m *DefaultManager) BuildContext(ctx context.Context, sessionID, query string) ([]model.Message, error) {
	log.Printf("[Memory:BuildContext] Starting - SessionID=%s, Query='%s'", sessionID, truncateStr(query, 50))

	var messages []model.Message

	// 1. Get recent conversation history from this session
	recentMemories, err := m.memoryRepo.GetRecentBySessionID(sessionID, 10)
	if err != nil {
		log.Printf("[Memory:BuildContext] Error getting recent memories: %v", err)
		return nil, fmt.Errorf("failed to get recent memories: %w", err)
	}
	log.Printf("[Memory:BuildContext] Got %d recent memories", len(recentMemories))

	// 2. Search for relevant knowledge (global, extracted facts)
	var knowledgeResults []model.KnowledgeSearchResult
	if m.embedProvider != nil && query != "" {
		log.Printf("[Memory:BuildContext] Searching for relevant knowledge...")
		knowledgeResults, _ = m.SearchKnowledge(ctx, SearchOptions{
			Query:      query,
			Categories: []model.KnowledgeCategory{model.CategoryPersonalInfo, model.CategoryPreference, model.CategoryFact},
			ActiveOnly: true,
			MinScore:   0.5,
			Limit:      20,
		})
		log.Printf("[Memory:BuildContext] Found %d knowledge results", len(knowledgeResults))
	}

	// 3. Build context from knowledge
	var knowledgeParts []string
	for _, kr := range knowledgeResults {
		// Skip if same content as query
		if kr.Knowledge.Content == query {
			log.Printf("[Memory:BuildContext] Skip (same as query) - ID=%s", kr.Knowledge.ID)
			continue
		}
		// Include knowledge with sufficient score
		if kr.Score > 0.5 {
			knowledgeParts = append(knowledgeParts, kr.Knowledge.Content)
			log.Printf("[Memory:BuildContext] Include knowledge - ID=%s, Score=%.3f, Content='%s'",
				kr.Knowledge.ID, kr.Score, truncateStr(kr.Knowledge.Content, 50))
			if len(knowledgeParts) >= 8 {
				log.Printf("[Memory:BuildContext] Reached max knowledge parts (8)")
				break
			}
		}
	}

	// 4. Build system message with knowledge
	if len(knowledgeParts) > 0 {
		contextContent := "已知用户信息:\n"
		for _, part := range knowledgeParts {
			contextContent += "- " + part + "\n"
		}
		messages = append(messages, model.Message{
			Role:    model.RoleSystem,
			Content: contextContent,
		})
		log.Printf("[Memory:BuildContext] Added system message with %d knowledge parts", len(knowledgeParts))
	}

	// 5. Add recent conversation history
	historyCount := 0
	for _, mem := range recentMemories {
		if mem.Role == model.RoleUser || mem.Role == model.RoleAssistant {
			messages = append(messages, model.Message{
				Role:    mem.Role,
				Content: mem.Content,
			})
			historyCount++
		}
	}
	log.Printf("[Memory:BuildContext] Added %d history messages", historyCount)

	log.Printf("[Memory:BuildContext] Complete - Total messages=%d", len(messages))
	return messages, nil
}

// ProcessConversation extracts and stores knowledge from a conversation
func (m *DefaultManager) ProcessConversation(ctx context.Context, sessionID, userMsg, assistantResp string, llm adapter.LLMAdapter) error {
	log.Printf("[Knowledge:Process] Starting - UserMsg='%s', AssistantResp='%s'",
		truncateStr(userMsg, 50), truncateStr(assistantResp, 50))

	if m.embedProvider == nil {
		log.Printf("[Knowledge:Process] Skipping - no embedding provider")
		return nil
	}

	// 1. Extract facts from conversation
	log.Printf("[Knowledge:Process] Extracting facts via LLM...")
	facts, err := m.processor.ExtractFacts(ctx, userMsg, assistantResp, llm)
	if err != nil {
		log.Printf("[Knowledge:Process] Error extracting facts: %v", err)
		return fmt.Errorf("failed to extract facts: %w", err)
	}

	if len(facts) == 0 {
		log.Printf("[Knowledge:Process] No facts extracted")
		return nil
	}
	log.Printf("[Knowledge:Process] Extracted %d facts", len(facts))

	// 2. Process each fact
	for i, fact := range facts {
		log.Printf("[Knowledge:Process] Processing fact %d/%d - Category=%s, Importance=%.2f, Content='%s'",
			i+1, len(facts), fact.Category, fact.Importance, truncateStr(fact.Content, 50))

		// Search for similar existing knowledge
		similar, err := m.SearchKnowledge(ctx, SearchOptions{
			Query:      fact.Content,
			ActiveOnly: true,
			MinScore:   0.7,
			Limit:      5,
		})
		if err != nil {
			log.Printf("[Knowledge:Process] Error searching similar: %v", err)
			continue
		}
		log.Printf("[Knowledge:Process] Found %d similar knowledge", len(similar))

		// Detect conflicts
		log.Printf("[Knowledge:Process] Detecting conflicts via LLM...")
		conflict, err := m.processor.DetectConflict(ctx, fact, similar, llm)
		if err != nil {
			log.Printf("[Knowledge:Process] Error detecting conflict: %v", err)
			continue
		}
		log.Printf("[Knowledge:Process] Conflict result - HasConflict=%v, Action=%s, ConflictingID=%s",
			conflict.HasConflict, conflict.Action, conflict.ConflictingID)

		switch conflict.Action {
		case ActionSkip:
			log.Printf("[Knowledge:Process] SKIP (duplicate) - Content='%s'", truncateStr(fact.Content, 50))
			continue

		case ActionUpdate:
			log.Printf("[Knowledge:Process] UPDATE - Old='%s' -> New='%s'",
				truncateStr(conflict.OldContent, 30), truncateStr(fact.Content, 30))
			newKnowledge, err := m.AddKnowledge(ctx, AddKnowledgeOptions{
				Content:    fact.Content,
				Category:   fact.Category,
				Source:     model.SourceExtracted,
				Importance: fact.Importance,
			})
			if err != nil {
				log.Printf("[Knowledge:Process] Error creating new knowledge: %v", err)
				continue
			}
			if err := m.SupersedeKnowledge(ctx, conflict.ConflictingID, newKnowledge.ID); err != nil {
				log.Printf("[Knowledge:Process] Error superseding old knowledge: %v", err)
			} else {
				log.Printf("[Knowledge:Process] Superseded %s with %s", conflict.ConflictingID, newKnowledge.ID)
			}

		case ActionCreate:
			log.Printf("[Knowledge:Process] CREATE - Content='%s'", truncateStr(fact.Content, 50))
			newKnowledge, err := m.AddKnowledge(ctx, AddKnowledgeOptions{
				Content:    fact.Content,
				Category:   fact.Category,
				Source:     model.SourceExtracted,
				Importance: fact.Importance,
			})
			if err != nil {
				log.Printf("[Knowledge:Process] Error creating knowledge: %v", err)
				continue
			}
			log.Printf("[Knowledge:Process] Created knowledge ID=%s", newKnowledge.ID)
		}
	}

	log.Printf("[Knowledge:Process] Complete")
	return nil
}

// SupersedeKnowledge marks old knowledge as superseded by new one
func (m *DefaultManager) SupersedeKnowledge(ctx context.Context, oldID, newID string) error {
	log.Printf("[Knowledge:Supersede] Starting - OldID=%s, NewID=%s", oldID, newID)

	// Update database
	if err := m.knowledgeRepo.Supersede(oldID, newID); err != nil {
		log.Printf("[Knowledge:Supersede] Error updating DB: %v", err)
		return fmt.Errorf("failed to supersede in db: %w", err)
	}
	log.Printf("[Knowledge:Supersede] Updated DB")

	// Update vector store metadata
	if doc, ok := m.vectorStore.Get(oldID); ok {
		metadata := doc.MetaData
		if metadata == nil {
			metadata = &vector.DocumentMetadata{}
		}
		metadata.IsActive = false
		if err := m.vectorStore.UpdateMetadata(oldID, metadata); err != nil {
			log.Printf("[Knowledge:Supersede] Error updating vector metadata: %v", err)
			return fmt.Errorf("failed to update vector metadata: %w", err)
		}
		log.Printf("[Knowledge:Supersede] Updated vector metadata - IsActive=false")
	} else {
		log.Printf("[Knowledge:Supersede] Old document not found in vector store")
	}

	log.Printf("[Knowledge:Supersede] Complete")
	return nil
}

// GetRecentMemories retrieves recent memories for a session
func (m *DefaultManager) GetRecentMemories(ctx context.Context, sessionID string, limit int) ([]model.Memory, error) {
	return m.memoryRepo.GetRecentBySessionID(sessionID, limit)
}

// truncateStr truncates a string to maxLen characters
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
