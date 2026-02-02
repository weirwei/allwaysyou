package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/allwaysyou/llm-agent/internal/config"
	"github.com/allwaysyou/llm-agent/internal/domain/entity"
)

type Provider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func New(cfg config.ProviderConfig) *Provider {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &Provider{
		apiKey:  cfg.APIKey,
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

type chatCompletionRequest struct {
	Model       string          `json:"model"`
	Messages    []chatMessage   `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	ID      string   `json:"id"`
	Choices []choice `json:"choices"`
	Usage   *usage   `json:"usage,omitempty"`
}

type choice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	Delta        chatMessage `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingResponse struct {
	Data []embeddingData `json:"data"`
}

type embeddingData struct {
	Embedding []float32 `json:"embedding"`
}

func (p *Provider) Chat(ctx context.Context, req *entity.ChatRequest) (*entity.ChatResponse, error) {
	messages := make([]chatMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = chatMessage{Role: m.Role, Content: m.Content}
	}

	apiReq := chatCompletionRequest{
		Model:       p.model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      false,
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	result := &entity.ChatResponse{
		ID:           apiResp.ID,
		Content:      apiResp.Choices[0].Message.Content,
		Role:         apiResp.Choices[0].Message.Role,
		FinishReason: apiResp.Choices[0].FinishReason,
	}

	if apiResp.Usage != nil {
		result.Usage = &entity.TokenUsage{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:      apiResp.Usage.TotalTokens,
		}
	}

	return result, nil
}

func (p *Provider) ChatStream(ctx context.Context, req *entity.ChatRequest) (<-chan *entity.ChatChunk, error) {
	messages := make([]chatMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = chatMessage{Role: m.Role, Content: m.Content}
	}

	apiReq := chatCompletionRequest{
		Model:       p.model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      true,
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan *entity.ChatChunk)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var streamResp chatCompletionResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				ch <- &entity.ChatChunk{Error: fmt.Errorf("decode stream: %w", err)}
				return
			}

			if len(streamResp.Choices) > 0 {
				ch <- &entity.ChatChunk{
					ID:           streamResp.ID,
					Content:      streamResp.Choices[0].Delta.Content,
					FinishReason: streamResp.Choices[0].FinishReason,
				}
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- &entity.ChatChunk{Error: fmt.Errorf("scan stream: %w", err)}
		}
	}()

	return ch, nil
}

func (p *Provider) Embedding(ctx context.Context, text string) ([]float32, error) {
	apiReq := embeddingRequest{
		Model: "text-embedding-3-small",
		Input: text,
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(apiResp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings in response")
	}

	return apiResp.Data[0].Embedding, nil
}

func (p *Provider) GetModelInfo() entity.ModelInfo {
	return entity.ModelInfo{
		Provider:    "openai",
		Model:       p.model,
		MaxTokens:   4096,
		ContextSize: 128000,
	}
}
