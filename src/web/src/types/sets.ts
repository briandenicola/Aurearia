// sets types. Split out of the former single-file src/types/index.ts;
// re-exported from '@/types' so existing imports keep working.
import type { Coin } from '@/types/coin'

export interface CollectionSetOption {
  id: number
  name: string
  color: string
  filterValue: string
  source: 'tag' | 'set'
}

export type CoinSetType = 'standard' | 'goal' | 'smart' | 'agentic'

export type LegacyCoinSetType = 'open' | 'defined' | 'tracker' | 'dynamic'

export type CoinSetTypeResponse = CoinSetType | LegacyCoinSetType

export function normalizeCoinSetType(setType: CoinSetTypeResponse): CoinSetType {
  if (setType === 'open') return 'standard'
  if (setType === 'defined') return 'goal'
  if (setType === 'tracker' || setType === 'dynamic') return 'agentic'
  return setType
}

export interface CoinSet {
  id: number
  userId: number
  name: string
  description?: string
  color: string
  icon?: string
  setType: CoinSetTypeResponse
  parentSetId?: number | null
  targetCompletionDate?: string | null
  createdAt: string
  updatedAt: string
}

export interface CoinSetSummary {
  id: number
  name: string
  color: string
  icon?: string
  setType: CoinSetTypeResponse
  coinCount: number
  totalValue: number
  completionPercentage?: number | null
  valueChangePercent?: number | null
  agenticStatus?: string | null
  pinned?: boolean
  pinnedAt?: string | null
}

export interface CoinSetDetail extends CoinSetSummary {
  description?: string
  parentSetId?: number | null
  targetCompletionDate?: string | null
  totalInvested: number
  avgValuePerCoin?: number | null
  highestValueCoinId?: number | null
  agenticPrompt?: string | null
  agenticStatus?: string | null
}

export type CoinRecommendationTargetType = 'set' | 'tag'

export type CoinRecommendationStatus = 'pending' | 'accepted' | 'rejected' | 'dismissed'

export interface CoinRecommendation {
  id: number
  targetType: CoinRecommendationTargetType
  targetId: number
  targetName: string
  score: number
  confidence: 'high' | 'medium' | 'low'
  reasons: string[]
  status: CoinRecommendationStatus
}

export interface CreateCoinSetRequest {
  name: string
  description?: string
  color?: string
  icon?: string
  setType: CoinSetType
  parentSetId?: number | null
  targetCompletionDate?: string | null
  smartCriteria?: Record<string, unknown> | null
  templateId?: string | null
  agenticPrompt?: string | null
}

export interface CreateCoinSetFromCsvRequest extends CreateCoinSetRequest {
  csv: string
}

export interface SetBuilderRun {
  id: number
  userId: number
  prompt: string
  status: 'queued' | 'running' | 'completed' | 'failed'
  provider?: string
  model?: string
  transcriptSummary?: string
  errorMessage?: string
  terminationReason?: string
  usedTurns?: number
  createdAt: string
  updatedAt: string
}

export interface CreateSetBuilderRunRequest {
  prompt: string
}

export interface SetProposalSlot {
  id: number
  proposalId: number
  label: string
  criteria?: Record<string, unknown> | null
  group?: string
  sortOrder: number
  verificationStatus: 'verified' | 'unverified'
  sourceNote?: string
  validationNote?: string
  createdAt: string
  updatedAt: string
}

export interface UpdateSetProposalSlotRequest {
  label: string
  criteria?: Record<string, unknown> | null
  group?: string
  sortOrder: number
  verificationStatus: 'verified' | 'unverified'
  sourceNote?: string
  validationNote?: string
}

export interface UpdateSetProposalRequest {
  proposedName: string
  description?: string
  color?: string
  selectedScope?: string
  slots: UpdateSetProposalSlotRequest[]
}

export interface RegenerateSetProposalRequest {
  feedback: string
}

export interface SetProposalPrematchSummary {
  estimatedFilled?: number
  estimatedTotal?: number
  notes?: string
}

export interface SetProposal {
  id: number
  userId: number
  builderRunId: number
  run?: SetBuilderRun
  originalPrompt: string
  status: 'pending' | 'approved' | 'rejected' | 'expired' | 'creation_failed'
  proposedName: string
  proposedSlug?: string
  description?: string
  color: string
  selectedScope?: string
  scopeOptions?: {
    scopeSummary?: string
    groupBy?: string
    options?: Array<{
      label: string
      description?: string
      estimated_slot_count?: number
      recommended?: boolean
    }>
  } | null
  rosterPayload?: {
    transcriptSummary?: string
    turnsUsed?: number
  } | null
  preMatchSummary?: SetProposalPrematchSummary | null
  expiresAt: string
  rejectedAt?: string | null
  rejectionReason?: string
  approvalSetId?: number | null
  errorMessage?: string
  slots?: SetProposalSlot[]
  createdAt: string
  updatedAt: string
}

export type UpdateCoinSetRequest = Partial<CreateCoinSetRequest> & { pinned?: boolean }

export interface AddCoinToSetRequest {
  coinId: number
  notes?: string
  targetId?: number
}

export interface ReorderSetCoinsRequest {
  coinIds: number[]
}

// US2: Defined/Goal Sets and Completion
export interface CoinSetTarget {
  id: number
  setId: number
  label: string
  year?: number | null
  mintMark?: string | null
  denomination?: string | null
  country?: string | null
  material?: string | null
  matchRules?: Record<string, unknown> | null
  sortOrder: number
  createdAt?: string
}

export interface CoinSetCompletion {
  totalTargets: number
  completedTargets: number
  completionPercentage: number
  missingTargets: CoinSetTarget[]
  targets?: CoinSetTarget[]
  targetMatches?: Array<{
    target: CoinSetTarget
    coin?: Coin | null
  }>
  collectionItems?: number
  wishlistItems?: number
}

export interface CoinSetTemplate {
  id: string
  name: string
  category: string
  description: string
  version: number
  targets?: CoinSetTemplateTarget[]
}

export interface CoinSetTemplateTarget {
  label: string
  year?: number | null
  mintMark?: string | null
  denomination?: string | null
  country?: string | null
  material?: string | null
  sortOrder: number
}

// US3: Snapshots and Trends
export interface CoinSetSnapshot {
  id?: number
  setId?: number
  userId?: number
  snapshotDate: string
  totalValue: number
  totalInvested: number
  coinCount: number
  completionPercentage?: number | null
  avgValuePerCoin?: number | null
  highestValueCoinId?: number | null
}

export interface CoinSetAnalytics {
  roiPercent?: number | null
  bestPerformerCoinId?: number | null
  worstPerformerCoinId?: number | null
  acquisitionRatePerMonth?: number | null
  projectedCompletionDate?: string | null
}

export interface CoinSetComparison {
  setId: number
  name: string
  startValue: number
  endValue: number
  valueChange: number
  valueChangePercent: number
  completionChange?: number | null
}

export type SmartCriteriaOperator = 'and' | 'or'

export type SmartCriteriaRuleOp = 'eq' | 'neq' | 'contains' | 'startsWith' | 'in' | 'between' | 'gte' | 'lte' | 'isNull' | 'isNotNull'

export interface SmartCriteriaRule {
  field: string
  op: SmartCriteriaRuleOp
  value?: unknown
}

export interface SmartCriteriaGroup {
  operator: SmartCriteriaOperator
  rules: Array<SmartCriteriaRule | SmartCriteriaGroup>
}

export interface SmartSetPreview {
  coinIds: number[]
  coinCount: number
  totalValue: number
}

export interface SmartCriteriaTemplate {
  id: number
  userId: number
  name: string
  description: string
  criteria: SmartCriteriaGroup
  createdAt: string
  updatedAt: string
}

export interface SuggestedSmartCriteria {
  id: string
  name: string
  description: string
  criteria: SmartCriteriaGroup
}
