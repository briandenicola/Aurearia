// agent endpoints. Split out of the former monolithic client.ts;
// re-exported from '@/api/client' so existing imports keep working.
import { api, refreshAccessToken, formatAgentServiceError } from '@/api/http'
import type {
  AgentChatAppContext,
  AgentChatMessage,
  ApplyDeepIdentificationProposalInput,
  CoinCopilotCapability,
  CoinCopilotEvidenceKind,
  CoinCopilotEvent,
  CoinCopilotRunEnvelope,
  CoinCopilotRunStatus,
  CoinCopilotSpecialistCapability,
  CoinCopilotSpecialistResult,
  CoinCopilotStreamEnd,
  CoinCopilotStreamTruncated,
  CoinCopilotThreadEnvelope,
  CoinSuggestion,
  CollectionChatResponse,
  CreateDeepIdentificationJobInput,
  DeepApplyResult,
  DeepIdentificationCapability,
  DeepJobEnvelope,
  DeepJobListResponse,
  DeepProposal,
  DeepProviderId,
  ListDeepIdentificationJobsParams,
  PortfolioSummary,
  UpdateDeepIdentificationProposalInput,
} from '@/types'
import { appendOptionalFormValue } from '@/api/endpoints/_shared'

export const getValuationPrompt = () => api.get<{ prompt: string; default: string }>('/agent/valuation-prompt')

export const getPortfolioSummary = () => api.get<PortfolioSummary>('/agent/portfolio-summary')

// Agent

export async function agentChatStream(
  message: string,
  history: AgentChatMessage[],
  onText: (text: string) => void,
  onDone: (message: string, suggestions: CoinSuggestion[], collection?: CollectionChatResponse) => void,
  onError: (error: string) => void,
  onStatus?: (status: string) => void,
  appContext?: AgentChatAppContext,
) {
  const baseURL = import.meta.env.VITE_API_BASE_URL || ''

  async function fetchWithAuthRetry(url: string, init: RequestInit): Promise<Response> {
    const firstHeaders = new Headers(init.headers ?? {})
    const token = localStorage.getItem('token')
    if (token) {
      firstHeaders.set('Authorization', `Bearer ${token}`)
    }

    const firstResp = await fetch(url, { ...init, headers: firstHeaders })
    if (firstResp.status !== 401) {
      return firstResp
    }

    const refreshedToken = await refreshAccessToken()
    const retryHeaders = new Headers(init.headers ?? {})
    retryHeaders.set('Authorization', `Bearer ${refreshedToken}`)
    return fetch(url, { ...init, headers: retryHeaders })
  }

  try {
    const resp = await fetchWithAuthRetry(`${baseURL}/api/agent/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ message, history, appContext }),
    })

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: `HTTP ${resp.status}` }))
      onError(formatAgentServiceError(err, `HTTP ${resp.status}`))
      return
    }

    const reader = resp.body?.getReader()
    if (!reader) { onError('No response body'); return }

    const decoder = new TextDecoder()
    let buffer = ''
    let accumulatedText = ''
    let terminalSent = false

    const sendDone = (
      finalMessage?: string,
      suggestions?: CoinSuggestion[],
      collection?: CollectionChatResponse,
    ) => {
      if (terminalSent) return
      terminalSent = true
      onDone(finalMessage || accumulatedText, Array.isArray(suggestions) ? suggestions : [], collection)
    }

    const sendError = (message: string) => {
      if (terminalSent) return
      terminalSent = true
      onError(message)
    }

    const handleDataLine = (line: string) => {
      if (!line.startsWith('data:')) return
      const data = line.replace(/^data:\s*/, '').trim()
      if (!data || data === '[DONE]') return

      try {
        const event = JSON.parse(data)
        if (event.type === 'text' && typeof event.text === 'string') {
          accumulatedText += event.text
          onText(event.text)
        } else if (event.type === 'status' && typeof event.message === 'string') {
          onStatus?.(event.message)
        } else if (event.type === 'done') {
          sendDone(
            typeof event.message === 'string' ? event.message : undefined,
            event.suggestions,
            event.collection,
          )
        } else if (event.type === 'error') {
          sendError(formatAgentServiceError(
            typeof event.message === 'string' ? event.message : '',
            'Agent stream error',
          ))
        }
      } catch {
        // Ignore malformed stream chunks.
      }
    }

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })

      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        handleDataLine(line)
      }
    }

    buffer += decoder.decode()
    if (buffer.trim()) {
      handleDataLine(buffer.trim())
    }

    if (!terminalSent) {
      if (accumulatedText.trim()) {
        sendDone(accumulatedText, [])
      } else {
        sendError('Stream ended unexpectedly')
      }
    }
  } catch (err: unknown) {
    onError(err instanceof Error ? err.message : 'Stream failed')
  }
}

export const commitCollectionProposal = (proposalId: string, proposalToken: string) =>
  api.post(`/agent/collection/proposals/${proposalId}/commit`, {
    proposalToken,
    confirm: true,
  })

export const cancelCollectionProposal = (proposalId: string) =>
  api.post(`/agent/collection/proposals/${proposalId}/cancel`, {})

export interface AnthropicModel {
  id: string
  name: string
}

export const getAnthropicModels = () => api.get<AnthropicModel[]>('/agent/models')

export const getCoinSearchPrompt = () => api.get<{ prompt: string; default: string }>('/agent/coin-search-prompt')

export const getCoinShowsPrompt = () => api.get<{ prompt: string; default: string }>('/agent/coin-shows-prompt')

// Agent Conversations
export interface ConversationSummary {
  id: number
  title: string
  createdAt: string
  updatedAt: string
}

export interface SavedConversation {
  id: number
  userId: number
  title: string
  messages: string
  createdAt: string
  updatedAt: string
}

export const listConversations = () => api.get<ConversationSummary[]>('/agent/conversations')

export const getConversation = (id: number) => api.get<SavedConversation>(`/agent/conversations/${id}`)

export const saveConversation = (data: { id?: number; title: string; messages: string }) =>
  api.post<SavedConversation>('/agent/conversations', data)

export const deleteConversation = (id: number) => api.delete(`/agent/conversations/${id}`)

export const getOllamaStatus = () =>
  api.get<{ available: boolean; model: string; url: string; message: string }>('/ollama-status')

export const getAIStatus = () =>
  api.get<{ available: boolean; provider: string; model: string; message: string }>('/ai-status')

// Agent status
export const getAgentStatus = () =>
  api.get<{ provider: string; configured: boolean }>('/agent/status')

export const getCoinCopilotCapability = () =>
  api.get<CoinCopilotCapability>('/agent/copilot/capability')

export const startCoinCopilotRun = (
  input: { goal: string; threadId?: string; appContext?: AgentChatAppContext },
  idempotencyKey: string,
) => api.post<CoinCopilotRunEnvelope>('/agent/copilot/runs', input, {
  headers: { 'Idempotency-Key': idempotencyKey },
})

export const getCoinCopilotRun = (runId: string) =>
  api.get<CoinCopilotRunEnvelope>(`/agent/copilot/runs/${runId}`)

export const getCoinCopilotThread = (threadId: string) =>
  api.get<CoinCopilotThreadEnvelope>(`/agent/copilot/threads/${threadId}`)

export const deleteCoinCopilotThread = (threadId: string) =>
  api.delete<void>(`/agent/copilot/threads/${threadId}`)

export const cancelCoinCopilotRun = (runId: string) =>
  api.post<CoinCopilotRunEnvelope>(`/agent/copilot/runs/${runId}/cancel`)

export const resumeCoinCopilotRun = (
  runId: string,
  input: { answer: string; expectedCheckpointVersion: number },
  idempotencyKey: string,
) => api.post<CoinCopilotRunEnvelope>(`/agent/copilot/runs/${runId}/resume`, input, {
  headers: { 'Idempotency-Key': idempotencyKey },
})

type CoinCopilotStreamHandlers = {
  onEvent: (event: CoinCopilotEvent) => void | boolean
  onTruncated?: (event: CoinCopilotStreamTruncated) => void
  onEnd?: (event: CoinCopilotStreamEnd) => void
}

type CoinCopilotStreamOptions = {
  since?: number
  signal?: AbortSignal
  seenSeqs?: Set<number>
}

const COPILOT_EVENT_TYPES = new Set<CoinCopilotEvent['type']>([
  'run_started',
  'plan_updated',
  'tool_started',
  'tool_completed',
  'clarification_required',
  'run_paused',
  'run_resumed',
  'run_cancelled',
  'run_completed',
  'run_failed',
])

const COPILOT_RUN_STATUSES = new Set<CoinCopilotRunStatus>([
  'queued',
  'running',
  'paused',
  'cancel_requested',
  'completed',
  'failed',
  'cancelled',
])

const COPILOT_TERMINAL_STATUSES = new Set<CoinCopilotRunStatus>(['completed', 'failed', 'cancelled'])

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isNonNegativeInteger(value: unknown) {
  return Number.isSafeInteger(value) && Number(value) >= 0
}

function hasOnlyKeys(value: Record<string, unknown>, keys: readonly string[]) {
  return Object.keys(value).every(key => keys.includes(key))
}

const SPECIALIST_KINDS: Record<CoinCopilotSpecialistCapability, CoinCopilotEvidenceKind> = {
  market_search: 'dealer_listing',
  auction_search: 'auction_lot',
  price_trends: 'sale_observation',
  similar_lots: 'similar_lot',
}

function isSafeSpecialistUrl(value: unknown): value is string {
  if (typeof value !== 'string' || value.length > 2048) return false
  try {
    const url = new URL(value)
    const host = url.hostname.replace(/^\[|\]$/g, '').toLowerCase()
    const unsafeIPv4 = /^(?:0|10|127|169\.254|192\.168)\./.test(host) ||
      /^172\.(?:1[6-9]|2\d|3[01])\./.test(host)
    return url.protocol === 'https:' && !url.username && !url.password &&
      host !== 'localhost' && host !== '::1' && !host.endsWith('.localhost') &&
      !host.endsWith('.local') && !unsafeIPv4
  } catch {
    return false
  }
}

function isBoundedStrings(value: unknown, maximum: number, itemMaximum = 500): value is string[] {
  return Array.isArray(value) && value.length <= maximum &&
    value.every(item => typeof item === 'string' && item.length <= itemMaximum)
}

function isSpecialistEvidence(value: unknown, expectedKind: CoinCopilotEvidenceKind): boolean {
  if (!isRecord(value) || !hasOnlyKeys(value, [
    'kind', 'title', 'sourceUrl', 'observedAt', 'confidence', 'verificationState',
    'facts', 'matchedAttributes', 'materialDifferences',
  ])) return false
  if (
    value.kind !== expectedKind ||
    typeof value.title !== 'string' ||
    value.title.length === 0 ||
    value.title.length > 300 ||
    !isSafeSpecialistUrl(value.sourceUrl) ||
    typeof value.observedAt !== 'string' ||
    !['high', 'medium', 'low'].includes(String(value.confidence)) ||
    !['verified', 'partial'].includes(String(value.verificationState)) ||
    !isBoundedStrings(value.facts, 10) ||
    !isBoundedStrings(value.matchedAttributes, 20, 200) ||
    !isBoundedStrings(value.materialDifferences, 20, 200)
  ) return false
  return expectedKind !== 'similar_lot' || value.matchedAttributes.length > 0
}

function isNullableFiniteNumber(value: unknown): boolean {
  return value === null || (typeof value === 'number' && Number.isFinite(value))
}

function isSpecialistTrend(value: unknown): boolean {
  if (!isRecord(value) || !hasOnlyKeys(value, [
    'state', 'sampleSize', 'dateFrom', 'dateTo', 'currency', 'priceBasis',
    'low', 'median', 'high', 'confidence', 'limitations', 'supportingSourceIds',
  ])) return false
  return ['rising', 'stable', 'declining', 'unknown'].includes(String(value.state)) &&
    isNonNegativeInteger(value.sampleSize) &&
    (value.dateFrom === null || typeof value.dateFrom === 'string') &&
    (value.dateTo === null || typeof value.dateTo === 'string') &&
    (value.currency === null || typeof value.currency === 'string') &&
    [null, 'hammer', 'realized_including_premium'].includes(value.priceBasis as null | string) &&
    isNullableFiniteNumber(value.low) &&
    isNullableFiniteNumber(value.median) &&
    isNullableFiniteNumber(value.high) &&
    ['high', 'medium', 'low'].includes(String(value.confidence)) &&
    isBoundedStrings(value.limitations, 10) &&
    isBoundedStrings(value.supportingSourceIds, 10, 2048) &&
    value.supportingSourceIds.every(isSafeSpecialistUrl)
}

function isSpecialistResult(value: unknown, toolName: string): value is CoinCopilotSpecialistResult {
  if (!isRecord(value) || !hasOnlyKeys(value, [
    'capability', 'outcome', 'items', 'trend', 'warnings', 'truncation',
  ])) return false
  const capability = value.capability as CoinCopilotSpecialistCapability
  if (!(capability in SPECIALIST_KINDS) || capability !== toolName) return false
  if (!['complete', 'partial', 'no_match', 'unavailable'].includes(String(value.outcome)) ||
      !Array.isArray(value.items) || value.items.length > 10 ||
      !value.items.every(item => isSpecialistEvidence(item, SPECIALIST_KINDS[capability])) ||
      !isBoundedStrings(value.warnings, 10)) return false
  if (['complete', 'partial'].includes(String(value.outcome)) !== (value.items.length > 0)) return false
  if (capability === 'price_trends' ? !isSpecialistTrend(value.trend) : value.trend !== null) return false
  if (!isRecord(value.truncation) || !hasOnlyKeys(value.truncation, [
    'truncated', 'originalBytes', 'persistedBytes', 'digest', 'omittedItems',
  ])) return false
  return typeof value.truncation.truncated === 'boolean' &&
    isNonNegativeInteger(value.truncation.originalBytes) &&
    isNonNegativeInteger(value.truncation.persistedBytes) &&
    typeof value.truncation.digest === 'string' &&
    /^[a-f0-9]{64}$/.test(value.truncation.digest) &&
    isNonNegativeInteger(value.truncation.omittedItems)
}

function isCoinCopilotUsage(value: unknown) {
  if (!isRecord(value)) return false
  return isNonNegativeInteger(value.iterations) && isNonNegativeInteger(value.toolCalls) &&
    isNonNegativeInteger(value.inputTokens) && isNonNegativeInteger(value.outputTokens) &&
    !('estimatedCostMicros' in value)
}

function isCoinCopilotLimits(value: unknown) {
  if (!isRecord(value)) return false
  return isNonNegativeInteger(value.maxIterations) && isNonNegativeInteger(value.maxToolCalls) &&
    isNonNegativeInteger(value.maxConcurrentTools) && isNonNegativeInteger(value.hardTimeoutSeconds) &&
    isNonNegativeInteger(value.maxPersistedToolResultBytes) && !('maxEstimatedCostMicros' in value)
}

function isSafeCopilotEvent(value: unknown, eventType: string, eventId?: string): value is CoinCopilotEvent {
  if (!isRecord(value) || !COPILOT_EVENT_TYPES.has(eventType as CoinCopilotEvent['type'])) return false
  if (value.type !== eventType || !Number.isSafeInteger(value.seq) || Number(value.seq) < 1) return false
  if (eventId !== undefined && Number(eventId) !== value.seq) return false
  if (typeof value.threadId !== 'string' || typeof value.runId !== 'string' ||
      typeof value.executionId !== 'string' || typeof value.ts !== 'string' || !isRecord(value.payload)) {
    return false
  }

  const payload = value.payload
  switch (eventType) {
    case 'run_started':
      return payload.status === 'running' && typeof payload.executionId === 'string' &&
        isNonNegativeInteger(payload.attempt) && isCoinCopilotLimits(payload.limits)
    case 'plan_updated':
      return Array.isArray(payload.plan) && payload.plan.every((item) =>
        isRecord(item) && typeof item.id === 'string' && typeof item.title === 'string' &&
        ['pending', 'in_progress', 'completed', 'skipped', 'failed'].includes(String(item.status)))
    case 'tool_started':
      return typeof payload.toolCallId === 'string' && typeof payload.toolName === 'string' && typeof payload.stepId === 'string'
    case 'tool_completed':
      return typeof payload.toolCallId === 'string' && typeof payload.toolName === 'string' &&
        typeof payload.stepId === 'string' && ['succeeded', 'failed', 'cancelled', 'rejected'].includes(String(payload.status)) &&
        Number.isFinite(payload.durationMs) && typeof payload.resultSummary === 'string' && typeof payload.truncated === 'boolean' &&
        (!Object.hasOwn(payload, 'specialistResult') ||
          isSpecialistResult(payload.specialistResult, payload.toolName))
    case 'clarification_required':
      return typeof payload.question === 'string' && ['text', 'single_choice', 'boolean'].includes(String(payload.inputType)) &&
        Array.isArray(payload.choices) && payload.choices.every(choice => typeof choice === 'string') &&
        isNonNegativeInteger(payload.checkpointVersion)
    case 'run_paused':
      return payload.reason === 'clarification_required' && isNonNegativeInteger(payload.checkpointVersion) &&
        typeof payload.resumeDeadline === 'string'
    case 'run_resumed':
      return typeof payload.executionId === 'string' && isNonNegativeInteger(payload.attempt) &&
        isNonNegativeInteger(payload.checkpointVersion)
    case 'run_cancelled':
      return payload.reason === 'owner_cancelled'
    case 'run_completed':
      return typeof payload.answer === 'string' && isCoinCopilotUsage(payload.usage)
    case 'run_failed':
      return typeof payload.code === 'string' && typeof payload.message === 'string' &&
        typeof payload.retryable === 'boolean' && isCoinCopilotUsage(payload.usage)
    default:
      return false
  }
}

export function createCoinCopilotSSEParser(
  handlers: CoinCopilotStreamHandlers,
  seenSeqs = new Set<number>(),
) {
  let buffer = ''
  let stopped = false

  function handleFrame(frame: string) {
    if (!frame.trim() || frame.startsWith(':')) return
    let eventType = 'message'
    let eventId: string | undefined
    const dataLines: string[] = []
    for (const rawLine of frame.split('\n')) {
      const line = rawLine.replace(/\r$/, '')
      if (!line || line.startsWith(':')) continue
      if (line.startsWith('event:')) eventType = line.slice(6).trim()
      else if (line.startsWith('id:')) eventId = line.slice(3).trim()
      else if (line.startsWith('data:')) dataLines.push(line.slice(5).trimStart())
    }
    if (dataLines.length === 0) return

    let data: unknown
    try {
      data = JSON.parse(dataLines.join('\n'))
    } catch {
      return
    }

    if (eventType === 'stream_truncated') {
      if (isRecord(data) && typeof data.runId === 'string' &&
          COPILOT_RUN_STATUSES.has(data.status as CoinCopilotRunStatus) &&
          Number.isSafeInteger(data.earliestSeq) && Number.isSafeInteger(data.lastSeq)) {
        handlers.onTruncated?.(data as unknown as CoinCopilotStreamTruncated)
      }
      return
    }
    if (eventType === 'end') {
      if (isRecord(data) && typeof data.runId === 'string' &&
          COPILOT_TERMINAL_STATUSES.has(data.status as CoinCopilotRunStatus)) {
        stopped = true
        handlers.onEnd?.(data as unknown as CoinCopilotStreamEnd)
      }
      return
    }
    if (!isSafeCopilotEvent(data, eventType, eventId) || seenSeqs.has(data.seq)) return

    seenSeqs.add(data.seq)
    if (handlers.onEvent(data) === false) stopped = true
  }

  return {
    push(chunk: string) {
      if (stopped) return
      buffer += chunk
      const frames = buffer.split(/\r?\n\r?\n/)
      buffer = frames.pop() ?? ''
      for (const frame of frames) handleFrame(frame)
    },
    finish() {
      if (!stopped && buffer.trim()) handleFrame(buffer)
      buffer = ''
    },
    get stopped() {
      return stopped
    },
  }
}

async function fetchCopilotStreamWithAuthRetry(url: string, signal?: AbortSignal): Promise<Response> {
  const headersFor = (token: string | null) => {
    const headers = new Headers({ Accept: 'text/event-stream' })
    if (token) headers.set('Authorization', `Bearer ${token}`)
    return headers
  }
  const first = await fetch(url, { headers: headersFor(localStorage.getItem('token')), signal })
  if (first.status !== 401) return first
  const refreshed = await refreshAccessToken()
  return fetch(url, { headers: headersFor(refreshed), signal })
}

export async function streamCoinCopilotRunEvents(
  runId: string,
  handlers: CoinCopilotStreamHandlers,
  options: CoinCopilotStreamOptions = {},
) {
  const baseURL = import.meta.env.VITE_API_BASE_URL || ''
  const since = options.since ?? 0
  const query = since > 0 ? `?since=${encodeURIComponent(String(since))}` : ''
  const response = await fetchCopilotStreamWithAuthRetry(
    `${baseURL}/api/agent/copilot/runs/${encodeURIComponent(runId)}/events${query}`,
    options.signal,
  )
  if (!response.ok) {
    const body = await response.json().catch(() => null)
    throw new Error(formatAgentServiceError(body, `Unable to open Coin Copilot stream (HTTP ${response.status}).`))
  }
  const reader = response.body?.getReader()
  if (!reader) throw new Error('Streaming is not supported in this browser.')

  const parser = createCoinCopilotSSEParser(handlers, options.seenSeqs)
  const decoder = new TextDecoder()
  while (!parser.stopped) {
    const { done, value } = await reader.read()
    if (done) break
    parser.push(decoder.decode(value, { stream: true }))
  }
  parser.push(decoder.decode())
  parser.finish()
  if (parser.stopped) await reader.cancel().catch(() => undefined)
}

// Deep Agentic Coin Identification (344-deep-agentic-coin-identification).
export async function createDeepIdentificationJob(input: CreateDeepIdentificationJobInput) {
  const formData = new FormData()
  if (input.coinId !== undefined) formData.append('coinId', String(input.coinId))
  if (input.obverseImage) formData.append('obverse', input.obverseImage)
  if (input.reverseImage) formData.append('reverse', input.reverseImage)
  for (const hint of input.hintImages ?? []) {
    formData.append('hints', hint)
  }
  appendOptionalFormValue(formData, 'notes', input.notes)
  if (input.providers && input.providers.length > 0) {
    formData.append('providers', input.providers.join(','))
  }
  return api.post<DeepJobEnvelope>('/deep-identification/jobs', formData)
}

export const listDeepIdentificationJobs = (params?: ListDeepIdentificationJobsParams) =>
  api.get<DeepJobListResponse>('/deep-identification/jobs', { params })

export const getDeepIdentificationCapability = () =>
  api.get<DeepIdentificationCapability>('/deep-identification/capability')

export const getDeepIdentificationJob = (id: number) =>
  api.get<DeepJobEnvelope>(`/deep-identification/jobs/${id}`)

export const cancelDeepIdentificationJob = (id: number) =>
  api.post<DeepJobEnvelope>(`/deep-identification/jobs/${id}/cancel`)

export const retryDeepIdentificationJob = (id: number, input?: { notes?: string; providers?: DeepProviderId[] }) =>
  api.post<DeepJobEnvelope>(`/deep-identification/jobs/${id}/retry`, {
    notes: input?.notes,
    providers: input?.providers,
  })

export const patchDeepIdentificationProposal = (id: number, input: UpdateDeepIdentificationProposalInput) =>
  api.patch<DeepProposal>(`/deep-identification/jobs/${id}/proposal`, input)

export const applyDeepIdentificationProposal = (id: number, input: ApplyDeepIdentificationProposalInput) =>
  api.post<DeepApplyResult>(`/deep-identification/jobs/${id}/apply`, input)

// Spec 354 T017/T044: hard-deletes a terminal job (204). Non-terminal → 409,
// non-owner/missing → 404 — surfaced to the caller via getApiErrorMessage.
export const deleteDeepIdentificationJob = (id: number) =>
  api.delete<void>(`/deep-identification/jobs/${id}`)

export default api
