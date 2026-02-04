package service

import (
	"fmt"
	"time"

	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/pkg/crypto"
	"github.com/allwaysyou/llm-agent/internal/repository"
	"github.com/google/uuid"
)

// ProviderService handles Provider business logic
type ProviderService struct {
	repo            *repository.ProviderRepository
	modelConfigRepo *repository.ModelConfigRepository
	encryptor       *crypto.Encryptor
}

// NewProviderService creates a new provider service
func NewProviderService(repo *repository.ProviderRepository, modelConfigRepo *repository.ModelConfigRepository, encryptor *crypto.Encryptor) *ProviderService {
	return &ProviderService{
		repo:            repo,
		modelConfigRepo: modelConfigRepo,
		encryptor:       encryptor,
	}
}

// Create creates a new provider
func (s *ProviderService) Create(req *model.CreateProviderRequest) (*model.Provider, error) {
	// Encrypt API key
	encryptedKey, err := s.encryptor.Encrypt(req.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	provider := &model.Provider{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Type:      req.Type,
		APIKey:    encryptedKey,
		BaseURL:   req.BaseURL,
		Enabled:   enabled,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(provider); err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return provider, nil
}

// GetByID retrieves a provider by ID
func (s *ProviderService) GetByID(id string) (*model.Provider, error) {
	return s.repo.GetByID(id)
}

// GetByIDWithModels retrieves a provider by ID with its models
func (s *ProviderService) GetByIDWithModels(id string) (*model.ProviderResponse, error) {
	provider, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, nil
	}

	models, err := s.modelConfigRepo.GetByProvider(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get models: %w", err)
	}

	resp := provider.ToResponse()
	resp.Models = make([]model.ModelConfigResponse, len(models))
	for i, m := range models {
		resp.Models[i] = m.ToResponse()
	}

	return &resp, nil
}

// GetAll retrieves all providers
func (s *ProviderService) GetAll() ([]model.Provider, error) {
	return s.repo.GetAll()
}

// GetEnabled retrieves all enabled providers
func (s *ProviderService) GetEnabled() ([]model.Provider, error) {
	return s.repo.GetEnabled()
}

// Update updates a provider
func (s *ProviderService) Update(id string, req *model.UpdateProviderRequest) (*model.Provider, error) {
	provider, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, fmt.Errorf("provider not found")
	}

	// Update fields
	if req.Name != "" {
		provider.Name = req.Name
	}
	if req.Type != "" {
		provider.Type = req.Type
	}
	if req.APIKey != "" {
		encryptedKey, err := s.encryptor.Encrypt(req.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt API key: %w", err)
		}
		provider.APIKey = encryptedKey
	}
	if req.BaseURL != "" {
		provider.BaseURL = req.BaseURL
	}
	if req.Enabled != nil {
		provider.Enabled = *req.Enabled
	}

	provider.UpdatedAt = time.Now()

	if err := s.repo.Update(provider); err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	return provider, nil
}

// Delete deletes a provider and all its models
func (s *ProviderService) Delete(id string) error {
	// First delete all models associated with this provider
	if err := s.modelConfigRepo.DeleteByProvider(id); err != nil {
		return fmt.Errorf("failed to delete provider models: %w", err)
	}
	return s.repo.Delete(id)
}

// DecryptAPIKey decrypts the API key for a provider
func (s *ProviderService) DecryptAPIKey(encryptedKey string) (string, error) {
	return s.encryptor.Decrypt(encryptedKey)
}

// GetDecryptedAPIKey gets the decrypted API key for a provider
func (s *ProviderService) GetDecryptedAPIKey(id string) (string, error) {
	provider, err := s.repo.GetByID(id)
	if err != nil {
		return "", err
	}
	if provider == nil {
		return "", fmt.Errorf("provider not found")
	}
	return s.encryptor.Decrypt(provider.APIKey)
}
