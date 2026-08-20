// deep-identification types. Split out of the former single-file src/types/index.ts;
// re-exported from '@/types' so existing imports keep working.

export interface DeepIdentificationObservabilitySummary {
  jobsByTerminalStatus: Record<string, number>
  partialSuccessRate: number
  duration: { p50Ms: number; p95Ms: number }
  providers: Record<string, {
    statusCounts: Record<string, number>
    latency: { p50Ms: number; p95Ms: number }
  }>
  activeSseStreams: number
  reconnectCount: number
  truncationCount: number
  queueDepth: number
  hintDeletion: { success: number; failure: number }
  janitor: { recoverySweeps: number; retentionSweeps: number; failures: number }
}

// Deep Agentic Coin Identification (344-deep-agentic-coin-identification).
// Contract anchor: specs/344-deep-agentic-coin-identification/contracts/deep-identification.openapi.yaml
export type DeepProviderId = 'nomisma' | 'numista' | 'ngc' | 'ocre' | 'rpc'

export type DeepJobStatus = 'queued' | 'running' | 'partial' | 'completed' | 'failed' | 'cancelled'

export type DeepJobSource = 'intake' | 'saved_coin'

export type DeepProviderStatus =
  | 'pending'
  | 'running'
  | 'contributed'
  | 'no_match'
  | 'failed'
  | 'timed_out'
  | 'skipped'
  | 'not_automated'
  | 'unavailable'

export interface DeepJob {
  id: number
  coinId?: number | null
  source: DeepJobSource
  status: DeepJobStatus
  partialSuccess: boolean
  selectedProviders?: DeepProviderId[]
  requestedProviders?: DeepProviderId[]
  routerRationale?: string
  retryOfJobId?: number | null
  cancelRequested: boolean
  lastSeq: number
  eventsAvailable: boolean
  failureCode?: string
  failureMessage?: string
  appliedCoinId?: number | null
  appliedDraftId?: number | null
  appliedAt?: string | null
  /**
   * Server-computed (spec 354 T024/T026): true when `appliedCoinId` still
   * resolves to an existing, non-deleted owned coin. False when the linked
   * coin was deleted, or when nothing has been applied yet. Drives whether
   * the history/detail UI treats an "applied" job as re-appliable.
   */
  appliedCoinExists?: boolean
  startedAt?: string | null
  completedAt?: string | null
  expiresAt: string
  createdAt: string
}

export interface DeepReportCoverage {
  provider: DeepProviderId
  status: DeepProviderStatus
  note?: string
  linkOut?: string | null
}

export interface DeepClaim {
  field: string
  value: string
  confidence?: number
  /**
   * Absent for an image-sourced claim (contract `vision-hypothesis.md` §3:
   * "an image ref has no citation to validate"). Every provider-cited claim
   * still carries one — `citation` stays effectively required for those.
   */
  citation?: string
  excerpt?: string
}

export interface DeepReportDisagreement {
  field: string
  claims: DeepClaim[]
  resolution: 'unresolved' | 'preferred'
}

export interface DeepReportAttribution {
  provider: DeepProviderId
  text: string
  identifier?: string | null
}

/**
 * One typed, bounded-confidence value the vision hypothesis supports for a
 * single coin field (contract `vision-hypothesis.md` §1). A field the images
 * do not support is omitted entirely — never guessed at low confidence.
 */

export interface HypothesisFieldValue {
  value: string
  confidence: number
}

/**
 * The vision node's typed output — "what the images alone said" before any
 * provider/catalogue evidence was combined (FR-008, RD-6). Field names
 * mirror the coin-field vocabulary shared with `DeepProposal.fields`.
 */

export interface CoinHypothesis {
  ruler?: HypothesisFieldValue
  denomination?: HypothesisFieldValue
  material?: HypothesisFieldValue
  mint?: HypothesisFieldValue
  dateRange?: HypothesisFieldValue
  era?: HypothesisFieldValue
  obverseInscription?: HypothesisFieldValue
  reverseInscription?: HypothesisFieldValue
  obverseDescription?: HypothesisFieldValue
  reverseDescription?: HypothesisFieldValue
  diameterMm?: HypothesisFieldValue
  weightGrams?: HypothesisFieldValue
  notes?: HypothesisFieldValue
  coin_type?: HypothesisFieldValue
  /** Short bounded prose for the narrative writer only — never itself a proposed field value. */
  observations?: string
  /**
   * `false` means the images could not be read clearly enough to produce
   * findings — a distinct, plainly-stated state from "the pipeline dropped
   * the result" (the original undiagnosable failure this panel exists to fix).
   */
  legible: boolean
}

export interface DeepReport {
  schemaVersion: number
  narrative: string
  coverage: DeepReportCoverage[]
  disagreements?: DeepReportDisagreement[]
  unresolvedQuestions?: string[]
  attributions?: DeepReportAttribution[]
  partialSuccess: boolean
  generatedAt: string
  /**
   * Typed outcome of the quick-evidence (NGC quick-lookup) pass that runs
   * before the main Deep Analysis pipeline. Optional — absent on reports
   * generated before 351 T014/T015/T016 landed. `unavailable` means the
   * lookup did not complete (timeout/error), distinct from `no_data`
   * (it completed and genuinely found nothing).
   */
  quickLookupOutcome?: 'ok' | 'no_data' | 'unavailable'
  /**
   * Additive, optional (contract `vision-hypothesis.md` §4, FR-008). Present
   * when the vision call produced anything; absent on reports persisted
   * before Feature 351. The key name is deliberately snake_case — Go passes
   * the Python synthesis JSON through verbatim for this key, so the wire
   * shape genuinely is `image_hypothesis`, not `imageHypothesis`, unlike the
   * rest of this interface.
   */
  image_hypothesis?: CoinHypothesis
}

export interface DeepProposalFieldEntry {
  proposed: unknown
  confidence?: number
  evidence?: DeepClaim[]
  ownerEdited: boolean
  ownerValue: unknown
  accepted: boolean | null
}

export interface DeepProposal {
  schemaVersion: number
  targetCoinId?: number | null
  fields: Record<string, DeepProposalFieldEntry>
  sourceReportGeneratedAt?: string
}

export interface DeepJobEnvelope {
  job: DeepJob
  reused?: boolean
  report?: DeepReport | null
  proposal?: DeepProposal | null
}

export interface DeepJobListResponse {
  jobs: DeepJob[]
  nextCursor?: string
}

export interface DeepIdentificationCapability {
  enabled: boolean
  providers: DeepProviderId[]
}

export interface CreateDeepIdentificationJobInput {
  coinId?: number
  obverseImage?: File | null
  reverseImage?: File | null
  hintImages?: File[]
  notes?: string
  providers?: DeepProviderId[]
}

export interface DeepProposalFieldEdit {
  ownerValue?: unknown
  accepted?: boolean | null
}

export interface UpdateDeepIdentificationProposalInput {
  fields: Record<string, DeepProposalFieldEdit>
}

export type DeepApplyTarget = 'draft' | 'coin'

export interface ApplyDeepIdentificationProposalInput {
  target: DeepApplyTarget
  fields?: string[]
}

export interface DeepApplyResult {
  jobId: number
  draftId?: number | null
  coinId?: number | null
  appliedFields: string[]
  appliedAt: string
}

export interface ListDeepIdentificationJobsParams {
  coinId?: number
  activeOnly?: boolean
  status?: DeepJobStatus
  limit?: number
  cursor?: string
}

// SSE envelope from GET /api/deep-identification/jobs/{id}/events
// (contracts/sse-events.md §1/§2). Not a passthrough of the internal
// LangGraph stream - a persisted, replayable, application-owned shape.
export type DeepStreamEventType =
  | 'job_accepted'
  | 'status_changed'
  | 'router_selected'
  | 'provider_started'
  | 'provider_result'
  | 'evaluation'
  | 'synthesis_started'
  | 'progress'
  | 'terminal'

export interface DeepStreamEvent {
  seq: number
  jobId: number
  type: DeepStreamEventType | string
  ts: string
  payload: Record<string, unknown>
}

export interface DeepStreamTruncatedPayload {
  status: DeepJobStatus
  earliestSeq: number
  lastSeq: number
}

export interface DeepStreamTerminalPayload {
  status: DeepJobStatus
  partialSuccess: boolean
  failureCode?: string
  hasReport: boolean
  hasProposal: boolean
}
