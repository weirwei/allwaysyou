package model

import "time"

// LLMProvider represents the LLM service provider
type LLMProvider string

const (
	ProviderOpenAI  LLMProvider = "openai"
	ProviderClaude  LLMProvider = "claude"
	ProviderAzure   LLMProvider = "azure"
	ProviderCustom  LLMProvider = "custom"
)

// LLMConfig represents a configuration for an LLM provider
type LLMConfig struct {
	ID          string      `json:"id" gorm:"primaryKey"`
	Name        string      `json:"name" gorm:"not null"`
	Provider    LLMProvider `json:"provider" gorm:"not null"`
	APIKey      string      `json:"api_key,omitempty" gorm:"not null"` // Encrypted
	BaseURL     string      `json:"base_url,omitempty"`
	Model       string      `json:"model" gorm:"not null"`
	MaxTokens   int         `json:"max_tokens" gorm:"default:4096"`
	Temperature float64     `json:"temperature" gorm:"default:0.7"`
	IsDefault   bool        `json:"is_default" gorm:"default:false"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// CreateLLMConfigRequest represents the request to create a new LLM config
type CreateLLMConfigRequest struct {
	Name        string      `json:"name" binding:"required"`
	Provider    LLMProvider `json:"provider" binding:"required"`
	APIKey      string      `json:"api_key" binding:"required"`
	BaseURL     string      `json:"base_url"`
	Model       string      `json:"model" binding:"required"`
	MaxTokens   int         `json:"max_tokens"`
	Temperature float64     `json:"temperature"`
	IsDefault   bool        `json:"is_default"`
}

// UpdateLLMConfigRequest represents the request to update an LLM config
type UpdateLLMConfigRequest struct {
	Name        string      `json:"name"`
	Provider    LLMProvider `json:"provider"`
	APIKey      string      `json:"api_key"`
	BaseURL     string      `json:"base_url"`
	Model       string      `json:"model"`
	MaxTokens   *int        `json:"max_tokens"`
	Temperature *float64    `json:"temperature"`
	IsDefault   *bool       `json:"is_default"`
}

// LLMConfigResponse represents the response for an LLM config (without sensitive data)
type LLMConfigResponse struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Provider    LLMProvider `json:"provider"`
	BaseURL     string      `json:"base_url,omitempty"`
	Model       string      `json:"model"`
	MaxTokens   int         `json:"max_tokens"`
	Temperature float64     `json:"temperature"`
	IsDefault   bool        `json:"is_default"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// ToResponse converts LLMConfig to LLMConfigResponse (excludes API key)
func (c *LLMConfig) ToResponse() LLMConfigResponse {
	return LLMConfigResponse{
		ID:          c.ID,
		Name:        c.Name,
		Provider:    c.Provider,
		BaseURL:     c.BaseURL,
		Model:       c.Model,
		MaxTokens:   c.MaxTokens,
		Temperature: c.Temperature,
		IsDefault:   c.IsDefault,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}
