package repository

import (
	"errors"
	"fmt"

	"github.com/allwaysyou/llm-agent/internal/model"
	"gorm.io/gorm"
)

// KnowledgeRepository handles knowledge persistence
type KnowledgeRepository struct {
	db *DB
}

// NewKnowledgeRepository creates a new knowledge repository
func NewKnowledgeRepository(db *DB) *KnowledgeRepository {
	return &KnowledgeRepository{db: db}
}

// Create creates a new knowledge entry
func (r *KnowledgeRepository) Create(knowledge *model.Knowledge) error {
	return r.db.Create(knowledge).Error
}

// GetByID retrieves a knowledge entry by ID
func (r *KnowledgeRepository) GetByID(id string) (*model.Knowledge, error) {
	var knowledge model.Knowledge
	if err := r.db.First(&knowledge, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &knowledge, nil
}

// Update updates a knowledge entry
func (r *KnowledgeRepository) Update(knowledge *model.Knowledge) error {
	return r.db.Save(knowledge).Error
}

// Delete deletes a knowledge entry by ID
func (r *KnowledgeRepository) Delete(id string) error {
	return r.db.Delete(&model.Knowledge{}, "id = ?", id).Error
}

// Supersede marks a knowledge entry as superseded by another
func (r *KnowledgeRepository) Supersede(oldID, newID string) error {
	return r.db.Model(&model.Knowledge{}).
		Where("id = ?", oldID).
		Updates(map[string]any{
			"superseded_by": newID,
		}).Error
}

// GetAll retrieves all knowledge entries
func (r *KnowledgeRepository) GetAll(limit int) ([]model.Knowledge, error) {
	var knowledge []model.Knowledge
	query := r.db.Order("created_at desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&knowledge).Error; err != nil {
		return nil, err
	}
	return knowledge, nil
}

// GetAllActive retrieves all active knowledge (not superseded)
func (r *KnowledgeRepository) GetAllActive(limit int) ([]model.Knowledge, error) {
	var knowledge []model.Knowledge
	query := r.db.Where("superseded_by = '' OR superseded_by IS NULL").
		Order("created_at desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&knowledge).Error; err != nil {
		return nil, err
	}
	return knowledge, nil
}

// Count returns the total number of knowledge entries
func (r *KnowledgeRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&model.Knowledge{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountActive returns the number of active knowledge entries
func (r *KnowledgeRepository) CountActive() (int64, error) {
	var count int64
	if err := r.db.Model(&model.Knowledge{}).
		Where("superseded_by = '' OR superseded_by IS NULL").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// RecordHit records a hit for a knowledge entry (increments hit_count and updates last_hit_at)
func (r *KnowledgeRepository) RecordHit(id string) error {
	return r.db.Model(&model.Knowledge{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"hit_count":   gorm.Expr("hit_count + 1"),
			"last_hit_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}

// PromoteToLongTerm promotes a mid-term knowledge to long-term
func (r *KnowledgeRepository) PromoteToLongTerm(id string) error {
	return r.db.Model(&model.Knowledge{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"tier":        model.TierLongTerm,
			"promoted_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}

// GetMidTermByTier retrieves knowledge by tier
func (r *KnowledgeRepository) GetByTier(tier model.KnowledgeTier, activeOnly bool, limit int) ([]model.Knowledge, error) {
	var knowledge []model.Knowledge
	query := r.db.Where("tier = ?", tier)
	if activeOnly {
		query = query.Where("superseded_by = '' OR superseded_by IS NULL")
	}
	query = query.Order("created_at desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&knowledge).Error; err != nil {
		return nil, err
	}
	return knowledge, nil
}

// GetMidTermReadyForPromotion gets mid-term knowledge that has enough hits to be promoted
func (r *KnowledgeRepository) GetMidTermReadyForPromotion(minHits int) ([]model.Knowledge, error) {
	var knowledge []model.Knowledge
	if err := r.db.Where("tier = ? AND hit_count >= ? AND (superseded_by = '' OR superseded_by IS NULL)",
		model.TierMidTerm, minHits).
		Find(&knowledge).Error; err != nil {
		return nil, err
	}
	return knowledge, nil
}

// DeleteExpiredMidTerm deletes mid-term knowledge that hasn't been hit for a specified number of days
func (r *KnowledgeRepository) DeleteExpiredMidTerm(expireDays int) (int64, error) {
	// For mid-term memories without any hit, use created_at as reference
	// SQLite datetime format: datetime('now', '-7 days')
	modifier := fmt.Sprintf("-%d days", expireDays)
	result := r.db.Where("tier = ? AND ((last_hit_at IS NULL AND created_at < datetime('now', ?)) OR (last_hit_at IS NOT NULL AND last_hit_at < datetime('now', ?))) AND (superseded_by = '' OR superseded_by IS NULL)",
		model.TierMidTerm, modifier, modifier).
		Delete(&model.Knowledge{})
	return result.RowsAffected, result.Error
}
