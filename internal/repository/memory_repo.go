package repository

import (
	"errors"

	"github.com/allwaysyou/llm-agent/internal/model"
	"gorm.io/gorm"
)

// MemoryRepository handles memory persistence
type MemoryRepository struct {
	db *DB
}

// NewMemoryRepository creates a new memory repository
func NewMemoryRepository(db *DB) *MemoryRepository {
	return &MemoryRepository{db: db}
}

// Create creates a new memory
func (r *MemoryRepository) Create(memory *model.Memory) error {
	return r.db.Create(memory).Error
}

// CreateBatch creates multiple memories
func (r *MemoryRepository) CreateBatch(memories []model.Memory) error {
	if len(memories) == 0 {
		return nil
	}
	return r.db.Create(&memories).Error
}

// GetByID retrieves a memory by ID
func (r *MemoryRepository) GetByID(id string) (*model.Memory, error) {
	var memory model.Memory
	if err := r.db.First(&memory, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &memory, nil
}

// GetBySessionID retrieves all memories for a session, ordered by creation time
func (r *MemoryRepository) GetBySessionID(sessionID string, limit int) ([]model.Memory, error) {
	var memories []model.Memory
	query := r.db.Where("session_id = ?", sessionID).Order("created_at asc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&memories).Error; err != nil {
		return nil, err
	}
	return memories, nil
}

// GetRecentBySessionID retrieves the most recent memories for a session
func (r *MemoryRepository) GetRecentBySessionID(sessionID string, limit int) ([]model.Memory, error) {
	var memories []model.Memory
	if err := r.db.Where("session_id = ?", sessionID).
		Order("created_at desc").
		Limit(limit).
		Find(&memories).Error; err != nil {
		return nil, err
	}
	// Reverse to get chronological order
	for i, j := 0, len(memories)-1; i < j; i, j = i+1, j-1 {
		memories[i], memories[j] = memories[j], memories[i]
	}
	return memories, nil
}

// Delete deletes a memory by ID
func (r *MemoryRepository) Delete(id string) error {
	return r.db.Delete(&model.Memory{}, "id = ?", id).Error
}

// DeleteBySessionID deletes all memories for a session
func (r *MemoryRepository) DeleteBySessionID(sessionID string) error {
	return r.db.Delete(&model.Memory{}, "session_id = ?", sessionID).Error
}

// CountBySessionID counts memories for a session
func (r *MemoryRepository) CountBySessionID(sessionID string) (int64, error) {
	var count int64
	if err := r.db.Model(&model.Memory{}).Where("session_id = ?", sessionID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
