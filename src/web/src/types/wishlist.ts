// wishlist types. Split out of the former single-file src/types/index.ts;
// re-exported from '@/types' so existing imports keep working.
import type { Coin, CoinMutationPayload } from '@/types/coin'

export type WishlistSearchAlertCadence = 'manual' | 'daily' | 'weekly' | 'monthly'

export type AlertRunStatus = 'queued' | 'running' | 'completed' | 'failed' | 'partial' | 'rate_limited' | 'cancelled'

export type CandidateProvenanceStatus = 'verified' | 'partial' | 'unverified'

export type AlertCandidateState = 'active' | 'dismissed' | 'converted' | 'suppressed' | 'needs_review'

export type CandidateDismissalReason = 'irrelevant' | 'duplicate' | 'price_too_high' | 'poor_provenance' | 'other'

export interface WishlistSearchAlertCriteria {
  rulerOrIssuer: string
  coinType: string
  dateFrom: number | null
  dateTo: number | null
  mint: string
  mintLocationId: number | null
  material: string
  gradeOrCondition: string
  priceMin: number | null
  priceMax: number | null
  currency: string
  dealerPreference: string
  sourceFilters: string[]
  keywords: string
  notes: string
}

export interface WishlistSearchAlert {
  id: number
  userId: number
  name: string
  rulerOrIssuer: string
  coinType: string
  dateFrom: number | null
  dateTo: number | null
  mint: string
  mintLocationId: number | null
  material: string
  gradeOrCondition: string
  priceMin: number | null
  priceMax: number | null
  currency: string
  dealerPreference: string
  sourceFilters: string[]
  keywords: string
  notes: string
  cadence: WishlistSearchAlertCadence
  isActive: boolean
  lastRunAt: string | null
  createdAt: string
  updatedAt: string
}

export interface WishlistSearchAlertInput {
  name: string
  criteria: WishlistSearchAlertCriteria
  cadence: WishlistSearchAlertCadence
  isActive: boolean
}

export interface WishlistSearchAlertListResponse {
  alerts: WishlistSearchAlert[]
  total: number
  page: number
  limit: number
}

export interface AlertRun {
  id: number
  alertId: number
  userId: number
  triggerType: 'manual' | 'scheduled'
  status: AlertRunStatus
  startedAt: string
  completedAt: string | null
  durationMs: number
  criteriaSnapshot: string
  resultCount: number
  newCount: number
  duplicateCount: number
  dismissedCount: number
  partialWarnings: string[]
  errorMessage: string
  rateLimitStatus: string
  createdAt: string
}

export interface AlertRunResult {
  runId: number
  alertId: number
  status: AlertRunStatus
  startedAt: string
  completedAt: string | null
  resultCount: number
  newCount: number
  duplicateCount: number
  dismissedCount: number
  partialWarnings: string[]
  rateLimitStatus: string
  errorMessage?: string
  candidates?: AlertCandidate[]
}

export interface AlertRunListResponse {
  runs: AlertRun[]
  total: number
  page: number
  limit: number
}

export interface CandidateProvenance {
  id: number
  candidateId: number
  field: string
  value: string
  sourceUrl: string
  observedAt: string
  confidence: 'high' | 'medium' | 'low' | string
  verificationState: CandidateProvenanceStatus
  notes: string
}

export interface AlertCandidate {
  id: number
  userId: number
  alertId: number
  runId: number
  sourceUrl: string
  canonicalSourceUrl: string
  sourceName: string
  title: string
  observedPrice: number | null
  observedCurrency: string
  reasonForMatch: string
  fields: Record<string, string>
  lastSeenAt: string
  firstSeenAt: string
  provenanceStatus: CandidateProvenanceStatus
  lifecycleState: AlertCandidateState
  duplicateOfCandidateId: number | null
  matchingWishlistCoinId: number | null
  convertedCoinId: number | null
  dismissalReason: string
  provenance?: CandidateProvenance[]
  createdAt: string
  updatedAt: string
}

export interface AlertCandidateListResponse {
  candidates: AlertCandidate[]
  total: number
  page: number
  limit: number
}

export interface DismissWishlistSearchAlertCandidateInput {
  reason: CandidateDismissalReason | ''
  notes?: string
}

export interface ConvertWishlistSearchAlertCandidateInput {
  coin: CoinMutationPayload
  acknowledgeDuplicateWarning: boolean
}

export interface ConvertWishlistSearchAlertCandidateResponse {
  coin?: Coin
  candidate: AlertCandidate
  warnings: string[]
}

export interface AdjustWishlistSearchAlertCriteriaInput {
  candidateIds: number[]
  criteria: WishlistSearchAlertCriteria
}
