package repository

import (
	"errors"

	"github.com/allwaysyou/llm-agent/internal/model"
	"gorm.io/gorm"
)

// ProviderRepository handles Provider persistence
type ProviderRepository struct {
	db *DB
}

// NewProviderRepository creates a new provider repository
func NewProviderRepository(db *DB) *ProviderRepository {
	return &ProviderRepository{db: db}
}

// Create creates a new provider
func (r *ProviderRepository) Create(provider *model.Provider) error {
	return r.db.Create(provider).Error
}

// GetByID retrieves a provider by ID
func (r *ProviderRepository) GetByID(id string) (*model.Provider, error) {
	var provider model.Provider
	if err := r.db.First(&provider, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &provider, nil
}

// GetAll retrieves all providers
func (r *ProviderRepository) GetAll() ([]model.Provider, error) {
	var providers []model.Provider
	if err := r.db.Order("created_at desc").Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

// GetEnabled retrieves all enabled providers
func (r *ProviderRepository) GetEnabled() ([]model.Provider, error) {
	var providers []model.Provider
	if err := r.db.Where("enabled = ?", true).Order("created_at desc").Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

// GetByType retrieves providers by type
func (r *ProviderRepository) GetByType(providerType model.ProviderType) ([]model.Provider, error) {
	var providers []model.Provider
	if err := r.db.Where("type = ?", providerType).Order("created_at desc").Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

// Update updates a provider
func (r *ProviderRepository) Update(provider *model.Provider) error {
	return r.db.Save(provider).Error
}

// Delete deletes a provider by ID
func (r *ProviderRepository) Delete(id string) error {
	return r.db.Delete(&model.Provider{}, "id = ?", id).Error
}
