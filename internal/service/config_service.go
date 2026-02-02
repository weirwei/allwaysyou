package service

import (
	"fmt"
	"time"

	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/pkg/crypto"
	"github.com/allwaysyou/llm-agent/internal/repository"
	"github.com/google/uuid"
)

// ConfigService handles LLM config business logic
type ConfigService struct {
	repo      *repository.ConfigRepository
	encryptor *crypto.Encryptor
}

// NewConfigService creates a new config service
func NewConfigService(repo *repository.ConfigRepository, encryptor *crypto.Encryptor) *ConfigService {
	return &ConfigService{
		repo:      repo,
		encryptor: encryptor,
	}
}

// Create creates a new LLM config
func (s *ConfigService) Create(req *model.CreateLLMConfigRequest) (*model.LLMConfig, error) {
	// Encrypt API key
	encryptedKey, err := s.encryptor.Encrypt(req.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}

	config := &model.LLMConfig{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Provider:    req.Provider,
		APIKey:      encryptedKey,
		BaseURL:     req.BaseURL,
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		IsDefault:   req.IsDefault,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set defaults
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}

	if err := s.repo.Create(config); err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	// If this is set as default, update others
	if config.IsDefault {
		if err := s.repo.SetDefault(config.ID); err != nil {
			return nil, fmt.Errorf("failed to set default: %w", err)
		}
	}

	return config, nil
}

// GetByID retrieves a config by ID
func (s *ConfigService) GetByID(id string) (*model.LLMConfig, error) {
	return s.repo.GetByID(id)
}

// GetAll retrieves all configs
func (s *ConfigService) GetAll() ([]model.LLMConfig, error) {
	return s.repo.GetAll()
}

// GetDefault retrieves the default config
func (s *ConfigService) GetDefault() (*model.LLMConfig, error) {
	return s.repo.GetDefault()
}

// Update updates a config
func (s *ConfigService) Update(id string, req *model.UpdateLLMConfigRequest) (*model.LLMConfig, error) {
	config, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("config not found")
	}

	// Update fields
	if req.Name != "" {
		config.Name = req.Name
	}
	if req.Provider != "" {
		config.Provider = req.Provider
	}
	if req.APIKey != "" {
		encryptedKey, err := s.encryptor.Encrypt(req.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt API key: %w", err)
		}
		config.APIKey = encryptedKey
	}
	if req.BaseURL != "" {
		config.BaseURL = req.BaseURL
	}
	if req.Model != "" {
		config.Model = req.Model
	}
	if req.MaxTokens != nil {
		config.MaxTokens = *req.MaxTokens
	}
	if req.Temperature != nil {
		config.Temperature = *req.Temperature
	}
	if req.IsDefault != nil && *req.IsDefault {
		config.IsDefault = true
	}

	config.UpdatedAt = time.Now()

	if err := s.repo.Update(config); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	// If this is set as default, update others
	if config.IsDefault {
		if err := s.repo.SetDefault(config.ID); err != nil {
			return nil, fmt.Errorf("failed to set default: %w", err)
		}
	}

	return config, nil
}

// Delete deletes a config
func (s *ConfigService) Delete(id string) error {
	return s.repo.Delete(id)
}

// DecryptAPIKey decrypts the API key for a config
func (s *ConfigService) DecryptAPIKey(encryptedKey string) (string, error) {
	return s.encryptor.Decrypt(encryptedKey)
}
