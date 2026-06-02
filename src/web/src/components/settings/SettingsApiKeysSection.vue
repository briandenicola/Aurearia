<template>
  <section class="settings-section card">
    <h2>API Keys</h2>
    <p class="setting-desc api-key-description">
      Generate API keys to access your collection from external tools and scripts. Use the <code>X-API-Key</code> header to authenticate.
    </p>

    <div class="apikey-generate">
      <input
        v-model="apiKeyName"
        type="text"
        class="form-input"
        placeholder="Key name (e.g. My Script)"
        :disabled="generatingKey"
      />
      <div class="apikey-scope-selector">
        <button
          type="button"
          class="chip"
          :class="{ active: apiKeyScope === 'read' }"
          :disabled="generatingKey"
          @click="apiKeyScope = 'read'"
        >
          Read
        </button>
        <button
          type="button"
          class="chip"
          :class="{ active: apiKeyScope === 'read,write' }"
          :disabled="generatingKey"
          @click="apiKeyScope = 'read,write'"
        >
          Read/Write
        </button>
      </div>
      <button
        class="btn btn-primary btn-sm icon-button"
        :disabled="!apiKeyName.trim() || generatingKey"
        @click="handleGenerateKey"
      >
        <KeyRound :size="14" />
        {{ generatingKey ? 'Generating...' : 'Generate Key' }}
      </button>
    </div>

    <div v-if="newlyGeneratedKey" class="apikey-reveal">
      <p class="apikey-reveal-warning">
        Copy this key now — it will not be shown again.
      </p>
      <div class="apikey-reveal-box">
        <code class="apikey-reveal-value">{{ newlyGeneratedKey }}</code>
        <button class="btn btn-secondary btn-sm icon-button" @click="copyKey">
          <Check v-if="keyCopied" :size="14" />
          <Clipboard v-else :size="14" />
          {{ keyCopied ? 'Copied' : 'Copy' }}
        </button>
      </div>
    </div>

    <p v-if="apiKeyMsg" class="msg" :class="{ error: apiKeyError }">{{ apiKeyMsg }}</p>

    <div v-if="apiKeys.length" class="apikey-list">
      <div
        v-for="key in apiKeys"
        :key="key.id"
        class="apikey-item"
        :class="{ revoked: key.revokedAt }"
      >
        <div class="apikey-item-info">
          <div class="apikey-title-row">
            <span class="apikey-item-name">{{ key.name }}</span>
            <span class="chip-sm capability-badge" :class="capabilityClass(key.capabilities)">
              {{ capabilityLabel(key.capabilities) }}
            </span>
          </div>
          <span class="apikey-item-meta">
            ...{{ key.keyPrefix }}
            · Created {{ formatDate(key.createdAt) }}
            <template v-if="key.lastUsedAt"> · Last used {{ formatDate(key.lastUsedAt) }}</template>
          </span>
        </div>
        <span v-if="key.revokedAt" class="chip-sm revoked-badge">Revoked</span>
        <button
          v-else
          class="btn btn-danger btn-sm"
          @click="handleRevokeKey(key.id)"
        >
          Revoke
        </button>
      </div>
    </div>
    <p v-else-if="!generatingKey" class="setting-desc no-api-keys">No API keys yet.</p>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Check, Clipboard, KeyRound } from 'lucide-vue-next'
import { generateApiKey, listApiKeys, revokeApiKey } from '@/api/client'
import type { ApiKey } from '@/types'

const apiKeys = ref<ApiKey[]>([])
const apiKeyName = ref('')
const apiKeyScope = ref<'read' | 'read,write'>('read')
const newlyGeneratedKey = ref('')
const keyCopied = ref(false)
const generatingKey = ref(false)
const apiKeyMsg = ref('')
const apiKeyError = ref(false)

async function loadApiKeys() {
  try {
    const res = await listApiKeys()
    apiKeys.value = res.data
  } catch {
    // silently fail on load
  }
}

async function handleGenerateKey() {
  if (!apiKeyName.value.trim()) return

  generatingKey.value = true
  apiKeyMsg.value = ''
  apiKeyError.value = false
  newlyGeneratedKey.value = ''
  keyCopied.value = false

  try {
    const res = await generateApiKey(apiKeyName.value.trim(), apiKeyScope.value)
    newlyGeneratedKey.value = res.data.key
    apiKeyName.value = ''
    apiKeyScope.value = 'read'
    await loadApiKeys()
  } catch {
    apiKeyMsg.value = 'Failed to generate API key'
    apiKeyError.value = true
  } finally {
    generatingKey.value = false
  }
}

async function copyKey() {
  try {
    await navigator.clipboard.writeText(newlyGeneratedKey.value)
    keyCopied.value = true
    setTimeout(() => { keyCopied.value = false }, 3000)
  } catch {
    const textarea = document.createElement('textarea')
    textarea.value = newlyGeneratedKey.value
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    keyCopied.value = true
    setTimeout(() => { keyCopied.value = false }, 3000)
  }
}

async function handleRevokeKey(id: number) {
  apiKeyMsg.value = ''
  apiKeyError.value = false
  try {
    await revokeApiKey(id)
    await loadApiKeys()
    newlyGeneratedKey.value = ''
  } catch {
    apiKeyMsg.value = 'Failed to revoke key'
    apiKeyError.value = true
  }
}

function capabilityLabel(capabilities: string): string {
  return capabilities === 'read,write' ? 'Read/Write' : 'Read'
}

function capabilityClass(capabilities: string): string {
  return capabilities === 'read,write' ? 'capability-readwrite' : 'capability-read'
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric',
  })
}

onMounted(() => {
  loadApiKeys()
})

defineExpose({ loadApiKeys })
</script>

<style scoped>
.settings-section h2 {
  font-size: 1.2rem;
  margin-bottom: 1.25rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
}

.setting-desc {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.api-key-description {
  margin-bottom: 1rem;
}

.no-api-keys {
  margin-top: 0.5rem;
}

.apikey-generate,
.apikey-scope-selector,
.apikey-reveal-box,
.apikey-title-row {
  display: flex;
  align-items: center;
}

.msg {
  font-size: 0.85rem;
  color: var(--accent-gold);
  margin: 0.5rem 0;
}

.msg.error {
  color: var(--cat-byzantine);
}

.apikey-generate {
  gap: 0.75rem;
  margin-bottom: 0.75rem;
  flex-wrap: wrap;
}

.apikey-generate .form-input {
  flex: 1;
  min-width: 200px;
}

.apikey-scope-selector {
  gap: 0.35rem;
}

.icon-button {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.apikey-reveal {
  background: var(--bg-primary);
  border: 1px solid var(--accent-gold-dim);
  border-radius: var(--radius-sm);
  padding: 0.75rem 1rem;
  margin-bottom: 0.75rem;
}

.apikey-reveal-warning {
  font-size: 0.8rem;
  color: var(--accent-gold);
  margin-bottom: 0.5rem;
  font-weight: 500;
}

.apikey-reveal-box {
  gap: 0.5rem;
}

.apikey-reveal-value {
  flex: 1;
  font-size: 0.8rem;
  background: var(--bg-card);
  padding: 0.4rem 0.6rem;
  border-radius: var(--radius-sm);
  word-break: break-all;
  user-select: all;
}

.apikey-list {
  margin-top: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.apikey-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.6rem 0;
  border-bottom: 1px solid var(--border-subtle);
  gap: 0.75rem;
}

.apikey-item:last-child {
  border-bottom: none;
}

.apikey-item.revoked {
  opacity: 0.5;
}

.apikey-item-info {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  min-width: 0;
}

.apikey-title-row {
  gap: 0.5rem;
}

.apikey-item-name {
  font-size: 0.9rem;
  font-weight: 500;
}

.apikey-item-meta {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.capability-badge,
.revoked-badge {
  flex-shrink: 0;
}

.capability-badge.capability-read {
  background: var(--bg-input);
  color: var(--text-secondary);
  border: 1px solid var(--border-subtle);
}

.capability-badge.capability-readwrite {
  background: var(--accent-gold-glow);
  color: var(--accent-gold);
  border: 1px solid var(--accent-gold-dim);
}

.revoked-badge {
  background: var(--bg-primary);
  color: var(--text-muted);
}
</style>
