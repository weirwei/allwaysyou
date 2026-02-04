package repository

import (
	"errors"

	"github.com/allwaysyou/llm-agent/internal/model"
	"gorm.io/gorm"
)

// ModelConfigRepository handles ModelConfig persistence
type ModelConfigRepository struct {
	db *DB
}

// NewModelConfigRepository creates a new model config repository
func NewModelConfigRepository(db *DB) *ModelConfigRepository {
	return &ModelConfigRepository{db: db}
}

// Create creates a new model config
func (r *ModelConfigRepository) Create(config *model.ModelConfig) error {
	return r.db.Create(config).Error
}

// GetByID retrieves a model config by ID
func (r *ModelConfigRepository) GetByID(id string) (*model.ModelConfig, error) {
	var config model.ModelConfig
	if err := r.db.Preload("Provider").First(&config, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// GetAll retrieves all model configs
func (r *ModelConfigRepository) GetAll() ([]model.ModelConfig, error) {
	var configs []model.ModelConfig
	if err := r.db.Preload("Provider").Order("created_at desc").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// GetByProvider retrieves all model configs for a specific provider
func (r *ModelConfigRepository) GetByProvider(providerID string) ([]model.ModelConfig, error) {
	var configs []model.ModelConfig
	if err := r.db.Preload("Provider").Where("provider_id = ?", providerID).Order("config_type, created_at desc").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// GetByType retrieves all model configs of a specific type
func (r *ModelConfigRepository) GetByType(configType model.ConfigType) ([]model.ModelConfig, error) {
	var configs []model.ModelConfig
	if err := r.db.Preload("Provider").Where("config_type = ?", configType).Order("created_at desc").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// GetDefaultByType retrieves the default model config for a specific type
func (r *ModelConfigRepository) GetDefaultByType(configType model.ConfigType) (*model.ModelConfig, error) {
	var config model.ModelConfig
	// First, try to find a default config of this type with an enabled provider
	if err := r.db.Preload("Provider").
		Joins("JOIN providers ON providers.id = model_configs.provider_id").
		Where("model_configs.config_type = ? AND model_configs.is_default = ? AND providers.enabled = ?", configType, true, true).
		First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// If no default, return the first config of this type with an enabled provider
			if err := r.db.Preload("Provider").
				Joins("JOIN providers ON providers.id = model_configs.provider_id").
				Where("model_configs.config_type = ? AND providers.enabled = ?", configType, true).
				First(&config).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, nil
				}
				return nil, err
			}
			return &config, nil
		}
		return nil, err
	}
	return &config, nil
}

// Update updates a model config
func (r *ModelConfigRepository) Update(config *model.ModelConfig) error {
	return r.db.Save(config).Error
}

// Delete deletes a model config by ID
func (r *ModelConfigRepository) Delete(id string) error {
	return r.db.Delete(&model.ModelConfig{}, "id = ?", id).Error
}

// DeleteByProvider deletes all model configs for a specific provider
func (r *ModelConfigRepository) DeleteByProvider(providerID string) error {
	return r.db.Delete(&model.ModelConfig{}, "provider_id = ?", providerID).Error
}

// SetDefault sets a model config as default and unsets others of the same type
func (r *ModelConfigRepository) SetDefault(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// First get the config to know its type
		var config model.ModelConfig
		if err := tx.First(&config, "id = ?", id).Error; err != nil {
			return err
		}

		// Unset defaults only for configs of the same type
		if err := tx.Model(&model.ModelConfig{}).Where("config_type = ? AND is_default = ?", config.ConfigType, true).Update("is_default", false).Error; err != nil {
			return err
		}
		// Set the new default
		return tx.Model(&model.ModelConfig{}).Where("id = ?", id).Update("is_default", true).Error
	})
}
