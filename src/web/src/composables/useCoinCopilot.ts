import { computed, ref, shallowRef } from 'vue'
import {
  cancelCoinCopilotRun,
  getApiErrorMessage,
  getCoinCopilotCapability,
  getCoinCopilotRun,
  getCoinCopilotThread,
  resumeCoinCopilotRun,
  startCoinCopilotRun,
  streamCoinCopilotRunEvents,
} from '@/api/client'
import type {
  AgentChatAppContext,
  CoinCopilotCapability,
  CoinCopilotClarification,
  DeepAnalysisHandoffResult,
  CoinCopilotEvent,
  CoinCopilotPlanItem,
  CoinCopilotRun,
  CoinCopilotSpecialistResult,
  CoinCopilotThread,
  CoinCopilotToolCompletedEvent,
} from '@/types'

const ACTIVE_RUN_KEY = 'coinCopilot:activeRun'
const MAX_RECONNECTS = 3

type StoredRunCursor = {
  runId: string
  threadId: string
  lastSeq: number
}

export type CoinCopilotToolProgress = {
  toolCallId: string
  toolName: string
  stepId: string
  status: 'running' | CoinCopilotToolCompletedEvent['payload']['status']
  durationMs?: number
  resultSummary?: string
  truncated?: boolean
  specialistResult?: CoinCopilotSpecialistResult
  deepAnalysisHandoffResult?: DeepAnalysisHandoffResult
}

export type CoinCopilotStartResult =
  | { accepted: false; fallback: true }
  | { accepted: false; fallback: false; error: string }
  | { accepted: true; status: CoinCopilotRun['status']; answer?: string; error?: string }

type UseCoinCopilotOptions = {
  onRecoveredThread?: (thread: CoinCopilotThread) => void
}

function createIdempotencyKey(prefix: 'start' | 'resume') {
  const random = globalThis.crypto?.randomUUID?.() ??
    `${Date.now()}-${Math.random().toString(36).slice(2)}`
  return `${prefix}-${random}`.slice(0, 128)
}

function responseStatus(error: unknown): number | undefined {
  if (typeof error !== 'object' || error === null) return undefined
  return (error as { response?: { status?: number } }).response?.status
}

function actionableError(error: unknown, fallback: string) {
  const message = getApiErrorMessage(error).trim()
  if (/capacity|queue|conflicts with current state/i.test(message)) {
    return 'Another Coin Copilot run is active. Reopen it or wait for it to finish.'
  }
  if (/checkpoint|stale/i.test(message)) {
    return 'This clarification is out of date. Reload the run before answering again.'
  }
  if (/not found/i.test(message)) {
    return 'This Coin Copilot run is no longer available.'
  }
  return message || fallback
}

function normalizeCapability(value: unknown): CoinCopilotCapability {
  if (typeof value !== 'object' || value === null) {
    return {
      mode: 'legacy',
      enabled: false,
      modelToolCallingSupported: false,
      reason: 'temporarily_unavailable',
    }
  }
  const candidate = value as Record<string, unknown>
  if (candidate.mode === 'copilot' &&
      candidate.enabled === true &&
      candidate.modelToolCallingSupported === true &&
      (candidate.reason === undefined || candidate.reason === null)) {
    return {
      mode: 'copilot',
      enabled: true,
      modelToolCallingSupported: true,
      reason: candidate.reason as null | undefined,
    }
  }
  const reasons = new Set([
    'disabled',
    'provider_unconfigured',
    'model_tool_calling_unsupported',
    'temporarily_unavailable',
  ])
  if (candidate.mode === 'legacy' &&
      typeof candidate.enabled === 'boolean' &&
      candidate.modelToolCallingSupported === false &&
      typeof candidate.reason === 'string' &&
      reasons.has(candidate.reason)) {
    return candidate as CoinCopilotCapability
  }
  return {
    mode: 'legacy',
    enabled: false,
    modelToolCallingSupported: false,
    reason: 'temporarily_unavailable',
  }
}

function readStoredCursor(): StoredRunCursor | null {
  try {
    const raw = sessionStorage.getItem(ACTIVE_RUN_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as Partial<StoredRunCursor>
    if (typeof parsed.runId !== 'string' || typeof parsed.threadId !== 'string' ||
        !Number.isSafeInteger(parsed.lastSeq) || Number(parsed.lastSeq) < 0) {
      sessionStorage.removeItem(ACTIVE_RUN_KEY)
      return null
    }
    return parsed as StoredRunCursor
  } catch {
    sessionStorage.removeItem(ACTIVE_RUN_KEY)
    return null
  }
}

export function useCoinCopilot(options: UseCoinCopilotOptions = {}) {
  const capability = shallowRef<CoinCopilotCapability | null>(null)
  const capabilityLoaded = ref(false)
  const run = shallowRef<CoinCopilotRun | null>(null)
  const plan = shallowRef<CoinCopilotPlanItem[]>([])
  const tools = shallowRef<CoinCopilotToolProgress[]>([])
  const clarification = shallowRef<CoinCopilotClarification | null>(null)
  const lastSeq = ref(0)
  const connected = ref(false)
  const reconnecting = ref(false)
  const truncated = ref(false)
  const error = ref('')

  const seenSeqs = new Set<number>()
  let abortController: AbortController | null = null
  let capabilityPromise: Promise<CoinCopilotCapability> | null = null

  const active = computed(() => capability.value?.mode === 'copilot')
  const canCancel = computed(() =>
    run.value?.status === 'queued' || run.value?.status === 'running' || run.value?.status === 'paused')
  const canResume = computed(() => run.value?.status === 'paused' && clarification.value !== null)
  const terminal = computed(() =>
    run.value?.status === 'completed' || run.value?.status === 'failed' || run.value?.status === 'cancelled')
  const paused = () => run.value?.status === 'paused'
  const hasUnappliedEvents = () => Boolean(run.value && lastSeq.value < run.value.lastSeq)
  const shouldStream = (replayPaused: boolean) => Boolean(
    run.value && (hasUnappliedEvents() || (!terminal.value && (!paused() || replayPaused))),
  )

  function persistCursor() {
    if (!run.value || (terminal.value && !hasUnappliedEvents())) {
      sessionStorage.removeItem(ACTIVE_RUN_KEY)
      return
    }
    sessionStorage.setItem(ACTIVE_RUN_KEY, JSON.stringify({
      runId: run.value.id,
      threadId: run.value.threadId,
      lastSeq: lastSeq.value,
    } satisfies StoredRunCursor))
  }

  function disconnect() {
    abortController?.abort()
    abortController = null
    connected.value = false
    reconnecting.value = false
  }

  function clearRun() {
    disconnect()
    run.value = null
    plan.value = []
    tools.value = []
    clarification.value = null
    lastSeq.value = 0
    truncated.value = false
    error.value = ''
    seenSeqs.clear()
    sessionStorage.removeItem(ACTIVE_RUN_KEY)
  }

  async function resolveCapability(force = false): Promise<CoinCopilotCapability> {
    if (!force && capability.value) return capability.value
    if (!force && capabilityPromise) return capabilityPromise
    capabilityPromise = getCoinCopilotCapability()
      .then((response) => normalizeCapability(response.data))
      .catch(() => ({
        mode: 'legacy',
        enabled: false,
        modelToolCallingSupported: false,
        reason: 'temporarily_unavailable',
      } satisfies CoinCopilotCapability))
      .then((resolved) => {
        capability.value = resolved
        capabilityLoaded.value = true
        capabilityPromise = null
        return resolved
      })
    return capabilityPromise
  }

  function updateTool(event: CoinCopilotEvent) {
    if (event.type === 'tool_started') {
      const existing = tools.value.find(item => item.toolCallId === event.payload.toolCallId)
      if (existing) return
      tools.value = [...tools.value, {
        toolCallId: event.payload.toolCallId,
        toolName: event.payload.toolName,
        stepId: event.payload.stepId,
        status: 'running',
      }]
      return
    }
    if (event.type !== 'tool_completed') return
    const next = tools.value.filter(item => item.toolCallId !== event.payload.toolCallId)
    next.push({
      toolCallId: event.payload.toolCallId,
      toolName: event.payload.toolName,
      stepId: event.payload.stepId,
      status: event.payload.status,
      durationMs: event.payload.durationMs,
      resultSummary: event.payload.resultSummary,
      truncated: event.payload.truncated,
      specialistResult: event.payload.specialistResult,
      deepAnalysisHandoffResult: event.payload.deepAnalysisHandoffResult,
    })
    tools.value = next
  }

  function applyEvent(event: CoinCopilotEvent): boolean {
    lastSeq.value = Math.max(lastSeq.value, event.seq)
    updateTool(event)
    if (!run.value) return false

    switch (event.type) {
      case 'run_started':
        run.value = { ...run.value, status: 'running', attempt: event.payload.attempt }
        break
      case 'plan_updated':
        plan.value = event.payload.plan
        break
      case 'clarification_required':
        clarification.value = event.payload
        break
      case 'run_paused':
        run.value = {
          ...run.value,
          status: 'paused',
          checkpointVersion: event.payload.checkpointVersion,
          resumeDeadline: event.payload.resumeDeadline,
        }
        persistCursor()
        return false
      case 'run_resumed':
        clarification.value = null
        run.value = {
          ...run.value,
          status: 'running',
          attempt: event.payload.attempt,
          checkpointVersion: event.payload.checkpointVersion,
        }
        break
      case 'run_cancelled':
        run.value = { ...run.value, status: 'cancelled' }
        persistCursor()
        return false
      case 'run_completed':
        run.value = {
          ...run.value,
          status: 'completed',
          finalAnswer: event.payload.answer,
          usage: event.payload.usage,
        }
        persistCursor()
        return false
      case 'run_failed':
        run.value = {
          ...run.value,
          status: 'failed',
          failureCode: event.payload.code,
          failureMessage: event.payload.message,
          usage: event.payload.usage,
        }
        error.value = event.payload.message
        persistCursor()
        return false
    }
    persistCursor()
    return true
  }

  async function refreshRun() {
    if (!run.value) return null
    const response = await getCoinCopilotRun(run.value.id)
    run.value = response.data.run
    persistCursor()
    return run.value
  }

  async function connect(replayPaused = false): Promise<void> {
    if (!shouldStream(replayPaused)) return
    disconnect()
    error.value = ''
    let reconnectAttempt = 0

    while (shouldStream(replayPaused)) {
      const currentRun = run.value
      if (!currentRun) break
      abortController = new AbortController()
      const controller = abortController
      try {
        connected.value = true
        await streamCoinCopilotRunEvents(currentRun.id, {
          onEvent: applyEvent,
          onTruncated: () => { truncated.value = true },
          onEnd: (end) => {
            if (run.value) run.value = { ...run.value, status: end.status }
          },
        }, {
          since: lastSeq.value,
          signal: controller.signal,
          seenSeqs,
        })
        if (controller.signal.aborted) break
        if (!hasUnappliedEvents() && (terminal.value || paused())) break
        await refreshRun()
        if (!hasUnappliedEvents() && (terminal.value || paused())) break
        reconnectAttempt += 1
        if (reconnectAttempt > MAX_RECONNECTS) {
          error.value = 'Coin Copilot disconnected. Reopen the drawer to reconnect.'
          break
        }
        reconnecting.value = true
        await new Promise(resolve => setTimeout(resolve, reconnectAttempt * 250))
      } catch (streamError: unknown) {
        if (controller.signal.aborted) break
        try {
          await refreshRun()
        } catch (refreshError: unknown) {
          error.value = actionableError(refreshError, 'Coin Copilot could not reload the durable run.')
          break
        }
        if (!hasUnappliedEvents() && (terminal.value || paused())) break
        reconnectAttempt += 1
        if (reconnectAttempt > MAX_RECONNECTS) {
          error.value = actionableError(streamError, 'Coin Copilot disconnected. Reopen the drawer to reconnect.')
          break
        }
        reconnecting.value = true
        await new Promise(resolve => setTimeout(resolve, reconnectAttempt * 250))
      } finally {
        connected.value = false
      }
    }
    reconnecting.value = false
    abortController = null
    persistCursor()
  }

  async function start(
    goal: string,
    appContext?: AgentChatAppContext,
    threadId?: string,
  ): Promise<CoinCopilotStartResult> {
    const resolved = await resolveCapability()
    if (resolved.mode !== 'copilot') return { accepted: false, fallback: true }

    error.value = ''
    plan.value = []
    tools.value = []
    clarification.value = null
    lastSeq.value = 0
    truncated.value = false
    seenSeqs.clear()
    const key = createIdempotencyKey('start')
    const input = { goal, appContext, ...(threadId ? { threadId } : {}) }

    let response
    try {
      response = await startCoinCopilotRun(input, key)
    } catch (firstError: unknown) {
      if (responseStatus(firstError) === 503) return { accepted: false, fallback: true }
      if (responseStatus(firstError) !== undefined) {
        return { accepted: false, fallback: false, error: actionableError(firstError, 'Coin Copilot could not start.') }
      }
      try {
        response = await startCoinCopilotRun(input, key)
      } catch (retryError: unknown) {
        if (responseStatus(retryError) === 503) return { accepted: false, fallback: true }
        return {
          accepted: false,
          fallback: false,
          error: actionableError(retryError, 'Coin Copilot start could not be confirmed. Try again to safely reuse the request.'),
        }
      }
    }

    run.value = response.data.run
    persistCursor()
    await connect()
    return {
      accepted: true,
      status: run.value.status,
      answer: run.value.finalAnswer ?? undefined,
      error: run.value.failureMessage ?? (error.value || undefined),
    }
  }

  async function cancel() {
    if (!run.value || !canCancel.value) return false
    error.value = ''
    try {
      const response = await cancelCoinCopilotRun(run.value.id)
      run.value = response.data.run
      persistCursor()
      return true
    } catch (cancelError: unknown) {
      error.value = actionableError(cancelError, 'Coin Copilot could not cancel this run.')
      return false
    }
  }

  async function resume(answer: string) {
    if (!run.value || !clarification.value || !canResume.value || !answer.trim()) return false
    error.value = ''
    const key = createIdempotencyKey('resume')
    const checkpointVersion = clarification.value.checkpointVersion
    try {
      const response = await resumeCoinCopilotRun(run.value.id, {
        answer: answer.trim(),
        expectedCheckpointVersion: checkpointVersion,
      }, key)
      run.value = response.data.run
      clarification.value = null
      persistCursor()
      await connect()
      return true
    } catch (firstError: unknown) {
      if (responseStatus(firstError) === undefined) {
        try {
          const response = await resumeCoinCopilotRun(run.value.id, {
            answer: answer.trim(),
            expectedCheckpointVersion: checkpointVersion,
          }, key)
          run.value = response.data.run
          clarification.value = null
          persistCursor()
          await connect()
          return true
        } catch (retryError: unknown) {
          error.value = actionableError(retryError, 'Coin Copilot could not resume this run.')
          return false
        }
      }
      error.value = actionableError(firstError, 'Coin Copilot could not resume this run.')
      return false
    }
  }

  async function restoreActiveRun() {
    const stored = readStoredCursor()
    if (!stored) return false
    try {
      const [runResponse, threadResponse] = await Promise.all([
        getCoinCopilotRun(stored.runId),
        getCoinCopilotThread(stored.threadId),
      ])
      run.value = runResponse.data.run
      lastSeq.value = Math.min(stored.lastSeq, run.value.lastSeq)
      for (let seq = 1; seq <= lastSeq.value; seq += 1) seenSeqs.add(seq)
      options.onRecoveredThread?.(threadResponse.data.thread)
      if (terminal.value) {
        persistCursor()
        return true
      }
      if (run.value.status === 'paused') {
        lastSeq.value = 0
        seenSeqs.clear()
        await connect(true)
      } else {
        void connect()
      }
      return true
    } catch {
      clearRun()
      return false
    }
  }

  return {
    capability,
    capabilityLoaded,
    active,
    run,
    plan,
    tools,
    clarification,
    lastSeq,
    connected,
    reconnecting,
    truncated,
    error,
    canCancel,
    canResume,
    terminal,
    resolveCapability,
    restoreActiveRun,
    start,
    cancel,
    resume,
    disconnect,
    clearRun,
  }
}
