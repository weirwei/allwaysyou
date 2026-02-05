<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, computed, watch } from 'vue'
import { marked } from 'marked'
import * as api from './api'
import { getApiPort, setApiPort, getServerPort, saveServerPort, getSystemConfigs, updateSystemConfig } from './api'
import type { Message, Session, Provider, ModelConfig, CreateProviderRequest, UpdateProviderRequest, CreateModelConfigRequest, UpdateModelConfigRequest, ConfigType, ProviderType, Knowledge, LLMConfig, CreateConfigRequest, UpdateConfigRequest, SystemConfig } from './api'

// State
const sessions = ref<Session[]>([])
const currentSessionId = ref<string | null>(null)
const messages = ref<Message[]>([])
const inputText = ref('')
const isLoading = ref(false)
const currentView = ref<'chat' | 'settings'>('chat')
const sidebarCollapsed = ref(false)
const sidebarWidth = ref(280)
const isResizing = ref(false)

// Provider & Model state
const providers = ref<Provider[]>([])
const models = ref<ModelConfig[]>([])
const selectedProviderId = ref<string | null>(null)
const providerSearchQuery = ref('')
const showAddProvider = ref(false)
const editingProviderId = ref<string | null>(null)
const showAddModel = ref(false)
const editingModelId = ref<string | null>(null)
const testingProviderId = ref<string | null>(null)
const testingModelId = ref<string | null>(null)
const showApiKey = ref(false)

// Legacy config state (for backward compatibility)
const configs = ref<LLMConfig[]>([])

// Knowledge state
const knowledgeList = ref<Knowledge[]>([])
const showAddKnowledge = ref(false)
const editingKnowledgeId = ref<string | null>(null)
const newKnowledgeContent = ref('')

// Theme state
const currentTheme = ref('default')

// API settings
const apiPort = ref(getApiPort())

// Memory settings
const memoryConfigs = ref<SystemConfig[]>([])
const editingConfigs = ref<Record<string, string>>({})
const themes = [
  { id: 'default', name: 'Default', preview: ['#1A1A2E', '#6366F1', '#06B6D4'] },
  { id: 'monokai', name: 'Monokai', preview: ['#272822', '#f92672', '#a6e22e'] },
  { id: 'solarized', name: 'Solarized Dark', preview: ['#002b36', '#268bd2', '#2aa198'] },
  { id: 'solarized-light', name: 'Solarized Light', preview: ['#fdf6e3', '#268bd2', '#859900'] },
  { id: 'catppuccin-latte', name: 'Catppuccin Latte', preview: ['#eff1f5', '#8839ef', '#179299'] },
  { id: 'rose-peach', name: 'Rosé Peach', preview: ['#FFDCDC', '#e8787a', '#7fb3b3'] },
]

// Settings tab state
type SettingsTab = 'model' | 'system' | 'knowledge' | 'memory' | 'port' | 'theme'
const currentSettingsTab = ref<SettingsTab>('model')
const settingsTabs: { id: SettingsTab; name: string; icon: string }[] = [
  { id: 'model', name: '模型配置', icon: 'M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z' },
  { id: 'system', name: '系统模型', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z' },
  { id: 'knowledge', name: '知识管理', icon: 'M12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2zm1 15h-2v-6h2zm0-8h-2V7h2z' },
  { id: 'memory', name: '记忆设置', icon: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z' },
  { id: 'port', name: '端口设置', icon: 'M13 10V3L4 14h7v7l9-11h-7z' },
  { id: 'theme', name: '主题', icon: 'M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0z' },
]

// System model state
const showAddSystemModel = ref(false)
const editingSystemModelId = ref<string | null>(null)
const systemModelType = ref<'summarize' | 'embedding'>('summarize')

// Provider form
const newProvider = ref<CreateProviderRequest>({
  name: '',
  type: 'openai',
  api_key: '',
  base_url: '',
  enabled: true
})

// Model form
const newModel = ref<CreateModelConfigRequest>({
  provider_id: '',
  model: '',
  max_tokens: 4096,
  temperature: 0.7,
  config_type: 'chat',
  is_default: false
})

// Computed
const selectedProvider = computed(() =>
  providers.value.find(p => p.id === selectedProviderId.value)
)

const filteredProviders = computed(() => {
  if (!providerSearchQuery.value) return providers.value
  const query = providerSearchQuery.value.toLowerCase()
  return providers.value.filter(p =>
    p.name.toLowerCase().includes(query) ||
    p.type.toLowerCase().includes(query)
  )
})

const providerModels = computed(() => {
  if (!selectedProviderId.value) return []
  return models.value.filter(m => m.provider_id === selectedProviderId.value)
})

const chatModels = computed(() => {
  return providerModels.value.filter(m => m.config_type === 'chat')
})

// All models by type (for system models tab)
const summarizeModels = computed(() => {
  return models.value.filter(m => m.config_type === 'summarize')
})

const embeddingModels = computed(() => {
  return models.value.filter(m => m.config_type === 'embedding')
})

const apiUrlPreview = computed(() => {
  const base = newProvider.value.base_url || getDefaultBaseUrl(newProvider.value.type)
  if (!base) return ''
  const cleanBase = base.replace(/\/$/, '')
  return `${cleanBase}/chat/completions`
})

function getDefaultBaseUrl(type: ProviderType): string {
  switch (type) {
    case 'openai': return 'https://api.openai.com/v1'
    case 'claude': return 'https://api.anthropic.com/v1'
    case 'azure': return ''
    case 'ollama': return 'http://localhost:11434/v1'
    case 'custom': return ''
    default: return ''
  }
}

function getProviderIcon(type: ProviderType): string {
  switch (type) {
    case 'openai': return 'O'
    case 'claude': return 'C'
    case 'azure': return 'A'
    case 'ollama': return '🦙'
    case 'custom': return '⚙️'
    default: return '?'
  }
}

// Theme functions
function loadTheme() {
  const savedTheme = localStorage.getItem('appTheme')
  if (savedTheme) currentTheme.value = savedTheme
  applyTheme()
}

function applyTheme() {
  document.documentElement.setAttribute('data-theme', currentTheme.value)
}

function setTheme(themeId: string) {
  currentTheme.value = themeId
  localStorage.setItem('appTheme', themeId)
  applyTheme()
}

// API port functions
async function loadApiPort() {
  try {
    const port = await getServerPort()
    apiPort.value = String(port)
    setApiPort(String(port)) // Sync to localStorage
  } catch (e) {
    // Fallback to localStorage
    apiPort.value = getApiPort()
  }
}

async function saveApiPort() {
  const port = apiPort.value.trim()
  if (!/^\d+$/.test(port) || parseInt(port) < 1 || parseInt(port) > 65535) {
    showToast('请输入有效的端口号 (1-65535)', 'error')
    return
  }
  try {
    await saveServerPort(parseInt(port))
    setApiPort(port) // Also save to localStorage for frontend use
    showToast('API 端口已保存，重启应用后生效', 'success')
  } catch (e: any) {
    showToast(e.message || '保存失败', 'error')
  }
}

// Memory config functions
async function loadMemoryConfigs() {
  try {
    memoryConfigs.value = await getSystemConfigs('memory')
    // Initialize editing values
    editingConfigs.value = {}
    memoryConfigs.value.forEach(config => {
      editingConfigs.value[config.key] = config.value
    })
  } catch (e) {
    console.error('Failed to load memory configs:', e)
  }
}

async function saveMemoryConfig(key: string) {
  const value = editingConfigs.value[key]
  if (!value) return

  try {
    await updateSystemConfig(key, value)
    // Update local value
    const config = memoryConfigs.value.find(c => c.key === key)
    if (config) {
      config.value = value
    }
    showToast('配置已保存', 'success')
  } catch (e: any) {
    showToast(e.message || '保存失败', 'error')
  }
}

async function saveAllMemoryConfigs() {
  try {
    for (const config of memoryConfigs.value) {
      await updateSystemConfig(config.key, editingConfigs.value[config.key])
    }
    await loadMemoryConfigs() // Reload to confirm
    showToast('所有配置已保存', 'success')
  } catch (e: any) {
    showToast(e.message || '保存失败', 'error')
  }
}

// Sidebar resize
function startResize(e: MouseEvent) {
  isResizing.value = true
  document.addEventListener('mousemove', handleResize)
  document.addEventListener('mouseup', stopResize)
}

function handleResize(e: MouseEvent) {
  if (!isResizing.value) return
  const newWidth = e.clientX
  const collapseThreshold = 150
  const minExpandedWidth = 200
  const maxExpandedWidth = 400

  if (sidebarCollapsed.value) {
    if (newWidth >= collapseThreshold) {
      sidebarCollapsed.value = false
      sidebarWidth.value = Math.max(minExpandedWidth, Math.min(newWidth, maxExpandedWidth))
    }
  } else {
    if (newWidth < collapseThreshold) {
      sidebarCollapsed.value = true
    } else {
      sidebarWidth.value = Math.max(minExpandedWidth, Math.min(newWidth, maxExpandedWidth))
    }
  }
}

function stopResize() {
  isResizing.value = false
  document.removeEventListener('mousemove', handleResize)
  document.removeEventListener('mouseup', stopResize)
}

// Window control functions
function windowClose() {
  (window as any).go?.main?.App?.Close?.()
}

function windowMinimize() {
  (window as any).go?.main?.App?.Minimize?.()
}

function windowMaximize() {
  (window as any).go?.main?.App?.Maximize?.()
}

// Global keyboard shortcuts
function handleGlobalKeydown(e: KeyboardEvent) {
  const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0
  const cmdOrCtrl = isMac ? e.metaKey : e.ctrlKey

  if (cmdOrCtrl) {
    switch (e.key.toLowerCase()) {
      case 'w':
        e.preventDefault()
        windowClose()
        break
      case 'm':
        e.preventDefault()
        windowMinimize()
        break
      case 'n':
        e.preventDefault()
        newChat()
        break
      case ',':
        e.preventDefault()
        currentView.value = 'settings'
        break
      case 'b':
        e.preventDefault()
        sidebarCollapsed.value = !sidebarCollapsed.value
        break
    }
  }

  if (e.key === 'Escape') {
    if (currentView.value === 'settings') {
      if (showAddProvider.value) {
        resetProviderForm()
      } else if (showAddModel.value) {
        resetModelForm()
      } else if (showAddKnowledge.value) {
        resetKnowledgeForm()
      } else {
        currentView.value = 'chat'
      }
    }
  }
}

// Toast notification
const toast = ref<{ message: string; type: 'error' | 'success' | 'info' } | null>(null)
let toastTimer: number | null = null

function showToast(message: string, type: 'error' | 'success' | 'info' = 'error') {
  toast.value = { message, type }
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = window.setTimeout(() => {
    toast.value = null
  }, 4000)
}

// Refs
const chatContainer = ref<HTMLElement | null>(null)

// Computed
const currentSession = computed(() =>
  sessions.value.find(s => s.id === currentSessionId.value)
)

// Methods
async function loadSessions() {
  try {
    sessions.value = await api.getSessions()
  } catch (e) {
    console.error('Failed to load sessions:', e)
  }
}

async function loadProviders() {
  try {
    providers.value = await api.getProviders()
    // Auto-select first provider if none selected
    if (providers.value.length > 0 && !selectedProviderId.value) {
      selectedProviderId.value = providers.value[0].id
    }
  } catch (e) {
    console.error('Failed to load providers:', e)
  }
}

async function loadModels() {
  try {
    models.value = await api.getModels()
  } catch (e) {
    console.error('Failed to load models:', e)
  }
}

async function loadConfigs() {
  try {
    configs.value = await api.getConfigs()
  } catch (e) {
    console.error('Failed to load configs:', e)
  }
}

async function selectSession(session: Session) {
  currentSessionId.value = session.id
  try {
    const data = await api.getSession(session.id)
    messages.value = data.messages || []
    scrollToBottom()
  } catch (e) {
    console.error('Failed to load session:', e)
  }
}

async function deleteSession(id: string, event: Event) {
  event.stopPropagation()
  event.preventDefault()

  try {
    await api.deleteSession(id)
    sessions.value = sessions.value.filter(s => s.id !== id)
    if (currentSessionId.value === id) {
      currentSessionId.value = null
      messages.value = []
    }
    showToast('Conversation deleted', 'success')
  } catch (e) {
    console.error('Failed to delete session:', e)
    showToast('Failed to delete conversation')
  }
}

async function deleteMessage(index: number) {
  const msg = messages.value[index]
  if (!msg) return

  if (!msg.id) {
    messages.value.splice(index, 1)
    return
  }

  if (!currentSessionId.value) return

  try {
    await api.deleteMessage(currentSessionId.value, msg.id)
    messages.value.splice(index, 1)
  } catch (e) {
    console.error('Failed to delete message:', e)
    showToast('Failed to delete message')
  }
}

function newChat() {
  currentSessionId.value = null
  messages.value = []
  inputText.value = ''
}

async function sendMessage() {
  if (!inputText.value.trim() || isLoading.value) return

  // Check if we have any config (new or legacy)
  const hasNewConfig = models.value.some(m => m.config_type === 'chat')
  const hasLegacyConfig = configs.value.some(c => c.config_type === 'chat')

  if (!hasNewConfig && !hasLegacyConfig) {
    showToast('Please add an LLM configuration first', 'info')
    currentView.value = 'settings'
    return
  }

  const userMessage: Message = { role: 'user', content: inputText.value.trim() }
  messages.value.push(userMessage)
  inputText.value = ''
  isLoading.value = true
  scrollToBottom()

  try {
    messages.value.push({ role: 'assistant', content: '' })
    const assistantIndex = messages.value.length - 1

    const { stream, sessionId } = await api.chatStream({
      session_id: currentSessionId.value || undefined,
      messages: [userMessage]
    })

    if (sessionId && !currentSessionId.value) {
      currentSessionId.value = sessionId
    }

    for await (const chunk of stream) {
      if (chunk.delta) {
        messages.value[assistantIndex].content += chunk.delta
        scrollToBottom()
      }
      if (chunk.done) {
        break
      }
    }

    await loadSessions()
  } catch (e: any) {
    console.error('Chat failed:', e)
    messages.value.pop()
    showToast(e.message || 'Failed to send message')
  } finally {
    isLoading.value = false
  }
}

// Provider management
function selectProvider(id: string) {
  selectedProviderId.value = id
  showAddProvider.value = false
  showAddModel.value = false
}

function resetProviderForm() {
  newProvider.value = {
    name: '',
    type: 'openai',
    api_key: '',
    base_url: '',
    enabled: true
  }
  editingProviderId.value = null
  showAddProvider.value = false
  showApiKey.value = false
}

function startAddProvider() {
  newProvider.value = {
    name: '',
    type: 'openai',
    api_key: '',
    base_url: '',
    enabled: true
  }
  editingProviderId.value = null
  showApiKey.value = false
  showAddProvider.value = true
}

function editProvider(provider: Provider) {
  editingProviderId.value = provider.id
  newProvider.value = {
    name: provider.name,
    type: provider.type,
    api_key: '',
    base_url: provider.base_url || '',
    enabled: provider.enabled
  }
  showAddProvider.value = true
}

async function saveProvider() {
  if (!newProvider.value.name) {
    showToast('请填写供应商名称', 'info')
    return
  }

  if (!editingProviderId.value && !newProvider.value.api_key && newProvider.value.type !== 'ollama') {
    showToast('请填写 API Key', 'info')
    return
  }

  const providerToSave = { ...newProvider.value }
  if (providerToSave.type === 'ollama' && !providerToSave.api_key) {
    providerToSave.api_key = 'ollama'
  }
  if (!providerToSave.base_url) {
    providerToSave.base_url = getDefaultBaseUrl(providerToSave.type)
  }

  try {
    if (editingProviderId.value) {
      const updateData: UpdateProviderRequest = {
        name: providerToSave.name,
        type: providerToSave.type,
        base_url: providerToSave.base_url,
        enabled: providerToSave.enabled
      }
      if (providerToSave.api_key) {
        updateData.api_key = providerToSave.api_key
      }
      await api.updateProvider(editingProviderId.value, updateData)
      showToast('供应商已更新', 'success')
    } else {
      const created = await api.createProvider(providerToSave)
      selectedProviderId.value = created.id
      showToast('供应商已添加', 'success')
    }
    await loadProviders()
    resetProviderForm()
  } catch (e: any) {
    showToast(e.message || '保存失败')
  }
}

async function removeProvider(id: string) {
  try {
    await api.deleteProvider(id)
    await Promise.all([loadProviders(), loadModels()])
    if (selectedProviderId.value === id) {
      selectedProviderId.value = providers.value.length > 0 ? providers.value[0].id : null
    }
    showToast('供应商已删除', 'success')
  } catch (e) {
    console.error('Failed to delete provider:', e)
    showToast('删除失败')
  }
}

async function toggleProviderEnabled(id: string, enabled: boolean) {
  try {
    await api.updateProvider(id, { enabled })
    await loadProviders()
  } catch (e) {
    console.error('Failed to toggle provider:', e)
  }
}

async function testProvider(id: string) {
  testingProviderId.value = id
  try {
    const result = await api.testProvider(id)
    if (result.success) {
      showToast('连接成功', 'success')
    } else {
      showToast(result.error || '连接失败')
    }
  } catch (e: any) {
    showToast(e.message || '测试失败')
  } finally {
    testingProviderId.value = null
  }
}

// Model management
function resetModelForm() {
  newModel.value = {
    provider_id: selectedProviderId.value || '',
    model: '',
    max_tokens: 4096,
    temperature: 0.7,
    config_type: 'chat',
    is_default: false
  }
  editingModelId.value = null
  showAddModel.value = false
}

function startAddModel() {
  newModel.value = {
    provider_id: selectedProviderId.value || '',
    model: '',
    max_tokens: 4096,
    temperature: 0.7,
    config_type: 'chat',
    is_default: !chatModels.value.length
  }
  editingModelId.value = null
  showAddModel.value = true
}

// System model functions
function startAddSystemModel(type: 'summarize' | 'embedding') {
  systemModelType.value = type
  const existingModels = type === 'summarize' ? summarizeModels.value : embeddingModels.value
  newModel.value = {
    provider_id: providers.value.length > 0 ? providers.value[0].id : '',
    model: '',
    max_tokens: type === 'embedding' ? 8192 : 4096,
    temperature: type === 'embedding' ? 0 : 0.7,
    config_type: type,
    is_default: !existingModels.length
  }
  editingSystemModelId.value = null
  showAddSystemModel.value = true
}

function editSystemModel(model: ModelConfig) {
  systemModelType.value = model.config_type as 'summarize' | 'embedding'
  editingSystemModelId.value = model.id
  newModel.value = {
    provider_id: model.provider_id,
    model: model.model,
    max_tokens: model.max_tokens,
    temperature: model.temperature,
    config_type: model.config_type,
    is_default: model.is_default
  }
  showAddSystemModel.value = true
}

function resetSystemModelForm() {
  showAddSystemModel.value = false
  editingSystemModelId.value = null
}

async function saveSystemModel() {
  if (!newModel.value.model) {
    showToast('请填写模型名称', 'info')
    return
  }
  if (!newModel.value.provider_id) {
    showToast('请选择供应商', 'info')
    return
  }

  try {
    if (editingSystemModelId.value) {
      const updateData: UpdateModelConfigRequest = {
        model: newModel.value.model,
        max_tokens: newModel.value.max_tokens,
        temperature: newModel.value.temperature,
        config_type: newModel.value.config_type,
        is_default: newModel.value.is_default
      }
      await api.updateModel(editingSystemModelId.value, updateData)
      showToast('模型已更新', 'success')
    } else {
      await api.createModel(newModel.value)
      showToast('模型已添加', 'success')
    }
    await loadModels()
    resetSystemModelForm()
  } catch (e: any) {
    showToast(e.message || '保存失败')
  }
}

function editModel(model: ModelConfig) {
  editingModelId.value = model.id
  newModel.value = {
    provider_id: model.provider_id,
    model: model.model,
    max_tokens: model.max_tokens,
    temperature: model.temperature,
    config_type: model.config_type,
    is_default: model.is_default
  }
  showAddModel.value = true
}

async function saveModel() {
  if (!newModel.value.model) {
    showToast('请填写模型名称', 'info')
    return
  }

  try {
    if (editingModelId.value) {
      const updateData: UpdateModelConfigRequest = {
        model: newModel.value.model,
        max_tokens: newModel.value.max_tokens,
        temperature: newModel.value.temperature,
        config_type: newModel.value.config_type,
        is_default: newModel.value.is_default
      }
      await api.updateModel(editingModelId.value, updateData)
      showToast('模型已更新', 'success')
    } else {
      await api.createModel(newModel.value)
      showToast('模型已添加', 'success')
    }
    await loadModels()
    resetModelForm()
  } catch (e: any) {
    showToast(e.message || '保存失败')
  }
}

async function removeModel(id: string) {
  try {
    await api.deleteModel(id)
    await loadModels()
    showToast('模型已删除', 'success')
  } catch (e) {
    console.error('Failed to delete model:', e)
    showToast('删除失败')
  }
}

async function setDefaultModel(id: string) {
  try {
    await api.setDefaultModel(id)
    await loadModels()
    showToast('已设为默认', 'success')
  } catch (e) {
    console.error('Failed to set default model:', e)
    showToast('设置失败')
  }
}

async function testModel(id: string) {
  testingModelId.value = id
  try {
    const result = await api.testModel(id)
    if (result.success) {
      showToast('测试成功', 'success')
    } else {
      showToast(result.error || '测试失败')
    }
  } catch (e: any) {
    showToast(e.message || '测试失败')
  } finally {
    testingModelId.value = null
  }
}

// Knowledge management
async function loadKnowledge() {
  try {
    knowledgeList.value = await api.getKnowledge(true, 100)
  } catch (e) {
    console.error('Failed to load knowledge:', e)
  }
}

function resetKnowledgeForm() {
  newKnowledgeContent.value = ''
  editingKnowledgeId.value = null
  showAddKnowledge.value = false
}

function editKnowledge(knowledge: Knowledge) {
  editingKnowledgeId.value = knowledge.id
  newKnowledgeContent.value = knowledge.content
  showAddKnowledge.value = true
}

async function saveKnowledge() {
  if (!newKnowledgeContent.value.trim()) {
    showToast('请输入记忆内容', 'info')
    return
  }

  try {
    if (editingKnowledgeId.value) {
      await api.updateKnowledge(editingKnowledgeId.value, newKnowledgeContent.value)
      showToast('记忆已更新', 'success')
    } else {
      await api.createKnowledge(newKnowledgeContent.value)
      showToast('记忆已添加', 'success')
    }
    await loadKnowledge()
    resetKnowledgeForm()
  } catch (e: any) {
    showToast(e.message || '保存失败')
  }
}

async function removeKnowledge(id: string) {
  try {
    await api.deleteKnowledge(id)
    await loadKnowledge()
    showToast('记忆已删除', 'success')
  } catch (e) {
    console.error('Failed to delete knowledge:', e)
    showToast('删除失败')
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (chatContainer.value) {
      chatContainer.value.scrollTop = chatContainer.value.scrollHeight
    }
  })
}

function renderMarkdown(content: string): string {
  return marked(content) as string
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}

// Watch for tab changes to reload data
watch(currentSettingsTab, (newTab) => {
  if (newTab === 'knowledge') {
    loadKnowledge()
  } else if (newTab === 'memory') {
    loadMemoryConfigs()
  }
})

// Lifecycle
onMounted(async () => {
  loadTheme()
  loadApiPort()
  window.addEventListener('keydown', handleGlobalKeydown)
  await Promise.all([loadSessions(), loadProviders(), loadModels(), loadConfigs(), loadKnowledge()])
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleGlobalKeydown)
})
</script>

<template>
  <div class="app-container" :class="{ 'sidebar-collapsed': sidebarCollapsed, 'is-resizing': isResizing }">
    <!-- Sidebar -->
    <div class="sidebar" :style="{ width: sidebarCollapsed ? '100px' : sidebarWidth + 'px' }">
      <div class="sidebar-titlebar">
        <div class="titlebar-drag-region"></div>
      </div>

      <div class="sidebar-content">
        <div class="sidebar-brand">
          <div class="brand-logo">
            <img src="./assets/logo.svg" alt="AllWaysYou" width="64" height="64" />
          </div>
          <span v-if="!sidebarCollapsed" class="brand-name">AllWaysYou</span>
          <button class="sidebar-toggle" @click.stop="sidebarCollapsed = !sidebarCollapsed" :title="sidebarCollapsed ? 'Expand' : 'Collapse'">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
              <line x1="9" y1="3" x2="9" y2="21"></line>
              <polyline v-if="sidebarCollapsed" points="14 9 17 12 14 15"></polyline>
              <polyline v-else points="17 9 14 12 17 15"></polyline>
            </svg>
          </button>
        </div>

        <button class="new-chat-btn" @click="newChat" :title="sidebarCollapsed ? 'New Chat' : ''">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          <span v-if="!sidebarCollapsed">New Chat</span>
        </button>

        <div class="session-list">
          <div
            v-for="session in sessions"
            :key="session.id"
            class="session-item"
            :class="{ active: session.id === currentSessionId }"
            @click="selectSession(session)"
            :title="sidebarCollapsed ? (session.title || 'New Chat') : ''"
          >
            <span class="session-icon">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
              </svg>
            </span>
            <span class="session-title" v-if="!sidebarCollapsed">{{ session.title || 'New Chat' }}</span>
            <button class="delete-btn" v-if="!sidebarCollapsed" @click="deleteSession(session.id, $event)">
              &times;
            </button>
          </div>
        </div>

        <button class="settings-btn" :class="{ active: currentView === 'settings' }" @click="currentView = 'settings'" :title="sidebarCollapsed ? 'Settings' : ''">
          <span v-if="!sidebarCollapsed">Settings</span>
        </button>
      </div>

      <div
        class="sidebar-resize-handle"
        @mousedown="startResize"
      ></div>
    </div>

    <!-- Main content -->
    <div class="main-content">
      <!-- Chat View -->
      <template v-if="currentView === 'chat'">
        <div ref="chatContainer" class="chat-container">
          <template v-if="messages.length > 0">
            <div
              v-for="(msg, index) in messages"
              :key="msg.id || index"
              class="message-wrapper"
              :class="msg.role"
            >
              <div class="message" :class="msg.role">
                <div v-if="msg.role === 'assistant'" v-html="renderMarkdown(msg.content)"></div>
                <template v-else>{{ msg.content }}</template>
              </div>
              <button
                class="message-delete-btn"
                @click="deleteMessage(index)"
                title="Delete message"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="3 6 5 6 21 6"></polyline>
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                </svg>
              </button>
            </div>

            <div v-if="isLoading && !messages[messages.length - 1]?.content" class="typing-indicator">
              <span></span>
              <span></span>
              <span></span>
            </div>
          </template>

          <div v-else class="empty-state">
            <img src="./assets/logo.svg" alt="AllWaysYou" class="empty-state-logo" />
            <h2>Hello, I'm here to help</h2>
            <p>Ask me anything or select a previous conversation to continue</p>
          </div>
        </div>

        <div class="input-area">
          <div class="input-container">
            <textarea
              v-model="inputText"
              placeholder="Type your message..."
              @keydown="handleKeydown"
              :disabled="isLoading"
              rows="1"
            ></textarea>
            <button @click="sendMessage" :disabled="isLoading || !inputText.trim()">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="22" y1="2" x2="11" y2="13"></line>
                <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
              </svg>
              Send
            </button>
          </div>
        </div>
      </template>

      <!-- Settings View -->
      <template v-else-if="currentView === 'settings'">
        <div class="settings-page">
          <div class="settings-titlebar"></div>
          <div class="settings-header">
            <button class="back-btn" @click="currentView = 'chat'" title="Back to chat">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="15 18 9 12 15 6"></polyline>
              </svg>
            </button>
            <h1>Settings</h1>
          </div>

          <!-- Settings Tabs -->
          <div class="settings-tabs">
            <button
              v-for="tab in settingsTabs"
              :key="tab.id"
              class="settings-tab"
              :class="{ active: currentSettingsTab === tab.id }"
              @click="currentSettingsTab = tab.id"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path :d="tab.icon"></path>
              </svg>
              {{ tab.name }}
            </button>
          </div>

          <div class="settings-content">
            <!-- Model Configuration Tab (New Provider+Model UI) -->
            <template v-if="currentSettingsTab === 'model'">
              <div class="model-config-layout">
                <!-- Left: Provider List -->
                <div class="provider-list-panel">
                  <div class="provider-search">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <circle cx="11" cy="11" r="8"></circle>
                      <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
                    </svg>
                    <input v-model="providerSearchQuery" placeholder="搜索供应商..." />
                  </div>

                  <div class="provider-items">
                    <div
                      v-for="provider in filteredProviders"
                      :key="provider.id"
                      class="provider-item"
                      :class="{ active: provider.id === selectedProviderId }"
                      @click="selectProvider(provider.id)"
                    >
                      <span class="provider-icon">{{ getProviderIcon(provider.type) }}</span>
                      <span class="provider-name">{{ provider.name }}</span>
                      <label class="provider-toggle" @click.stop>
                        <input
                          type="checkbox"
                          :checked="provider.enabled"
                          @change="toggleProviderEnabled(provider.id, ($event.target as HTMLInputElement).checked)"
                        />
                        <span class="toggle-slider"></span>
                      </label>
                    </div>
                  </div>

                  <button class="btn-add-provider" @click="startAddProvider()">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                      <line x1="12" y1="5" x2="12" y2="19"></line>
                      <line x1="5" y1="12" x2="19" y2="12"></line>
                    </svg>
                    添加供应商
                  </button>
                </div>

                <!-- Right: Provider Details & Models -->
                <div class="provider-detail-panel">
                  <template v-if="showAddProvider">
                    <!-- Add/Edit Provider Form -->
                    <div class="form-header">
                      <button class="back-btn" @click="resetProviderForm" title="Back">
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <polyline points="15 18 9 12 15 6"></polyline>
                        </svg>
                      </button>
                      <h2>{{ editingProviderId ? '编辑供应商' : '添加供应商' }}</h2>
                    </div>

                    <div class="provider-form">
                      <div class="form-group">
                        <label>名称 *</label>
                        <input v-model="newProvider.name" placeholder="例如: My OpenAI" />
                      </div>

                      <div class="form-group">
                        <label>类型 *</label>
                        <select v-model="newProvider.type">
                          <option value="openai">OpenAI</option>
                          <option value="claude">Claude (Anthropic)</option>
                          <option value="azure">Azure OpenAI</option>
                          <option value="ollama">Ollama (本地)</option>
                          <option value="custom">自定义 (OpenAI 兼容)</option>
                        </select>
                      </div>

                      <div class="form-group" v-if="newProvider.type !== 'ollama'">
                        <label>API 密钥 {{ editingProviderId ? '(留空保持不变)' : '*' }}</label>
                        <div class="api-key-input">
                          <input
                            v-model="newProvider.api_key"
                            :type="showApiKey ? 'text' : 'password'"
                            :placeholder="editingProviderId ? '••••••••' : 'sk-...'"
                          />
                          <button class="btn-icon" @click="showApiKey = !showApiKey" type="button">
                            <svg v-if="showApiKey" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                              <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"></path>
                              <line x1="1" y1="1" x2="23" y2="23"></line>
                            </svg>
                            <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                              <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path>
                              <circle cx="12" cy="12" r="3"></circle>
                            </svg>
                          </button>
                        </div>
                      </div>

                      <div class="form-group">
                        <label>API 地址</label>
                        <input v-model="newProvider.base_url" :placeholder="getDefaultBaseUrl(newProvider.type)" />
                        <span v-if="apiUrlPreview" class="url-preview">预览: {{ apiUrlPreview }}</span>
                      </div>

                      <div class="form-actions">
                        <button class="btn-secondary" @click="resetProviderForm">取消</button>
                        <button class="btn-primary" @click="saveProvider">{{ editingProviderId ? '更新' : '保存' }}</button>
                      </div>
                    </div>
                  </template>

                  <template v-else-if="selectedProvider">
                    <!-- Provider Header -->
                    <div class="provider-header">
                      <div class="provider-title">
                        <span class="provider-icon-large">{{ getProviderIcon(selectedProvider.type) }}</span>
                        <h2>{{ selectedProvider.name }}</h2>
                      </div>
                      <div class="provider-actions">
                        <button
                          class="btn-icon btn-test"
                          :class="{ testing: testingProviderId === selectedProvider.id }"
                          @click="testProvider(selectedProvider.id)"
                          :disabled="testingProviderId === selectedProvider.id"
                          title="测试连接"
                        >
                          <svg v-if="testingProviderId !== selectedProvider.id" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polygon points="5 3 19 12 5 21 5 3"></polygon>
                          </svg>
                          <svg v-else class="spin" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="12" y1="2" x2="12" y2="6"></line>
                            <line x1="12" y1="18" x2="12" y2="22"></line>
                            <line x1="4.93" y1="4.93" x2="7.76" y2="7.76"></line>
                            <line x1="16.24" y1="16.24" x2="19.07" y2="19.07"></line>
                            <line x1="2" y1="12" x2="6" y2="12"></line>
                            <line x1="18" y1="12" x2="22" y2="12"></line>
                            <line x1="4.93" y1="19.07" x2="7.76" y2="16.24"></line>
                            <line x1="16.24" y1="7.76" x2="19.07" y2="4.93"></line>
                          </svg>
                        </button>
                        <button class="btn-icon" @click="editProvider(selectedProvider)" title="编辑">
                          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                            <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                          </svg>
                        </button>
                        <button class="btn-icon btn-danger" @click="removeProvider(selectedProvider.id)" title="删除">
                          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="3 6 5 6 21 6"></polyline>
                            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                          </svg>
                        </button>
                        <label class="provider-toggle">
                          <input
                            type="checkbox"
                            :checked="selectedProvider.enabled"
                            @change="toggleProviderEnabled(selectedProvider.id, ($event.target as HTMLInputElement).checked)"
                          />
                          <span class="toggle-slider"></span>
                        </label>
                      </div>
                    </div>

                    <!-- Provider Info -->
                    <div class="provider-info">
                      <div class="info-row">
                        <span class="info-label">类型</span>
                        <span class="info-value">{{ selectedProvider.type }}</span>
                      </div>
                      <div class="info-row">
                        <span class="info-label">API 地址</span>
                        <span class="info-value">{{ selectedProvider.base_url || getDefaultBaseUrl(selectedProvider.type) }}</span>
                      </div>
                      <div class="info-row">
                        <span class="info-label">API 密钥</span>
                        <span class="info-value">{{ selectedProvider.has_api_key ? '••••••••' : '未设置' }}</span>
                      </div>
                    </div>

                    <div class="models-divider"></div>

                    <!-- Models Section -->
                    <template v-if="showAddModel">
                      <!-- Add/Edit Model Form -->
                      <div class="form-header">
                        <button class="back-btn" @click="resetModelForm" title="Back">
                          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="15 18 9 12 15 6"></polyline>
                          </svg>
                        </button>
                        <h2>{{ editingModelId ? '编辑模型' : '添加模型' }}</h2>
                      </div>

                      <div class="model-form">
                        <div class="form-group">
                          <label>模型名称 *</label>
                          <input v-model="newModel.model" placeholder="例如: gpt-4o-mini" />
                        </div>

                        <div class="form-group">
                          <label>用途 *</label>
                          <select v-model="newModel.config_type">
                            <option value="chat">聊天模型</option>
                            <option value="summarize">总结模型</option>
                            <option value="embedding">向量模型</option>
                          </select>
                        </div>

                        <div class="form-group" v-if="newModel.config_type !== 'embedding'">
                          <label>最大 Tokens</label>
                          <input v-model.number="newModel.max_tokens" type="number" />
                        </div>

                        <div class="form-group" v-if="newModel.config_type !== 'embedding'">
                          <label>温度 (0-2)</label>
                          <input v-model.number="newModel.temperature" type="number" step="0.1" min="0" max="2" />
                        </div>

                        <div class="form-group">
                          <label class="checkbox-label">
                            <input type="checkbox" v-model="newModel.is_default" />
                            <span>设为该类型的默认模型</span>
                          </label>
                        </div>

                        <div class="form-actions">
                          <button class="btn-secondary" @click="resetModelForm">取消</button>
                          <button class="btn-primary" @click="saveModel">{{ editingModelId ? '更新' : '保存' }}</button>
                        </div>
                      </div>
                    </template>

                    <template v-else>
                      <div class="models-header">
                        <h3>聊天模型 ({{ chatModels.length }})</h3>
                        <button class="btn-add-model" @click="startAddModel()">
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                            <line x1="12" y1="5" x2="12" y2="19"></line>
                            <line x1="5" y1="12" x2="19" y2="12"></line>
                          </svg>
                          添加模型
                        </button>
                      </div>

                      <div class="models-list">
                        <div
                          v-for="model in chatModels"
                          :key="model.id"
                          class="model-item"
                        >
                          <span class="model-name">
                            {{ model.model }}
                            <span v-if="model.is_default" class="default-badge">默认</span>
                          </span>
                          <div class="model-actions">
                            <button
                              class="btn-icon btn-test"
                              :class="{ testing: testingModelId === model.id }"
                              @click="testModel(model.id)"
                              :disabled="testingModelId === model.id"
                              title="测试"
                            >
                              <svg v-if="testingModelId !== model.id" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <polygon points="5 3 19 12 5 21 5 3"></polygon>
                              </svg>
                              <svg v-else class="spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <line x1="12" y1="2" x2="12" y2="6"></line>
                                <line x1="12" y1="18" x2="12" y2="22"></line>
                              </svg>
                            </button>
                            <button class="btn-icon" @click="editModel(model)" title="编辑">
                              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                              </svg>
                            </button>
                            <button class="btn-icon btn-danger" @click="removeModel(model.id)" title="删除">
                              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <polyline points="3 6 5 6 21 6"></polyline>
                                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                              </svg>
                            </button>
                            <button v-if="!model.is_default" class="btn-default" @click="setDefaultModel(model.id)" title="设为默认">
                              设为默认
                            </button>
                          </div>
                        </div>

                        <p v-if="chatModels.length === 0" class="empty-models-text">
                          暂无聊天模型，点击上方按钮添加
                        </p>
                      </div>
                    </template>
                  </template>

                  <template v-else>
                    <div class="no-provider-selected">
                      <p>请从左侧选择或添加一个供应商</p>
                    </div>
                  </template>
                </div>
              </div>
            </template>

            <!-- System Models Tab -->
            <template v-else-if="currentSettingsTab === 'system'">
              <div class="system-models-container">
                <template v-if="showAddSystemModel">
                  <!-- Add/Edit System Model Form -->
                  <div class="form-header">
                    <button class="back-btn" @click="resetSystemModelForm" title="Back">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="15 18 9 12 15 6"></polyline>
                      </svg>
                    </button>
                    <h2>{{ editingSystemModelId ? '编辑' : '添加' }}{{ systemModelType === 'summarize' ? '总结' : '向量' }}模型</h2>
                  </div>

                  <div class="system-model-form">
                    <div class="form-group">
                      <label>供应商 *</label>
                      <select v-model="newModel.provider_id">
                        <option value="" disabled>选择供应商</option>
                        <option v-for="p in providers" :key="p.id" :value="p.id">{{ p.name }}</option>
                      </select>
                    </div>

                    <div class="form-group">
                      <label>模型名称 *</label>
                      <input v-model="newModel.model" :placeholder="systemModelType === 'embedding' ? '例如: text-embedding-3-small' : '例如: gpt-3.5-turbo'" />
                    </div>

                    <div class="form-group" v-if="systemModelType !== 'embedding'">
                      <label>最大 Tokens</label>
                      <input v-model.number="newModel.max_tokens" type="number" />
                    </div>

                    <div class="form-group" v-if="systemModelType !== 'embedding'">
                      <label>温度 (0-2)</label>
                      <input v-model.number="newModel.temperature" type="number" step="0.1" min="0" max="2" />
                    </div>

                    <div class="form-group">
                      <label class="checkbox-label">
                        <input type="checkbox" v-model="newModel.is_default" />
                        <span>设为默认{{ systemModelType === 'summarize' ? '总结' : '向量' }}模型</span>
                      </label>
                    </div>

                    <div class="form-actions">
                      <button class="btn-secondary" @click="resetSystemModelForm">取消</button>
                      <button class="btn-primary" @click="saveSystemModel">{{ editingSystemModelId ? '更新' : '保存' }}</button>
                    </div>
                  </div>
                </template>

                <template v-else>
                  <!-- Summarize Models Section -->
                  <div class="system-model-section">
                    <div class="system-model-header">
                      <div class="system-model-title">
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                          <path d="M14 2v6h6"></path>
                          <line x1="16" y1="13" x2="8" y2="13"></line>
                          <line x1="16" y1="17" x2="8" y2="17"></line>
                        </svg>
                        <h3>总结模型</h3>
                      </div>
                      <p class="system-model-desc">用于生成对话摘要和标题</p>
                      <button class="btn-add-model" @click="startAddSystemModel('summarize')">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                          <line x1="12" y1="5" x2="12" y2="19"></line>
                          <line x1="5" y1="12" x2="19" y2="12"></line>
                        </svg>
                        添加
                      </button>
                    </div>
                    <div class="system-model-list">
                      <div v-for="model in summarizeModels" :key="model.id" class="model-item">
                        <div class="model-info">
                          <span class="model-name">
                            {{ model.model }}
                            <span v-if="model.is_default" class="default-badge">默认</span>
                          </span>
                          <span class="model-provider">{{ providers.find(p => p.id === model.provider_id)?.name || '未知供应商' }}</span>
                        </div>
                        <div class="model-actions">
                          <button
                            class="btn-icon btn-test"
                            :class="{ testing: testingModelId === model.id }"
                            @click="testModel(model.id)"
                            :disabled="testingModelId === model.id"
                            title="测试"
                          >
                            <svg v-if="testingModelId !== model.id" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                              <polygon points="5 3 19 12 5 21 5 3"></polygon>
                            </svg>
                            <svg v-else class="spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                              <line x1="12" y1="2" x2="12" y2="6"></line>
                              <line x1="12" y1="18" x2="12" y2="22"></line>
                            </svg>
                          </button>
                          <button class="btn-icon" @click="editSystemModel(model)" title="编辑">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                              <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                              <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                            </svg>
                          </button>
                          <button class="btn-icon btn-danger" @click="removeModel(model.id)" title="删除">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                              <polyline points="3 6 5 6 21 6"></polyline>
                              <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                            </svg>
                          </button>
                          <button v-if="!model.is_default" class="btn-default" @click="setDefaultModel(model.id)" title="设为默认">
                            设为默认
                          </button>
                        </div>
                      </div>
                      <p v-if="summarizeModels.length === 0" class="empty-models-text">暂未配置总结模型</p>
                    </div>
                  </div>

                  <!-- Embedding Models Section -->
                  <div class="system-model-section">
                    <div class="system-model-header">
                      <div class="system-model-title">
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <path d="M12 2L2 7l10 5 10-5-10-5z"></path>
                          <path d="M2 17l10 5 10-5"></path>
                          <path d="M2 12l10 5 10-5"></path>
                        </svg>
                        <h3>向量模型</h3>
                      </div>
                      <p class="system-model-desc">用于记忆检索和语义搜索</p>
                      <button class="btn-add-model" @click="startAddSystemModel('embedding')">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                          <line x1="12" y1="5" x2="12" y2="19"></line>
                          <line x1="5" y1="12" x2="19" y2="12"></line>
                        </svg>
                        添加
                      </button>
                    </div>
                    <div class="system-model-list">
                      <div v-for="model in embeddingModels" :key="model.id" class="model-item">
                        <div class="model-info">
                          <span class="model-name">
                            {{ model.model }}
                            <span v-if="model.is_default" class="default-badge">默认</span>
                          </span>
                          <span class="model-provider">{{ providers.find(p => p.id === model.provider_id)?.name || '未知供应商' }}</span>
                        </div>
                        <div class="model-actions">
                          <button
                            class="btn-icon btn-test"
                            :class="{ testing: testingModelId === model.id }"
                            @click="testModel(model.id)"
                            :disabled="testingModelId === model.id"
                            title="测试"
                          >
                            <svg v-if="testingModelId !== model.id" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                              <polygon points="5 3 19 12 5 21 5 3"></polygon>
                            </svg>
                            <svg v-else class="spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                              <line x1="12" y1="2" x2="12" y2="6"></line>
                              <line x1="12" y1="18" x2="12" y2="22"></line>
                            </svg>
                          </button>
                          <button class="btn-icon" @click="editSystemModel(model)" title="编辑">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                              <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                              <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                            </svg>
                          </button>
                          <button class="btn-icon btn-danger" @click="removeModel(model.id)" title="删除">
                            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                              <polyline points="3 6 5 6 21 6"></polyline>
                              <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                            </svg>
                          </button>
                          <button v-if="!model.is_default" class="btn-default" @click="setDefaultModel(model.id)" title="设为默认">
                            设为默认
                          </button>
                        </div>
                      </div>
                      <p v-if="embeddingModels.length === 0" class="empty-models-text">暂未配置向量模型</p>
                    </div>
                  </div>
                </template>
              </div>
            </template>

            <!-- Knowledge Management Tab -->
            <template v-else-if="currentSettingsTab === 'knowledge'">
              <template v-if="!showAddKnowledge">
                <div class="knowledge-list">
                  <div
                    v-for="knowledge in knowledgeList"
                    :key="knowledge.id"
                    class="knowledge-item"
                  >
                    <div class="knowledge-content">{{ knowledge.content }}</div>
                    <div class="knowledge-meta">
                      <span class="knowledge-date">{{ new Date(knowledge.created_at).toLocaleDateString() }}</span>
                    </div>
                    <div class="knowledge-actions">
                      <button class="btn-icon" @click="editKnowledge(knowledge)" title="Edit">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                          <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                          <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                        </svg>
                      </button>
                      <button class="btn-icon btn-danger" @click="removeKnowledge(knowledge.id)" title="Delete">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                          <polyline points="3 6 5 6 21 6"></polyline>
                          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                        </svg>
                      </button>
                    </div>
                  </div>

                  <p v-if="knowledgeList.length === 0" class="empty-config-text">
                    暂无记忆条目
                  </p>
                </div>

                <button class="btn-add-config" @click="showAddKnowledge = true; editingKnowledgeId = null; newKnowledgeContent = ''">
                  <span class="icon-wrapper">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                      <line x1="12" y1="5" x2="12" y2="19"></line>
                      <line x1="5" y1="12" x2="19" y2="12"></line>
                    </svg>
                  </span>
                  添加记忆
                </button>
              </template>

              <!-- Add/Edit Knowledge Form -->
              <template v-else>
                <div class="form-header">
                  <button class="back-btn" @click="resetKnowledgeForm" title="Back">
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <polyline points="15 18 9 12 15 6"></polyline>
                    </svg>
                  </button>
                  <h2>{{ editingKnowledgeId ? '编辑记忆' : '添加记忆' }}</h2>
                </div>

                <div class="knowledge-form">
                  <div class="form-group">
                    <label>记忆内容 *</label>
                    <textarea
                      v-model="newKnowledgeContent"
                      placeholder="输入需要记住的内容，例如：用户偏好使用Vue 3和TypeScript"
                      rows="5"
                    ></textarea>
                  </div>

                  <div class="form-actions">
                    <button class="btn-secondary" @click="resetKnowledgeForm">Cancel</button>
                    <button class="btn-primary" @click="saveKnowledge">{{ editingKnowledgeId ? 'Update' : 'Save' }}</button>
                  </div>
                </div>
              </template>
            </template>

            <!-- Memory Settings Tab -->
            <template v-else-if="currentSettingsTab === 'memory'">
              <div class="memory-settings-container">
                <div class="settings-section">
                  <h3 class="settings-section-title">记忆系统配置</h3>
                  <div class="memory-configs-list">
                    <div v-for="config in memoryConfigs" :key="config.key" class="config-item">
                      <div class="config-info">
                        <label class="config-label">{{ config.label }}</label>
                        <span class="config-hint">{{ config.hint }}</span>
                      </div>
                      <div class="config-input-group">
                        <input
                          v-model="editingConfigs[config.key]"
                          :type="config.type === 'number' ? 'number' : 'text'"
                          :step="config.type === 'number' ? '0.01' : undefined"
                          class="config-input"
                        />
                        <button class="btn-icon btn-save" @click="saveMemoryConfig(config.key)" title="保存">
                          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M19 21H5a2 2 0 01-2-2V5a2 2 0 012-2h11l5 5v11a2 2 0 01-2 2z"></path>
                            <polyline points="17 21 17 13 7 13 7 21"></polyline>
                            <polyline points="7 3 7 8 15 8"></polyline>
                          </svg>
                        </button>
                      </div>
                    </div>
                  </div>
                  <div class="form-actions" style="margin-top: var(--space-lg);">
                    <button class="btn-primary" @click="saveAllMemoryConfigs">保存所有配置</button>
                  </div>
                </div>
              </div>
            </template>

            <!-- Port Settings Tab -->
            <template v-else-if="currentSettingsTab === 'port'">
              <div class="port-settings-container">
                <div class="settings-section">
                  <h3 class="settings-section-title">端口设置</h3>
                  <div class="api-port-setting">
                    <div class="form-group">
                      <label>后端 API 端口</label>
                      <div class="api-port-input">
                        <input
                          v-model="apiPort"
                          type="text"
                          placeholder="18080"
                          @keyup.enter="saveApiPort"
                        />
                        <button class="btn-primary" @click="saveApiPort">保存</button>
                      </div>
                      <span class="form-hint">修改后需重启应用生效</span>
                    </div>
                  </div>
                </div>
              </div>
            </template>

            <!-- Theme Settings Tab -->
            <template v-else-if="currentSettingsTab === 'theme'">
              <div class="theme-list">
                <button
                  v-for="theme in themes"
                  :key="theme.id"
                  class="theme-option"
                  :class="{ active: currentTheme === theme.id }"
                  @click="setTheme(theme.id)"
                >
                  <div class="theme-preview">
                    <div class="theme-preview-bg" :style="{ backgroundColor: theme.preview[0] }">
                      <div class="theme-preview-accent" :style="{ backgroundColor: theme.preview[1] }"></div>
                      <div class="theme-preview-secondary" :style="{ backgroundColor: theme.preview[2] }"></div>
                    </div>
                  </div>
                  <span class="theme-name">{{ theme.name }}</span>
                  <svg v-if="currentTheme === theme.id" class="theme-check" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                    <polyline points="20 6 9 17 4 12"></polyline>
                  </svg>
                </button>
              </div>
            </template>
          </div>
        </div>
      </template>
    </div>
  </div>

  <!-- Toast Notification -->
  <Transition name="toast">
    <div v-if="toast" class="toast" :class="toast.type">
      <span class="toast-icon">
        <svg v-if="toast.type === 'error'" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="15" y1="9" x2="9" y2="15"></line>
          <line x1="9" y1="9" x2="15" y2="15"></line>
        </svg>
        <svg v-else-if="toast.type === 'success'" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
          <polyline points="22 4 12 14.01 9 11.01"></polyline>
        </svg>
        <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="12" y1="16" x2="12" y2="12"></line>
          <line x1="12" y1="8" x2="12.01" y2="8"></line>
        </svg>
      </span>
      <span class="toast-message">{{ toast.message }}</span>
      <button class="toast-close" @click="toast = null">&times;</button>
    </div>
  </Transition>
</template>
