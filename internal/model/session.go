package model

import "time"

// Session represents a chat session
type Session struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Title     string    `json:"title"`
	ConfigID  string    `json:"config_id" gorm:"index"` // LLM config used
	Summary   string    `json:"summary"`                // Session summary for long-term memory
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateSessionRequest represents the request to create a new session
type CreateSessionRequest struct {
	Title    string `json:"title"`
	ConfigID string `json:"config_id"`
}

// SessionResponse represents the response for a session
type SessionResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	ConfigID  string    `json:"config_id"`
	Summary   string    `json:"summary,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts Session to SessionResponse
func (s *Session) ToResponse() SessionResponse {
	return SessionResponse{
		ID:        s.ID,
		Title:     s.Title,
		ConfigID:  s.ConfigID,
		Summary:   s.Summary,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
