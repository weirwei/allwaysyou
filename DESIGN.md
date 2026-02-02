# Private LLM Agent - 设计方案

## 1. 项目概述

构建一个私人 LLM Agent，具备以下核心能力：
- 支持多种大模型 API 配置（OpenAI、Claude、本地模型等）
- 长期记忆存储与检索
- 对话上下文管理
- RESTful API 接口

## 2. 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Layer                             │
│              (Web UI / CLI / Mobile App / SDK)                   │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                         API Gateway                              │
│                    (REST + WebSocket)                            │
│              Authentication / Rate Limiting                      │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Core Agent Service                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Session   │  │   Memory    │  │      LLM Provider       │  │
│  │   Manager   │  │   Manager   │  │       Abstraction       │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Prompt    │  │   Config    │  │      Tool/Plugin        │  │
│  │   Builder   │  │   Manager   │  │       Framework         │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Storage Layer                              │
├──────────────────┬──────────────────┬───────────────────────────┤
│    PostgreSQL    │      Redis       │     Vector Database       │
│  (Structured)    │   (Cache/MQ)     │    (Semantic Search)      │
└──────────────────┴──────────────────┴───────────────────────────┘
```

## 3. 核心模块设计

### 3.1 LLM Provider 抽象层

支持多种 LLM 提供商的统一接口：

```go
// LLMProvider 定义统一的 LLM 调用接口
type LLMProvider interface {
    // Chat 发送对话请求
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    // ChatStream 流式对话
    ChatStream(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error)
    // Embedding 生成文本向量
    Embedding(ctx context.Context, text string) ([]float64, error)
    // GetModel 获取当前模型信息
    GetModel() ModelInfo
}

// ChatRequest 对话请求
type ChatRequest struct {
    Messages    []Message
    MaxTokens   int
    Temperature float64
    Tools       []Tool
}

// ProviderConfig 提供商配置
type ProviderConfig struct {
    Type     string  // openai, claude, ollama, etc.
    APIKey   string
    BaseURL  string
    Model    string
    Timeout  int
}
```

**支持的提供商：**
- OpenAI (GPT-4, GPT-3.5)
- Anthropic (Claude)
- Ollama (本地模型)
- Azure OpenAI
- 自定义 OpenAI 兼容 API

### 3.2 记忆系统

采用三层记忆架构：

```
┌─────────────────────────────────────────────────────────┐
│                   Memory Architecture                    │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │           Working Memory (工作记忆)              │   │
│  │   - 当前对话上下文                               │   │
│  │   - 最近 N 轮对话                                │   │
│  │   - 存储: Redis                                  │   │
│  └─────────────────────────────────────────────────┘   │
│                          │                              │
│                          ▼                              │
│  ┌─────────────────────────────────────────────────┐   │
│  │          Episodic Memory (情景记忆)              │   │
│  │   - 完整对话历史                                 │   │
│  │   - 带时间戳和元数据                            │   │
│  │   - 存储: PostgreSQL                            │   │
│  └─────────────────────────────────────────────────┘   │
│                          │                              │
│                          ▼                              │
│  ┌─────────────────────────────────────────────────┐   │
│  │          Semantic Memory (语义记忆)              │   │
│  │   - 向量化的知识片段                            │   │
│  │   - 语义相似度检索                              │   │
│  │   - 存储: pgvector / Milvus                     │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

```go
// MemoryManager 记忆管理器
type MemoryManager interface {
    // SaveMessage 保存消息到记忆
    SaveMessage(ctx context.Context, sessionID string, msg Message) error
    // GetRecentMessages 获取最近的消息
    GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]Message, error)
    // SearchRelevantMemories 语义搜索相关记忆
    SearchRelevantMemories(ctx context.Context, query string, limit int) ([]MemoryFragment, error)
    // Summarize 生成记忆摘要
    Summarize(ctx context.Context, sessionID string) (string, error)
    // Forget 遗忘指定记忆
    Forget(ctx context.Context, memoryID string) error
}

// MemoryFragment 记忆片段
type MemoryFragment struct {
    ID        string
    Content   string
    Embedding []float64
    Metadata  map[string]interface{}
    CreatedAt time.Time
    Score     float64  // 相关性得分
}
```

### 3.3 配置管理

```yaml
# config.yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"  # debug / release

auth:
  enabled: true
  jwt_secret: "${JWT_SECRET}"
  token_expiry: "24h"

providers:
  default: "openai"
  openai:
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
    model: "gpt-4"
    timeout: 60
  claude:
    api_key: "${CLAUDE_API_KEY}"
    base_url: "https://api.anthropic.com"
    model: "claude-3-opus-20240229"
  ollama:
    base_url: "http://localhost:11434"
    model: "llama2"

memory:
  working:
    max_messages: 20
    ttl: "1h"
  episodic:
    retention_days: 365
  semantic:
    embedding_model: "text-embedding-3-small"
    similarity_threshold: 0.7
    max_results: 10

database:
  postgres:
    host: "${DB_HOST}"
    port: 5432
    user: "${DB_USER}"
    password: "${DB_PASSWORD}"
    database: "llm_agent"
  redis:
    host: "${REDIS_HOST}"
    port: 6379
    password: "${REDIS_PASSWORD}"
    db: 0
```

### 3.4 API 设计

#### RESTful API

```
POST   /api/v1/chat                    # 发送消息
POST   /api/v1/chat/stream             # 流式对话 (SSE)
GET    /api/v1/sessions                # 获取会话列表
POST   /api/v1/sessions                # 创建会话
GET    /api/v1/sessions/:id            # 获取会话详情
DELETE /api/v1/sessions/:id            # 删除会话
GET    /api/v1/sessions/:id/messages   # 获取会话消息
POST   /api/v1/memory/search           # 搜索记忆
DELETE /api/v1/memory/:id              # 删除记忆
GET    /api/v1/providers               # 获取可用提供商
PUT    /api/v1/providers/:name         # 更新提供商配置
GET    /api/v1/config                  # 获取配置
PUT    /api/v1/config                  # 更新配置
```

#### WebSocket

```
WS /ws/chat/:session_id   # 实时对话
```

### 3.5 数据模型

```sql
-- 会话表
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    title VARCHAR(255),
    provider VARCHAR(50) NOT NULL,
    model VARCHAR(100) NOT NULL,
    system_prompt TEXT,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 消息表
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES sessions(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,  -- user, assistant, system
    content TEXT NOT NULL,
    tokens_used INT,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 记忆向量表 (使用 pgvector)
CREATE TABLE memory_vectors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1536),  -- OpenAI embedding dimension
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX ON memory_vectors USING ivfflat (embedding vector_cosine_ops);

-- 提供商配置表
CREATE TABLE provider_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    name VARCHAR(50) NOT NULL,
    config JSONB NOT NULL,  -- encrypted
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, name)
);
```

## 4. 技术选型

| 组件 | 技术选择 | 理由 |
|------|---------|------|
| 后端语言 | Go 1.22+ | 高性能、并发支持好 |
| Web 框架 | Gin | 轻量、高性能、生态丰富 |
| ORM | GORM | 功能全面、支持 pgvector |
| 配置管理 | Viper | 支持多种配置源、热重载 |
| 日志 | Zap | 高性能结构化日志 |
| 数据库 | PostgreSQL + pgvector | 关系型 + 向量存储一体 |
| 缓存 | Redis | 会话缓存、消息队列 |
| API 文档 | Swagger/OpenAPI | 自动生成 API 文档 |
| 容器化 | Docker + Compose | 简化部署 |

## 5. 项目结构

```
.
├── cmd/
│   └── server/
│       └── main.go              # 程序入口
├── internal/
│   ├── api/
│   │   ├── handler/             # HTTP 处理器
│   │   ├── middleware/          # 中间件
│   │   └── router.go            # 路由配置
│   ├── config/
│   │   └── config.go            # 配置加载
│   ├── domain/
│   │   ├── entity/              # 领域实体
│   │   └── repository/          # 仓储接口
│   ├── infrastructure/
│   │   ├── database/            # 数据库实现
│   │   ├── cache/               # 缓存实现
│   │   └── llm/                 # LLM 提供商实现
│   │       ├── openai/
│   │       ├── claude/
│   │       └── ollama/
│   ├── service/
│   │   ├── agent/               # Agent 核心服务
│   │   ├── memory/              # 记忆服务
│   │   └── session/             # 会话服务
│   └── pkg/
│       ├── errors/              # 错误处理
│       └── utils/               # 工具函数
├── migrations/                   # 数据库迁移
├── configs/
│   └── config.yaml              # 配置文件
├── scripts/                      # 脚本
├── Dockerfile
├── docker-compose.yaml
├── Makefile
├── go.mod
└── go.sum
```

## 6. 核心流程

### 6.1 对话流程

```
用户发送消息
       │
       ▼
┌──────────────────┐
│   接收请求       │
│   验证 Token     │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  加载会话上下文   │
│  (Working Memory)│
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  语义搜索相关记忆 │
│ (Semantic Memory)│
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│   构建 Prompt    │
│  System + Context│
│  + Memories      │
│  + User Message  │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│   调用 LLM API   │
│  (Provider)      │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│   保存消息       │
│ (Episodic Memory)│
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  异步向量化存储   │
│ (Semantic Memory)│
└────────┬─────────┘
         │
         ▼
    返回响应
```

### 6.2 记忆检索流程

```
用户消息
    │
    ▼
┌────────────────┐
│  生成 Embedding │
└───────┬────────┘
        │
        ▼
┌────────────────┐
│  向量相似度搜索 │
│  Top K 结果    │
└───────┬────────┘
        │
        ▼
┌────────────────┐
│  相关性过滤     │
│  score > 0.7   │
└───────┬────────┘
        │
        ▼
┌────────────────┐
│  重排序 & 去重  │
└───────┬────────┘
        │
        ▼
  返回相关记忆
```

## 7. 安全设计

1. **API 密钥加密存储** - 使用 AES-256 加密存储用户的 API Key
2. **JWT 认证** - 所有 API 需要 Bearer Token 认证
3. **Rate Limiting** - 防止 API 滥用
4. **输入验证** - 严格的请求参数校验
5. **日志脱敏** - 敏感信息不记录到日志

## 8. 部署方案

### Docker Compose 部署

```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis

  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_DB: llm_agent
      POSTGRES_USER: agent
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

## 9. 开发计划

### Phase 1: 基础框架 (Week 1)
- [ ] 项目初始化、目录结构
- [ ] 配置管理模块
- [ ] 数据库连接和迁移
- [ ] 基础 API 框架

### Phase 2: LLM 集成 (Week 2)
- [ ] LLM Provider 抽象层
- [ ] OpenAI Provider 实现
- [ ] 基础对话功能
- [ ] 流式响应支持

### Phase 3: 记忆系统 (Week 3)
- [ ] Working Memory (Redis)
- [ ] Episodic Memory (PostgreSQL)
- [ ] Semantic Memory (pgvector)
- [ ] 记忆检索和注入

### Phase 4: 完善和优化 (Week 4)
- [ ] 多 Provider 支持 (Claude, Ollama)
- [ ] API 文档生成
- [ ] Docker 部署
- [ ] 性能优化

## 10. 扩展考虑

- **Tool/Function Calling**: 支持 LLM 调用外部工具
- **RAG**: 集成文档检索增强生成
- **Multi-Agent**: 支持多 Agent 协作
- **Web UI**: 提供 Web 管理界面
- **插件系统**: 支持自定义插件扩展
