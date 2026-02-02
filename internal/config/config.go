package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig              `mapstructure:"server"`
	Auth      AuthConfig                `mapstructure:"auth"`
	Providers map[string]ProviderConfig `mapstructure:"providers"`
	Memory    MemoryConfig              `mapstructure:"memory"`
	Database  DatabaseConfig            `mapstructure:"database"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type AuthConfig struct {
	Enabled    bool          `mapstructure:"enabled"`
	JWTSecret  string        `mapstructure:"jwt_secret"`
	TokenExpiry time.Duration `mapstructure:"token_expiry"`
}

type ProviderConfig struct {
	Type    string  `mapstructure:"type"`
	APIKey  string  `mapstructure:"api_key"`
	BaseURL string  `mapstructure:"base_url"`
	Model   string  `mapstructure:"model"`
	Timeout int     `mapstructure:"timeout"`
}

type MemoryConfig struct {
	Working  WorkingMemoryConfig  `mapstructure:"working"`
	Episodic EpisodicMemoryConfig `mapstructure:"episodic"`
	Semantic SemanticMemoryConfig `mapstructure:"semantic"`
}

type WorkingMemoryConfig struct {
	MaxMessages int           `mapstructure:"max_messages"`
	TTL         time.Duration `mapstructure:"ttl"`
}

type EpisodicMemoryConfig struct {
	RetentionDays int `mapstructure:"retention_days"`
}

type SemanticMemoryConfig struct {
	EmbeddingModel      string  `mapstructure:"embedding_model"`
	SimilarityThreshold float64 `mapstructure:"similarity_threshold"`
	MaxResults          int     `mapstructure:"max_results"`
}

type DatabaseConfig struct {
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

func Load(configPath string) (*Config, error) {
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath(".")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Override with environment variables
	cfg.overrideFromEnv()

	return &cfg, nil
}

func (c *Config) overrideFromEnv() {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		c.Auth.JWTSecret = secret
	}
	if host := os.Getenv("DB_HOST"); host != "" {
		c.Database.Postgres.Host = host
	}
	if user := os.Getenv("DB_USER"); user != "" {
		c.Database.Postgres.User = user
	}
	if pass := os.Getenv("DB_PASSWORD"); pass != "" {
		c.Database.Postgres.Password = pass
	}
	if host := os.Getenv("REDIS_HOST"); host != "" {
		c.Database.Redis.Host = host
	}
	if pass := os.Getenv("REDIS_PASSWORD"); pass != "" {
		c.Database.Redis.Password = pass
	}

	// Override provider API keys
	for name, provider := range c.Providers {
		envKey := "OPENAI_API_KEY"
		if name == "claude" {
			envKey = "CLAUDE_API_KEY"
		}
		if key := os.Getenv(envKey); key != "" {
			provider.APIKey = key
			c.Providers[name] = provider
		}
	}
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
			Mode: "debug",
		},
		Auth: AuthConfig{
			Enabled:     false,
			TokenExpiry: 24 * time.Hour,
		},
		Providers: map[string]ProviderConfig{
			"openai": {
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
				Model:   "gpt-4",
				Timeout: 60,
			},
		},
		Memory: MemoryConfig{
			Working: WorkingMemoryConfig{
				MaxMessages: 20,
				TTL:         time.Hour,
			},
			Episodic: EpisodicMemoryConfig{
				RetentionDays: 365,
			},
			Semantic: SemanticMemoryConfig{
				EmbeddingModel:      "text-embedding-3-small",
				SimilarityThreshold: 0.7,
				MaxResults:          10,
			},
		},
		Database: DatabaseConfig{
			Postgres: PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "agent",
				Password: "",
				Database: "llm_agent",
				SSLMode:  "disable",
			},
			Redis: RedisConfig{
				Host:     "localhost",
				Port:     6379,
				Password: "",
				DB:       0,
			},
		},
	}
}
