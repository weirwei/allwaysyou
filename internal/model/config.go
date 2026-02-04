package model

import "time"

// ProviderType represents the LLM service provider type
type ProviderType string

const (
	ProviderTypeOpenAI ProviderType = "openai"
	ProviderTypeClaude ProviderType = "claude"
	ProviderTypeAzure  ProviderType = "azure"
	ProviderTypeOllama ProviderType = "ollama"
	ProviderTypeCustom ProviderType = "custom"
)

// ConfigType represents the purpose of an LLM configuration
type ConfigType string

const (
	ConfigTypeChat      ConfigType = "chat"      // For chat/conversation
	ConfigTypeSummarize ConfigType = "summarize" // For memory summarization
	ConfigTypeEmbedding ConfigType = "embedding" // For vector embeddings
)

// Provider represents an LLM service provider
type Provider struct {
	ID        string       `json:"id" gorm:"primaryKey"`
	Name      string       `json:"name" gorm:"not null"`
	Type      ProviderType `json:"type" gorm:"not null"`
	APIKey    string       `json:"-" gorm:"column:api_key"` // Encrypted, hidden from JSON
	BaseURL   string       `json:"base_url"`
	Enabled   bool         `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// ModelConfig represents a model configuration associated with a provider
type ModelConfig struct {
	ID          string     `json:"id" gorm:"primaryKey"`
	ProviderID  string     `json:"provider_id" gorm:"not null;index"`
	Model       string     `json:"model" gorm:"not null"`
	MaxTokens   int        `json:"max_tokens" gorm:"default:4096"`
	Temperature float64    `json:"temperature" gorm:"default:0.7"`
	ConfigType  ConfigType `json:"config_type" gorm:"default:chat"`
	IsDefault   bool       `json:"is_default" gorm:"default:false"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	Provider *Provider `json:"provider,omitempty" gorm:"foreignKey:ProviderID"`
}

// CreateProviderRequest represents the request to create a new provider
type CreateProviderRequest struct {
	Name    string       `json:"name" binding:"required"`
	Type    ProviderType `json:"type" binding:"required"`
	APIKey  string       `json:"api_key" binding:"required"`
	BaseURL string       `json:"base_url"`
	Enabled *bool        `json:"enabled"`
}

// UpdateProviderRequest represents the request to update a provider
type UpdateProviderRequest struct {
	Name    string       `json:"name"`
	Type    ProviderType `json:"type"`
	APIKey  string       `json:"api_key"`
	BaseURL string       `json:"base_url"`
	Enabled *bool        `json:"enabled"`
}

// ProviderResponse represents the response for a provider (without sensitive data)
type ProviderResponse struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Type      ProviderType `json:"type"`
	BaseURL   string       `json:"base_url"`
	Enabled   bool         `json:"enabled"`
	HasAPIKey bool         `json:"has_api_key"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	Models    []ModelConfigResponse `json:"models,omitempty"`
}

// ToResponse converts Provider to ProviderResponse (excludes API key)
func (p *Provider) ToResponse() ProviderResponse {
	return ProviderResponse{
		ID:        p.ID,
		Name:      p.Name,
		Type:      p.Type,
		BaseURL:   p.BaseURL,
		Enabled:   p.Enabled,
		HasAPIKey: p.APIKey != "",
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// CreateModelConfigRequest represents the request to create a new model config
type CreateModelConfigRequest struct {
	ProviderID  string     `json:"provider_id" binding:"required"`
	Model       string     `json:"model" binding:"required"`
	MaxTokens   int        `json:"max_tokens"`
	Temperature float64    `json:"temperature"`
	ConfigType  ConfigType `json:"config_type"`
	IsDefault   bool       `json:"is_default"`
}

// UpdateModelConfigRequest represents the request to update a model config
type UpdateModelConfigRequest struct {
	Model       string     `json:"model"`
	MaxTokens   *int       `json:"max_tokens"`
	Temperature *float64   `json:"temperature"`
	ConfigType  ConfigType `json:"config_type"`
	IsDefault   *bool      `json:"is_default"`
}

// ModelConfigResponse represents the response for a model config
type ModelConfigResponse struct {
	ID          string            `json:"id"`
	ProviderID  string            `json:"provider_id"`
	Model       string            `json:"model"`
	MaxTokens   int               `json:"max_tokens"`
	Temperature float64           `json:"temperature"`
	ConfigType  ConfigType        `json:"config_type"`
	IsDefault   bool              `json:"is_default"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Provider    *ProviderResponse `json:"provider,omitempty"`
}

// ToResponse converts ModelConfig to ModelConfigResponse
func (m *ModelConfig) ToResponse() ModelConfigResponse {
	resp := ModelConfigResponse{
		ID:          m.ID,
		ProviderID:  m.ProviderID,
		Model:       m.Model,
		MaxTokens:   m.MaxTokens,
		Temperature: m.Temperature,
		ConfigType:  m.ConfigType,
		IsDefault:   m.IsDefault,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	if m.Provider != nil {
		providerResp := m.Provider.ToResponse()
		resp.Provider = &providerResp
	}
	return resp
}

// ---- Backward compatibility aliases ----
// These are kept for migration purposes and will be removed later

// LLMProvider is an alias for ProviderType (deprecated)
type LLMProvider = ProviderType

const (
	ProviderOpenAI  = ProviderTypeOpenAI
	ProviderClaude  = ProviderTypeClaude
	ProviderAzure   = ProviderTypeAzure
	ProviderOllama  = ProviderTypeOllama
	ProviderCustom  = ProviderTypeCustom
)

// LLMConfig is kept for backward compatibility during migration
// This will be removed after migration is complete
type LLMConfig struct {
	ID          string       `json:"id" gorm:"primaryKey"`
	Name        string       `json:"name" gorm:"not null"`
	Provider    ProviderType `json:"provider" gorm:"not null"`
	APIKey      string       `json:"api_key,omitempty" gorm:"not null"` // Encrypted
	BaseURL     string       `json:"base_url,omitempty"`
	Model       string       `json:"model" gorm:"not null"`
	MaxTokens   int          `json:"max_tokens" gorm:"default:4096"`
	Temperature float64      `json:"temperature" gorm:"default:0.7"`
	IsDefault   bool         `json:"is_default" gorm:"default:false"`
	ConfigType  ConfigType   `json:"config_type" gorm:"default:chat"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// CreateLLMConfigRequest is kept for backward compatibility
type CreateLLMConfigRequest struct {
	Name        string       `json:"name" binding:"required"`
	Provider    ProviderType `json:"provider" binding:"required"`
	APIKey      string       `json:"api_key" binding:"required"`
	BaseURL     string       `json:"base_url"`
	Model       string       `json:"model" binding:"required"`
	MaxTokens   int          `json:"max_tokens"`
	Temperature float64      `json:"temperature"`
	IsDefault   bool         `json:"is_default"`
	ConfigType  ConfigType   `json:"config_type"`
}

// UpdateLLMConfigRequest is kept for backward compatibility
type UpdateLLMConfigRequest struct {
	Name        string       `json:"name"`
	Provider    ProviderType `json:"provider"`
	APIKey      string       `json:"api_key"`
	BaseURL     string       `json:"base_url"`
	Model       string       `json:"model"`
	MaxTokens   *int         `json:"max_tokens"`
	Temperature *float64     `json:"temperature"`
	IsDefault   *bool        `json:"is_default"`
	ConfigType  ConfigType   `json:"config_type"`
}

// LLMConfigResponse is kept for backward compatibility
type LLMConfigResponse struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Provider    ProviderType `json:"provider"`
	BaseURL     string       `json:"base_url,omitempty"`
	Model       string       `json:"model"`
	MaxTokens   int          `json:"max_tokens"`
	Temperature float64      `json:"temperature"`
	IsDefault   bool         `json:"is_default"`
	ConfigType  ConfigType   `json:"config_type"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
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
		ConfigType:  c.ConfigType,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}
