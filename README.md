# LLM Agent

一个具有长期记忆能力的私人 LLM Agent，使用 Golang 构建后端，Vue 3 构建前端。

## 功能特性

### 核心功能
- **多模型支持**: OpenAI, Claude (Anthropic), Azure OpenAI, 以及任何 OpenAI 兼容 API
- **API Key 加密**: 使用 AES-256-GCM 加密存储敏感凭证
- **流式响应**: 支持 Server-Sent Events (SSE) 实时流式输出
- **RESTful API**: 完整的 REST API，易于与任何前端集成

### 记忆系统
- **会话管理**: 创建、管理、删除对话会话
- **短期记忆**: 会话内上下文自动保持
- **长期记忆**: 基于向量相似度的语义记忆检索
- **记忆摘要**: 自动生成对话摘要用于记忆压缩

### Web 界面
- 现代化深色主题 UI
- 实时流式对话显示
- 会话历史管理
- LLM 配置管理界面
- Markdown 渲染支持

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+ (用于构建前端)
- SQLite (内置，无需单独安装)

### 构建

```bash
# 构建后端
make build

# 构建前端
cd web && npm install && npm run build && cd ..
```

### 运行

```bash
# 设置加密密钥 (32字节)
export LLM_AGENT_ENCRYPTION_KEY="your-32-byte-encryption-key-here"

# 运行服务器
./bin/llm-agent -config ./configs/config.yaml
```

服务启动后访问:
- Web UI: http://localhost:8080/
- API: http://localhost:8080/api/v1

### 配置

编辑 `configs/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "debug"  # debug, release

database:
  path: "./data/llm.db"

vector:
  path: "./data/chroma"
  collection: "memories"

embedding:
  provider: "openai"
  model: "text-embedding-3-small"
```

## API 文档

### 配置管理

#### 创建 LLM 配置
```bash
curl -X POST http://localhost:8080/api/v1/configs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "OpenAI GPT-4",
    "provider": "openai",
    "api_key": "sk-your-api-key",
    "model": "gpt-4o-mini",
    "max_tokens": 4096,
    "temperature": 0.7,
    "is_default": true
  }'
```

支持的 provider:
- `openai` - OpenAI API
- `claude` - Anthropic Claude API
- `azure` - Azure OpenAI Service
- `custom` - 任何 OpenAI 兼容 API

#### 获取配置列表
```bash
curl http://localhost:8080/api/v1/configs
```

### 对话

#### 发送消息
```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "你好!"}]
  }'
```

#### 流式对话
```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "你好!"}],
    "stream": true
  }'
```

#### 继续现有会话
```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "existing-session-id",
    "messages": [{"role": "user", "content": "继续之前的话题"}]
  }'
```

### 会话管理

#### 获取会话列表
```bash
curl http://localhost:8080/api/v1/sessions
```

#### 获取会话详情(包含消息历史)
```bash
curl http://localhost:8080/api/v1/sessions/{session_id}
```

#### 删除会话
```bash
curl -X DELETE http://localhost:8080/api/v1/sessions/{session_id}
```

#### 生成会话摘要
```bash
curl -X POST http://localhost:8080/api/v1/sessions/{session_id}/summarize
```

### 记忆搜索

#### 语义搜索记忆
```bash
curl "http://localhost:8080/api/v1/memories/search?query=关于项目进度&limit=5"
```

## 项目结构

```
.
├── cmd/server/                # 应用入口
│   └── main.go
├── internal/
│   ├── adapter/               # LLM 提供商适配器
│   │   ├── adapter.go         # 适配器接口
│   │   ├── openai.go          # OpenAI 适配器
│   │   ├── claude.go          # Claude 适配器
│   │   └── azure.go           # Azure 适配器
│   ├── config/                # 配置管理
│   ├── handler/               # HTTP 处理器
│   ├── model/                 # 数据模型
│   ├── pkg/
│   │   ├── crypto/            # 加密工具
│   │   ├── embedding/         # 向量嵌入
│   │   └── vector/            # 向量存储
│   ├── repository/            # 数据持久化
│   └── service/               # 业务逻辑
├── web/                       # Vue 3 前端
│   ├── src/
│   │   ├── App.vue            # 主应用组件
│   │   ├── api/               # API 客户端
│   │   └── assets/            # 样式文件
│   └── dist/                  # 构建输出
├── configs/                   # 配置文件
├── data/                      # 数据目录 (gitignored)
├── Makefile
└── README.md
```

## 技术栈

### 后端
- **语言**: Go 1.21+
- **Web 框架**: Gin
- **ORM**: GORM + SQLite
- **配置**: Viper
- **加密**: AES-256-GCM

### 前端
- **框架**: Vue 3 + TypeScript
- **构建工具**: Vite
- **Markdown**: marked

### 存储
- **关系数据库**: SQLite (会话、配置、消息)
- **向量存储**: 内置向量存储 (语义搜索)

## 安全说明

1. **API Key 加密**: 所有 API Key 使用 AES-256-GCM 加密存储
2. **本地部署**: 所有数据存储在本地，不上传云端
3. **生产环境**: 务必通过环境变量设置 `LLM_AGENT_ENCRYPTION_KEY`

## 开发

### 热重载开发

后端 (需要 air):
```bash
air
```

前端:
```bash
cd web && npm run dev
```

### 运行测试
```bash
make test
```

## License

MIT
