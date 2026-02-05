# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Private LLM Agent with long-term memory capabilities. Go backend (Gin) + Vue 3 frontend. Supports multiple LLM providers (OpenAI, Claude, Azure, Ollama, custom OpenAI-compatible APIs) with encrypted API key storage and semantic memory search.

## Common Commands

### Backend
```bash
make build          # Build binary to bin/llm-agent
make run            # Run with config: ./cmd/server -config ./configs/config.yaml
make dev            # Hot reload development (requires air)
make test           # Run all tests: go test -v ./...
make deps           # Download and tidy Go dependencies
```

### Frontend
```bash
cd web
npm install         # Install dependencies
npm run dev         # Vite dev server
npm run build       # Production build to dist/
```

### Desktop App
```bash
make desktop-dev    # Wails dev mode
make desktop-build  # Build macOS universal binary
```

### Running
```bash
export LLM_AGENT_ENCRYPTION_KEY="your-32-byte-key"
./bin/llm-agent -config ./configs/config.yaml
# Web UI: http://localhost:8080/
# API: http://localhost:8080/api/v1
```

## Architecture

### Layered Structure
```
cmd/server/main.go          → CLI server entry point
desktop/app.go              → Desktop app (Wails) entry point
internal/server/setup.go    → Shared server initialization code
internal/handler/           → HTTP handlers (Gin routes)
internal/service/           → Business logic (ChatService, ProviderService, ModelConfigService, MemoryService, SummarizeService)
internal/repository/        → Data access layer (GORM + SQLite)
internal/adapter/           → LLM provider adapters (implements LLMAdapter interface)
internal/pkg/               → Shared packages (crypto, embedding, vector, memory)
web/                        → Vue 3 + TypeScript frontend
```

### Key Patterns
- **Adapter Pattern**: `internal/adapter/adapter.go` defines `LLMAdapter` interface; implementations in `openai.go`, `claude.go`, `azure.go`, `ollama.go`
- **Factory Pattern**: `AdapterFactory` creates provider-specific adapters based on config
- **Repository Pattern**: `ProviderRepository`, `ModelConfigRepository`, `SessionRepository`, `MemoryRepository` abstract data access

### Data Flow
1. HTTP request → Handler → Service → Repository/Adapter
2. Chat requests: Handler → ChatService → LLMAdapter (streaming via SSE)
3. Memory retrieval: MemoryService uses vector store for semantic search (cosine similarity)

### Storage
- **SQLite** (`data/llm.db`): Providers, model configs, sessions, messages metadata
- **Vector Store** (`data/chroma/`): JSON-persisted in-memory vector store for embeddings
- **Encryption**: AES-256-GCM for API keys (`internal/pkg/crypto/`)

## API Endpoints

### Provider Management
- `GET /api/v1/providers` - List all providers
- `POST /api/v1/providers` - Create provider (type: openai, claude, azure, ollama, custom)
- `GET /api/v1/providers/:id` - Get provider with models
- `PUT /api/v1/providers/:id` - Update provider
- `DELETE /api/v1/providers/:id` - Delete provider
- `POST /api/v1/providers/:id/test` - Test provider connection

### Model Configuration
- `GET /api/v1/models` - List all models (optional: ?provider_id=...)
- `POST /api/v1/models` - Create model config
- `PUT /api/v1/models/:id` - Update model config
- `DELETE /api/v1/models/:id` - Delete model config
- `POST /api/v1/models/:id/default` - Set as default for type
- `POST /api/v1/models/:id/test` - Test model

### Chat
- `POST /api/v1/chat` - Send message (supports `stream: true` for SSE)

### Sessions
- `GET /api/v1/sessions` - List sessions
- `GET /api/v1/sessions/:id` - Get session with message history
- `DELETE /api/v1/sessions/:id` - Delete session
- `POST /api/v1/sessions/:id/summarize` - Generate session summary

### Knowledge & Memory
- `GET /api/v1/knowledge` - List knowledge entries
- `POST /api/v1/knowledge` - Create knowledge
- `PUT /api/v1/knowledge/:id` - Update knowledge
- `DELETE /api/v1/knowledge/:id` - Delete knowledge
- `GET /api/v1/memories/search?query=...&limit=5` - Semantic memory search

## Configuration

Main config at `configs/config.yaml`. Encryption key must be set via `LLM_AGENT_ENCRYPTION_KEY` environment variable (32 bytes for AES-256).
