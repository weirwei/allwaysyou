package repository

import (
	"errors"

	"github.com/allwaysyou/llm-agent/internal/model"
	"gorm.io/gorm"
)

// ConfigRepository handles LLM config persistence
type ConfigRepository struct {
	db *DB
}

// NewConfigRepository creates a new config repository
func NewConfigRepository(db *DB) *ConfigRepository {
	return &ConfigRepository{db: db}
}

// Create creates a new LLM config
func (r *ConfigRepository) Create(config *model.LLMConfig) error {
	return r.db.Create(config).Error
}

// GetByID retrieves an LLM config by ID
func (r *ConfigRepository) GetByID(id string) (*model.LLMConfig, error) {
	var config model.LLMConfig
	if err := r.db.First(&config, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// GetAll retrieves all LLM configs
func (r *ConfigRepository) GetAll() ([]model.LLMConfig, error) {
	var configs []model.LLMConfig
	if err := r.db.Order("created_at desc").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// GetDefault retrieves the default LLM config (for backward compatibility, returns chat type)
func (r *ConfigRepository) GetDefault() (*model.LLMConfig, error) {
	return r.GetDefaultByType(model.ConfigTypeChat)
}

// GetByType retrieves all LLM configs of a specific type
func (r *ConfigRepository) GetByType(configType model.ConfigType) ([]model.LLMConfig, error) {
	var configs []model.LLMConfig
	if err := r.db.Where("config_type = ?", configType).Order("created_at desc").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// GetDefaultByType retrieves the default LLM config for a specific type
func (r *ConfigRepository) GetDefaultByType(configType model.ConfigType) (*model.LLMConfig, error) {
	var config model.LLMConfig
	if err := r.db.First(&config, "config_type = ? AND is_default = ?", configType, true).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// If no default, return the first config of this type
			if err := r.db.First(&config, "config_type = ?", configType).Error; err != nil {
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

// Update updates an LLM config
func (r *ConfigRepository) Update(config *model.LLMConfig) error {
	return r.db.Save(config).Error
}

// Delete deletes an LLM config by ID
func (r *ConfigRepository) Delete(id string) error {
	return r.db.Delete(&model.LLMConfig{}, "id = ?", id).Error
}

// SetDefault sets a config as default and unsets others of the same type
func (r *ConfigRepository) SetDefault(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// First get the config to know its type
		var config model.LLMConfig
		if err := tx.First(&config, "id = ?", id).Error; err != nil {
			return err
		}

		// Unset defaults only for configs of the same type
		if err := tx.Model(&model.LLMConfig{}).Where("config_type = ? AND is_default = ?", config.ConfigType, true).Update("is_default", false).Error; err != nil {
			return err
		}
		// Set the new default
		return tx.Model(&model.LLMConfig{}).Where("id = ?", id).Update("is_default", true).Error
	})
}
