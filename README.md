# LLM Agent

一个支持多大模型 API 配置和长期记忆的私人 LLM Agent。

## 特性

- **多 LLM Provider 支持**: OpenAI, Claude, Ollama 等
- **三层记忆系统**:
  - Working Memory: 当前对话上下文 (Redis)
  - Episodic Memory: 完整对话历史 (PostgreSQL)
  - Semantic Memory: 向量化语义检索 (pgvector)
- **流式响应**: 支持 SSE 流式输出
- **RESTful API**: 完整的 API 接口

## 快速开始

### 使用 Docker Compose

```bash
# 设置环境变量
export OPENAI_API_KEY=your-api-key
export DB_PASSWORD=your-db-password

# 启动服务
docker-compose up -d
```

### 本地开发

```bash
# 安装依赖
make deps

# 启动 PostgreSQL 和 Redis
docker-compose up -d postgres redis

# 运行服务
make run
```

## API 接口

### 对话

```bash
# 发送消息
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello, how are you?",
    "provider": "openai"
  }'

# 流式对话
curl -X POST http://localhost:8080/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Tell me a story",
    "session_id": "uuid-here"
  }'
```

### 会话管理

```bash
# 获取会话列表
curl http://localhost:8080/api/v1/sessions

# 获取会话详情
curl http://localhost:8080/api/v1/sessions/{id}

# 删除会话
curl -X DELETE http://localhost:8080/api/v1/sessions/{id}
```

### 提供商

```bash
# 获取可用提供商
curl http://localhost:8080/api/v1/providers
```

## 配置

配置文件位于 `configs/config.yaml`，支持通过环境变量覆盖：

- `OPENAI_API_KEY`: OpenAI API 密钥
- `CLAUDE_API_KEY`: Claude API 密钥
- `DB_HOST`, `DB_USER`, `DB_PASSWORD`: 数据库配置
- `REDIS_HOST`, `REDIS_PASSWORD`: Redis 配置
- `JWT_SECRET`: JWT 密钥（启用认证时）

## 项目结构

```
.
├── cmd/server/          # 程序入口
├── internal/
│   ├── api/             # HTTP API
│   ├── config/          # 配置管理
│   ├── domain/          # 领域模型
│   ├── infrastructure/  # 基础设施
│   └── service/         # 业务逻辑
├── configs/             # 配置文件
├── Dockerfile
├── docker-compose.yaml
└── Makefile
```

## License

MIT
