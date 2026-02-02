package entity

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID           uuid.UUID              `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID       uuid.UUID              `json:"user_id" gorm:"type:uuid;index"`
	Title        string                 `json:"title" gorm:"type:varchar(255)"`
	Provider     string                 `json:"provider" gorm:"type:varchar(50)"`
	Model        string                 `json:"model" gorm:"type:varchar(100)"`
	SystemPrompt string                 `json:"system_prompt,omitempty" gorm:"type:text"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb;serializer:json"`
	CreatedAt    time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time              `json:"updated_at" gorm:"autoUpdateTime"`
	Messages     []Message              `json:"messages,omitempty" gorm:"foreignKey:SessionID"`
}

func NewSession(userID uuid.UUID, provider, model string) *Session {
	return &Session{
		ID:        uuid.New(),
		UserID:    userID,
		Provider:  provider,
		Model:     model,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (s *Session) TableName() string {
	return "sessions"
}
