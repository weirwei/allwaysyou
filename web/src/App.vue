<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, computed, watch } from 'vue'
import { marked } from 'marked'
import * as api from './api'
import type { Message, Session, LLMConfig, CreateConfigRequest, UpdateConfigRequest, ConfigType, Knowledge } from './api'

// State
const sessions = ref<Session[]>([])
const currentSessionId = ref<string | null>(null)
const messages = ref<Message[]>([])
const inputText = ref('')
const isLoading = ref(false)
const currentView = ref<'chat' | 'settings'>('chat')
const configs = ref<LLMConfig[]>([])
const showAddConfig = ref(false)
const editingConfigId = ref<string | null>(null)
const testingConfigId = ref<string | null>(null)
const sidebarCollapsed = ref(false)
const sidebarWidth = ref(280)
const isResizing = ref(false)

// Knowledge state
const knowledgeList = ref<Knowledge[]>([])
const showAddKnowledge = ref(false)
const editingKnowledgeId = ref<string | null>(null)
const newKnowledgeContent = ref('')

// Theme state
const currentTheme = ref('default')
const themes = [
  { id: 'default', name: 'Default', preview: ['#1A1A2E', '#6366F1', '#06B6D4'] },
  { id: 'monokai', name: 'Monokai', preview: ['#272822', '#f92672', '#a6e22e'] },
  { id: 'solarized', name: 'Solarized Dark', preview: ['#002b36', '#268bd2', '#2aa198'] },
  { id: 'solarized-light', name: 'Solarized Light', preview: ['#fdf6e3', '#268bd2', '#859900'] },
  { id: 'catppuccin-latte', name: 'Catppuccin Latte', preview: ['#eff1f5', '#8839ef', '#179299'] },
  { id: 'rose-peach', name: 'Rosé Peach', preview: ['#FFDCDC', '#e8787a', '#7fb3b3'] },
]

// Settings tab state
type SettingsTab = 'chat' | 'summarize' | 'embedding' | 'knowledge' | 'theme'
const currentSettingsTab = ref<SettingsTab>('chat')
const settingsTabs: { id: SettingsTab; name: string; icon: string }[] = [
  { id: 'chat', name: '聊天模型', icon: 'M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z' },
  { id: 'summarize', name: '总结模型', icon: 'M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z M14 2v6h6 M16 13H8 M16 17H8 M10 9H8' },
  { id: 'embedding', name: '向量模型', icon: 'M12 2L2 7l10 5 10-5-10-5z M2 17l10 5 10-5 M2 12l10 5 10-5' },
  { id: 'knowledge', name: '记忆管理', icon: 'M12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2zm1 15h-2v-6h2zm0-8h-2V7h2z' },
  { id: 'theme', name: '主题', icon: 'M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0z' },
]

// Config type for model tabs
const currentConfigTab = computed<ConfigType>(() => {
  if (currentSettingsTab.value === 'chat' || currentSettingsTab.value === 'summarize' || currentSettingsTab.value === 'embedding') {
    return currentSettingsTab.value
  }
  return 'chat'
})

// Filtered configs by current tab
const filteredConfigs = computed(() =>
  configs.value.filter(c => c.config_type === currentConfigTab.value)
)

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
    // Currently collapsed - expand if dragged past threshold
    if (newWidth >= collapseThreshold) {
      sidebarCollapsed.value = false
      sidebarWidth.value = Math.max(minExpandedWidth, Math.min(newWidth, maxExpandedWidth))
    }
  } else {
    // Currently expanded
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
      case 'w': // Close window
        e.preventDefault()
        windowClose()
        break
      case 'm': // Minimize
        e.preventDefault()
        windowMinimize()
        break
      case 'n': // New chat
        e.preventDefault()
        newChat()
        break
      case ',': // Open settings
        e.preventDefault()
        currentView.value = 'settings'
        break
      case 'b': // Toggle sidebar
        e.preventDefault()
        sidebarCollapsed.value = !sidebarCollapsed.value
        break
    }
  }

  // Escape to go back
  if (e.key === 'Escape') {
    if (currentView.value === 'settings') {
      if (showAddConfig.value) {
        resetConfigForm()
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

// New config form
const newConfig = ref<CreateConfigRequest>({
  name: '',
  provider: 'openai',
  api_key: '',
  base_url: '',
  model: 'gpt-4o-mini',
  max_tokens: 4096,
  temperature: 0.7,
  is_default: true,
  config_type: 'chat'
})

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

  // For messages without ID (streaming messages not yet saved), just remove from UI
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

  if (configs.value.length === 0) {
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
    // Add placeholder for assistant message
    messages.value.push({ role: 'assistant', content: '' })
    const assistantIndex = messages.value.length - 1

    // Stream response
    const { stream, sessionId } = await api.chatStream({
      session_id: currentSessionId.value || undefined,
      messages: [userMessage]
    })

    // Update session ID immediately if this is a new chat
    if (sessionId && !currentSessionId.value) {
      currentSessionId.value = sessionId
    }

    for await (const chunk of stream) {
      if (chunk.delta) {
        // Update through reactive array to ensure Vue detects changes
        messages.value[assistantIndex].content += chunk.delta
        scrollToBottom()
      }
      if (chunk.done) {
        break
      }
    }

    // Reload sessions to get the new/updated session
    await loadSessions()
  } catch (e: any) {
    console.error('Chat failed:', e)
    messages.value.pop() // Remove empty assistant message
    showToast(e.message || 'Failed to send message')
  } finally {
    isLoading.value = false
  }
}

function resetConfigForm() {
  newConfig.value = {
    name: '',
    provider: 'openai',
    api_key: '',
    base_url: '',
    model: currentConfigTab.value === 'embedding' ? 'text-embedding-3-small' : 'gpt-4o-mini',
    max_tokens: 4096,
    temperature: 0.7,
    is_default: filteredConfigs.value.length === 0,
    config_type: currentConfigTab.value
  }
  editingConfigId.value = null
  showAddConfig.value = false
}

function editConfig(config: LLMConfig) {
  editingConfigId.value = config.id
  newConfig.value = {
    name: config.name,
    provider: config.provider,
    api_key: '', // Don't show existing API key for security
    base_url: config.base_url || '',
    model: config.model,
    max_tokens: config.max_tokens,
    temperature: config.temperature,
    is_default: config.is_default,
    config_type: config.config_type
  }
  showAddConfig.value = true
}

async function saveConfig() {
  if (!newConfig.value.name) {
    showToast('Please fill in the name field', 'info')
    return
  }

  // For new config, API key is required (except for Ollama)
  if (!editingConfigId.value && !newConfig.value.api_key && newConfig.value.provider !== 'ollama') {
    showToast('Please fill in the API key field', 'info')
    return
  }

  // For Ollama, set a placeholder API key if not provided
  const configToSave = { ...newConfig.value }
  if (configToSave.provider === 'ollama' && !configToSave.api_key) {
    configToSave.api_key = 'ollama'
  }
  // Set default base URL for Ollama if not provided
  if (configToSave.provider === 'ollama' && !configToSave.base_url) {
    configToSave.base_url = 'http://localhost:11434/v1'
  }

  try {
    if (editingConfigId.value) {
      // Update existing config
      const updateData: UpdateConfigRequest = {
        name: configToSave.name,
        provider: configToSave.provider,
        base_url: configToSave.base_url,
        model: configToSave.model,
        max_tokens: configToSave.max_tokens,
        temperature: configToSave.temperature,
        is_default: configToSave.is_default
      }
      // Only include API key if user entered a new one
      if (configToSave.api_key) {
        updateData.api_key = configToSave.api_key
      }
      await api.updateConfig(editingConfigId.value, updateData)
    } else {
      // Create new config
      await api.createConfig(configToSave)
    }
    await loadConfigs()
    resetConfigForm()
  } catch (e: any) {
    showToast(e.message || 'Failed to save config')
  }
}

async function removeConfig(id: string) {
  try {
    await api.deleteConfig(id)
    await loadConfigs()
    showToast('配置已删除', 'success')
  } catch (e) {
    console.error('Failed to delete config:', e)
    showToast('删除失败')
  }
}

async function testConfig(id: string) {
  testingConfigId.value = id
  try {
    const result = await api.testConfig(id)
    if (result.success) {
      showToast('连接成功', 'success')
    } else {
      showToast(result.error || '连接失败')
    }
  } catch (e: any) {
    showToast(e.message || '测试失败')
  } finally {
    testingConfigId.value = null
  }
}

async function selectConfig(id: string) {
  try {
    await api.setDefaultConfig(id)
    await loadConfigs()
  } catch (e) {
    console.error('Failed to set default config:', e)
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
  }
})

// Lifecycle
onMounted(async () => {
  loadTheme()
  // Add global keyboard shortcut listener
  window.addEventListener('keydown', handleGlobalKeydown)
  await Promise.all([loadSessions(), loadConfigs(), loadKnowledge()])
})

// Cleanup on unmount
onUnmounted(() => {
  window.removeEventListener('keydown', handleGlobalKeydown)
})
</script>

<template>
  <div class="app-container" :class="{ 'sidebar-collapsed': sidebarCollapsed, 'is-resizing': isResizing }">
    <!-- Sidebar -->
    <div class="sidebar" :style="{ width: sidebarCollapsed ? '100px' : sidebarWidth + 'px' }">
      <!-- Title bar area - native macOS traffic lights -->
      <div class="sidebar-titlebar">
        <div class="titlebar-drag-region"></div>
      </div>

      <div class="sidebar-content">
        <!-- Brand row: logo + name + toggle -->
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

      <!-- Resize handle - always visible -->
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
          <!-- Titlebar drag region for macOS -->
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
              @click="currentSettingsTab = tab.id; if (['chat', 'summarize', 'embedding'].includes(tab.id)) { showAddConfig = false; showAddKnowledge = false; } else if (tab.id === 'knowledge') { showAddConfig = false; }"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path :d="tab.icon"></path>
              </svg>
              {{ tab.name }}
            </button>
          </div>

          <div class="settings-content">
            <!-- Model Config Tabs (chat, summarize, embedding) -->
            <template v-if="['chat', 'summarize', 'embedding'].includes(currentSettingsTab)">
              <template v-if="!showAddConfig">
                <div class="config-list">
                  <div
                    v-for="config in filteredConfigs"
                    :key="config.id"
                    class="config-item"
                    :class="{ active: config.is_default }"
                    @click="selectConfig(config.id)"
                  >
                    <div class="config-info">
                      <div class="config-name">
                        {{ config.name }}
                        <span v-if="config.is_default" class="default-badge">Default</span>
                      </div>
                      <div class="config-detail">{{ config.provider }} · {{ config.model }}</div>
                    </div>
                    <div class="config-actions">
                      <button
                        class="btn-icon btn-test"
                        :class="{ testing: testingConfigId === config.id }"
                        @click.stop="testConfig(config.id)"
                        :disabled="testingConfigId === config.id"
                        title="Test"
                      >
                        <svg v-if="testingConfigId !== config.id" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                          <polygon points="5 3 19 12 5 21 5 3"></polygon>
                        </svg>
                        <svg v-else class="spin" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
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
                      <button class="btn-icon" @click.stop="editConfig(config)" title="Edit">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                          <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                          <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                        </svg>
                      </button>
                      <button class="btn-icon btn-danger" @click.stop="removeConfig(config.id)" title="Delete">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                          <polyline points="3 6 5 6 21 6"></polyline>
                          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                        </svg>
                      </button>
                    </div>
                  </div>

                  <p v-if="filteredConfigs.length === 0" class="empty-config-text">
                    暂无{{ settingsTabs.find(t => t.id === currentSettingsTab)?.name }}配置
                  </p>
                </div>

                <button class="btn-add-config" @click="showAddConfig = true; editingConfigId = null; newConfig.config_type = currentConfigTab">
                  <span class="icon-wrapper">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                      <line x1="12" y1="5" x2="12" y2="19"></line>
                      <line x1="5" y1="12" x2="19" y2="12"></line>
                    </svg>
                  </span>
                  添加{{ settingsTabs.find(t => t.id === currentSettingsTab)?.name }}配置
                </button>
              </template>

              <!-- Add/Edit Config Form -->
              <template v-else>
                <div class="form-header">
                  <button class="back-btn" @click="resetConfigForm" title="Back">
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <polyline points="15 18 9 12 15 6"></polyline>
                    </svg>
                  </button>
                  <h2>{{ editingConfigId ? '编辑配置' : `添加${settingsTabs.find(t => t.id === currentSettingsTab)?.name}配置` }}</h2>
                </div>

                <div class="config-form">
                  <div class="form-group">
                    <label>Name *</label>
                    <input v-model="newConfig.name" placeholder="My OpenAI Config" />
                  </div>

                  <div class="form-group">
                    <label>Provider *</label>
                    <select v-model="newConfig.provider">
                      <option value="openai">OpenAI</option>
                      <option value="claude">Claude (Anthropic)</option>
                      <option value="azure">Azure OpenAI</option>
                      <option value="ollama">Ollama (Local)</option>
                      <option value="custom">Custom (OpenAI Compatible)</option>
                    </select>
                  </div>

                  <div class="form-group" v-if="newConfig.provider !== 'ollama'">
                    <label>API Key {{ editingConfigId ? '(leave empty to keep current)' : '*' }}</label>
                    <input v-model="newConfig.api_key" type="password" :placeholder="editingConfigId ? '••••••••' : 'sk-...'" />
                  </div>

                  <div class="form-group">
                    <label>Base URL (optional)</label>
                    <input v-model="newConfig.base_url" placeholder="https://api.openai.com/v1" />
                  </div>

                  <div class="form-group">
                    <label>Model *</label>
                    <input v-model="newConfig.model" placeholder="gpt-4o-mini" />
                  </div>

                  <div class="form-group" v-if="currentSettingsTab !== 'embedding'">
                    <label>Max Tokens</label>
                    <input v-model.number="newConfig.max_tokens" type="number" />
                  </div>

                  <div class="form-group" v-if="currentSettingsTab !== 'embedding'">
                    <label>Temperature (0-2)</label>
                    <input v-model.number="newConfig.temperature" type="number" step="0.1" min="0" max="2" />
                  </div>

                  <div class="form-group" v-if="!editingConfigId || !configs.find(c => c.id === editingConfigId)?.is_default">
                    <label class="checkbox-label">
                      <input type="checkbox" v-model="newConfig.is_default" />
                      <span>Set as default configuration</span>
                    </label>
                  </div>

                  <div class="form-actions">
                    <button class="btn-secondary" @click="resetConfigForm">Cancel</button>
                    <button
                      v-if="editingConfigId"
                      class="btn-test-config"
                      :class="{ testing: testingConfigId === editingConfigId }"
                      :disabled="testingConfigId === editingConfigId"
                      @click="testConfig(editingConfigId)"
                    >
                      <svg v-if="testingConfigId !== editingConfigId" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <polygon points="5 3 19 12 5 21 5 3"></polygon>
                      </svg>
                      <svg v-else class="spin" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <line x1="12" y1="2" x2="12" y2="6"></line>
                        <line x1="12" y1="18" x2="12" y2="22"></line>
                        <line x1="4.93" y1="4.93" x2="7.76" y2="7.76"></line>
                        <line x1="16.24" y1="16.24" x2="19.07" y2="19.07"></line>
                        <line x1="2" y1="12" x2="6" y2="12"></line>
                        <line x1="18" y1="12" x2="22" y2="12"></line>
                        <line x1="4.93" y1="19.07" x2="7.76" y2="16.24"></line>
                        <line x1="16.24" y1="7.76" x2="19.07" y2="4.93"></line>
                      </svg>
                      Test
                    </button>
                    <button class="btn-primary" @click="saveConfig">{{ editingConfigId ? 'Update' : 'Save' }}</button>
                  </div>
                </div>
              </template>
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
