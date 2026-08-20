// agent types. Split out of the former single-file src/types/index.ts;
// re-exported from '@/types' so existing imports keep working.
import type { CoinReferenceInput } from '@/types/coin'

export type AIJobStatus = string

export type AIJobType = 'coin_analysis' | 'coin_grading' | 'coin_value_estimate' | 'value_estimate' | 'valuation' | (string & {})

export interface CoinGradingResult {
  gradingReport: string
}

export interface AIJob {
  id: string
  userId?: number
  coinId: number
  jobType: AIJobType
  side?: 'obverse' | 'reverse' | null
  status: AIJobStatus
  result?: unknown
  errorMessage?: string | null
  createdAt: string
  updatedAt: string
  startedAt?: string | null
  completedAt?: string | null
}

export interface AIJobStartResponse {
  id?: string | number
  jobId?: string | number
  job?: AIJob
  status: AIJobStatus
  jobType: string
  coinId: number
  side?: 'obverse' | 'reverse' | null
  result?: unknown
  errorMessage?: string | null
  createdAt?: string
  updatedAt?: string
  startedAt?: string | null
  completedAt?: string | null
}

export interface CoinShow {
  name: string
  dates: string
  location: string
  venue: string
  url: string
  description: string
  entryFee: string
  notableDealers: string[]
}

export interface AgentChatMessage {
  role: 'user' | 'assistant'
  content: string
}

export interface AgentChatAppContext {
  route?: string
  activeCoinId?: number
}

export interface CollectionCoinSummary {
  id: number
  name: string
  category?: string
  era?: string
  ruler?: string
  material?: string
  currentValue?: number | null
}

export interface CollectionAggregateSummary {
  totalCoins: number
  totalWishlist: number
  totalSold: number
  totalCurrentUsd: number
  totalPurchaseUsd: number
}

export interface CollectionReadResult {
  resultType: string
  total?: number
  coins?: CollectionCoinSummary[]
  aggregate?: CollectionAggregateSummary
}

export interface CollectionDisambiguation {
  message: string
  candidates: CollectionCoinSummary[]
}

export interface CollectionProposalPreview {
  proposalId: string
  proposalToken: string
  coinId: number
  coinName: string
  changedFields: string[]
  changes: Record<string, unknown>
  expiresAt: string
}

export interface CollectionChatResponse {
  kind: 'read_result' | 'proposal' | 'disambiguation' | 'validation_error'
  message: string
  readResult?: CollectionReadResult
  disambiguation?: CollectionDisambiguation
  proposal?: CollectionProposalPreview
  errorCode?: string
}

export interface CoinSuggestion {
  name: string
  description: string
  category: string
  era: string
  ruler: string
  material: string
  denomination: string
  estPrice: string
  imageUrl: string
  sourceUrl: string
  sourceName: string
  candidateReferences?: CoinReferenceInput[]
}

export interface AgentChatResponse {
  message: string
  suggestions: CoinSuggestion[]
  collection?: CollectionChatResponse
}
