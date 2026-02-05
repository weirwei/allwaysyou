package service

import (
	"fmt"
	"strconv"

	"github.com/allwaysyou/llm-agent/internal/model"
	"github.com/allwaysyou/llm-agent/internal/repository"
)

// SystemConfigService handles system configuration operations
type SystemConfigService struct {
	repo *repository.SystemConfigRepository
}

// NewSystemConfigService creates a new system config service
func NewSystemConfigService(repo *repository.SystemConfigRepository) *SystemConfigService {
	return &SystemConfigService{repo: repo}
}

// GetAll retrieves all system configs
func (s *SystemConfigService) GetAll() ([]model.SystemConfig, error) {
	return s.repo.GetAll()
}

// GetByCategory retrieves system configs by category
func (s *SystemConfigService) GetByCategory(category string) ([]model.SystemConfig, error) {
	return s.repo.GetByCategory(category)
}

// Get retrieves a system config by key
func (s *SystemConfigService) Get(key string) (*model.SystemConfig, error) {
	return s.repo.Get(key)
}

// Update updates a system config value with validation
func (s *SystemConfigService) Update(key string, value string) error {
	// Get existing config to check type
	config, err := s.repo.Get(key)
	if err != nil {
		return fmt.Errorf("config not found: %w", err)
	}

	// Validate based on type
	if err := s.validateValue(config.Type, value); err != nil {
		return err
	}

	return s.repo.Update(key, value)
}

// validateValue validates the value based on its type
func (s *SystemConfigService) validateValue(valueType, value string) error {
	switch valueType {
	case "number":
		_, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid number: %s", value)
		}
	case "boolean":
		_, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean: %s", value)
		}
	}
	return nil
}

// InitDefaults initializes default configs
func (s *SystemConfigService) InitDefaults() error {
	return s.repo.InitDefaults()
}
