// Wire shapes for the coin viewer MCP App. Coin fields mirror the Go
// services.CollectionCoinSummary returned by search_collection and get_coin.

export interface CoinSummary {
  id: number
  name: string
  category?: string
  denomination?: string
  era?: string
  ruler?: string
  mint?: string
  material?: string
  grade?: string
  weightGrams?: number | null
  diameterMm?: number | null
  purchasePrice?: number | null
  currentValue?: number | null
  missingFields?: string[] | null
}

export interface CoinImageRef {
  coinId: number
  imageId: number
  imageType: string
  isPrimary: boolean
  mimeType: string
}

export interface CoinImage extends CoinImageRef {
  src: string
}

export type ContentBlock =
  | { type: 'text'; text: string }
  | { type: 'image'; data: string; mimeType: string }
  | { type: string; [key: string]: unknown }

export interface CallToolResult {
  content?: ContentBlock[]
  structuredContent?: unknown
  isError?: boolean
}

export interface HostContext {
  theme?: 'light' | 'dark'
  styles?: { variables?: Record<string, string | undefined> }
}
