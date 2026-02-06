# AllWaysYou

<p align="center">
  <img src="web/src/assets/logo.svg" width="120" height="120" alt="AllWaysYou Logo">
</p>

<p align="center">
  一个具有长期记忆能力的私人 LLM Agent，支持多模型配置和语义记忆检索。
</p>

## 功能特性

### 核心功能
- **多模型支持**: OpenAI, Claude, Azure OpenAI, Ollama, 以及任何 OpenAI 兼容 API
- **多模型配置**: 分别配置聊天模型、总结模型、向量模型
- **API Key 加密**: 使用 AES-256-GCM 加密存储敏感凭证
- **流式响应**: 支持 Server-Sent Events (SSE) 实时流式输出
- **桌面应用**: 基于 Wails 的原生桌面应用 (macOS)

### 记忆系统
- **会话管理**: 创建、管理、删除对话会话
- **短期记忆**: 会话内上下文自动保持
- **长期记忆**: 基于向量相似度的语义记忆检索
- **分层记忆**:
  - 中期记忆（观察区）: 中等置信度信息，多次命中后自动提升
  - 长期记忆: 高置信度的重要信息
- **智能提取**:
  - 关键信号检测（"我是..."、"我喜欢..."、"记住..."等）
  - 置信度过滤，自动丢弃低价值临时信息
- **知识管理**: 手动添加、编辑、删除知识条目
- **记忆摘要**: 自动生成对话摘要用于记忆压缩

### Web 界面
- 现代化深色主题 UI
- 实时流式对话显示
- 会话历史管理
- LLM 配置管理界面（按类型分组）
- 知识库管理界面
- Markdown 渲染支持

---

## 系统架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              AllWaysYou                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────┐    ┌─────────────────────────────────────────────────┐ │
│  │   Desktop App   │    │                   Backend (Go)                   │ │
│  │    (Wails)      │    │                                                  │ │
│  │                 │    │  ┌─────────────────────────────────────────────┐ │ │
│  │  ┌───────────┐  │    │  │              Handler Layer                  │ │ │
│  │  │  Vue 3    │  │◄──►│  │  ConfigHandler │ ChatHandler │ SessionHandler│ │ │
│  │  │ Frontend  │  │    │  │  MemoryHandler                              │ │ │
│  │  └───────────┘  │    │  └─────────────────────────────────────────────┘ │ │
│  │                 │    │                       │                          │ │
│  └─────────────────┘    │                       ▼                          │ │
│                         │  ┌─────────────────────────────────────────────┐ │ │
│                         │  │              Service Layer                   │ │ │
│                         │  │  ConfigService │ ChatService │ MemoryService │ │ │
│                         │  │  SummarizeService                           │ │ │
│                         │  └─────────────────────────────────────────────┘ │ │
│                         │                       │                          │ │
│                         │         ┌─────────────┼─────────────┐            │ │
│                         │         ▼             ▼             ▼            │ │
│                         │  ┌───────────┐ ┌───────────┐ ┌───────────────┐  │ │
│                         │  │ Repository│ │  Adapter  │ │  Memory Mgr   │  │ │
│                         │  │   Layer   │ │  Factory  │ │               │  │ │
│                         │  └─────┬─────┘ └─────┬─────┘ └───────┬───────┘  │ │
│                         │        │             │               │          │ │
│                         └────────┼─────────────┼───────────────┼──────────┘ │
│                                  │             │               │            │
│  ┌───────────────────────────────┼─────────────┼───────────────┼──────────┐ │
│  │                Storage        │             │               │          │ │
│  │  ┌─────────────┐  ┌──────────▼────┐  ┌─────▼─────┐  ┌──────▼──────┐   │ │
│  │  │   SQLite    │  │   LLM APIs    │  │  Vector   │  │  Embedding  │   │ │
│  │  │  Database   │  │               │  │   Store   │  │   Provider  │   │ │
│  │  │             │  │ ┌───────────┐ │  │  (JSON)   │  │             │   │ │
│  │  │ • Configs   │  │ │  OpenAI   │ │  │           │  │ • Ollama    │   │ │
│  │  │ • Sessions  │  │ │  Claude   │ │  │           │  │ • OpenAI    │   │ │
│  │  │ • Messages  │  │ │  Azure    │ │  │           │  │             │   │ │
│  │  │ • Knowledge │  │ │  Ollama   │ │  │           │  │             │   │ │
│  │  │ • Memories  │  │ │  Custom   │ │  │           │  │             │   │ │
│  │  └─────────────┘  │ └───────────┘ │  └───────────┘  └─────────────┘   │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 对话流程

```
┌──────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   用户   │     │   Frontend   │     │   Backend    │     │   LLM API    │
└────┬─────┘     └──────┬───────┘     └──────┬───────┘     └──────┬───────┘
     │                  │                    │                    │
     │  1. 发送消息     │                    │                    │
     │─────────────────►│                    │                    │
     │                  │  2. POST /chat     │                    │
     │                  │───────────────────►│                    │
     │                  │                    │                    │
     │                  │           ┌────────┴────────┐           │
     │                  │           │ 3. 检索相关知识  │           │
     │                  │           │   (向量搜索)    │           │
     │                  │           └────────┬────────┘           │
     │                  │                    │                    │
     │                  │           ┌────────┴────────┐           │
     │                  │           │ 4. 构建上下文   │           │
     │                  │           │  • 系统提示词   │           │
     │                  │           │  • 相关知识     │           │
     │                  │           │  • 历史消息     │           │
     │                  │           └────────┬────────┘           │
     │                  │                    │                    │
     │                  │                    │  5. 流式请求       │
     │                  │                    │───────────────────►│
     │                  │                    │                    │
     │                  │                    │  6. SSE 响应       │
     │                  │  7. SSE 转发       │◄───────────────────│
     │  8. 实时显示     │◄───────────────────│                    │
     │◄─────────────────│                    │                    │
     │                  │                    │                    │
     │                  │           ┌────────┴────────┐           │
     │                  │           │ 9. 保存消息     │           │
     │                  │           │  • 用户消息     │           │
     │                  │           │  • AI 回复      │           │
     │                  │           └────────┬────────┘           │
     │                  │                    │                    │
     │                  │           ┌────────┴────────┐           │
     │                  │           │10. 异步处理     │           │
     │                  │           │  • 提取知识     │           │
     │                  │           │  • 更新向量     │           │
     │                  │           └─────────────────┘           │
     │                  │                    │                    │
```

---

## 记忆系统详解

### 分层记忆架构

```
┌─────────────────────────────────────────────────────────────┐
│                      记忆系统                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │  短期记忆   │  │  中期记忆   │  │      长期记忆       │  │
│  │  (Memory)   │  │  (Mid-term) │  │    (Long-term)      │  │
│  ├─────────────┤  ├─────────────┤  ├─────────────────────┤  │
│  │ 会话级消息  │  │ 观察区信息  │  │  确认的重要信息     │  │
│  │ 自动保持    │  │ 命中3次提升 │  │  个人信息/偏好      │  │
│  │ 随会话清理  │  │ 7天未用过期 │  │  永久保存           │  │
│  └─────────────┘  └──────┬──────┘  └─────────────────────┘  │
│                          │                 ▲                 │
│                          └─────提升────────┘                 │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 智能提取机制

**关键信号检测** - 只在检测到以下模式时触发记忆提取：
- 身份信息: "我是..."、"我叫..."、"我名字是..."
- 偏好习惯: "我喜欢..."、"我偏好..."、"我习惯..."
- 长期特征: "以后都..."、"总是..."、"从不..."
- 显式请求: "记住..."、"别忘了..."、"帮我记..."
- 个人背景: "我住在..."、"我在...工作"

**置信度过滤** - 根据提取信息的重要性分级处理：

| 置信度 (importance) | 处理方式 |
|---------------------|----------|
| ≥ 0.7 | 直接存入长期记忆 |
| 0.4 - 0.7 | 存入中期观察区 |
| < 0.4 | 丢弃（临时操作细节等） |

---

## 多模型配置

系统支持三种类型的模型配置，各司其职：

| 配置类型 | 用途 | 推荐模型 |
|---------|------|---------|
| **Chat** | 对话交互 | GPT-4, Claude-3, Llama-3 |
| **Summarize** | 记忆总结 | GPT-3.5, Llama-3 |
| **Embedding** | 向量生成 | nomic-embed-text, text-embedding-3-small |

```
┌─────────────────────────────────────────────────────────────────┐
│                        Model Configs                             │
├─────────────────┬─────────────────┬─────────────────────────────┤
│   Chat Models   │ Summarize Models│     Embedding Models        │
├─────────────────┼─────────────────┼─────────────────────────────┤
│ • GPT-4o        │ • GPT-3.5       │ • nomic-embed-text (Ollama) │
│ • Claude-3      │ • Llama-3       │ • text-embedding-3-small    │
│ • Llama-3.2     │                 │ • mxbai-embed-large         │
│ • Qwen-2.5      │                 │                             │
└─────────────────┴─────────────────┴─────────────────────────────┘
```

---

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+
- Wails CLI v2 (桌面应用)
- Ollama (可选，本地模型)

### 桌面应用 (推荐)

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 开发模式
cd desktop && wails dev

# 构建生产版本
cd desktop && wails build
```

### 服务器模式

```bash
# 构建
make build
cd web && npm install && npm run build && cd ..

# 设置加密密钥
export LLM_AGENT_ENCRYPTION_KEY="your-32-byte-encryption-key-here"

# 运行
./bin/llm-agent -config ./configs/config.yaml
```

访问:
- Web UI: http://localhost:8080/
- API: http://localhost:8080/api/v1

---

## 项目结构

```
.
├── cmd/server/                # 服务器入口
│   └── main.go
├── desktop/                   # Wails 桌面应用
│   ├── app.go                 # 应用主逻辑
│   ├── build/                 # 构建配置和图标
│   │   └── darwin/            # macOS 配置
│   └── frontend/              # 前端引用
├── internal/
│   ├── adapter/               # LLM 提供商适配器
│   │   ├── adapter.go         # 适配器接口
│   │   ├── openai.go          # OpenAI
│   │   ├── claude.go          # Claude
│   │   ├── azure.go           # Azure OpenAI
│   │   ├── ollama.go          # Ollama
│   │   └── custom.go          # 自定义 OpenAI 兼容
│   ├── config/                # 配置加载
│   ├── handler/               # HTTP 处理器
│   ├── model/                 # 数据模型
│   ├── pkg/
│   │   ├── crypto/            # AES-256-GCM 加密
│   │   ├── embedding/         # 向量嵌入提供商
│   │   ├── memory/            # 记忆管理器
│   │   └── vector/            # 向量存储
│   ├── repository/            # 数据持久化 (GORM)
│   └── service/               # 业务逻辑
├── web/                       # Vue 3 前端
│   ├── src/
│   │   ├── App.vue            # 主应用组件
│   │   ├── api/               # API 客户端
│   │   └── assets/            # 样式和资源
│   └── dist/                  # 构建输出
├── configs/                   # 配置文件
└── data/                      # 数据目录 (gitignored)
    ├── llm.db                 # SQLite 数据库
    └── chroma/                # 向量存储
```

---

## API 文档

### 配置管理

```bash
# 创建配置
POST /api/v1/configs
{
  "name": "GPT-4",
  "provider": "openai",      # openai, claude, azure, ollama, custom
  "api_key": "sk-xxx",
  "model": "gpt-4o-mini",
  "config_type": "chat",     # chat, summarize, embedding
  "is_default": true
}

# 获取配置列表
GET /api/v1/configs

# 测试配置
POST /api/v1/configs/:id/test

# 删除配置
DELETE /api/v1/configs/:id
```

### 对话

```bash
# 发送消息 (流式)
POST /api/v1/chat
{
  "session_id": "xxx",       # 可选，继续现有会话
  "messages": [{"role": "user", "content": "你好"}],
  "stream": true
}

# 响应头包含: X-Session-ID
```

### 会话管理

```bash
# 获取会话列表
GET /api/v1/sessions

# 获取会话详情
GET /api/v1/sessions/:id

# 删除会话
DELETE /api/v1/sessions/:id

# 删除单条消息
DELETE /api/v1/sessions/:id/messages/:messageId
```

### 知识管理

```bash
# 获取知识列表
GET /api/v1/knowledge?active_only=true&limit=100

# 创建知识
POST /api/v1/knowledge
{ "content": "重要信息..." }

# 更新知识
PUT /api/v1/knowledge/:id
{ "content": "更新后的信息..." }

# 删除知识
DELETE /api/v1/knowledge/:id
```

### 记忆搜索

```bash
# 语义搜索
GET /api/v1/memories/search?query=关于项目&limit=5
```

---

## 技术栈

| 层级 | 技术 |
|-----|------|
| **桌面框架** | Wails v2 |
| **后端** | Go 1.21+, Gin, GORM |
| **前端** | Vue 3, TypeScript, Vite |
| **数据库** | SQLite |
| **向量存储** | 内置 JSON 向量存储 |
| **加密** | AES-256-GCM |
| **LLM** | OpenAI, Claude, Azure, Ollama |

---

## 配置说明

编辑 `configs/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "debug"

database:
  path: "./data/llm.db"

vector:
  path: "./data/chroma"

embedding:
  provider: "ollama"                    # ollama, openai
  model: "nomic-embed-text"
  base_url: "http://localhost:11434"

memory:
  context_relevance_threshold: 0.5      # 知识相关性阈值
  max_knowledge_in_context: 8           # 上下文中最大知识条数
  long_term_threshold: 0.7              # 长期记忆置信度阈值
  mid_term_threshold: 0.4               # 中期记忆置信度阈值
  mid_term_promote_hits: 3              # 中期记忆提升所需命中次数
  mid_term_expire_days: 7               # 中期记忆过期天数

llm:
  max_tokens: 4096
  temperature: 0.7
```

---

## 安全说明

1. **API Key 加密**: 所有 API Key 使用 AES-256-GCM 加密存储
2. **本地部署**: 所有数据存储在本地，不上传云端
3. **生产环境**: 务必通过环境变量设置 `LLM_AGENT_ENCRYPTION_KEY`

---

## License

MIT
