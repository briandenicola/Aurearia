<template>
  <section class="card">
    <h2 class="text-xl font-medium mb-5 pb-3 border-b border-border-subtle">API Keys</h2>
    <p class="text-sm text-text-muted mb-4">
      Generate API keys to access your collection from external tools and scripts. Use the <code>X-API-Key</code> header to authenticate.
    </p>

    <!-- Generate form -->
    <div class="flex flex-wrap items-center gap-3 mb-3">
      <input
        v-model="apiKeyName"
        type="text"
        class="form-input flex-1 min-w-[200px]"
        placeholder="Key name (e.g. My Script)"
        :disabled="generatingKey"
      />
      <div class="flex items-center gap-[0.35rem]">
        <button
          type="button"
          class="chip"
          :class="{ active: apiKeyScope === 'read' }"
          :disabled="generatingKey"
          @click="apiKeyScope = 'read'"
        >Read</button>
        <button
          type="button"
          class="chip"
          :class="{ active: apiKeyScope === 'read,write' }"
          :disabled="generatingKey"
          @click="apiKeyScope = 'read,write'"
        >Read/Write</button>
      </div>
      <button
        class="btn btn-primary btn-sm inline-flex items-center gap-[0.35rem]"
        :disabled="!apiKeyName.trim() || generatingKey"
        @click="handleGenerateKey"
      >
        <KeyRound :size="14" />
        {{ generatingKey ? 'Generating...' : 'Generate Key' }}
      </button>
    </div>

    <!-- Newly generated key reveal -->
    <div
      v-if="newlyGeneratedKey"
      class="bg-surface border border-gold-dim rounded-sm px-4 py-3 mb-3"
    >
      <p class="text-chip text-gold font-medium mb-2">
        Copy this key now — it will not be shown again.
      </p>
      <div class="flex items-center gap-2">
        <code class="flex-1 text-chip bg-card px-[0.6rem] py-[0.4rem] rounded-sm break-all select-all">
          {{ newlyGeneratedKey }}
        </code>
        <button
          class="btn btn-secondary btn-sm inline-flex items-center gap-[0.35rem]"
          @click="copyKey"
        >
          <Check v-if="keyCopied" :size="14" />
          <Clipboard v-else :size="14" />
          {{ keyCopied ? 'Copied' : 'Copy' }}
        </button>
      </div>
    </div>

    <p
      v-if="apiKeyMsg"
      class="text-body my-2"
      :class="apiKeyError ? 'text-byzantine' : 'text-gold'"
    >{{ apiKeyMsg }}</p>

    <!-- Key list -->
    <div v-if="apiKeys.length" class="flex flex-col gap-2 mt-4">
      <div
        v-for="key in apiKeys"
        :key="key.id"
        class="flex justify-between items-center py-[0.6rem] border-b border-border-subtle last:border-0 gap-3"
        :class="{ 'opacity-50': key.revokedAt }"
      >
        <div class="flex flex-col gap-[0.1rem] min-w-0">
          <div class="flex items-center gap-2">
            <span class="text-base font-medium">{{ key.name }}</span>
            <span
              class="chip-sm shrink-0"
              :class="key.capabilities === 'read,write'
                ? 'bg-gold-glow text-gold border border-gold-dim'
                : 'bg-input text-text-secondary border border-border-subtle'"
            >
              {{ capabilityLabel(key.capabilities) }}
            </span>
          </div>
          <span class="text-sm text-text-muted">
            ...{{ key.keyPrefix }}
            · Created {{ formatDate(key.createdAt) }}
            <template v-if="key.lastUsedAt"> · Last used {{ formatDate(key.lastUsedAt) }}</template>
          </span>
        </div>
        <span
          v-if="key.revokedAt"
          class="chip-sm shrink-0 bg-surface text-text-muted"
        >Revoked</span>
        <button
          v-else
          class="btn btn-danger btn-sm"
          @click="handleRevokeKey(key.id)"
        >Revoke</button>
      </div>
    </div>
    <p v-else-if="!generatingKey" class="text-sm text-text-muted mt-2">No API keys yet.</p>
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

