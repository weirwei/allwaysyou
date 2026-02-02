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
	azureAPIVersion = "2024-02-01"
)

// AzureAdapter implements LLMAdapter for Azure OpenAI API
type AzureAdapter struct {
	apiKey       string
	baseURL      string // Azure endpoint: https://{resource}.openai.azure.com
	deploymentID string // Azure deployment name
	maxTokens    int
	temperature  float64
	client       *http.Client
}

// NewAzureAdapter creates a new Azure OpenAI adapter
func NewAzureAdapter(cfg AdapterConfig) (LLMAdapter, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("Azure endpoint (base_url) is required")
	}

	if cfg.Model == "" {
		return nil, fmt.Errorf("Azure deployment ID (model) is required")
	}

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	temperature := cfg.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	return &AzureAdapter{
		apiKey:       cfg.APIKey,
		baseURL:      strings.TrimSuffix(cfg.BaseURL, "/"),
		deploymentID: cfg.Model,
		maxTokens:    maxTokens,
		temperature:  temperature,
		client:       &http.Client{},
	}, nil
}

func (a *AzureAdapter) buildURL(endpoint string) string {
	return fmt.Sprintf("%s/openai/deployments/%s/%s?api-version=%s",
		a.baseURL, a.deploymentID, endpoint, azureAPIVersion)
}

func (a *AzureAdapter) Chat(ctx context.Context, messages []model.Message) (*model.ChatResponse, error) {
	reqBody := openaiRequest{
		Model:       a.deploymentID,
		Messages:    convertToOpenAIMessages(messages),
		MaxTokens:   a.maxTokens,
		Temperature: a.temperature,
		Stream:      false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.buildURL("chat/completions"), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", a.apiKey)

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

func (a *AzureAdapter) ChatStream(ctx context.Context, messages []model.Message) (<-chan model.StreamChunk, error) {
	reqBody := openaiRequest{
		Model:       a.deploymentID,
		Messages:    convertToOpenAIMessages(messages),
		MaxTokens:   a.maxTokens,
		Temperature: a.temperature,
		Stream:      true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.buildURL("chat/completions"), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", a.apiKey)
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

func (a *AzureAdapter) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	reqBody := openaiEmbeddingRequest{
		Model: a.deploymentID,
		Input: text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.buildURL("embeddings"), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", a.apiKey)

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

func (a *AzureAdapter) CountTokens(text string) int {
	return len(text) / 4
}

func (a *AzureAdapter) Name() string {
	return "azure"
}

func (a *AzureAdapter) Provider() model.LLMProvider {
	return model.ProviderAzure
}
