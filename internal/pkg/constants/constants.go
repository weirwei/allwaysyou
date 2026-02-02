package constants

// Embedding provider defaults
const (
	DefaultOpenAIBaseURL = "https://api.openai.com/v1"
	DefaultOpenAIModel   = "text-embedding-3-small"
	DefaultOllamaBaseURL = "http://localhost:11434"
	DefaultOllamaModel   = "nomic-embed-text"
)

// Session defaults
const (
	DefaultSessionTitle = "New Chat"
)

// Log truncation lengths
const (
	TruncateLengthShort  = 30
	TruncateLengthMedium = 50
)

// Vector store metadata keys
const (
	MetadataKeyType     = "type"
	MetadataKeyCategory = "category"
	MetadataKeySource   = "source"
	MetadataKeyIsActive = "is_active"
	MetadataValueTrue   = "true"
	MetadataValueFalse  = "false"
)

// Vector store document roles
const (
	RoleKnowledge = "knowledge"
)

// Context building
const (
	KnowledgeContextPrefix = "已知用户信息:\n"
	KnowledgeContextItem   = "- "
)

// LLM prompts for fact extraction
const (
	FactExtractionPrompt = `分析以下对话，提取用户透露的关键信息。

对话:
用户: %s
助手: %s

请以JSON数组格式返回提取的事实，每个事实包含:
- content: 事实内容（简洁的陈述句）
- category: 类别（personal_info=个人信息, preference=偏好, fact=事实, event=事件）
- importance: 重要性(0-1)

示例输出:
[
  {"content": "用户名字是张三", "category": "personal_info", "importance": 0.9},
  {"content": "用户喜欢喝咖啡", "category": "preference", "importance": 0.6}
]

如果没有值得记住的信息，返回空数组: []

注意：只提取用户明确说出的信息，不要推断。`

	ConflictDetectionPrompt = `判断新信息是否与已有信息冲突或重复。

新信息: %s

已有信息:
%s

请回答(JSON格式):
{
  "is_duplicate": true/false,  // 是否完全重复
  "is_conflict": true/false,   // 是否存在冲突(新信息更新了旧信息)
  "conflict_index": -1         // 冲突的已有信息索引(0开始)，无冲突则为-1
}

示例:
- 新"住在上海" vs 旧"住在北京" -> conflict=true
- 新"喜欢咖啡" vs 旧"喜欢喝咖啡" -> duplicate=true
- 新"养了一只猫" vs 旧"喜欢运动" -> 都是false`
)
