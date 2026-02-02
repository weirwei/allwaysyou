package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/allwaysyou/llm-agent/internal/adapter"
	"github.com/allwaysyou/llm-agent/internal/config"
	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/pkg/constants"
)

// Processor handles LLM-based memory processing
type Processor struct {
	config config.MemoryConfig
}

// NewProcessor creates a new memory processor
func NewProcessor(cfg config.MemoryConfig) *Processor {
	return &Processor{config: cfg}
}

// ExtractFacts extracts key facts from a conversation using LLM
func (p *Processor) ExtractFacts(ctx context.Context, userMsg, assistantResp string, llm adapter.LLMAdapter) ([]ExtractedFact, error) {
	log.Printf("[Processor:ExtractFacts] Starting - UserMsg='%.50s...', AssistantResp='%.50s...'",
		userMsg, assistantResp)

	messages := []model.Message{
		{
			Role:    model.RoleUser,
			Content: fmt.Sprintf(constants.FactExtractionPrompt, userMsg, assistantResp),
		},
	}

	log.Printf("[Processor:ExtractFacts] Calling LLM...")
	resp, err := llm.Chat(ctx, messages)
	if err != nil {
		log.Printf("[Processor:ExtractFacts] LLM call failed: %v", err)
		return nil, fmt.Errorf("LLM extraction failed: %w", err)
	}

	// Parse JSON response
	content := resp.Message.Content
	log.Printf("[Processor:ExtractFacts] LLM response: %s", content)

	// Extract JSON from response (handle markdown code blocks)
	content = extractJSON(content)
	log.Printf("[Processor:ExtractFacts] Extracted JSON: %s", content)

	var facts []ExtractedFact
	if err := json.Unmarshal([]byte(content), &facts); err != nil {
		log.Printf("[Processor:ExtractFacts] JSON parse error: %v", err)
		// If parsing fails, return empty
		return nil, nil
	}

	// Convert category strings to proper type
	for i := range facts {
		facts[i].Category = normalizeCategory(string(facts[i].Category))
		if facts[i].Importance <= 0 || facts[i].Importance > 1 {
			facts[i].Importance = p.config.DefaultImportance
		}
		log.Printf("[Processor:ExtractFacts] Fact %d: Category=%s, Importance=%.2f, Content='%s'",
			i, facts[i].Category, facts[i].Importance, facts[i].Content)
	}

	log.Printf("[Processor:ExtractFacts] Extracted %d facts", len(facts))
	return facts, nil
}

// DetectConflict checks if a new fact conflicts with existing knowledge
func (p *Processor) DetectConflict(ctx context.Context, newFact ExtractedFact, existingKnowledge []model.KnowledgeSearchResult, llm adapter.LLMAdapter) (*ConflictResult, error) {
	log.Printf("[Processor:DetectConflict] Starting - NewFact='%s', ExistingCount=%d",
		newFact.Content, len(existingKnowledge))

	if len(existingKnowledge) == 0 {
		log.Printf("[Processor:DetectConflict] No existing knowledge -> CREATE")
		return &ConflictResult{HasConflict: false, Action: ActionCreate}, nil
	}

	// Find knowledge with high similarity
	var candidates []string
	var candidateIDs []string
	for _, k := range existingKnowledge {
		log.Printf("[Processor:DetectConflict] Checking knowledge - ID=%s, Score=%.3f, IsActive=%v, Content='%s'",
			k.Knowledge.ID, k.Score, k.Knowledge.IsActive(), k.Knowledge.Content)
		if k.Score > p.config.ConflictDetectionThreshold && k.Knowledge.IsActive() {
			candidates = append(candidates, k.Knowledge.Content)
			candidateIDs = append(candidateIDs, k.Knowledge.ID)
		}
	}

	if len(candidates) == 0 {
		log.Printf("[Processor:DetectConflict] No high-similarity candidates (>%.2f) -> CREATE", p.config.ConflictDetectionThreshold)
		return &ConflictResult{HasConflict: false, Action: ActionCreate}, nil
	}
	log.Printf("[Processor:DetectConflict] Found %d high-similarity candidates", len(candidates))

	// Use LLM to detect conflict
	existingStr := ""
	for i, c := range candidates {
		existingStr += fmt.Sprintf("%d. %s\n", i, c)
	}

	messages := []model.Message{
		{
			Role:    model.RoleUser,
			Content: fmt.Sprintf(constants.ConflictDetectionPrompt, newFact.Content, existingStr),
		},
	}

	log.Printf("[Processor:DetectConflict] Calling LLM to detect conflict...")
	resp, err := llm.Chat(ctx, messages)
	if err != nil {
		log.Printf("[Processor:DetectConflict] LLM call failed: %v -> default to CREATE", err)
		// On error, default to create
		return &ConflictResult{HasConflict: false, Action: ActionCreate}, nil
	}

	log.Printf("[Processor:DetectConflict] LLM response: %s", resp.Message.Content)
	content := extractJSON(resp.Message.Content)
	log.Printf("[Processor:DetectConflict] Extracted JSON: %s", content)

	var result struct {
		IsDuplicate   bool `json:"is_duplicate"`
		IsConflict    bool `json:"is_conflict"`
		ConflictIndex int  `json:"conflict_index"`
	}
	result.ConflictIndex = -1

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		log.Printf("[Processor:DetectConflict] JSON parse error: %v -> default to CREATE", err)
		return &ConflictResult{HasConflict: false, Action: ActionCreate}, nil
	}

	log.Printf("[Processor:DetectConflict] Parsed result - IsDuplicate=%v, IsConflict=%v, ConflictIndex=%d",
		result.IsDuplicate, result.IsConflict, result.ConflictIndex)

	if result.IsDuplicate {
		log.Printf("[Processor:DetectConflict] Result: SKIP (duplicate)")
		return &ConflictResult{HasConflict: false, Action: ActionSkip}, nil
	}

	if result.IsConflict && result.ConflictIndex >= 0 && result.ConflictIndex < len(candidateIDs) {
		log.Printf("[Processor:DetectConflict] Result: UPDATE - ConflictingID=%s, OldContent='%s'",
			candidateIDs[result.ConflictIndex], candidates[result.ConflictIndex])
		return &ConflictResult{
			HasConflict:   true,
			ConflictingID: candidateIDs[result.ConflictIndex],
			OldContent:    candidates[result.ConflictIndex],
			Action:        ActionUpdate,
		}, nil
	}

	log.Printf("[Processor:DetectConflict] Result: CREATE (no conflict)")
	return &ConflictResult{HasConflict: false, Action: ActionCreate}, nil
}

// extractJSON extracts JSON from a string that might contain markdown code blocks
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// Remove markdown code blocks
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}

	return strings.TrimSpace(s)
}

// normalizeCategory normalizes category string to KnowledgeCategory
func normalizeCategory(s string) model.KnowledgeCategory {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "personal_info", "personalinfo", "personal":
		return model.CategoryPersonalInfo
	case "preference", "pref":
		return model.CategoryPreference
	case "fact", "facts":
		return model.CategoryFact
	case "event", "events":
		return model.CategoryEvent
	default:
		return model.CategoryFact
	}
}
