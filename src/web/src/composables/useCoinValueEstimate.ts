import { type MaybeRefOrGetter, computed, onMounted, onUnmounted, ref, toValue } from 'vue'
import { estimateCoinValue, getAIJob, getCoinAIJobs, updateCoin } from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import { useNotifications } from '@/composables/useNotifications'
import { useToast } from '@/composables/useToast'
import type { AIJob, AIJobStartResponse, ValueEstimate } from '@/types'

const POLL_INTERVAL_MS = 3_000

/**
 * AI value-estimate seam extracted from `CoinActionsPanel.vue` (F6
 * god-component cleanup). Owns the full background-job lifecycle for a
 * coin's value estimate: kicking it off, resuming it on mount (both from
 * the server's own active-job list and from a `sessionStorage` fallback so
 * a page refresh mid-estimate doesn't lose track of it), polling until it
 * finishes, and applying the result to the coin's current value. This is
 * one seam, not several, because every piece of it - start, resume, poll,
 * finish - reads and writes the same `estimating`/`valueEstimate` state.
 */
export function useCoinValueEstimate(
  coinId: MaybeRefOrGetter<number>,
  options: { onApplied?: () => void } = {},
) {
  const { showAlert } = useDialog()
  const { refresh: refreshNotifications } = useNotifications()
  const { showToast } = useToast()

  const estimating = ref(false)
  const valueEstimate = ref<ValueEstimate | null>(null)
  const estimateError = ref('')
  const activeEstimateJob = ref<AIJob | null>(null)
  let estimatePollTimer: ReturnType<typeof setTimeout> | null = null
  let unmounted = false

  const estimateStatusMessage = computed(() => {
    const status = activeEstimateJob.value?.status
    if (!status) return ''
    return `Value estimate ${formatStatus(status)}. This will continue in the background; you can leave this page.`
  })

  onMounted(() => {
    void resumeEstimateJob()
  })

  onUnmounted(() => {
    unmounted = true
    clearEstimatePollTimer()
  })

  async function handleEstimateValue() {
    clearEstimatePollTimer()
    estimating.value = true
    estimateError.value = ''
    valueEstimate.value = null
    activeEstimateJob.value = null
    try {
      const res = await estimateCoinValue(toValue(coinId))
      const job = normalizeStartedJob(res.data)
      rememberEstimateJob(job.id)
      showToast('Value estimate queued. You can leave this page; we will notify you when it is done.', 'info')
      await pollEstimateJob(job.id, job)
    } catch (err: unknown) {
      estimateError.value = err instanceof Error ? err.message : 'Failed to estimate value'
      if (typeof err === 'object' && err !== null && 'response' in err) {
        const axiosErr = err as { response?: { data?: { error?: string } } }
        estimateError.value = axiosErr.response?.data?.error || estimateError.value
      }
      estimating.value = false
    }
  }

  async function handleApplyEstimate() {
    if (!valueEstimate.value) return
    try {
      await updateCoin(toValue(coinId), { currentValue: valueEstimate.value.estimatedValue }, { source: 'estimate' })
      valueEstimate.value = null
      options.onApplied?.()
    } catch {
      await showAlert('Failed to update coin value', { title: 'Error' })
    }
  }

  function dismissEstimate() {
    valueEstimate.value = null
  }

  async function resumeEstimateJob() {
    try {
      const res = await getCoinAIJobs(toValue(coinId), true)
      const jobs = normalizeJobList(res.data)
      const activeJob = jobs.find((job) => isEstimateJob(job) && !isTerminalStatus(job.status))
      if (activeJob?.id) {
        estimating.value = true
        estimateError.value = ''
        await pollEstimateJob(activeJob.id, activeJob)
        return
      }
    } catch {
      // Stored job ID below still lets this component recover after navigation.
    }

    const jobId = sessionStorage.getItem(estimateJobStorageKey())
    if (!jobId) return
    try {
      const res = await getAIJob(jobId)
      if (!isEstimateJob(res.data)) return
      if (isTerminalStatus(res.data.status)) {
        await finishEstimateJob(res.data)
      } else {
        estimating.value = true
        estimateError.value = ''
        await pollEstimateJob(jobId, res.data)
      }
    } catch {
      sessionStorage.removeItem(estimateJobStorageKey())
    }
  }

  async function pollEstimateJob(jobId: string, knownJob?: AIJob) {
    if (unmounted) return
    if (knownJob) activeEstimateJob.value = knownJob
    try {
      const res = await getAIJob(jobId)
      activeEstimateJob.value = res.data
      if (isTerminalStatus(res.data.status)) {
        await finishEstimateJob(res.data)
        return
      }
    } catch {
      // Keep polling through transient failures; the backend job still owns the work.
    }
    scheduleEstimatePoll(jobId)
  }

  function scheduleEstimatePoll(jobId: string) {
    clearEstimatePollTimer()
    estimatePollTimer = setTimeout(() => {
      void pollEstimateJob(jobId)
    }, POLL_INTERVAL_MS)
  }

  async function finishEstimateJob(job: AIJob) {
    clearEstimatePollTimer()
    sessionStorage.removeItem(estimateJobStorageKey())
    activeEstimateJob.value = job
    estimating.value = false
    if (isFailedStatus(job.status)) {
      estimateError.value = job.errorMessage || 'Value estimate failed. Please retry.'
      showToast(estimateError.value, 'error')
      return
    }

    const parsed = parseValueEstimate(job.result)
    if (!parsed) {
      estimateError.value = 'No estimate returned from AI'
      return
    }
    valueEstimate.value = parsed
    activeEstimateJob.value = null
    showToast('Value estimate ready.', 'success')
    await refreshNotifications()
  }

  function clearEstimatePollTimer() {
    if (estimatePollTimer) {
      clearTimeout(estimatePollTimer)
      estimatePollTimer = null
    }
  }

  function isEstimateJob(job: AIJob) {
    return job.coinId === toValue(coinId) && /(estimate|value|valuation)/i.test(job.jobType)
  }

  function rememberEstimateJob(jobId: string) {
    sessionStorage.setItem(estimateJobStorageKey(), jobId)
  }

  function estimateJobStorageKey() {
    return `aiJob:value:${toValue(coinId)}`
  }

  return {
    estimating,
    estimateStatusMessage,
    estimateError,
    valueEstimate,
    handleEstimateValue,
    handleApplyEstimate,
    dismissEstimate,
  }
}

function parseValueEstimate(result: unknown): ValueEstimate | null {
  const raw = unwrapEstimateResult(result)
  if (!raw || typeof raw !== 'object') return null
  const data = raw as Record<string, unknown>
  const estimatedValue = Number(data.estimatedValue ?? data.estimated_value ?? data.value ?? 0)
  const confidenceValue = typeof data.confidence === 'string' ? data.confidence.toLowerCase() : 'medium'
  const confidence: ValueEstimate['confidence'] = confidenceValue === 'high' || confidenceValue === 'low' ? confidenceValue : 'medium'
  const reasoning = typeof data.reasoning === 'string'
    ? data.reasoning
    : typeof data.summary === 'string'
      ? data.summary
      : ''
  const comparables = Array.isArray(data.comparables)
    ? data.comparables.map(normalizeComparable).filter((item): item is ValueEstimate['comparables'][number] => item !== null)
    : []

  if (!estimatedValue && !reasoning) return null
  return {
    estimatedValue,
    confidence,
    reasoning,
    comparables,
  }
}

function unwrapEstimateResult(result: unknown): unknown {
  if (typeof result === 'string') {
    try {
      return unwrapEstimateResult(JSON.parse(result))
    } catch {
      return { reasoning: result }
    }
  }
  if (result && typeof result === 'object') {
    const data = result as Record<string, unknown>
    return data.valueEstimate ?? data.estimate ?? data.result ?? result
  }
  return result
}

function normalizeComparable(item: unknown): ValueEstimate['comparables'][number] | null {
  if (!item || typeof item !== 'object') return null
  const data = item as Record<string, unknown>
  return {
    source: String(data.source ?? data.title ?? 'Comparable'),
    price: String(data.price ?? data.value ?? ''),
    url: String(data.url ?? ''),
  }
}

function normalizeStartedJob(job: AIJobStartResponse): AIJob {
  const data = job.job ?? job
  const id = String(('jobId' in data ? data.jobId : data.id) ?? '')
  if (!id) throw new Error('Missing AI job ID')
  return {
    id,
    coinId: data.coinId,
    jobType: data.jobType,
    side: data.side,
    status: data.status,
    result: data.result,
    errorMessage: data.errorMessage,
    createdAt: data.createdAt ?? '',
    updatedAt: data.updatedAt ?? '',
    startedAt: data.startedAt,
    completedAt: data.completedAt,
  }
}

function normalizeJobList(data: AIJob[] | { jobs?: AIJob[] }): AIJob[] {
  return Array.isArray(data) ? data : data.jobs ?? []
}

function isTerminalStatus(status: string) {
  return ['completed', 'succeeded', 'success', 'failed', 'error', 'cancelled', 'canceled'].includes(status.toLowerCase())
}

function isFailedStatus(status: string) {
  return ['failed', 'error', 'cancelled', 'canceled'].includes(status.toLowerCase())
}

function formatStatus(status: string) {
  const normalized = status.toLowerCase()
  if (normalized === 'queued' || normalized === 'pending') return 'queued'
  if (normalized === 'running' || normalized === 'processing') return 'in progress'
  return normalized
}

export type CoinValueEstimate = ReturnType<typeof useCoinValueEstimate>
