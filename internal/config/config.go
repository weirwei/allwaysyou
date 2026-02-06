package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Vector     VectorConfig     `mapstructure:"vector"`
	Encryption EncryptionConfig `mapstructure:"encryption"`
	Embedding  EmbeddingConfig  `mapstructure:"embedding"`
	Memory     MemoryConfig     `mapstructure:"memory"`
	LLM        LLMDefaults      `mapstructure:"llm"`
	Log        LogConfig        `mapstructure:"log"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type VectorConfig struct {
	Path       string `mapstructure:"path"`
	Collection string `mapstructure:"collection"`
}

type EncryptionConfig struct {
	Key string `mapstructure:"key"`
}

type EmbeddingConfig struct {
	Provider string `mapstructure:"provider"`
	Model    string `mapstructure:"model"`
	BaseURL  string `mapstructure:"base_url"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// MemoryConfig contains memory system configuration
type MemoryConfig struct {
	// Search thresholds
	ConflictDetectionThreshold float32 `mapstructure:"conflict_detection_threshold"` // Threshold for detecting similar knowledge (default: 0.85)
	SimilarKnowledgeThreshold  float32 `mapstructure:"similar_knowledge_threshold"`  // Threshold for similar knowledge search (default: 0.7)
	ContextRelevanceThreshold  float32 `mapstructure:"context_relevance_threshold"`  // Min score for including in context (default: 0.5)

	// Confidence thresholds for tiered memory
	LongTermThreshold float32 `mapstructure:"long_term_threshold"` // Min importance for long-term memory (default: 0.7)
	MidTermThreshold  float32 `mapstructure:"mid_term_threshold"`  // Min importance for mid-term memory (default: 0.4)

	// Mid-term memory promotion settings
	MidTermPromoteHits int `mapstructure:"mid_term_promote_hits"` // Hits required to promote to long-term (default: 3)
	MidTermExpireDays  int `mapstructure:"mid_term_expire_days"`  // Days before mid-term memory expires (default: 7)

	// Limits
	DefaultSearchLimit    int `mapstructure:"default_search_limit"`     // Default limit for search queries (default: 10)
	ContextKnowledgeLimit int `mapstructure:"context_knowledge_limit"`  // Max knowledge items in context (default: 20)
	MaxKnowledgeInContext int `mapstructure:"max_knowledge_in_context"` // Max knowledge parts to include (default: 8)
	RecentMemoryLimit     int `mapstructure:"recent_memory_limit"`      // Recent conversation history limit (default: 10)
	ConflictCheckLimit    int `mapstructure:"conflict_check_limit"`     // Limit for conflict detection search (default: 5)

	// Default values
	DefaultImportance float32 `mapstructure:"default_importance"` // Default importance for extracted facts (default: 0.5)
}

// LLMDefaults contains default LLM configuration
type LLMDefaults struct {
	MaxTokens        int     `mapstructure:"max_tokens"`         // Default max tokens (default: 4096)
	Temperature      float32 `mapstructure:"temperature"`        // Default temperature (default: 0.7)
	StreamBufferSize int     `mapstructure:"stream_buffer_size"` // Stream channel buffer size (default: 100)
	TitleMaxLength   int     `mapstructure:"title_max_length"`   // Max length for session titles (default: 50)
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// Environment variable support
	v.SetEnvPrefix("LLM_AGENT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Override encryption key from environment if set
	if envKey := os.Getenv("LLM_AGENT_ENCRYPTION_KEY"); envKey != "" {
		cfg.Encryption.Key = envKey
	}

	// Apply default values for Memory config
	cfg.Memory.applyDefaults()
	cfg.LLM.applyDefaults()

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Path == "" {
		return fmt.Errorf("database path is required")
	}

	return nil
}

func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// applyDefaults sets default values for MemoryConfig if not specified
func (m *MemoryConfig) applyDefaults() {
	if m.ConflictDetectionThreshold <= 0 {
		m.ConflictDetectionThreshold = 0.85
	}
	if m.SimilarKnowledgeThreshold <= 0 {
		m.SimilarKnowledgeThreshold = 0.7
	}
	if m.ContextRelevanceThreshold <= 0 {
		m.ContextRelevanceThreshold = 0.5
	}
	// Confidence thresholds for tiered memory
	if m.LongTermThreshold <= 0 {
		m.LongTermThreshold = 0.7
	}
	if m.MidTermThreshold <= 0 {
		m.MidTermThreshold = 0.4
	}
	// Mid-term memory settings
	if m.MidTermPromoteHits <= 0 {
		m.MidTermPromoteHits = 3
	}
	if m.MidTermExpireDays <= 0 {
		m.MidTermExpireDays = 7
	}
	if m.DefaultSearchLimit <= 0 {
		m.DefaultSearchLimit = 10
	}
	if m.ContextKnowledgeLimit <= 0 {
		m.ContextKnowledgeLimit = 20
	}
	if m.MaxKnowledgeInContext <= 0 {
		m.MaxKnowledgeInContext = 8
	}
	if m.RecentMemoryLimit <= 0 {
		m.RecentMemoryLimit = 10
	}
	if m.ConflictCheckLimit <= 0 {
		m.ConflictCheckLimit = 5
	}
	if m.DefaultImportance <= 0 {
		m.DefaultImportance = 0.5
	}
}

// applyDefaults sets default values for LLMDefaults if not specified
func (l *LLMDefaults) applyDefaults() {
	if l.MaxTokens <= 0 {
		l.MaxTokens = 4096
	}
	if l.Temperature <= 0 {
		l.Temperature = 0.7
	}
	if l.StreamBufferSize <= 0 {
		l.StreamBufferSize = 100
	}
	if l.TitleMaxLength <= 0 {
		l.TitleMaxLength = 50
	}
}
