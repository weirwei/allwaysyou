package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

type MemoryVector struct {
	ID        uuid.UUID              `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID              `json:"user_id" gorm:"type:uuid;index"`
	Content   string                 `json:"content" gorm:"type:text"`
	Embedding pgvector.Vector        `json:"-" gorm:"type:vector(1536)"`
	Metadata  map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb;serializer:json"`
	CreatedAt time.Time              `json:"created_at" gorm:"autoCreateTime"`
}

func NewMemoryVector(userID uuid.UUID, content string, embedding []float32) *MemoryVector {
	return &MemoryVector{
		ID:        uuid.New(),
		UserID:    userID,
		Content:   content,
		Embedding: pgvector.NewVector(embedding),
		CreatedAt: time.Now(),
	}
}

func (m *MemoryVector) TableName() string {
	return "memory_vectors"
}

// MemoryFragment represents a retrieved memory with relevance score
type MemoryFragment struct {
	ID        uuid.UUID              `json:"id"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	Score     float64                `json:"score"`
}
