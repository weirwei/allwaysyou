package entity

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

type Message struct {
	ID         uuid.UUID              `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SessionID  uuid.UUID              `json:"session_id" gorm:"type:uuid;index"`
	Role       Role                   `json:"role" gorm:"type:varchar(20)"`
	Content    string                 `json:"content" gorm:"type:text"`
	TokensUsed int                    `json:"tokens_used,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb;serializer:json"`
	CreatedAt  time.Time              `json:"created_at" gorm:"autoCreateTime"`
}

func NewMessage(sessionID uuid.UUID, role Role, content string) *Message {
	return &Message{
		ID:        uuid.New(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	}
}

func (m *Message) TableName() string {
	return "messages"
}
