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

export interface LLMConfig {
  id: string
  name: string
  provider: 'openai' | 'claude' | 'azure' | 'custom'
  base_url?: string
  model: string
  max_tokens: number
  temperature: number
  is_default: boolean
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
