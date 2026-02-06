package model

import "time"

// KnowledgeCategory represents the category of extracted knowledge
type KnowledgeCategory string

const (
	CategoryPersonalInfo KnowledgeCategory = "personal_info" // 个人信息：姓名、年龄、职业等
	CategoryPreference   KnowledgeCategory = "preference"    // 偏好：喜好、习惯等
	CategoryFact         KnowledgeCategory = "fact"          // 事实：具体事件、陈述等
	CategoryEvent        KnowledgeCategory = "event"         // 事件：发生的事情、计划等
)

// KnowledgeSource represents how the knowledge was created
type KnowledgeSource string

const (
	SourceExtracted KnowledgeSource = "extracted" // LLM 提取的知识
	SourceManual    KnowledgeSource = "manual"    // 手动添加
)

// KnowledgeTier represents the memory tier
type KnowledgeTier string

const (
	TierMidTerm  KnowledgeTier = "mid"  // 中期记忆 - 观察区，待验证
	TierLongTerm KnowledgeTier = "long" // 长期记忆 - 已确认的重要信息
)

// Knowledge represents extracted user knowledge (long-term memory)
// Unlike conversation messages, knowledge is global and not tied to a specific session
type Knowledge struct {
	ID           string         `json:"id" gorm:"primaryKey"`
	Content      string         `json:"content" gorm:"not null"`
	SupersededBy string         `json:"superseded_by" gorm:"index"` // 被哪条知识取代 (空=有效)
	Tier         KnowledgeTier  `json:"tier" gorm:"default:long"`   // 记忆层级
	HitCount     int            `json:"hit_count" gorm:"default:0"` // 命中次数（用于中期记忆提升）
	LastHitAt    *time.Time     `json:"last_hit_at"`                // 最后命中时间
	PromotedAt   *time.Time     `json:"promoted_at"`                // 从中期提升为长期的时间
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// IsActive returns true if the knowledge has not been superseded
func (k *Knowledge) IsActive() bool {
	return k.SupersededBy == ""
}

// IsMidTerm returns true if the knowledge is in mid-term tier
func (k *Knowledge) IsMidTerm() bool {
	return k.Tier == TierMidTerm
}

// IsLongTerm returns true if the knowledge is in long-term tier
func (k *Knowledge) IsLongTerm() bool {
	return k.Tier == TierLongTerm || k.Tier == "" // default is long-term
}
