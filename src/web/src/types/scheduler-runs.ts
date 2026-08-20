// scheduler-runs types. Split out of the former single-file src/types/index.ts;
// re-exported from '@/types' so existing imports keep working.
import type { Coin } from '@/types/coin'

export interface AvailabilityRunSummary {
  runId: number
  coinsChecked: number
  available: number
  unavailable: number
  unknown: number
  durationMs: number
}

export interface AvailabilityResult {
  id: number
  runId: number
  coinId: number
  coinName: string
  url: string
  status: string
  reason: string
  httpStatus: number | null
  agentUsed: boolean
  checkedAt: string
}

export interface AvailabilityRun {
  id: number
  userId: number
  userName?: string
  triggerType: string
  triggerUserId: number | null
  cycleId?: number | null
  status: string
  failMessage?: string
  coinsChecked: number
  available: number
  unavailable: number
  unknown: number
  errors: number
  durationMs: number
  startedAt: string
  completedAt: string | null
  results?: AvailabilityResult[]
  createdAt: string
}

// Availability cycle — parent roll-up of per-user AvailabilityRun children for
// admin/scheduled wishlist availability checks (Feature 353). Coin-level counts live on
// each child AvailabilityRun; the cycle itself only rolls up child status counts.
export interface AvailabilityCycle {
  id: number
  triggerType: string
  triggerUserId: number | null
  status: string
  totalChildren: number
  queuedChildren: number
  runningChildren: number
  completedChildren: number
  failedChildren: number
  failMessage?: string
  startedAt: string
  completedAt: string | null
  createdAt: string
  children?: AvailabilityRun[]
}

export interface AvailabilityCycleListResponse {
  cycles: AvailabilityCycle[]
  total: number
  page: number
  limit: number
}

// GET /admin/availability-cycles/{id} returns the cycle itself with `children` populated
// (each child is a full AvailabilityRun with no per-coin `results` preloaded).
export type AvailabilityCycleDetail = AvailabilityCycle & { children: AvailabilityRun[] }

export interface AvailabilityCycleTriggerResponse {
  cycleId: number
  status: string
  message: string
}

export interface AvailabilityRunListResponse {
  runs: AvailabilityRun[]
  total: number
  page: number
  limit: number
}

export interface ValuationResult {
  id: number
  runId: number
  coinId: number
  coinName: string
  previousValue: number | null
  estimatedValue: number
  confidence: string
  reasoning: string
  changeExplanation?: string | null
  status: string
  errorMessage?: string
  checkedAt: string
}

export interface ValuationRun {
  id: number
  userId: number
  triggerType: string
  triggerUserId: number | null
  status: string
  totalCoins: number
  coinsChecked: number
  coinsUpdated: number
  coinsSkipped: number
  errors: number
  durationMs: number
  startedAt: string
  completedAt: string | null
  errorMessage?: string
  results?: ValuationResult[]
  createdAt: string
}

export interface CollectionHealthSnapshotRunResult {
  message?: string
  users?: number
  snapshotsCreated?: number
  skipped?: number
  errors?: number
  durationMs?: number
}

export interface CollectionHealthSnapshotRun {
  id: number
  triggerType: 'manual' | 'scheduled'
  status: 'running' | 'success' | 'error'
  usersEligible: number
  usersSnapshotted: number
  usersFailed: number
  durationMs: number
  startedAt: string
  completedAt: string | null
  errorMessage?: string
  createdAt: string
}

export interface SchedulerStatus {
  name: string
  enabled: boolean
  isRunning: boolean
  nextRunIn: number
}

export interface CoinOfDayRun {
  id: number
  triggerType: 'manual' | 'scheduled'
  triggerUserId: number | null
  status: 'queued' | 'running' | 'completed' | 'failed'
  startedAt: string
  completedAt: string | null
  picked: number
  skipped: number
  errors: number
  errorMessage: string
  createdAt: string
  updatedAt: string
}

export type FeaturedCoinSourceType = 'owned' | 'wishlist'

export interface FeaturedCoin {
  id: number
  userId: number
  coinId: number
  coin?: Coin
  summary: string
  /** Spec 354 D5: 'owned' preserves today's behavior; 'wishlist' surfaces the "Move to Collection" CTA. */
  sourceType: FeaturedCoinSourceType
  featuredAt: string
  createdAt: string
}
