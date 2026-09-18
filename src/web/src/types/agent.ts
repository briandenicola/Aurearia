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

export type CoinCopilotCapabilityReason =
  | 'disabled'
  | 'provider_unconfigured'
  | 'model_tool_calling_unsupported'
  | 'temporarily_unavailable'

export type CoinCopilotCapability =
  | {
      mode: 'copilot'
      enabled: true
      modelToolCallingSupported: true
      reason?: null
    }
  | {
      mode: 'legacy'
      enabled: boolean
      modelToolCallingSupported: false
      reason: CoinCopilotCapabilityReason
    }

export type CoinCopilotRunStatus =
  | 'queued'
  | 'running'
  | 'paused'
  | 'cancel_requested'
  | 'completed'
  | 'failed'
  | 'cancelled'

export interface CoinCopilotUsage {
  iterations: number
  toolCalls: number
  inputTokens: number
  outputTokens: number
}

export type CoinCopilotPlanStatus = 'pending' | 'in_progress' | 'completed' | 'skipped' | 'failed'

export interface CoinCopilotPlanItem {
  id: string
  title: string
  status: CoinCopilotPlanStatus
}

export type CoinCopilotSpecialistCapability =
  | 'market_search'
  | 'auction_search'
  | 'price_trends'
  | 'similar_lots'

export type CoinCopilotSpecialistOutcome = 'complete' | 'partial' | 'no_match' | 'unavailable'
export type CoinCopilotEvidenceKind = 'dealer_listing' | 'auction_lot' | 'sale_observation' | 'similar_lot'
export type CoinCopilotEvidenceConfidence = 'high' | 'medium' | 'low'

export interface CoinCopilotSpecialistEvidence {
  kind: CoinCopilotEvidenceKind
  title: string
  sourceUrl: string
  observedAt: string
  confidence: CoinCopilotEvidenceConfidence
  verificationState: 'verified' | 'partial'
  facts: string[]
  matchedAttributes: string[]
  materialDifferences: string[]
}

export interface CoinCopilotPriceTrend {
  state: 'rising' | 'stable' | 'declining' | 'unknown'
  sampleSize: number
  dateFrom: string | null
  dateTo: string | null
  currency: string | null
  priceBasis: 'hammer' | 'realized_including_premium' | null
  low: number | null
  median: number | null
  high: number | null
  confidence: CoinCopilotEvidenceConfidence
  limitations: string[]
  supportingSourceIds: string[]
}

export interface CoinCopilotSpecialistResult {
  capability: CoinCopilotSpecialistCapability
  outcome: CoinCopilotSpecialistOutcome
  items: CoinCopilotSpecialistEvidence[]
  trend: CoinCopilotPriceTrend | null
  warnings: string[]
  truncation: {
    truncated: boolean
    originalBytes: number
    persistedBytes: number
    digest: string
    omittedItems: number
  }
}

export type CoinCopilotClarificationInputType = 'text' | 'single_choice' | 'boolean'

export interface CoinCopilotClarification {
  question: string
  inputType: CoinCopilotClarificationInputType
  choices: string[]
  checkpointVersion: number
}

export interface CoinCopilotCheckpointSummary {
  version: number
  plan: CoinCopilotPlanItem[]
  pendingClarification: CoinCopilotClarification | null
}

export interface CoinCopilotRun {
  id: string
  threadId: string
  status: CoinCopilotRunStatus
  goal: string
  checkpointVersion: number
  lastSeq: number
  attempt: number
  finalAnswer: string | null
  failureCode: string | null
  failureMessage: string | null
  resumeDeadline: string | null
  usage: CoinCopilotUsage
  createdAt: string
  updatedAt: string
}

export interface CoinCopilotRunEnvelope {
  run: CoinCopilotRun
  reused: boolean
}

export interface CoinCopilotThreadMessage {
  role: 'user' | 'assistant'
  content: string
  runId: string | null
  createdAt: string
}

export interface CoinCopilotThread {
  id: string
  title: string
  messages: CoinCopilotThreadMessage[]
  runs: CoinCopilotRun[]
  createdAt: string
  updatedAt: string
}

export interface CoinCopilotThreadEnvelope {
  thread: CoinCopilotThread
}

export interface CoinCopilotRunLimits {
  maxIterations: number
  maxToolCalls: number
  maxConcurrentTools: number
  hardTimeoutSeconds: number
  maxPersistedToolResultBytes: number
}

interface CoinCopilotEventBase<TType extends string, TPayload> {
  seq: number
  threadId: string
  runId: string
  executionId: string
  type: TType
  ts: string
  payload: TPayload
}

export type CoinCopilotRunStartedEvent = CoinCopilotEventBase<'run_started', {
  status: 'running'
  executionId: string
  attempt: number
  limits: CoinCopilotRunLimits
}>

export type CoinCopilotPlanUpdatedEvent = CoinCopilotEventBase<'plan_updated', {
  plan: CoinCopilotPlanItem[]
}>

export type CoinCopilotToolStartedEvent = CoinCopilotEventBase<'tool_started', {
  toolCallId: string
  toolName: string
  stepId: string
}>

export type CoinCopilotToolCompletedEvent = CoinCopilotEventBase<'tool_completed', {
  toolCallId: string
  toolName: string
  stepId: string
  status: 'succeeded' | 'failed' | 'cancelled' | 'rejected'
  durationMs: number
  resultSummary: string
  truncated: boolean
  specialistResult?: CoinCopilotSpecialistResult
}>

export type CoinCopilotClarificationRequiredEvent = CoinCopilotEventBase<'clarification_required', CoinCopilotClarification>

export type CoinCopilotRunPausedEvent = CoinCopilotEventBase<'run_paused', {
  reason: 'clarification_required'
  checkpointVersion: number
  resumeDeadline: string
}>

export type CoinCopilotRunResumedEvent = CoinCopilotEventBase<'run_resumed', {
  executionId: string
  attempt: number
  checkpointVersion: number
}>

export type CoinCopilotRunCancelledEvent = CoinCopilotEventBase<'run_cancelled', {
  reason: 'owner_cancelled'
}>

export type CoinCopilotRunCompletedEvent = CoinCopilotEventBase<'run_completed', {
  answer: string
  usage: CoinCopilotUsage
}>

export type CoinCopilotFailureCode =
  | 'agent_unavailable'
  | 'execution_lost'
  | 'invalid_agent_frame'
  | 'invalid_tool_call'
  | 'iteration_limit_exceeded'
  | 'tool_limit_exceeded'
  | 'time_limit_exceeded'
  | 'model_tool_calling_unsupported'
  | 'resume_window_expired'
  | 'internal'

export type CoinCopilotRunFailedEvent = CoinCopilotEventBase<'run_failed', {
  code: CoinCopilotFailureCode
  message: string
  retryable: boolean
  usage: CoinCopilotUsage
}>

export type CoinCopilotEvent =
  | CoinCopilotRunStartedEvent
  | CoinCopilotPlanUpdatedEvent
  | CoinCopilotToolStartedEvent
  | CoinCopilotToolCompletedEvent
  | CoinCopilotClarificationRequiredEvent
  | CoinCopilotRunPausedEvent
  | CoinCopilotRunResumedEvent
  | CoinCopilotRunCancelledEvent
  | CoinCopilotRunCompletedEvent
  | CoinCopilotRunFailedEvent

export interface CoinCopilotStreamTruncated {
  runId: string
  status: CoinCopilotRunStatus
  earliestSeq: number
  lastSeq: number
}

export interface CoinCopilotStreamEnd {
  runId: string
  status: CoinCopilotRunStatus
}
