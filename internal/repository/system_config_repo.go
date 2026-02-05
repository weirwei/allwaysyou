package repository

import (
	"github.com/allwaysyou/llm-agent/internal/model"
)

// SystemConfigRepository handles system configuration data operations
type SystemConfigRepository struct {
	db *DB
}

// NewSystemConfigRepository creates a new system config repository
func NewSystemConfigRepository(db *DB) *SystemConfigRepository {
	return &SystemConfigRepository{db: db}
}

// GetAll retrieves all system configs
func (r *SystemConfigRepository) GetAll() ([]model.SystemConfig, error) {
	var configs []model.SystemConfig
	if err := r.db.Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// GetByCategory retrieves system configs by category
func (r *SystemConfigRepository) GetByCategory(category string) ([]model.SystemConfig, error) {
	var configs []model.SystemConfig
	if err := r.db.Where("category = ?", category).Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// Get retrieves a system config by key
func (r *SystemConfigRepository) Get(key string) (*model.SystemConfig, error) {
	var config model.SystemConfig
	if err := r.db.First(&config, "key = ?", key).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// Upsert creates or updates a system config
func (r *SystemConfigRepository) Upsert(config *model.SystemConfig) error {
	return r.db.Save(config).Error
}

// Update updates a system config value
func (r *SystemConfigRepository) Update(key string, value string) error {
	return r.db.Model(&model.SystemConfig{}).Where("key = ?", key).Update("value", value).Error
}

// InitDefaults initializes default memory configs if they don't exist
func (r *SystemConfigRepository) InitDefaults() error {
	defaults := []model.SystemConfig{
		{
			Key:      "memory.conflict_detection_threshold",
			Value:    "0.85",
			Type:     "number",
			Category: "memory",
			Label:    "冲突检测阈值",
			Hint:     "用于检测相似知识的阈值 (0-1)",
		},
		{
			Key:      "memory.similar_knowledge_threshold",
			Value:    "0.7",
			Type:     "number",
			Category: "memory",
			Label:    "相似知识阈值",
			Hint:     "用于搜索相似知识的阈值 (0-1)",
		},
		{
			Key:      "memory.context_relevance_threshold",
			Value:    "0.5",
			Type:     "number",
			Category: "memory",
			Label:    "上下文相关阈值",
			Hint:     "包含在上下文中的最小分数 (0-1)",
		},
		{
			Key:      "memory.default_search_limit",
			Value:    "10",
			Type:     "number",
			Category: "memory",
			Label:    "默认搜索限制",
			Hint:     "默认搜索查询限制数量",
		},
		{
			Key:      "memory.context_knowledge_limit",
			Value:    "20",
			Type:     "number",
			Category: "memory",
			Label:    "上下文知识限制",
			Hint:     "上下文中最大知识项数量",
		},
		{
			Key:      "memory.max_knowledge_in_context",
			Value:    "8",
			Type:     "number",
			Category: "memory",
			Label:    "最大知识数量",
			Hint:     "包含的最大知识部分数",
		},
		{
			Key:      "memory.recent_memory_limit",
			Value:    "10",
			Type:     "number",
			Category: "memory",
			Label:    "近期记忆限制",
			Hint:     "近期对话历史限制数量",
		},
		{
			Key:      "memory.conflict_check_limit",
			Value:    "5",
			Type:     "number",
			Category: "memory",
			Label:    "冲突检查限制",
			Hint:     "冲突检测搜索的限制数量",
		},
		{
			Key:      "memory.default_importance",
			Value:    "0.5",
			Type:     "number",
			Category: "memory",
			Label:    "默认重要性",
			Hint:     "提取事实的默认重要性 (0-1)",
		},
	}

	for _, config := range defaults {
		// Only create if not exists
		var existing model.SystemConfig
		if err := r.db.First(&existing, "key = ?", config.Key).Error; err != nil {
			// Not found, create it
			if err := r.db.Create(&config).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
