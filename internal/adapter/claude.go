package adapter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/google/uuid"
)

const (
	defaultClaudeBaseURL = "https://api.anthropic.com/v1"
	defaultClaudeModel   = "claude-3-5-sonnet-20241022"
	claudeAPIVersion     = "2023-06-01"
)

// ClaudeAdapter implements LLMAdapter for Anthropic Claude API
type ClaudeAdapter struct {
	apiKey      string
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
	client      *http.Client
}

// NewClaudeAdapter creates a new Claude adapter
func NewClaudeAdapter(cfg AdapterConfig) (LLMAdapter, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultClaudeBaseURL
	}

	modelName := cfg.Model
	if modelName == "" {
		modelName = defaultClaudeModel
	}

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	temperature := cfg.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	return &ClaudeAdapter{
		apiKey:      cfg.APIKey,
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		model:       modelName,
		maxTokens:   maxTokens,
		temperature: temperature,
		client:      &http.Client{},
	}, nil
}

// claudeRequest represents a Claude API request
type claudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Messages    []claudeMessage `json:"messages"`
	System      string          `json:"system,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeResponse represents a Claude API response
type claudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// claudeStreamEvent represents a streaming event
type claudeStreamEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
	Delta struct {
		Type string `json:"type,omitempty"`
		Text string `json:"text,omitempty"`
	} `json:"delta,omitempty"`
	Message *claudeResponse `json:"message,omitempty"`
	Usage   *struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
}

func (a *ClaudeAdapter) Chat(ctx context.Context, messages []model.Message) (*model.ChatResponse, error) {
	claudeMessages, systemPrompt := convertToClaudeMessages(messages)

	reqBody := claudeRequest{
		Model:       a.model,
		MaxTokens:   a.maxTokens,
		Messages:    claudeMessages,
		System:      systemPrompt,
		Temperature: a.temperature,
		Stream:      false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", claudeAPIVersion)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var claudeResp claudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	content := ""
	for _, c := range claudeResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return &model.ChatResponse{
		ID: claudeResp.ID,
		Message: model.Message{
			Role:    model.RoleAssistant,
			Content: content,
		},
		Usage: &model.Usage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
	}, nil
}

func (a *ClaudeAdapter) ChatStream(ctx context.Context, messages []model.Message) (<-chan model.StreamChunk, error) {
	claudeMessages, systemPrompt := convertToClaudeMessages(messages)

	reqBody := claudeRequest{
		Model:       a.model,
		MaxTokens:   a.maxTokens,
		Messages:    claudeMessages,
		System:      systemPrompt,
		Temperature: a.temperature,
		Stream:      true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", claudeAPIVersion)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	ch := make(chan model.StreamChunk, 100)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		id := uuid.New().String()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					ch <- model.StreamChunk{ID: id, Done: true}
				}
				return
			}

			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- model.StreamChunk{ID: id, Done: true}
				return
			}

			var event claudeStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "content_block_delta":
				if event.Delta.Type == "text_delta" {
					ch <- model.StreamChunk{
						ID:    id,
						Delta: event.Delta.Text,
						Done:  false,
					}
				}
			case "message_stop":
				ch <- model.StreamChunk{ID: id, Done: true}
				return
			}
		}
	}()

	return ch, nil
}

func (a *ClaudeAdapter) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Claude doesn't have a native embedding API
	// Return an error indicating to use a different embedding provider
	return nil, fmt.Errorf("Claude does not support embeddings, use OpenAI embedding provider instead")
}

func (a *ClaudeAdapter) CountTokens(text string) int {
	// Rough estimation for Claude
	return len(text) / 4
}

func (a *ClaudeAdapter) Name() string {
	return "claude"
}

func (a *ClaudeAdapter) Provider() model.LLMProvider {
	return model.ProviderClaude
}

// convertToClaudeMessages converts model.Message to Claude format
// Also extracts system prompts since Claude handles them separately
func convertToClaudeMessages(messages []model.Message) ([]claudeMessage, string) {
	var claudeMessages []claudeMessage
	var systemPrompt string

	for _, msg := range messages {
		if msg.Role == model.RoleSystem {
			if systemPrompt != "" {
				systemPrompt += "\n\n"
			}
			systemPrompt += msg.Content
		} else {
			role := string(msg.Role)
			// Claude uses "user" and "assistant" roles
			claudeMessages = append(claudeMessages, claudeMessage{
				Role:    role,
				Content: msg.Content,
			})
		}
	}

	return claudeMessages, systemPrompt
}
