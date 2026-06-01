<template>
  <section class="settings-section card">
    <h2>Backups & API Keys</h2>

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
        <span class="setting-desc">Import coins from a JSON or CSV file</span>
      </div>
      <div class="import-actions">
        <label class="btn btn-secondary btn-sm import-btn">
          Import
          <input type="file" accept=".json,.csv,text/csv" hidden @change="handleImport" />
        </label>
        <button class="btn btn-secondary btn-sm" @click="downloadCsvTemplate">CSV Template</button>
        <button class="btn btn-secondary btn-sm" @click="openCsvGuide">Guide</button>
      </div>
    </div>

    <p v-if="dataMsg" class="msg" :class="{ error: dataError }">{{ dataMsg }}</p>

    <h3>API Keys</h3>
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
import { useRouter } from 'vue-router'
import { Check, Clipboard, KeyRound } from 'lucide-vue-next'
import {
  exportCollection, exportCatalogPDF, importCollection,
  generateApiKey, listApiKeys, revokeApiKey,
} from '@/api/client'
import { CATEGORIES, MATERIALS } from '@/types'
import type { Coin, ApiKey, Category, Material } from '@/types'

const router = useRouter()

const exporting = ref(false)
const exportingPdf = ref(false)
const dataMsg = ref('')
const dataError = ref(false)

const CSV_IMPORT_DEFAULTS: Partial<Coin> = {
  category: 'Roman',
  material: 'Silver',
  denomination: '',
  ruler: '',
  era: '',
  mint: '',
  weightGrams: null,
  diameterMm: null,
  grade: '',
  obverseInscription: '',
  reverseInscription: '',
  obverseDescription: '',
  reverseDescription: '',
  rarityRating: '',
  purchasePrice: null,
  currentValue: null,
  purchaseDate: null,
  purchaseLocation: '',
  storageLocationId: null,
  storageLocation: null,
  notes: '',
  referenceUrl: '',
  referenceText: 'Store Link',
  isWishlist: false,
  isSold: false,
  soldPrice: null,
  soldDate: null,
  soldTo: '',
  isPrivate: false,
}

function normalizeHeader(value: string): string {
  return value.trim().toLowerCase().replace(/[\s_-]+/g, '')
}

function parseCsvRows(text: string): string[][] {
  const rows: string[][] = []
  let row: string[] = []
  let cell = ''
  let inQuotes = false

  for (let i = 0; i < text.length; i++) {
    const char = text[i]
    const next = text[i + 1]

    if (char === '"') {
      if (inQuotes && next === '"') {
        cell += '"'
        i++
      } else {
        inQuotes = !inQuotes
      }
      continue
    }

    if (char === ',' && !inQuotes) {
      row.push(cell)
      cell = ''
      continue
    }

    if ((char === '\n' || char === '\r') && !inQuotes) {
      row.push(cell)
      cell = ''
      if (row.some(value => value.trim() !== '')) {
        rows.push(row)
      }
      row = []
      if (char === '\r' && next === '\n') {
        i++
      }
      continue
    }

    cell += char
  }

  row.push(cell)
  if (row.some(value => value.trim() !== '')) {
    rows.push(row)
  }

  return rows
}

function getValue(row: Record<string, string>, keys: string[]): string {
  for (const key of keys) {
    const normalized = normalizeHeader(key)
    const value = row[normalized]
    if (value !== undefined && value.trim() !== '') {
      return value.trim()
    }
  }
  return ''
}

function parseOptionalNumber(value: string): number | null {
  if (!value) return null
  const parsed = Number.parseFloat(value)
  return Number.isFinite(parsed) ? parsed : null
}

function parseBoolean(value: string): boolean {
  if (!value) return false
  return ['1', 'true', 'yes', 'y'].includes(value.trim().toLowerCase())
}

function parseDateString(value: string): string | null {
  if (!value) return null
  const trimmed = value.trim()
  if (/^\d{4}-\d{2}-\d{2}$/.test(trimmed)) {
    return `${trimmed}T00:00:00Z`
  }
  return trimmed
}

function parseCategory(value: string): Category {
  if (!value) return 'Roman'
  const match = CATEGORIES.find(category => category.toLowerCase() === value.trim().toLowerCase())
  return match ?? 'Other'
}

function parseMaterial(value: string): Material {
  if (!value) return 'Silver'
  const match = MATERIALS.find(material => material.toLowerCase() === value.trim().toLowerCase())
  return match ?? 'Other'
}

function parseCsvCoins(text: string): { coins: Partial<Coin>[]; skippedRows: number } {
  const rows = parseCsvRows(text)
  if (rows.length < 2) {
    throw new Error('CSV must include a header row and at least one data row.')
  }

  const headers = rows[0]?.map(normalizeHeader) ?? []
  const coins: Partial<Coin>[] = []
  let skippedRows = 0

  for (const row of rows.slice(1)) {
    const values = headers.reduce<Record<string, string>>((acc, header, index) => {
      acc[header] = row[index]?.trim() ?? ''
      return acc
    }, {})

    const name = getValue(values, ['name', 'coinName', 'title'])
    if (!name) {
      skippedRows++
      continue
    }

    const purchasePrice = parseOptionalNumber(getValue(values, ['purchasePrice', 'pricePaid']))
    const currentValue = parseOptionalNumber(getValue(values, ['currentValue', 'estimatedValue'])) ?? purchasePrice

    coins.push({
      ...CSV_IMPORT_DEFAULTS,
      name,
      category: parseCategory(getValue(values, ['category'])),
      material: parseMaterial(getValue(values, ['material'])),
      denomination: getValue(values, ['denomination']),
      ruler: getValue(values, ['ruler', 'emperor', 'issuer']),
      era: getValue(values, ['era', 'date']),
      mint: getValue(values, ['mint']),
      weightGrams: parseOptionalNumber(getValue(values, ['weightGrams', 'weight', 'weight_g'])),
      diameterMm: parseOptionalNumber(getValue(values, ['diameterMm', 'diameter', 'diameter_mm'])),
      grade: getValue(values, ['grade', 'condition']),
      obverseInscription: getValue(values, ['obverseInscription', 'obverseLegend']),
      reverseInscription: getValue(values, ['reverseInscription', 'reverseLegend']),
      obverseDescription: getValue(values, ['obverseDescription', 'obverse']),
      reverseDescription: getValue(values, ['reverseDescription', 'reverse']),
      rarityRating: getValue(values, ['rarityRating', 'reference', 'catalogReference']),
      purchasePrice,
      currentValue,
      purchaseDate: parseDateString(getValue(values, ['purchaseDate', 'acquiredDate'])),
      purchaseLocation: getValue(values, ['purchaseLocation', 'store', 'dealer']),
      notes: getValue(values, ['notes']),
      referenceUrl: getValue(values, ['referenceUrl', 'url']),
      referenceText: getValue(values, ['referenceText', 'referenceLabel']) || 'Store Link',
      isWishlist: parseBoolean(getValue(values, ['isWishlist', 'wishlist'])),
      isSold: parseBoolean(getValue(values, ['isSold', 'sold'])),
      soldPrice: parseOptionalNumber(getValue(values, ['soldPrice'])),
      soldDate: parseDateString(getValue(values, ['soldDate'])),
      soldTo: getValue(values, ['soldTo']),
      isPrivate: parseBoolean(getValue(values, ['isPrivate', 'private'])),
    })
  }

  return { coins, skippedRows }
}

function escapeCsvValue(value: string): string {
  if (value.includes('"') || value.includes(',') || value.includes('\n')) {
    return `"${value.replaceAll('"', '""')}"`
  }
  return value
}

function downloadCsvTemplate() {
  const headers = [
    'name', 'category', 'material', 'denomination', 'ruler', 'era', 'mint', 'weightGrams', 'diameterMm',
    'grade', 'purchasePrice', 'currentValue', 'purchaseDate', 'purchaseLocation', 'notes',
    'referenceUrl', 'referenceText', 'isWishlist',
  ]
  const sampleRows = [
    ['Augustus Denarius', 'Roman', 'Silver', 'Denarius', 'Augustus', '27 BC - 14 AD', 'Rome', '3.82', '19.5', 'VF', '450', '600', '2024-03-15', 'Heritage Auctions', 'Strong portrait with clear legend', 'https://www.acsearch.info/', 'ACSearch', 'false'],
    ['Constantius II Follis', 'Roman', 'Bronze', 'Follis', 'Constantius II', '337-361 AD', 'Antioch', '2.9', '18.1', 'F', '35', '45', '2025-01-20', 'Local show', 'Entry-level late Roman bronze', '', 'Store Link', 'false'],
  ]

  const lines = [
    headers.map(escapeCsvValue).join(','),
    ...sampleRows.map(row => row.map(escapeCsvValue).join(',')),
  ]
  const csv = `${lines.join('\n')}\n`
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'coin-import-template.csv'
  link.click()
  URL.revokeObjectURL(url)
}

function openCsvGuide() {
  router.push({ path: '/settings', query: { tab: 'help' } })
}

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
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  dataMsg.value = ''
  dataError.value = false

  try {
    const text = await file.text()
    let coins: Partial<Coin>[] = []
    let skippedRows = 0

    if (file.name.toLowerCase().endsWith('.csv')) {
      const parsed = parseCsvCoins(text)
      coins = parsed.coins
      skippedRows = parsed.skippedRows
    } else {
      const parsed = JSON.parse(text)
      if (!Array.isArray(parsed)) {
        throw new Error('JSON import must be an array of coins')
      }
      coins = parsed as Partial<Coin>[]
    }

    if (coins.length === 0) {
      throw new Error('No valid rows found')
    }

    const res = await importCollection(coins)
    dataMsg.value = skippedRows > 0
      ? `Imported ${res.data.imported} coins (${skippedRows} skipped rows missing a name)`
      : `Imported ${res.data.imported} coins`
  } catch {
    dataMsg.value = 'Import failed — ensure the file is valid JSON or CSV'
    dataError.value = true
  } finally {
    input.value = ''
  }
}

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

.settings-section h3 {
  font-size: 0.9rem;
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

.api-key-description {
  margin-bottom: 1rem;
}

.no-api-keys {
  margin-top: 0.5rem;
}

.import-btn {
  cursor: pointer;
}

.import-actions,
.apikey-generate,
.apikey-scope-selector,
.apikey-reveal-box,
.apikey-title-row {
  display: flex;
  align-items: center;
}

.import-actions {
  gap: 0.5rem;
  flex-wrap: wrap;
  justify-content: flex-end;
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

@media (max-width: 640px) {
  .setting-item {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
