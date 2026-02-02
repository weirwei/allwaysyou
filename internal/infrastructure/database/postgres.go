package database

import (
	"context"
	"fmt"

	"github.com/allwaysyou/llm-agent/internal/config"
	"github.com/allwaysyou/llm-agent/internal/domain/entity"
	"github.com/allwaysyou/llm-agent/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PostgresDB struct {
	db *gorm.DB
}

func NewPostgres(cfg config.PostgresConfig) (*PostgresDB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	// Enable pgvector extension
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		return nil, fmt.Errorf("create vector extension: %w", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(&entity.Session{}, &entity.Message{}, &entity.MemoryVector{}); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	return &PostgresDB{db: db}, nil
}

func (p *PostgresDB) Close() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// SessionRepository implementation
type SessionRepo struct {
	db *gorm.DB
}

func NewSessionRepository(pg *PostgresDB) repository.SessionRepository {
	return &SessionRepo{db: pg.db}
}

func (r *SessionRepo) Create(ctx context.Context, session *entity.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *SessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Session, error) {
	var session entity.Session
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepo) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Session, error) {
	var sessions []*entity.Session
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *SessionRepo) Update(ctx context.Context, session *entity.Session) error {
	return r.db.WithContext(ctx).Save(session).Error
}

func (r *SessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.Session{}, id).Error
}

// MessageRepository implementation
type MessageRepo struct {
	db *gorm.DB
}

func NewMessageRepository(pg *PostgresDB) repository.MessageRepository {
	return &MessageRepo{db: pg.db}
}

func (r *MessageRepo) Create(ctx context.Context, message *entity.Message) error {
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *MessageRepo) GetBySessionID(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*entity.Message, error) {
	var messages []*entity.Message
	query := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at ASC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *MessageRepo) DeleteBySessionID(ctx context.Context, sessionID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("session_id = ?", sessionID).Delete(&entity.Message{}).Error
}

// MemoryRepository implementation
type MemoryRepo struct {
	db *gorm.DB
}

func NewMemoryRepository(pg *PostgresDB) repository.MemoryRepository {
	return &MemoryRepo{db: pg.db}
}

func (r *MemoryRepo) Create(ctx context.Context, memory *entity.MemoryVector) error {
	return r.db.WithContext(ctx).Create(memory).Error
}

func (r *MemoryRepo) SearchSimilar(ctx context.Context, userID uuid.UUID, embedding []float32, limit int, threshold float64) ([]*entity.MemoryFragment, error) {
	vec := pgvector.NewVector(embedding)

	var results []struct {
		entity.MemoryVector
		Score float64 `gorm:"column:score"`
	}

	err := r.db.WithContext(ctx).
		Table("memory_vectors").
		Select("*, 1 - (embedding <=> ?) as score", vec).
		Where("user_id = ?", userID).
		Where("1 - (embedding <=> ?) > ?", vec, threshold).
		Order("score DESC").
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	fragments := make([]*entity.MemoryFragment, len(results))
	for i, r := range results {
		fragments[i] = &entity.MemoryFragment{
			ID:        r.ID,
			Content:   r.Content,
			Metadata:  r.Metadata,
			CreatedAt: r.CreatedAt,
			Score:     r.Score,
		}
	}

	return fragments, nil
}

func (r *MemoryRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.MemoryVector{}, id).Error
}

func (r *MemoryRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entity.MemoryVector{}).Error
}
