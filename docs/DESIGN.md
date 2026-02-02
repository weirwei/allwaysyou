# 私人 LLM Agent 设计方案

## 1. 项目概述

构建一个支持多种大模型 API 的私人 LLM Agent，具备长期记忆能力，可用于个人知识管理和智能对话。

## 2. 核心功能

### 2.1 多模型支持
- 支持 OpenAI API (GPT-4, GPT-3.5)
- 支持 Anthropic API (Claude)
- 支持 Azure OpenAI
- 支持自定义 OpenAI 兼容接口
- 可扩展的模型适配器架构

### 2.2 配置管理
- API Key 安全存储（加密）
- 多配置文件支持
- 运行时动态切换模型
- 配置热更新

### 2.3 长期记忆系统
- **短期记忆**: 会话上下文窗口
- **长期记忆**: 向量数据库持久化存储
- **记忆检索**: 基于语义相似度的记忆召回
- **记忆压缩**: 自动摘要和归纳

## 3. 技术架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Client Layer                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   Web UI    │  │   CLI Tool  │  │  REST API   │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      API Gateway (Gin)                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │    Auth     │  │ Rate Limit  │  │   Logger    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Core Service Layer                       │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                   Agent Service                      │    │
│  │  ┌───────────┐  ┌───────────┐  ┌───────────┐        │    │
│  │  │  Planner  │  │  Memory   │  │  Executor │        │    │
│  │  └───────────┘  └───────────┘  └───────────┘        │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                   LLM Adapter                        │    │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐    │    │
│  │  │ OpenAI  │ │ Claude  │ │  Azure  │ │ Custom  │    │    │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘    │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Storage Layer                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   SQLite    │  │  ChromaDB   │  │ File Store  │          │
│  │ (Metadata)  │  │  (Vectors)  │  │  (Config)   │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────────────────────┘
```

## 4. 技术选型

| 组件 | 技术选择 | 理由 |
|------|----------|------|
| 后端语言 | **Golang** | 指定要求，高性能，并发支持好 |
| Web 框架 | **Gin** | 轻量级，性能优秀，生态成熟 |
| 向量数据库 | **ChromaDB** (嵌入式) | 轻量，支持持久化，适合私人部署 |
| 关系数据库 | **SQLite** | 零配置，单文件，适合私人使用 |
| 配置管理 | **Viper** | Go 生态最流行的配置库 |
| 加密 | **AES-256-GCM** | 安全的对称加密算法 |
| 嵌入模型 | **本地嵌入** (可选远程) | 支持 OpenAI embedding 或本地模型 |
| 前端 | **Vue 3 + TypeScript** | 现代化，组件化，类型安全 |

## 5. 数据模型

### 5.1 配置模型
```go
type LLMConfig struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Provider    string    `json:"provider"`    // openai, claude, azure, custom
    APIKey      string    `json:"api_key"`     // 加密存储
    BaseURL     string    `json:"base_url"`    // API 端点
    Model       string    `json:"model"`       // 模型名称
    MaxTokens   int       `json:"max_tokens"`
    Temperature float64   `json:"temperature"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### 5.2 记忆模型
```go
type Memory struct {
    ID        string    `json:"id"`
    SessionID string    `json:"session_id"`
    Role      string    `json:"role"`      // user, assistant, system
    Content   string    `json:"content"`
    Embedding []float32 `json:"-"`         // 向量表示
    Metadata  map[string]interface{} `json:"metadata"`
    CreatedAt time.Time `json:"created_at"`
}

type Session struct {
    ID        string    `json:"id"`
    Title     string    `json:"title"`
    ConfigID  string    `json:"config_id"` // 使用的 LLM 配置
    Summary   string    `json:"summary"`   // 会话摘要
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

## 6. API 设计

### 6.1 配置管理
```
POST   /api/v1/configs          # 创建配置
GET    /api/v1/configs          # 获取配置列表
GET    /api/v1/configs/:id      # 获取配置详情
PUT    /api/v1/configs/:id      # 更新配置
DELETE /api/v1/configs/:id      # 删除配置
POST   /api/v1/configs/:id/test # 测试配置连通性
```

### 6.2 对话接口
```
POST   /api/v1/chat             # 发送消息（支持流式）
GET    /api/v1/sessions         # 获取会话列表
GET    /api/v1/sessions/:id     # 获取会话详情
DELETE /api/v1/sessions/:id     # 删除会话
POST   /api/v1/sessions/:id/summarize # 生成会话摘要
```

### 6.3 记忆管理
```
GET    /api/v1/memories/search  # 搜索相关记忆
POST   /api/v1/memories         # 手动添加记忆
DELETE /api/v1/memories/:id     # 删除记忆
```

## 7. 长期记忆实现方案

### 7.1 记忆存储流程
```
用户输入 → 生成 Embedding → 存入向量数据库 → 同时存入关系数据库（元数据）
```

### 7.2 记忆检索流程
```
用户新消息 → 生成 Embedding → 向量相似度搜索 → 获取相关历史记忆 → 构建增强 Prompt
```

### 7.3 记忆压缩策略
- 当会话超过 N 轮对话时，自动生成摘要
- 定期对历史记忆进行聚类和归纳
- 支持用户手动标记重要记忆

### 7.4 Prompt 构建
```
System Prompt:
  - 基础人设
  - 长期记忆摘要

Context:
  - 相关历史记忆（向量检索）
  - 当前会话上下文

User Message:
  - 用户当前输入
```

## 8. 项目结构

```
allwaysyou/
├── cmd/
│   └── server/
│       └── main.go           # 程序入口
├── internal/
│   ├── config/
│   │   └── config.go         # 配置加载
│   ├── handler/
│   │   ├── chat.go           # 对话处理
│   │   ├── config.go         # 配置管理
│   │   └── memory.go         # 记忆管理
│   ├── service/
│   │   ├── agent.go          # Agent 核心逻辑
│   │   ├── memory.go         # 记忆服务
│   │   └── llm.go            # LLM 调用服务
│   ├── adapter/
│   │   ├── adapter.go        # 适配器接口
│   │   ├── openai.go         # OpenAI 适配器
│   │   ├── claude.go         # Claude 适配器
│   │   └── azure.go          # Azure 适配器
│   ├── repository/
│   │   ├── config.go         # 配置存储
│   │   ├── session.go        # 会话存储
│   │   └── memory.go         # 记忆存储
│   ├── model/
│   │   ├── config.go         # 配置模型
│   │   ├── session.go        # 会话模型
│   │   └── memory.go         # 记忆模型
│   └── pkg/
│       ├── crypto/
│       │   └── crypto.go     # 加密工具
│       ├── embedding/
│       │   └── embedding.go  # 向量嵌入
│       └── vector/
│           └── chroma.go     # 向量数据库客户端
├── web/                       # 前端代码 (Vue 3)
│   ├── src/
│   │   ├── views/
│   │   ├── components/
│   │   └── api/
│   └── package.json
├── configs/
│   └── config.yaml           # 默认配置
├── data/                      # 数据目录
│   ├── llm.db                # SQLite 数据库
│   └── chroma/               # ChromaDB 数据
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 9. 安全考虑

1. **API Key 加密**: 使用 AES-256-GCM 加密存储，密钥从环境变量读取
2. **本地部署**: 所有数据存储在本地，不上传云端
3. **访问控制**: 支持简单的认证机制（可选）
4. **数据隔离**: 每个用户的数据独立存储

## 10. 部署方式

### 10.1 本地运行
```bash
# 编译
make build

# 运行
./bin/allwaysyou serve --config ./configs/config.yaml
```

### 10.2 Docker 部署
```bash
docker-compose up -d
```

## 11. 开发计划

### Phase 1: 核心功能
- [ ] 项目初始化，基础框架搭建
- [ ] LLM 适配器实现（OpenAI）
- [ ] 配置管理模块
- [ ] 基础对话功能

### Phase 2: 记忆系统
- [ ] 向量数据库集成
- [ ] 记忆存储和检索
- [ ] 记忆压缩和摘要

### Phase 3: 完善体验
- [ ] Web UI 开发
- [ ] 更多 LLM 适配器
- [ ] 性能优化

## 12. 技术难点

1. **长期记忆的有效检索**: 需要平衡召回率和精确率
2. **上下文窗口管理**: 合理分配系统提示、历史记忆和当前对话的 token 配额
3. **记忆去重和压缩**: 避免冗余记忆占用存储和检索资源

---

**下一步**: 请确认设计方案，我将开始实现 Phase 1 的核心功能。
