package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/allwaysyou/llm-agent/internal/config"
	"github.com/allwaysyou/llm-agent/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedis(cfg config.RedisConfig, ttl time.Duration) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}

	return &RedisCache{
		client: client,
		ttl:    ttl,
	}, nil
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}

func (r *RedisCache) sessionKey(sessionID uuid.UUID) string {
	return fmt.Sprintf("session:%s:messages", sessionID.String())
}

// SaveMessages saves messages to the working memory cache
func (r *RedisCache) SaveMessages(ctx context.Context, sessionID uuid.UUID, messages []*entity.Message) error {
	key := r.sessionKey(sessionID)

	data, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("marshal messages: %w", err)
	}

	return r.client.Set(ctx, key, data, r.ttl).Err()
}

// GetMessages retrieves messages from the working memory cache
func (r *RedisCache) GetMessages(ctx context.Context, sessionID uuid.UUID) ([]*entity.Message, error) {
	key := r.sessionKey(sessionID)

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("get messages: %w", err)
	}

	var messages []*entity.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("unmarshal messages: %w", err)
	}

	return messages, nil
}

// AppendMessage appends a single message to the working memory
func (r *RedisCache) AppendMessage(ctx context.Context, sessionID uuid.UUID, message *entity.Message, maxMessages int) error {
	messages, err := r.GetMessages(ctx, sessionID)
	if err != nil {
		return err
	}

	messages = append(messages, message)

	// Trim to max messages (keep system message if present)
	if len(messages) > maxMessages {
		startIdx := 0
		if messages[0].Role == entity.RoleSystem {
			startIdx = 1
		}
		messages = append(messages[:startIdx], messages[len(messages)-(maxMessages-startIdx):]...)
	}

	return r.SaveMessages(ctx, sessionID, messages)
}

// ClearSession clears the session's working memory
func (r *RedisCache) ClearSession(ctx context.Context, sessionID uuid.UUID) error {
	key := r.sessionKey(sessionID)
	return r.client.Del(ctx, key).Err()
}

// ExtendTTL extends the TTL of a session's working memory
func (r *RedisCache) ExtendTTL(ctx context.Context, sessionID uuid.UUID) error {
	key := r.sessionKey(sessionID)
	return r.client.Expire(ctx, key, r.ttl).Err()
}
