package repository

import (
	"errors"

	"github.com/allwaysyou/llm-agent/internal/model"
	"gorm.io/gorm"
)

// SessionRepository handles session persistence
type SessionRepository struct {
	db *DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session
func (r *SessionRepository) Create(session *model.Session) error {
	return r.db.Create(session).Error
}

// GetByID retrieves a session by ID
func (r *SessionRepository) GetByID(id string) (*model.Session, error) {
	var session model.Session
	if err := r.db.First(&session, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

// GetAll retrieves all sessions
func (r *SessionRepository) GetAll(limit, offset int) ([]model.Session, error) {
	var sessions []model.Session
	query := r.db.Order("updated_at desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// Update updates a session
func (r *SessionRepository) Update(session *model.Session) error {
	return r.db.Save(session).Error
}

// Delete deletes a session by ID
func (r *SessionRepository) Delete(id string) error {
	return r.db.Delete(&model.Session{}, "id = ?", id).Error
}

// UpdateSummary updates the summary of a session
func (r *SessionRepository) UpdateSummary(id string, summary string) error {
	return r.db.Model(&model.Session{}).Where("id = ?", id).Update("summary", summary).Error
}
