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

export function detailRows(coin: CoinSummary): Array<[string, string]> {
  const rows: Array<[string, string | null | undefined]> = [
    ['Ruler', coin.ruler],
    ['Denomination', coin.denomination],
    ['Era', coin.era],
    ['Mint', coin.mint],
    ['Material', coin.material],
    ['Grade', coin.grade],
    ['Category', coin.category],
    ['Weight', formatNumber(coin.weightGrams, ' g')],
    ['Diameter', formatNumber(coin.diameterMm, ' mm')],
    ['Purchase price', formatNumber(coin.purchasePrice)],
    ['Current value', formatNumber(coin.currentValue)],
  ]
  return rows.filter((row): row is [string, string] => Boolean(row[1]))
}
