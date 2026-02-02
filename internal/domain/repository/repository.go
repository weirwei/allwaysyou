package repository

import (
	"context"

	"github.com/allwaysyou/llm-agent/internal/domain/entity"
	"github.com/google/uuid"
)

// SessionRepository defines the interface for session persistence
type SessionRepository interface {
	Create(ctx context.Context, session *entity.Session) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Session, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Session, error)
	Update(ctx context.Context, session *entity.Session) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// MessageRepository defines the interface for message persistence
type MessageRepository interface {
	Create(ctx context.Context, message *entity.Message) error
	GetBySessionID(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*entity.Message, error)
	DeleteBySessionID(ctx context.Context, sessionID uuid.UUID) error
}

// MemoryRepository defines the interface for memory vector persistence
type MemoryRepository interface {
	Create(ctx context.Context, memory *entity.MemoryVector) error
	SearchSimilar(ctx context.Context, userID uuid.UUID, embedding []float32, limit int, threshold float64) ([]*entity.MemoryFragment, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}
