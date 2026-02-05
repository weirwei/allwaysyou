package model

import "time"

// SystemConfig represents a system configuration entry
type SystemConfig struct {
	Key       string    `json:"key" gorm:"primaryKey"`
	Value     string    `json:"value" gorm:"not null"`
	Type      string    `json:"type" gorm:"default:string"` // string, number, boolean
	Category  string    `json:"category" gorm:"index"`      // memory, llm, server, etc.
	Label     string    `json:"label"`                      // Display label
	Hint      string    `json:"hint"`                       // Help text
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SystemConfigRequest represents the request to update system config
type SystemConfigRequest struct {
	Value string `json:"value" binding:"required"`
}

// SystemConfigResponse represents the response for system config
type SystemConfigResponse struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Type     string `json:"type"`
	Category string `json:"category"`
	Label    string `json:"label"`
	Hint     string `json:"hint"`
}

// ToResponse converts SystemConfig to SystemConfigResponse
func (c *SystemConfig) ToResponse() SystemConfigResponse {
	return SystemConfigResponse{
		Key:      c.Key,
		Value:    c.Value,
		Type:     c.Type,
		Category: c.Category,
		Label:    c.Label,
		Hint:     c.Hint,
	}
}
