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

export function coinsFrom(result: CallToolResult | null | undefined): CoinSummary[] | null {
  const coins = structured(result)?.coins
  return Array.isArray(coins) ? (coins as CoinSummary[]) : null
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

export function subtitle(coin: CoinSummary): string {
  return [coin.ruler, coin.denomination, coin.era].filter(Boolean).join(' · ')
}

function formatNumber(value: number | null | undefined, suffix = ''): string | null {
  if (typeof value !== 'number') return null
  return value.toLocaleString(undefined, { maximumFractionDigits: 2 }) + suffix
}

// Same formatting as the web app's formatCurrency, which shows coin prices in
// US dollars; keep them in step so the viewer never disagrees with Aurearia.
function formatPrice(value: number | null | undefined): string | null {
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

// Rows the server reports as missing are left out even when they hold a
// placeholder (material "Other"), so a field is never shown and missing at once.
export function detailRows(coin: CoinSummary): Array<[string, string]> {
  const missing = new Set(coin.missingFields ?? [])
  const rows: Array<[string, string, string | null | undefined]> = [
    ['ruler', 'Ruler', coin.ruler],
    ['denomination', 'Denomination', coin.denomination],
    ['era', 'Era', coin.era],
    ['mint', 'Mint', coin.mint],
    ['material', 'Material', coin.material],
    ['grade', 'Grade', coin.grade],
    ['category', 'Category', coin.category],
    ['weightGrams', 'Weight', formatNumber(coin.weightGrams, ' g')],
    ['diameterMm', 'Diameter', formatNumber(coin.diameterMm, ' mm')],
    ['purchasePrice', 'Purchase price', formatPrice(coin.purchasePrice)],
    ['currentValue', 'Current value', formatPrice(coin.currentValue)],
  ]
  return rows
    .filter(([field, , value]) => Boolean(value) && !missing.has(field))
    .map(([, label, value]) => [label, value as string])
}
