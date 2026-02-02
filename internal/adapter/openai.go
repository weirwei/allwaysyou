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
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
	defaultOpenAIModel   = "gpt-4o-mini"
)

// OpenAIAdapter implements LLMAdapter for OpenAI API
type OpenAIAdapter struct {
	apiKey      string
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
	client      *http.Client
}

// NewOpenAIAdapter creates a new OpenAI adapter
func NewOpenAIAdapter(cfg AdapterConfig) (LLMAdapter, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}

	modelName := cfg.Model
	if modelName == "" {
		modelName = defaultOpenAIModel
	}

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	temperature := cfg.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	return &OpenAIAdapter{
		apiKey:      cfg.APIKey,
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		model:       modelName,
		maxTokens:   maxTokens,
		temperature: temperature,
		client:      &http.Client{},
	}, nil
}

// openaiRequest represents an OpenAI chat completion request
type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiResponse represents an OpenAI chat completion response
type openaiResponse struct {
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

// openaiStreamResponse represents a streaming response chunk
type openaiStreamResponse struct {
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

// openaiEmbeddingRequest represents an embedding request
type openaiEmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// openaiEmbeddingResponse represents an embedding response
type openaiEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (a *OpenAIAdapter) Chat(ctx context.Context, messages []model.Message) (*model.ChatResponse, error) {
	reqBody := openaiRequest{
		Model:       a.model,
		Messages:    convertToOpenAIMessages(messages),
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
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var openaiResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &model.ChatResponse{
		ID: openaiResp.ID,
		Message: model.Message{
			Role:    model.MessageRole(openaiResp.Choices[0].Message.Role),
			Content: openaiResp.Choices[0].Message.Content,
		},
		Usage: &model.Usage{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		},
	}, nil
}

func (a *OpenAIAdapter) ChatStream(ctx context.Context, messages []model.Message) (<-chan model.StreamChunk, error) {
	reqBody := openaiRequest{
		Model:       a.model,
		Messages:    convertToOpenAIMessages(messages),
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
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
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

			var streamResp openaiStreamResponse
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

func (a *OpenAIAdapter) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	reqBody := openaiEmbeddingRequest{
		Model: "text-embedding-3-small",
		Input: text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var embResp openaiEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	return embResp.Data[0].Embedding, nil
}

func (a *OpenAIAdapter) CountTokens(text string) int {
	// Rough estimation: ~4 characters per token for English
	// This is a simplified estimation; for production, use tiktoken
	return len(text) / 4
}

func (a *OpenAIAdapter) Name() string {
	return "openai"
}

func (a *OpenAIAdapter) Provider() model.LLMProvider {
	return model.ProviderOpenAI
}

func convertToOpenAIMessages(messages []model.Message) []openaiMessage {
	result := make([]openaiMessage, len(messages))
	for i, msg := range messages {
		result[i] = openaiMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}
	return result
}
