<template>
  <section class="settings-section card">
    <h2>Data Management</h2>
    <div class="setting-item">
      <div class="setting-info">
        <span class="setting-label">Export Collection</span>
        <span class="setting-desc">Download your collection data and photos as a zip archive</span>
      </div>
      <button class="btn btn-secondary btn-sm" :disabled="exporting" @click="handleExport">
        {{ exporting ? 'Exporting...' : 'Export ZIP' }}
      </button>
    </div>
    <div class="setting-item">
      <div class="setting-info">
        <span class="setting-label">PDF Catalog</span>
        <span class="setting-desc">Generate a styled PDF catalog with photos, grades, and valuations</span>
      </div>
      <button class="btn btn-secondary btn-sm" :disabled="exportingPdf" @click="handleExportPDF">
        {{ exportingPdf ? 'Generating...' : 'Export PDF' }}
      </button>
    </div>
    <div class="setting-item">
      <div class="setting-info">
        <span class="setting-label">Import Collection</span>
        <span class="setting-desc">Import coins from a JSON file</span>
      </div>
      <label class="btn btn-secondary btn-sm import-btn">
        📤 Import
        <input type="file" accept=".json" hidden @change="handleImport" />
      </label>
    </div>
    <p v-if="dataMsg" class="msg" :class="{ error: dataError }">{{ dataMsg }}</p>

    <h3>API Keys</h3>
    <p class="setting-desc" style="margin-bottom: 1rem">
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
      <button
        class="btn btn-primary btn-sm"
        :disabled="!apiKeyName.trim() || generatingKey"
        @click="handleGenerateKey"
      >
        {{ generatingKey ? 'Generating...' : '🔑 Generate Key' }}
      </button>
    </div>

    <div v-if="newlyGeneratedKey" class="apikey-reveal">
      <p class="apikey-reveal-warning">
        ⚠️ Copy this key now — it will not be shown again.
      </p>
      <div class="apikey-reveal-box">
        <code class="apikey-reveal-value">{{ newlyGeneratedKey }}</code>
        <button class="btn btn-secondary btn-sm" @click="copyKey">
          {{ keyCopied ? '✓ Copied' : '📋 Copy' }}
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
          <span class="apikey-item-name">{{ key.name }}</span>
          <span class="apikey-item-meta">
            ...{{ key.keyPrefix }}
            · Created {{ formatDate(key.createdAt) }}
            <template v-if="key.lastUsedAt"> · Last used {{ formatDate(key.lastUsedAt) }}</template>
          </span>
        </div>
        <span v-if="key.revokedAt" class="apikey-item-badge revoked-badge">Revoked</span>
        <button
          v-else
          class="btn btn-danger btn-sm"
          @click="handleRevokeKey(key.id)"
        >
          Revoke
        </button>
      </div>
    </div>
    <p v-else-if="!generatingKey" class="setting-desc" style="margin-top: 0.5rem">No API keys yet.</p>

    <h3 style="margin-top: 2rem">Tags</h3>
    <p class="setting-desc">Create custom tags to organize and filter your coins.</p>

    <div class="tag-create-form">
      <input
        v-model="newTagName"
        type="text"
        class="form-input"
        placeholder="New tag name..."
        maxlength="50"
        @keydown.enter="handleCreateTag"
      />
      <div class="tag-color-picker">
        <button
          v-for="c in TAG_COLORS"
          :key="c"
          class="color-swatch"
          :class="{ active: newTagColor === c }"
          :style="{ backgroundColor: c }"
          @click="newTagColor = c"
        ></button>
      </div>
      <button class="btn btn-primary btn-sm" @click="handleCreateTag" :disabled="!newTagName.trim()">Create Tag</button>
    </div>
    <p v-if="tagError" class="tag-error">{{ tagError }}</p>

    <div v-if="tagList.length" class="tag-list">
      <div v-for="tag in tagList" :key="tag.id" class="tag-list-item">
        <template v-if="editingTag?.id === tag.id">
          <input v-model="editTagName" class="form-input tag-edit-input" maxlength="50" @keydown.enter="handleSaveTag" />
          <div class="tag-color-picker">
            <button
              v-for="c in TAG_COLORS"
              :key="c"
              class="color-swatch sm"
              :class="{ active: editTagColor === c }"
              :style="{ backgroundColor: c }"
              @click="editTagColor = c"
            ></button>
          </div>
          <button class="btn btn-primary btn-sm" @click="handleSaveTag">Save</button>
          <button class="btn btn-secondary btn-sm" @click="editingTag = null">Cancel</button>
        </template>
        <template v-else>
          <span class="tag-preview" :style="{ backgroundColor: tag.color + '22', color: tag.color, borderColor: tag.color + '44' }">{{ tag.name }}</span>
          <div class="tag-actions">
            <button class="btn btn-secondary btn-sm" @click="startEditTag(tag)">Edit</button>
            <button class="btn btn-danger btn-sm" @click="handleDeleteTag(tag)">Delete</button>
          </div>
        </template>
      </div>
    </div>
    <p v-else class="empty-tags">No tags created yet. Create your first tag above.</p>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  exportCollection, exportCatalogPDF, importCollection,
  generateApiKey, listApiKeys, revokeApiKey,
  getTags, createTag, updateTag as updateTagApi, deleteTag,
} from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import type { Coin, ApiKey, Tag } from '@/types'

const { showConfirm } = useDialog()

// Data export/import
const exporting = ref(false)
const exportingPdf = ref(false)
const dataMsg = ref('')
const dataError = ref(false)

async function handleExport() {
  exporting.value = true
  dataMsg.value = ''
  try {
    const res = await exportCollection()
    const blob = new Blob([res.data], { type: 'application/zip' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `ancient-coins-export-${new Date().toISOString().slice(0, 10)}.zip`
    a.click()
    URL.revokeObjectURL(url)
    dataMsg.value = 'Export downloaded'
  } catch {
    dataMsg.value = 'Export failed'
    dataError.value = true
  } finally {
    exporting.value = false
  }
}

async function handleExportPDF() {
  exportingPdf.value = true
  dataMsg.value = ''
  try {
    const res = await exportCatalogPDF()
    const blob = new Blob([res.data], { type: 'application/pdf' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `coin-catalog-${new Date().toISOString().slice(0, 10)}.pdf`
    a.click()
    URL.revokeObjectURL(url)
    dataMsg.value = 'PDF catalog downloaded'
  } catch {
    dataMsg.value = 'PDF generation failed'
    dataError.value = true
  } finally {
    exportingPdf.value = false
  }
}

async function handleImport(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return

  dataMsg.value = ''
  dataError.value = false

  try {
    const text = await file.text()
    const coins: Coin[] = JSON.parse(text)
    const res = await importCollection(coins)
    dataMsg.value = `Imported ${res.data.imported} coins`
  } catch {
    dataMsg.value = 'Import failed — ensure valid JSON format'
    dataError.value = true
  }
}

// API Keys
const apiKeys = ref<ApiKey[]>([])
const apiKeyName = ref('')
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
    const res = await generateApiKey(apiKeyName.value.trim())
    newlyGeneratedKey.value = res.data.key
    apiKeyName.value = ''
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

// Tag management
const tagList = ref<Tag[]>([])
const newTagName = ref('')
const newTagColor = ref('#6b7280')
const editingTag = ref<Tag | null>(null)
const editTagName = ref('')
const editTagColor = ref('')
const tagError = ref('')

const TAG_COLORS = ['#6b7280', '#ef4444', '#f59e0b', '#10b981', '#3b82f6', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316', '#6366f1']

async function loadTags() {
  try {
    const res = await getTags()
    tagList.value = res.data?.tags ?? []
  } catch { tagList.value = [] }
}

async function handleCreateTag() {
  tagError.value = ''
  const name = newTagName.value.trim()
  if (!name) return
  try {
    await createTag({ name, color: newTagColor.value })
    newTagName.value = ''
    newTagColor.value = '#6b7280'
    await loadTags()
  } catch (e: unknown) {
    if (typeof e === 'object' && e !== null && 'response' in e) {
      const axiosErr = e as { response?: { data?: { error?: string } } }
      tagError.value = axiosErr.response?.data?.error ?? 'Failed to create tag'
    } else {
      tagError.value = 'Failed to create tag'
    }
  }
}

function startEditTag(tag: Tag) {
  editingTag.value = tag
  editTagName.value = tag.name
  editTagColor.value = tag.color
}

async function handleSaveTag() {
  tagError.value = ''
  if (!editingTag.value) return
  try {
    await updateTagApi(editingTag.value.id, { name: editTagName.value.trim(), color: editTagColor.value })
    editingTag.value = null
    await loadTags()
  } catch (e: unknown) {
    if (typeof e === 'object' && e !== null && 'response' in e) {
      const axiosErr = e as { response?: { data?: { error?: string } } }
      tagError.value = axiosErr.response?.data?.error ?? 'Failed to update tag'
    } else {
      tagError.value = 'Failed to update tag'
    }
  }
}

async function handleDeleteTag(tag: Tag) {
  const confirmed = await showConfirm(`Delete tag "${tag.name}"? It will be removed from all coins.`, { title: 'Delete Tag', variant: 'danger' })
  if (!confirmed) return
  try {
    await deleteTag(tag.id)
    await loadTags()
  } catch { /* ignore */ }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric',
  })
}

onMounted(() => {
  loadApiKeys()
  loadTags()
})

defineExpose({ loadApiKeys, loadTags })
</script>

<style scoped>
.settings-section h2 {
  font-size: 1.1rem;
  margin-bottom: 1.25rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
}

.settings-section h3 {
  font-size: 0.95rem;
  margin-top: 1.25rem;
  margin-bottom: 0.75rem;
  color: var(--text-secondary);
}

.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 0;
  border-bottom: 1px solid var(--border-subtle);
  gap: 1rem;
}

.setting-item:last-child {
  border-bottom: none;
}

.setting-info {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.setting-label {
  font-size: 0.9rem;
  font-weight: 500;
}

.setting-desc {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.import-btn {
  cursor: pointer;
}

.msg {
  font-size: 0.85rem;
  color: var(--accent-gold);
  margin: 0.5rem 0;
}

.msg.error {
  color: #e74c3c;
}

.apikey-generate {
  display: flex;
  gap: 0.75rem;
  align-items: center;
  margin-bottom: 0.75rem;
}

.apikey-generate .form-input {
  flex: 1;
  max-width: 280px;
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
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.apikey-reveal-value {
  flex: 1;
  font-size: 0.78rem;
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

.apikey-item-name {
  font-size: 0.9rem;
  font-weight: 500;
}

.apikey-item-meta {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.revoked-badge {
  font-size: 0.7rem;
  padding: 0.15rem 0.5rem;
  background: var(--bg-primary);
  border-radius: var(--radius-full);
  color: var(--text-muted);
}

.btn-danger {
  background: #e74c3c;
  color: #fff;
  border: none;
  cursor: pointer;
}

.btn-danger:hover {
  background: #c0392b;
}

/* Tag Manager */
.tag-create-form {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
  margin: 1rem 0;
}

.tag-create-form .form-input {
  flex: 1;
  min-width: 150px;
}

.tag-color-picker {
  display: flex;
  gap: 0.3rem;
  align-items: center;
}

.color-swatch {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  padding: 0;
}

.color-swatch.active {
  border-color: var(--text-primary);
  box-shadow: 0 0 0 2px var(--bg-card);
}

.color-swatch.sm {
  width: 18px;
  height: 18px;
}

.tag-error {
  color: #ef4444;
  font-size: 0.85rem;
  margin-top: 0.25rem;
}

.tag-list {
  margin-top: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.tag-list-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  flex-wrap: wrap;
}

.tag-preview {
  font-size: 0.8rem;
  padding: 0.2rem 0.6rem;
  border-radius: 9999px;
  border: 1px solid;
  flex-shrink: 0;
}

.tag-edit-input {
  flex: 1;
  min-width: 120px;
}

.tag-actions {
  margin-left: auto;
  display: flex;
  gap: 0.25rem;
}

.empty-tags {
  color: var(--text-secondary);
  font-size: 0.85rem;
  margin-top: 1rem;
}

@media (max-width: 640px) {
  .setting-item {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
