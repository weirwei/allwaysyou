package service

import (
	"fmt"
	"time"

	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/repository"
	"github.com/google/uuid"
)

// ModelConfigService handles ModelConfig business logic
type ModelConfigService struct {
	repo         *repository.ModelConfigRepository
	providerRepo *repository.ProviderRepository
}

// NewModelConfigService creates a new model config service
func NewModelConfigService(repo *repository.ModelConfigRepository, providerRepo *repository.ProviderRepository) *ModelConfigService {
	return &ModelConfigService{
		repo:         repo,
		providerRepo: providerRepo,
	}
}

// Create creates a new model config
func (s *ModelConfigService) Create(req *model.CreateModelConfigRequest) (*model.ModelConfig, error) {
	// Verify provider exists
	provider, err := s.providerRepo.GetByID(req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	if provider == nil {
		return nil, fmt.Errorf("provider not found")
	}

	config := &model.ModelConfig{
		ID:          uuid.New().String(),
		ProviderID:  req.ProviderID,
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		ConfigType:  req.ConfigType,
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
	if config.ConfigType == "" {
		config.ConfigType = model.ConfigTypeChat
	}

	if err := s.repo.Create(config); err != nil {
		return nil, fmt.Errorf("failed to create model config: %w", err)
	}

	// If this is set as default, update others
	if config.IsDefault {
		if err := s.repo.SetDefault(config.ID); err != nil {
			return nil, fmt.Errorf("failed to set default: %w", err)
		}
	}

	// Fetch with provider data
	return s.repo.GetByID(config.ID)
}

// GetByID retrieves a model config by ID
func (s *ModelConfigService) GetByID(id string) (*model.ModelConfig, error) {
	return s.repo.GetByID(id)
}

// GetAll retrieves all model configs
func (s *ModelConfigService) GetAll() ([]model.ModelConfig, error) {
	return s.repo.GetAll()
}

// GetByProvider retrieves all model configs for a specific provider
func (s *ModelConfigService) GetByProvider(providerID string) ([]model.ModelConfig, error) {
	return s.repo.GetByProvider(providerID)
}

// GetByType retrieves all model configs of a specific type
func (s *ModelConfigService) GetByType(configType model.ConfigType) ([]model.ModelConfig, error) {
	return s.repo.GetByType(configType)
}

// GetDefaultByType retrieves the default model config for a specific type
func (s *ModelConfigService) GetDefaultByType(configType model.ConfigType) (*model.ModelConfig, error) {
	return s.repo.GetDefaultByType(configType)
}

// Update updates a model config
func (s *ModelConfigService) Update(id string, req *model.UpdateModelConfigRequest) (*model.ModelConfig, error) {
	config, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("model config not found")
	}

	// Update fields
	if req.Model != "" {
		config.Model = req.Model
	}
	if req.MaxTokens != nil {
		config.MaxTokens = *req.MaxTokens
	}
	if req.Temperature != nil {
		config.Temperature = *req.Temperature
	}
	if req.ConfigType != "" {
		config.ConfigType = req.ConfigType
	}
	if req.IsDefault != nil && *req.IsDefault {
		config.IsDefault = true
	}

	config.UpdatedAt = time.Now()

	if err := s.repo.Update(config); err != nil {
		return nil, fmt.Errorf("failed to update model config: %w", err)
	}

	// If this is set as default, update others
	if config.IsDefault {
		if err := s.repo.SetDefault(config.ID); err != nil {
			return nil, fmt.Errorf("failed to set default: %w", err)
		}
	}

	return s.repo.GetByID(config.ID)
}

// Delete deletes a model config
func (s *ModelConfigService) Delete(id string) error {
	return s.repo.Delete(id)
}

// SetDefault sets a model config as default for its type
func (s *ModelConfigService) SetDefault(id string) error {
	return s.repo.SetDefault(id)
}
