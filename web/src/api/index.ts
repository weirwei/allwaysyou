// Detect Wails environment (dev mode uses port 34115, production embeds frontend)
const isWailsEnv = typeof window !== 'undefined' &&
  (window.location.port === '34115' || (window as any).go !== undefined)
const API_BASE = isWailsEnv ? 'http://127.0.0.1:18080/api/v1' : '/api/v1'

export interface Message {
  id?: string
  role: 'user' | 'assistant' | 'system'
  content: string
}

export interface Session {
  id: string
  title: string
  config_id: string
  summary: string
  created_at: string
  updated_at: string
}

export type ConfigType = 'chat' | 'summarize' | 'embedding'

export interface LLMConfig {
  id: string
  name: string
  provider: 'openai' | 'claude' | 'azure' | 'ollama' | 'custom'
  base_url?: string
  model: string
  max_tokens: number
  temperature: number
  is_default: boolean
  config_type: ConfigType
  created_at: string
  updated_at: string
}

export interface CreateConfigRequest {
  name: string
  provider: string
  api_key: string
  base_url?: string
  model: string
  max_tokens?: number
  temperature?: number
  is_default?: boolean
  config_type?: ConfigType
}

export interface UpdateConfigRequest {
  name?: string
  provider?: string
  api_key?: string
  base_url?: string
  model?: string
  max_tokens?: number
  temperature?: number
  is_default?: boolean
  config_type?: ConfigType
}

export interface ChatRequest {
  session_id?: string
  config_id?: string
  messages: Message[]
  stream?: boolean
}

export interface ChatResponse {
  id: string
  session_id: string
  message: Message
  usage?: {
    prompt_tokens: number
    completion_tokens: number
    total_tokens: number
  }
}

export interface StreamChunk {
  id: string
  delta: string
  done: boolean
}

// Config API
export async function getConfigs(): Promise<LLMConfig[]> {
  const res = await fetch(`${API_BASE}/configs`)
  if (!res.ok) throw new Error('Failed to fetch configs')
  return res.json()
}

export async function createConfig(config: CreateConfigRequest): Promise<LLMConfig> {
  const res = await fetch(`${API_BASE}/configs`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config)
  })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Failed to create config')
  }
  return res.json()
}

export async function deleteConfig(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/configs/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Failed to delete config')
}

export async function updateConfig(id: string, config: UpdateConfigRequest): Promise<LLMConfig> {
  const res = await fetch(`${API_BASE}/configs/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config)
  })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Failed to update config')
  }
  return res.json()
}

export async function setDefaultConfig(id: string): Promise<void> {
  await updateConfig(id, { is_default: true })
}

export interface TestConfigResult {
  success: boolean
  message?: string
  error?: string
}

export async function testConfig(id: string): Promise<TestConfigResult> {
  const res = await fetch(`${API_BASE}/configs/${id}/test`, { method: 'POST' })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Test failed')
  }
  return res.json()
}

// Session API
export async function getSessions(): Promise<Session[]> {
  const res = await fetch(`${API_BASE}/sessions`)
  if (!res.ok) throw new Error('Failed to fetch sessions')
  return res.json()
}

export async function getSession(id: string): Promise<{ session: Session; messages: Message[] }> {
  const res = await fetch(`${API_BASE}/sessions/${id}`)
  if (!res.ok) throw new Error('Failed to fetch session')
  return res.json()
}

export async function deleteSession(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/sessions/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Failed to delete session')
}

export async function deleteMessage(sessionId: string, messageId: string): Promise<void> {
  const res = await fetch(`${API_BASE}/sessions/${sessionId}/messages/${messageId}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Failed to delete message')
}

// Chat API
export async function chat(request: ChatRequest): Promise<ChatResponse> {
  const res = await fetch(`${API_BASE}/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request)
  })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Chat failed')
  }
  return res.json()
}

export interface StreamResult {
  stream: AsyncGenerator<StreamChunk>
  sessionId: string | null
}

export async function chatStream(request: ChatRequest): Promise<StreamResult> {
  const res = await fetch(`${API_BASE}/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ...request, stream: true })
  })

  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Chat failed')
  }

  const sessionId = res.headers.get('X-Session-ID')
  const reader = res.body?.getReader()
  if (!reader) throw new Error('No response body')

  const decoder = new TextDecoder()

  async function* generateChunks(): AsyncGenerator<StreamChunk> {
    let buffer = ''

    while (true) {
      const { done, value } = await reader!.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.startsWith('event: message')) continue
        if (line.startsWith('data: ')) {
          try {
            const data = JSON.parse(line.slice(6))
            yield data as StreamChunk
          } catch {
            // Skip invalid JSON
          }
        }
      }
    }
  }

  return {
    stream: generateChunks(),
    sessionId
  }
}

// Knowledge API
export interface Knowledge {
  id: string
  content: string
  superseded_by: string
  created_at: string
  updated_at: string
}

export async function getKnowledge(activeOnly = true, limit = 100): Promise<Knowledge[]> {
  const res = await fetch(`${API_BASE}/knowledge?active_only=${activeOnly}&limit=${limit}`)
  if (!res.ok) throw new Error('Failed to fetch knowledge')
  return res.json()
}

export async function createKnowledge(content: string): Promise<Knowledge> {
  const res = await fetch(`${API_BASE}/knowledge`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content })
  })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Failed to create knowledge')
  }
  return res.json()
}

export async function updateKnowledge(id: string, content: string): Promise<Knowledge> {
  const res = await fetch(`${API_BASE}/knowledge/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content })
  })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Failed to update knowledge')
  }
  return res.json()
}

export async function deleteKnowledge(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/knowledge/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Failed to delete knowledge')
}
