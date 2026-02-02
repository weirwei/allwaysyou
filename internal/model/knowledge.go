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

// Knowledge represents extracted user knowledge (long-term memory)
// Unlike conversation messages, knowledge is global and not tied to a specific session
type Knowledge struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	Content      string    `json:"content" gorm:"not null"`
	SupersededBy string    `json:"superseded_by" gorm:"index"` // 被哪条知识取代 (空=有效)
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// IsActive returns true if the knowledge has not been superseded
func (k *Knowledge) IsActive() bool {
	return k.SupersededBy == ""
}
