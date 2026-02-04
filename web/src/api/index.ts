// Detect Wails environment (dev mode uses port 34115, production embeds frontend)
const isWailsEnv = typeof window !== 'undefined' &&
  (window.location.port === '34115' || (window as any).go !== undefined)

// Get API port from localStorage, default to 18080
function getApiBase(): string {
  if (!isWailsEnv) return '/api/v1'
  const port = localStorage.getItem('apiPort') || '18080'
  return `http://127.0.0.1:${port}/api/v1`
}

// Export functions to get/set API port
export function getApiPort(): string {
  return localStorage.getItem('apiPort') || '18080'
}

export function setApiPort(port: string): void {
  localStorage.setItem('apiPort', port)
}

// Settings API - get port from backend
export async function getServerPort(): Promise<number> {
  const res = await fetch(`${getApiBase()}/settings/port`)
  if (!res.ok) throw new Error('Failed to get port')
  const data = await res.json()
  return data.port
}

// Settings API - save port to backend config
export async function saveServerPort(port: number): Promise<void> {
  const res = await fetch(`${getApiBase()}/settings/port`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ port })
  })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Failed to save port')
  }
}

// Use getter to allow dynamic port changes
const getApiBaseUrl = () => getApiBase()

// For backward compatibility, also export as constant (but functions will use getter)
let API_BASE = getApiBase()

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
export type ProviderType = 'openai' | 'claude' | 'azure' | 'ollama' | 'custom'

// Legacy LLMConfig (kept for backward compatibility)
export interface LLMConfig {
  id: string
  name: string
  provider: ProviderType
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

// New Provider types
export interface Provider {
  id: string
  name: string
  type: ProviderType
  base_url: string
  enabled: boolean
  has_api_key: boolean
  created_at: string
  updated_at: string
  models?: ModelConfig[]
}

export interface CreateProviderRequest {
  name: string
  type: ProviderType
  api_key: string
  base_url?: string
  enabled?: boolean
}

export interface UpdateProviderRequest {
  name?: string
  type?: ProviderType
  api_key?: string
  base_url?: string
  enabled?: boolean
}

// New ModelConfig types
export interface ModelConfig {
  id: string
  provider_id: string
  model: string
  max_tokens: number
  temperature: number
  config_type: ConfigType
  is_default: boolean
  created_at: string
  updated_at: string
  provider?: Provider
}

export interface CreateModelConfigRequest {
  provider_id: string
  model: string
  max_tokens?: number
  temperature?: number
  config_type?: ConfigType
  is_default?: boolean
}

export interface UpdateModelConfigRequest {
  model?: string
  max_tokens?: number
  temperature?: number
  config_type?: ConfigType
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

export interface TestResult {
  success: boolean
  message?: string
  error?: string
}

// Legacy Config API (kept for backward compatibility)
export async function getConfigs(): Promise<LLMConfig[]> {
  const res = await fetch(`${getApiBaseUrl()}/configs`)
  if (!res.ok) throw new Error('Failed to fetch configs')
  return res.json()
}

export async function createConfig(config: CreateConfigRequest): Promise<LLMConfig> {
  const res = await fetch(`${getApiBaseUrl()}/configs`, {
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
  const res = await fetch(`${getApiBaseUrl()}/configs/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Failed to delete config')
}

export async function updateConfig(id: string, config: UpdateConfigRequest): Promise<LLMConfig> {
  const res = await fetch(`${getApiBaseUrl()}/configs/${id}`, {
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
  const res = await fetch(`${getApiBaseUrl()}/configs/${id}/test`, { method: 'POST' })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Test failed')
  }
  return res.json()
}

// Provider API
export async function getProviders(): Promise<Provider[]> {
  const res = await fetch(`${getApiBaseUrl()}/providers`)
  if (!res.ok) throw new Error('Failed to fetch providers')
  return res.json()
}

export async function getProvider(id: string): Promise<Provider> {
  const res = await fetch(`${getApiBaseUrl()}/providers/${id}`)
  if (!res.ok) throw new Error('Failed to fetch provider')
  return res.json()
}

export async function createProvider(provider: CreateProviderRequest): Promise<Provider> {
  const res = await fetch(`${getApiBaseUrl()}/providers`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(provider)
  })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Failed to create provider')
  }
  return res.json()
}

export async function updateProvider(id: string, provider: UpdateProviderRequest): Promise<Provider> {
  const res = await fetch(`${getApiBaseUrl()}/providers/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(provider)
  })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Failed to update provider')
  }
  return res.json()
}

export async function deleteProvider(id: string): Promise<void> {
  const res = await fetch(`${getApiBaseUrl()}/providers/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Failed to delete provider')
}

export async function testProvider(id: string): Promise<TestResult> {
  const res = await fetch(`${getApiBaseUrl()}/providers/${id}/test`, { method: 'POST' })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Test failed')
  }
  return res.json()
}

// Model Config API
export async function getModels(providerId?: string): Promise<ModelConfig[]> {
  const url = providerId
    ? `${getApiBaseUrl()}/models?provider_id=${providerId}`
    : `${getApiBaseUrl()}/models`
  const res = await fetch(url)
  if (!res.ok) throw new Error('Failed to fetch models')
  return res.json()
}

export async function getModel(id: string): Promise<ModelConfig> {
  const res = await fetch(`${getApiBaseUrl()}/models/${id}`)
  if (!res.ok) throw new Error('Failed to fetch model')
  return res.json()
}

export async function createModel(model: CreateModelConfigRequest): Promise<ModelConfig> {
  const res = await fetch(`${getApiBaseUrl()}/models`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(model)
  })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Failed to create model')
  }
  return res.json()
}

export async function updateModel(id: string, model: UpdateModelConfigRequest): Promise<ModelConfig> {
  const res = await fetch(`${getApiBaseUrl()}/models/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(model)
  })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Failed to update model')
  }
  return res.json()
}

export async function deleteModel(id: string): Promise<void> {
  const res = await fetch(`${getApiBaseUrl()}/models/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Failed to delete model')
}

export async function setDefaultModel(id: string): Promise<void> {
  const res = await fetch(`${getApiBaseUrl()}/models/${id}/default`, { method: 'POST' })
  if (!res.ok) throw new Error('Failed to set default model')
}

export async function testModel(id: string): Promise<TestResult> {
  const res = await fetch(`${getApiBaseUrl()}/models/${id}/test`, { method: 'POST' })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Test failed')
  }
  return res.json()
}

// Session API
export async function getSessions(): Promise<Session[]> {
  const res = await fetch(`${getApiBaseUrl()}/sessions`)
  if (!res.ok) throw new Error('Failed to fetch sessions')
  return res.json()
}

export async function getSession(id: string): Promise<{ session: Session; messages: Message[] }> {
  const res = await fetch(`${getApiBaseUrl()}/sessions/${id}`)
  if (!res.ok) throw new Error('Failed to fetch session')
  return res.json()
}

export async function deleteSession(id: string): Promise<void> {
  const res = await fetch(`${getApiBaseUrl()}/sessions/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Failed to delete session')
}

export async function deleteMessage(sessionId: string, messageId: string): Promise<void> {
  const res = await fetch(`${getApiBaseUrl()}/sessions/${sessionId}/messages/${messageId}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Failed to delete message')
}

// Chat API
export async function chat(request: ChatRequest): Promise<ChatResponse> {
  const res = await fetch(`${getApiBaseUrl()}/chat`, {
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
  const res = await fetch(`${getApiBaseUrl()}/chat`, {
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
  const res = await fetch(`${getApiBaseUrl()}/knowledge?active_only=${activeOnly}&limit=${limit}`)
  if (!res.ok) throw new Error('Failed to fetch knowledge')
  return res.json()
}

export async function createKnowledge(content: string): Promise<Knowledge> {
  const res = await fetch(`${getApiBaseUrl()}/knowledge`, {
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
  const res = await fetch(`${getApiBaseUrl()}/knowledge/${id}`, {
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
  const res = await fetch(`${getApiBaseUrl()}/knowledge/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Failed to delete knowledge')
}
