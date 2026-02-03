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
	defaultOllamaBaseURL = "http://localhost:11434/v1"
	defaultOllamaModel   = "llama3.2"
)

// OllamaAdapter implements LLMAdapter for Ollama API (OpenAI-compatible)
type OllamaAdapter struct {
	apiKey      string
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
	client      *http.Client
}

// NewOllamaAdapter creates a new Ollama adapter
func NewOllamaAdapter(cfg AdapterConfig) (LLMAdapter, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	}

	modelName := cfg.Model
	if modelName == "" {
		modelName = defaultOllamaModel
	}

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	temperature := cfg.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	// Ollama doesn't require an API key, but we keep the field for compatibility
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = "ollama" // Placeholder for Ollama
	}

	return &OllamaAdapter{
		apiKey:      apiKey,
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		model:       modelName,
		maxTokens:   maxTokens,
		temperature: temperature,
		client:      &http.Client{},
	}, nil
}

// ollamaRequest represents an Ollama chat completion request (OpenAI-compatible)
type ollamaRequest struct {
	Model       string          `json:"model"`
	Messages    []ollamaMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaResponse represents an Ollama chat completion response
type ollamaResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// ollamaStreamResponse represents a streaming response chunk
type ollamaStreamResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Delta struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// ollamaNativeEmbeddingRequest represents Ollama's native embedding request
type ollamaNativeEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaNativeEmbeddingResponse represents Ollama's native embedding response
type ollamaNativeEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (a *OllamaAdapter) Chat(ctx context.Context, messages []model.Message) (*model.ChatResponse, error) {
	reqBody := ollamaRequest{
		Model:       a.model,
		Messages:    convertToOllamaMessages(messages),
		MaxTokens:   a.maxTokens,
		Temperature: a.temperature,
		Stream:      false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if a.apiKey != "" && a.apiKey != "ollama" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(ollamaResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &model.ChatResponse{
		ID: ollamaResp.ID,
		Message: model.Message{
			Role:    model.MessageRole(ollamaResp.Choices[0].Message.Role),
			Content: ollamaResp.Choices[0].Message.Content,
		},
		Usage: &model.Usage{
			PromptTokens:     ollamaResp.Usage.PromptTokens,
			CompletionTokens: ollamaResp.Usage.CompletionTokens,
			TotalTokens:      ollamaResp.Usage.TotalTokens,
		},
	}, nil
}

func (a *OllamaAdapter) ChatStream(ctx context.Context, messages []model.Message) (<-chan model.StreamChunk, error) {
	reqBody := ollamaRequest{
		Model:       a.model,
		Messages:    convertToOllamaMessages(messages),
		MaxTokens:   a.maxTokens,
		Temperature: a.temperature,
		Stream:      true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if a.apiKey != "" && a.apiKey != "ollama" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}
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
			if line == "" {
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- model.StreamChunk{ID: id, Done: true}
				return
			}

			var streamResp ollamaStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) > 0 {
				chunk := model.StreamChunk{
					ID:    id,
					Delta: streamResp.Choices[0].Delta.Content,
					Done:  streamResp.Choices[0].FinishReason != "",
				}

				if streamResp.Usage != nil {
					chunk.Usage = &model.Usage{
						PromptTokens:     streamResp.Usage.PromptTokens,
						CompletionTokens: streamResp.Usage.CompletionTokens,
						TotalTokens:      streamResp.Usage.TotalTokens,
					}
				}

				ch <- chunk
			}
		}
	}()

	return ch, nil
}

func (a *OllamaAdapter) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Use the configured model for embeddings
	embeddingModel := a.model
	if embeddingModel == "" {
		embeddingModel = "nomic-embed-text"
	}

	reqBody := ollamaNativeEmbeddingRequest{
		Model:  embeddingModel,
		Prompt: text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use Ollama's native embedding API endpoint
	// baseURL is like "http://localhost:11434/v1", we need "http://localhost:11434/api/embeddings"
	baseURL := strings.TrimSuffix(a.baseURL, "/v1")
	baseURL = strings.TrimSuffix(baseURL, "/")
	embeddingURL := baseURL + "/api/embeddings"

	req, err := http.NewRequestWithContext(ctx, "POST", embeddingURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var embResp ollamaNativeEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embResp.Embedding) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	return embResp.Embedding, nil
}

func (a *OllamaAdapter) CountTokens(text string) int {
	// Rough estimation: ~4 characters per token
	return len(text) / 4
}

func (a *OllamaAdapter) Name() string {
	return "ollama"
}

func (a *OllamaAdapter) Provider() model.LLMProvider {
	return model.ProviderOllama
}

func convertToOllamaMessages(messages []model.Message) []ollamaMessage {
	result := make([]ollamaMessage, len(messages))
	for i, msg := range messages {
		result[i] = ollamaMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}
	return result
}
