import type { CallToolResult, CoinImage, CoinImageRef, CoinSummary } from './types'

// Prefers structuredContent; falls back to the JSON text block that the Go
// SDK always includes, for hosts that drop structured content.
export function structured(result: CallToolResult | null | undefined): Record<string, unknown> | null {
  if (!result) return null
  if (result.structuredContent && typeof result.structuredContent === 'object') {
    return result.structuredContent as Record<string, unknown>
  }
  const text = result.content?.find(
    (block): block is { type: 'text'; text: string } => block.type === 'text' && typeof block.text === 'string',
  )
  if (!text) return null
  try {
    const parsed: unknown = JSON.parse(text.text)
    return parsed && typeof parsed === 'object' ? (parsed as Record<string, unknown>) : null
  } catch {
    return null
  }
}

export interface SearchPage {
  coins: CoinSummary[]
  // Total matches on the server; at least coins.length. Older servers send
  // no total, in which case the page is all there is.
  total: number
  query: string
}

export function searchPageFrom(result: CallToolResult | null | undefined): SearchPage | null {
  const data = structured(result)
  if (!data || !Array.isArray(data.coins)) return null
  const coins = data.coins as CoinSummary[]
  const total = typeof data.total === 'number' ? data.total : coins.length
  const query = typeof data.query === 'string' ? data.query : ''
  return { coins, total: Math.max(total, coins.length), query }
}

export function coinFrom(result: CallToolResult | null | undefined): CoinSummary | null {
  const coin = structured(result)?.coin
  return coin && typeof coin === 'object' ? (coin as CoinSummary) : null
}

// get_coin_images returns image blocks in content and, at the same index,
// their metadata in structuredContent.images.
export function imagesFrom(result: CallToolResult): CoinImage[] {
  const refs = (structured(result)?.images ?? []) as CoinImageRef[]
  const blocks = (result.content ?? []).filter(
    (block): block is { type: 'image'; data: string; mimeType: string } =>
      block.type === 'image' && typeof block.data === 'string' && typeof block.mimeType === 'string',
  )
  const images: CoinImage[] = []
  refs.forEach((ref, index) => {
    const block = blocks[index]
    if (block) images.push({ ...ref, src: `data:${block.mimeType};base64,${block.data}` })
  })
  return images
}

// Who and what: "Trajan · Denarius".
export function attribution(coin: CoinSummary): string {
  return [coin.ruler, coin.denomination].filter(Boolean).join(' · ')
}

// Eras arrive as lowercase enum values ("ancient"); mints and other fields
// are shown exactly as recorded.
function capitalize(value: string | undefined): string | undefined {
  return value ? value.charAt(0).toUpperCase() + value.slice(1) : value
}

// When and where: "Ancient · Rome".
export function context(coin: CoinSummary): string {
  return [capitalize(coin.era), coin.mint].filter(Boolean).join(' · ')
}

const MATERIAL_TONES = new Set(['gold', 'silver', 'bronze', 'copper', 'electrum'])

// A known metal for the material marker, or null for Other and unknowns.
export function materialTone(coin: CoinSummary): string | null {
  if (!coin.material || coin.missingFields?.includes('material')) return null
  const tone = coin.material.toLowerCase()
  return MATERIAL_TONES.has(tone) ? tone : null
}

function formatNumber(value: number | null | undefined, suffix = ''): string | null {
  if (typeof value !== 'number') return null
  return value.toLocaleString(undefined, { maximumFractionDigits: 2 }) + suffix
}

// Same formatting as the web app's formatCurrency, which shows coin prices in
// US dollars; keep them in step so the viewer never disagrees with Aurearia.
export function formatPrice(value: number | null | undefined): string | null {
  if (typeof value !== 'number') return null
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(value)
}

// Readable names for the keys in CollectionCoinSummary.missingFields.
const MISSING_FIELD_LABELS: Record<string, string> = {
  weightGrams: 'weight',
  diameterMm: 'diameter',
  purchasePrice: 'purchase price',
  currentValue: 'current value',
  purchaseDate: 'purchase date',
  storageLocation: 'storage location',
  vendorSku: 'vendor SKU',
  vendorInvoice: 'vendor invoice',
  referenceUrl: 'reference link',
  referenceText: 'reference',
}

export function missingLabels(coin: CoinSummary): string[] {
  return (coin.missingFields ?? []).map(
    (field) => MISSING_FIELD_LABELS[field] ?? field.replace(/([a-z])([A-Z])/g, '$1 $2').toLowerCase(),
  )
}

export interface FactGroup {
  title: string
  rows: Array<[string, string]>
}

// Facts grouped for the detail view. Fields the server reports as missing are
// left out even when they hold a placeholder (material "Other"), so a field is
// never shown and listed as missing at once; empty groups are dropped.
export function factGroups(coin: CoinSummary): FactGroup[] {
  const missing = new Set(coin.missingFields ?? [])
  const keep = (rows: Array<[string, string, string | null | undefined]>) =>
    rows
      .filter(([field, , value]) => Boolean(value) && !missing.has(field))
      .map(([, label, value]) => [label, value as string] as [string, string])
  const groups: FactGroup[] = [
    {
      title: 'Attribution',
      rows: keep([
        ['ruler', 'Ruler', coin.ruler],
        ['denomination', 'Denomination', coin.denomination],
        ['mint', 'Mint', coin.mint],
        ['era', 'Era', capitalize(coin.era)],
        ['category', 'Category', coin.category],
      ]),
    },
    {
      title: 'Physical',
      rows: keep([
        ['material', 'Material', coin.material],
        ['weightGrams', 'Weight', formatNumber(coin.weightGrams, ' g')],
        ['diameterMm', 'Diameter', formatNumber(coin.diameterMm, ' mm')],
        ['grade', 'Grade', coin.grade],
      ]),
    },
  ]
  return groups.filter((group) => group.rows.length > 0)
}

export interface ValueFigure {
  label: string
  text: string
  tone?: 'up' | 'down'
}

// Current value, price paid and the change between them when both exist.
export function valueFigures(coin: CoinSummary): ValueFigure[] {
  const figures: ValueFigure[] = []
  const current = formatPrice(coin.currentValue)
  const paid = formatPrice(coin.purchasePrice)
  if (current) figures.push({ label: 'Value', text: current })
  if (paid) figures.push({ label: 'Paid', text: paid })
  if (typeof coin.currentValue === 'number' && typeof coin.purchasePrice === 'number' && coin.purchasePrice > 0) {
    const change = coin.currentValue - coin.purchasePrice
    const percent = Math.round((change / coin.purchasePrice) * 100)
    const sign = change > 0 ? '+' : change < 0 ? '−' : ''
    const amount = formatPrice(Math.abs(change)) as string
    figures.push({
      label: 'Change',
      text: `${sign}${amount} (${sign}${Math.abs(percent)}%)`,
      tone: change > 0 ? 'up' : change < 0 ? 'down' : undefined,
    })
  }
  return figures
}

// Cards and rows drop ".00" from whole amounts; the detail view keeps cents.
function formatCompactPrice(value: number | null | undefined): string | null {
  if (typeof value !== 'number') return null
  const cents = Number.isInteger(value) ? 0 : 2
  return new Intl.NumberFormat('en-US', {
    style: 'currency', currency: 'USD', minimumFractionDigits: cents, maximumFractionDigits: cents,
  }).format(value)
}

// The single figure a card has room for: what it is worth, else what was paid.
export function headlineValue(coin: CoinSummary): ValueFigure | null {
  const current = formatCompactPrice(coin.currentValue)
  if (current) return { label: 'Value', text: current }
  const paid = formatCompactPrice(coin.purchasePrice)
  return paid ? { label: 'Paid', text: paid } : null
}
