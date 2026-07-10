<template>
  <section class="card">
    <h2 class="mb-5 border-b border-border-subtle pb-3 text-lg">Backups</h2>

    <div class="flex flex-col gap-4 border-b border-border-subtle py-3 md:flex-row md:items-center md:justify-between last:border-0">
      <div class="flex min-w-0 flex-col gap-[0.15rem]">
        <span class="text-base font-medium text-text-primary">Export Collection</span>
        <span class="text-sm text-text-muted">Download your collection data and photos as a zip archive</span>
      </div>
      <button
        class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
        :disabled="exporting"
        @click="handleExport"
      >
        {{ exporting ? 'Exporting...' : 'Export ZIP' }}
      </button>
    </div>

    <div class="flex flex-col gap-4 border-b border-border-subtle py-3 md:flex-row md:items-center md:justify-between last:border-0">
      <div class="flex min-w-0 flex-col gap-[0.15rem]">
        <span class="text-base font-medium text-text-primary">PDF Catalog</span>
        <span class="text-sm text-text-muted">Generate a styled PDF catalog with photos, grades, and valuations</span>
      </div>
      <button
        class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
        :disabled="exportingPdf"
        @click="handleExportPDF"
      >
        {{ exportingPdf ? 'Generating...' : 'Export PDF' }}
      </button>
    </div>

    <div class="flex flex-col gap-4 border-b border-border-subtle py-3 md:flex-row md:items-center md:justify-between last:border-0">
      <div class="flex min-w-0 flex-col gap-[0.15rem]">
        <span class="text-base font-medium text-text-primary">Import Collection</span>
        <span class="text-sm text-text-muted">Import coins from a JSON or CSV file</span>
      </div>
      <div class="flex flex-wrap items-center justify-end gap-2">
        <label class="btn btn-secondary btn-sm inline-flex cursor-pointer focus-within:outline-2 focus-within:outline-gold focus-within:outline-offset-2">
          Import
          <input type="file" accept=".json,.csv,text/csv" hidden @change="handleImport" />
        </label>
        <button
          class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          @click="downloadCsvTemplate"
        >
          CSV Template
        </button>
        <button
          class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          @click="openCsvGuide"
        >
          Guide
        </button>
      </div>
    </div>

    <p v-if="dataMsg" class="my-2 text-body" :class="dataError ? 'text-[var(--cat-byzantine)]' : 'text-gold'">{{ dataMsg }}</p>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { exportCollection, exportCatalogPDF, importCollection } from '@/api/client'
import { CATEGORIES, MATERIALS } from '@/types'
import type { Coin, Category, Material } from '@/types'

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
</script>
