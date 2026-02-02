# LLM Agent 架构文档

## 时序图 - 聊天请求流程

```mermaid
sequenceDiagram
    participant User as 用户
    participant Handler as ChatHandler
    participant Service as ChatService
    participant Memory as MemoryManager
    participant Vector as VectorStore
    participant SQLite as SQLite DB
    participant LLM as LLM Adapter
    participant Processor as Processor

    User->>Handler: POST /api/v1/chat
    Handler->>Service: Chat(request)

    %% 获取配置
    Service->>SQLite: GetConfig()
    SQLite-->>Service: LLMConfig

    %% 构建上下文
    Service->>Memory: BuildContext(sessionID, query)
    Memory->>SQLite: GetRecentBySessionID(10)
    SQLite-->>Memory: 最近对话历史
    Memory->>Vector: SearchWithFilter(query)
    Vector-->>Memory: 相关知识记忆
    Memory-->>Service: 上下文消息[]

    %% 调用 LLM
    Service->>LLM: Chat(messages)
    LLM-->>Service: response

    %% 保存对话记忆
    Service->>Memory: SaveConversationMemory(user)
    Memory->>SQLite: Create(memory)
    Memory-->>Vector: Add(embedding) [async]

    Service->>Memory: SaveConversationMemory(assistant)
    Memory->>SQLite: Create(memory)
    Memory-->>Vector: Add(embedding) [async]

    %% 异步提取知识
    Service-->>Memory: ProcessConversation() [async]
    activate Memory
    Memory->>Processor: ExtractFacts(userMsg, assistantResp)
    Processor->>LLM: Chat(提取prompt)
    LLM-->>Processor: JSON facts[]
    Processor-->>Memory: ExtractedFact[]

    loop 每个 fact
        Memory->>Vector: Search(fact, minScore=0.7)
        Vector-->>Memory: 相似记忆[]
        Memory->>Processor: DetectConflict(fact, similar)
        Processor->>LLM: Chat(冲突检测prompt)
        LLM-->>Processor: {duplicate/conflict/create}

        alt 重复
            Memory->>Memory: Skip
        else 冲突更新
            Memory->>SQLite: Create(newMemory)
            Memory->>SQLite: Supersede(oldID, newID)
            Memory->>Vector: UpdateMetadata(IsActive=false)
            Memory->>Vector: Add(newEmbedding)
        else 新建
            Memory->>SQLite: Create(memory)
            Memory->>Vector: Add(embedding)
        end
    end
    deactivate Memory

    Service-->>Handler: ChatResponse
    Handler-->>User: JSON response
```

## 流程图 - 系统架构

```mermaid
flowchart TB
    subgraph Client["客户端"]
        WebUI["Vue 3 Web UI"]
        API["REST API 调用"]
    end

    subgraph Server["Go 后端服务"]
        subgraph Handlers["HTTP 层"]
            ChatH["ChatHandler"]
            ConfigH["ConfigHandler"]
            SessionH["SessionHandler"]
            MemoryH["MemoryHandler"]
        end

        subgraph Services["业务逻辑层"]
            ChatS["ChatService"]
            ConfigS["ConfigService"]
            SummarizeS["SummarizeService"]
        end

        subgraph MemorySystem["记忆系统"]
            Manager["MemoryManager"]
            Proc["Processor<br/>(LLM 处理)"]
        end

        subgraph Adapters["LLM 适配器"]
            Factory["AdapterFactory"]
            OpenAI["OpenAI"]
            Claude["Claude"]
            Azure["Azure"]
        end

        subgraph Storage["存储层"]
            subgraph SQLiteDB["SQLite"]
                Configs["configs 表"]
                Sessions["sessions 表"]
                Memories["memories 表"]
            end
            subgraph VectorDB["Vector Store"]
                Embeddings["embeddings<br/>(JSON文件)"]
            end
        end

        subgraph Packages["工具包"]
            Crypto["crypto<br/>(AES加密)"]
            Embed["embedding<br/>(Ollama/OpenAI)"]
        end
    end

    subgraph External["外部服务"]
        LLMProviders["LLM 提供商<br/>(OpenAI/Claude/Azure)"]
        OllamaEmbed["Ollama<br/>(本地embedding)"]
    end

    WebUI --> API
    API --> Handlers

    ChatH --> ChatS
    ConfigH --> ConfigS
    SessionH --> SQLiteDB
    MemoryH --> Manager

    ChatS --> Manager
    ChatS --> Factory

    Manager --> Proc
    Manager --> SQLiteDB
    Manager --> VectorDB
    Manager --> Embed

    Proc --> Factory

    Factory --> OpenAI
    Factory --> Claude
    Factory --> Azure

    ConfigS --> Crypto
    ConfigS --> SQLiteDB

    OpenAI --> LLMProviders
    Claude --> LLMProviders
    Azure --> LLMProviders

    Embed --> OllamaEmbed
    Embed --> LLMProviders
```

## 数据流程图 - 记忆写入

```mermaid
flowchart TD
    Start["用户消息 + AI回复"] --> Extract["1. LLM 提取事实"]
    Extract --> Facts{"有事实?"}

    Facts -->|否| End["结束"]
    Facts -->|是| Loop["遍历每个 fact"]

    Loop --> Search["2. 向量搜索相似记忆<br/>(score > 0.7)"]
    Search --> HasSimilar{"有相似?"}

    HasSimilar -->|否| Create["创建新记忆"]
    HasSimilar -->|是| Detect["3. LLM 检测冲突"]

    Detect --> Action{"判断结果"}

    Action -->|duplicate| Skip["跳过<br/>(完全重复)"]
    Action -->|conflict| Update["更新<br/>(新记忆取代旧记忆)"]
    Action -->|create| Create

    Create --> SaveDB["保存到 SQLite"]
    SaveDB --> GenEmbed["生成 Embedding"]
    GenEmbed --> SaveVector["保存到 Vector Store"]

    Update --> SaveDB2["保存新记忆到 SQLite"]
    SaveDB2 --> Supersede["标记旧记忆 SupersededBy"]
    Supersede --> UpdateVector["更新 Vector Store<br/>旧: IsActive=false"]
    UpdateVector --> GenEmbed2["生成新 Embedding"]
    GenEmbed2 --> SaveVector2["保存新向量"]

    Skip --> Next["下一个 fact"]
    SaveVector --> Next
    SaveVector2 --> Next

    Next --> Loop
    Next --> End
```

## 存储结构对比

```mermaid
flowchart LR
    subgraph SQLite["SQLite (结构化数据)"]
        subgraph MemoryTable["memories 表 (对话历史)"]
            M1["ID"]
            M2["SessionID"]
            M3["Role (user/assistant)"]
            M4["Content"]
            M5["CreatedAt"]
        end

        subgraph KnowledgeTable["knowledge 表 (长期记忆)"]
            K1["ID"]
            K2["Content"]
            K3["SupersededBy"]
            K4["CreatedAt"]
            K5["UpdatedAt"]
        end
    end

    subgraph Vector["Vector Store (语义数据)"]
        D1["Document"]
        D1 --> V1["ID (关联Knowledge)"]
        D1 --> V2["Content"]
        D1 --> V3["Embedding[]"]
        D1 --> V4["MetaData"]
        V4 --> VM1["Role = 'knowledge'"]
        V4 --> VM2["Category"]
        V4 --> VM3["Source"]
        V4 --> VM4["Importance"]
        V4 --> VM5["IsActive"]
    end

    K1 -.->|"ID 关联"| V1
```

### 数据分离说明

| 存储 | memories 表 | knowledge 表 | Vector Store |
|------|-------------|--------------|--------------|
| 用途 | 对话历史 | 长期知识 | 语义搜索 |
| 范围 | 会话级别 | 全局 | 全局 |
| 生命周期 | 短期 | 长期 | 长期 |
| 冲突检测 | 无 | 有 (SupersededBy) | 有 (IsActive) |
| 语义搜索 | 不支持 | 通过Vector Store | 支持 |

## 目录结构

```
.
├── cmd/server/main.go          # 入口，依赖注入
├── internal/
│   ├── adapter/                # LLM 适配器
│   │   ├── adapter.go          # 接口定义
│   │   ├── openai.go
│   │   ├── claude.go
│   │   └── azure.go
│   ├── handler/                # HTTP 处理器
│   ├── service/                # 业务逻辑
│   │   ├── chat_service.go
│   │   ├── config_service.go
│   │   └── memory_service.go
│   ├── repository/             # 数据访问
│   │   ├── memory_repo.go      # 对话历史存储
│   │   └── knowledge_repo.go   # 知识存储
│   ├── model/                  # 数据模型
│   │   ├── memory.go           # Memory (对话)
│   │   └── knowledge.go        # Knowledge (知识)
│   └── pkg/
│       ├── memory/             # 记忆管理
│       │   ├── manager.go      # MemoryManager (统一入口)
│       │   ├── processor.go    # LLM 提取与冲突检测
│       │   └── types.go        # 类型定义
│       ├── vector/             # 向量存储
│       ├── embedding/          # Embedding 提供商
│       └── crypto/             # 加密工具
├── web/                        # Vue 3 前端
├── data/
│   ├── llm.db                  # SQLite (memories + knowledge 表)
│   └── chroma/vectors.json     # 向量存储 (只存知识)
└── configs/config.yaml         # 配置文件
```
