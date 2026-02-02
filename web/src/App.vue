<script setup lang="ts">
import { ref, onMounted, nextTick, computed } from 'vue'
import { marked } from 'marked'
import * as api from './api'
import type { Message, Session, LLMConfig, CreateConfigRequest } from './api'

// State
const sessions = ref<Session[]>([])
const currentSessionId = ref<string | null>(null)
const messages = ref<Message[]>([])
const inputText = ref('')
const isLoading = ref(false)
const showSettings = ref(false)
const configs = ref<LLMConfig[]>([])
const showAddConfig = ref(false)

// New config form
const newConfig = ref<CreateConfigRequest>({
  name: '',
  provider: 'openai',
  api_key: '',
  base_url: '',
  model: 'gpt-4o-mini',
  max_tokens: 4096,
  temperature: 0.7,
  is_default: true
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
  if (!confirm('Delete this conversation?')) return

  try {
    await api.deleteSession(id)
    sessions.value = sessions.value.filter(s => s.id !== id)
    if (currentSessionId.value === id) {
      currentSessionId.value = null
      messages.value = []
    }
  } catch (e) {
    console.error('Failed to delete session:', e)
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
    alert('Please add an LLM configuration first')
    showSettings.value = true
    return
  }

  const userMessage: Message = { role: 'user', content: inputText.value.trim() }
  messages.value.push(userMessage)
  inputText.value = ''
  isLoading.value = true
  scrollToBottom()

  try {
    // Add placeholder for assistant message
    const assistantMessage: Message = { role: 'assistant', content: '' }
    messages.value.push(assistantMessage)

    // Stream response
    const stream = api.chatStream({
      session_id: currentSessionId.value || undefined,
      messages: [userMessage]
    })

    for await (const chunk of stream) {
      if (chunk.delta) {
        assistantMessage.content += chunk.delta
        scrollToBottom()
      }
      if (chunk.done) {
        break
      }
    }

    // Reload sessions to get the new/updated session
    await loadSessions()

    // If this was a new chat, set the current session
    if (!currentSessionId.value && sessions.value.length > 0) {
      currentSessionId.value = sessions.value[0].id
    }
  } catch (e: any) {
    console.error('Chat failed:', e)
    messages.value.pop() // Remove empty assistant message
    alert(e.message || 'Failed to send message')
  } finally {
    isLoading.value = false
  }
}

async function addConfig() {
  if (!newConfig.value.name || !newConfig.value.api_key) {
    alert('Please fill in required fields')
    return
  }

  try {
    await api.createConfig(newConfig.value)
    await loadConfigs()
    showAddConfig.value = false
    newConfig.value = {
      name: '',
      provider: 'openai',
      api_key: '',
      base_url: '',
      model: 'gpt-4o-mini',
      max_tokens: 4096,
      temperature: 0.7,
      is_default: configs.value.length === 0
    }
  } catch (e: any) {
    alert(e.message || 'Failed to add config')
  }
}

async function removeConfig(id: string) {
  if (!confirm('Delete this configuration?')) return

  try {
    await api.deleteConfig(id)
    await loadConfigs()
  } catch (e) {
    console.error('Failed to delete config:', e)
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

// Lifecycle
onMounted(async () => {
  await Promise.all([loadSessions(), loadConfigs()])
})
</script>

<template>
  <div class="sidebar">
    <div class="sidebar-header">
      <h1>LLM Agent</h1>
      <button class="new-chat-btn" @click="newChat">+ New Chat</button>
    </div>

    <div class="session-list">
      <div
        v-for="session in sessions"
        :key="session.id"
        class="session-item"
        :class="{ active: session.id === currentSessionId }"
        @click="selectSession(session)"
      >
        <span class="session-title">{{ session.title || 'New Chat' }}</span>
        <button class="delete-btn" @click="deleteSession(session.id, $event)">
          &times;
        </button>
      </div>
    </div>

    <button class="settings-btn" @click="showSettings = true">
      Settings
    </button>
  </div>

  <div class="main-content">
    <div ref="chatContainer" class="chat-container">
      <template v-if="messages.length > 0">
        <div
          v-for="(msg, index) in messages"
          :key="index"
          class="message"
          :class="msg.role"
        >
          <div v-if="msg.role === 'assistant'" v-html="renderMarkdown(msg.content)"></div>
          <template v-else>{{ msg.content }}</template>
        </div>

        <div v-if="isLoading && !messages[messages.length - 1]?.content" class="typing-indicator">
          <span></span>
          <span></span>
          <span></span>
        </div>
      </template>

      <div v-else class="empty-state">
        <h2>Welcome</h2>
        <p>Start a conversation or select a previous chat</p>
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
          Send
        </button>
      </div>
    </div>
  </div>

  <!-- Settings Modal -->
  <div v-if="showSettings" class="modal-overlay" @click.self="showSettings = false">
    <div class="modal">
      <h2>Settings</h2>

      <h3 style="margin-bottom: 15px; color: var(--text-secondary);">LLM Configurations</h3>

      <div class="config-list">
        <div
          v-for="config in configs"
          :key="config.id"
          class="config-item"
          :class="{ active: config.is_default }"
        >
          <div class="config-info">
            <div class="config-name">{{ config.name }}</div>
            <div class="config-detail">{{ config.provider }} - {{ config.model }}</div>
          </div>
          <div class="config-actions">
            <button class="btn-secondary" @click="removeConfig(config.id)">Delete</button>
          </div>
        </div>

        <p v-if="configs.length === 0" style="color: var(--text-secondary); text-align: center; padding: 20px;">
          No configurations yet. Add one to start chatting.
        </p>
      </div>

      <button v-if="!showAddConfig" class="btn-primary" style="width: 100%;" @click="showAddConfig = true">
        + Add Configuration
      </button>

      <template v-if="showAddConfig">
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
            <option value="custom">Custom (OpenAI Compatible)</option>
          </select>
        </div>

        <div class="form-group">
          <label>API Key *</label>
          <input v-model="newConfig.api_key" type="password" placeholder="sk-..." />
        </div>

        <div class="form-group">
          <label>Base URL (optional)</label>
          <input v-model="newConfig.base_url" placeholder="https://api.openai.com/v1" />
        </div>

        <div class="form-group">
          <label>Model *</label>
          <input v-model="newConfig.model" placeholder="gpt-4o-mini" />
        </div>

        <div class="form-group">
          <label>Max Tokens</label>
          <input v-model.number="newConfig.max_tokens" type="number" />
        </div>

        <div class="form-group">
          <label>Temperature (0-2)</label>
          <input v-model.number="newConfig.temperature" type="number" step="0.1" min="0" max="2" />
        </div>

        <div class="modal-actions">
          <button class="btn-secondary" @click="showAddConfig = false">Cancel</button>
          <button class="btn-primary" @click="addConfig">Save</button>
        </div>
      </template>

      <div v-if="!showAddConfig" class="modal-actions">
        <button class="btn-secondary" @click="showSettings = false">Close</button>
      </div>
    </div>
  </div>
</template>
