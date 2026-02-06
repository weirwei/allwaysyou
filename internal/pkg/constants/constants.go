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
	FactExtractionPrompt = `分析以下对话，提取用户透露的**值得长期记忆**的关键信息。

对话:
用户: %s
助手: %s

请以JSON数组格式返回提取的事实，每个事实包含:
- content: 事实内容（简洁的陈述句）
- category: 类别（personal_info=个人信息, preference=偏好, fact=事实, event=事件）
- importance: 重要性(0-1)

**应该保存的信息（长期记忆）：**
- 用户的个人信息（姓名、职业、住址等）
- 用户的长期偏好（喜好、习惯等）
- 用户的重要背景信息

**不应该保存的信息：**
- 当前操作的临时细节（如"开启了某模式"、"指定了某参数"）
- 一次性问题排查的场景描述
- 临时的技术配置或设置
- 只在当前对话有意义的上下文

示例输出:
[
  {"content": "用户名字是张三", "category": "personal_info", "importance": 0.9},
  {"content": "用户偏好使用Python编程", "category": "preference", "importance": 0.7}
]

如果没有值得**长期记忆**的信息，返回空数组: []

注意：只提取用户明确说出的、具有长期价值的信息，不要推断，不要保存临时操作细节。`

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
