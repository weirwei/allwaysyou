# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Private LLM Agent with long-term memory capabilities. Go backend (Gin) + Vue 3 frontend. Supports multiple LLM providers (OpenAI, Claude, Azure, custom OpenAI-compatible APIs) with encrypted API key storage and semantic memory search.

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
cmd/server/main.go          → Entry point, dependency wiring
internal/handler/           → HTTP handlers (Gin routes)
internal/service/           → Business logic (ChatService, ConfigService, MemoryService, SummarizeService)
internal/repository/        → Data access layer (GORM + SQLite)
internal/adapter/           → LLM provider adapters (implements LLMAdapter interface)
internal/pkg/               → Shared packages (crypto, embedding, vector)
web/                        → Vue 3 + TypeScript frontend
```

### Key Patterns
- **Adapter Pattern**: `internal/adapter/adapter.go` defines `LLMAdapter` interface; implementations in `openai.go`, `claude.go`, `azure.go`, `custom.go`
- **Factory Pattern**: `AdapterFactory` creates provider-specific adapters based on config
- **Repository Pattern**: `ConfigRepository`, `SessionRepository`, `MemoryRepository` abstract data access

### Data Flow
1. HTTP request → Handler → Service → Repository/Adapter
2. Chat requests: Handler → ChatService → LLMAdapter (streaming via SSE)
3. Memory retrieval: MemoryService uses vector store for semantic search (cosine similarity)

### Storage
- **SQLite** (`data/llm.db`): Configs, sessions, messages metadata
- **Vector Store** (`data/chroma/`): JSON-persisted in-memory vector store for embeddings
- **Encryption**: AES-256-GCM for API keys (`internal/pkg/crypto/`)

## API Endpoints

- `POST /api/v1/configs` - Create LLM config (provider: openai, claude, azure, custom)
- `POST /api/v1/chat` - Send message (supports `stream: true` for SSE)
- `GET /api/v1/sessions` - List sessions
- `GET /api/v1/sessions/:id` - Get session with message history
- `GET /api/v1/memories/search?query=...&limit=5` - Semantic memory search

## Configuration

Main config at `configs/config.yaml`. Encryption key must be set via `LLM_AGENT_ENCRYPTION_KEY` environment variable (32 bytes for AES-256).
