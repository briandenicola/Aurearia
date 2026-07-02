<template>
  <div class="ai-analysis-section">
    <div class="ai-analysis-content">
      <div class="ai-buttons">
        <button
          class="btn btn-primary btn-sm ai-action-btn"
          :disabled="busy || !hasObverse || !aiAvailable"
          aria-label="Analyze obverse"
          :title="!aiAvailable ? aiMessage : !hasObverse ? 'No obverse image' : ''"
          @click="handleAnalyze('obverse')"
        >
          {{ analyzingSide === 'obverse' ? 'Analyzing...' : 'Obverse' }}
        </button>
        <button
          class="btn btn-primary btn-sm ai-action-btn"
          :disabled="busy || !hasReverse || !aiAvailable"
          aria-label="Analyze reverse"
          :title="!aiAvailable ? aiMessage : !hasReverse ? 'No reverse image' : ''"
          @click="handleAnalyze('reverse')"
        >
          {{ analyzingSide === 'reverse' ? 'Analyzing...' : 'Reverse' }}
        </button>
        <button
          class="btn btn-primary btn-sm ai-action-btn"
          :disabled="busy || !canGradeCoin || !aiAvailable"
          aria-label="Grade coin"
          :title="gradingDisabledTitle"
          @click="handleGradeCoin"
        >
          {{ grading ? 'Grading...' : 'Grade' }}
        </button>
      </div>
      <p v-if="!aiAvailable" class="ai-unavailable">{{ aiMessage || 'AI unavailable — configure a provider in Admin → AI Configuration' }}</p>
      <p v-if="jobStatusMessage" class="ai-job-status">{{ jobStatusMessage }}</p>
      <div class="grading-panel">
        <div class="grading-copy">
          <p class="grading-limit">
            AI grading is an assisted estimate, not professional certification. Image quality and missing sides can reduce confidence. The saved coin grade is not changed automatically.
          </p>
          <p v-if="!canGradeCoin" class="grading-hint">
            Add coin photos before requesting a grading estimate.
          </p>
          <p v-if="gradingError" class="grading-error">{{ gradingError }}</p>
        </div>
        <div v-if="gradingReport" class="ai-result-section grading-result">
          <h5 class="ai-result-heading">Grading Report</h5>
          <div class="ai-content" v-html="renderedGradingReport"></div>
        </div>
      </div>

      <div v-if="obverseAnalysis" class="ai-result-section">
        <div class="ai-result-header">
          <h5 class="ai-result-heading">Obverse Analysis</h5>
          <button class="btn btn-ghost btn-xs" @click="handleDeleteAnalysis('obverse')">Remove</button>
        </div>
        <div class="ai-content" v-html="renderedObverse"></div>
      </div>

      <div v-if="reverseAnalysis" class="ai-result-section">
        <div class="ai-result-header">
          <h5 class="ai-result-heading">Reverse Analysis</h5>
          <button class="btn btn-ghost btn-xs" @click="handleDeleteAnalysis('reverse')">Remove</button>
        </div>
        <div class="ai-content" v-html="renderedReverse"></div>
      </div>

      <div v-if="aiAnalysis && !obverseAnalysis && !reverseAnalysis" class="ai-result-section">
        <div class="ai-content" v-html="renderedLegacy"></div>
      </div>

      <p v-if="!obverseAnalysis && !reverseAnalysis && !aiAnalysis && aiAvailable" class="ai-empty">
        Upload images and click an analyze button to get an expert assessment.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { analyzeCoin, deleteAnalysis, formatAgentServiceError, getAIJob, getAIStatus, getCoinAIJobs, gradeCoin } from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import { useNotifications } from '@/composables/useNotifications'
import { useToast } from '@/composables/useToast'
import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'
import type { AIJob, AIJobStartResponse, CoinGradingResult } from '@/types'

const props = defineProps<{
  coinId: number
  obverseAnalysis?: string | null
  reverseAnalysis?: string | null
  aiAnalysis?: string | null
  hasObverse: boolean
  hasReverse: boolean
}>()

const emit = defineEmits<{
  analysisUpdated: []
}>()

const { showConfirm, showAlert } = useDialog()
const { refresh: refreshNotifications } = useNotifications()
const { showToast } = useToast()
const md = new MarkdownIt({ html: false })
const POLL_INTERVAL_MS = 3_000

const analyzing = ref(false)
const analyzingSide = ref<string | null>(null)
const grading = ref(false)
const gradingReport = ref('')
const gradingError = ref('')
const aiAvailable = ref(true)
const aiMessage = ref('')
const activeJob = ref<AIJob | null>(null)
let pollTimer: ReturnType<typeof setTimeout> | null = null
let unmounted = false

const renderedObverse = computed(() => (props.obverseAnalysis ? DOMPurify.sanitize(md.render(props.obverseAnalysis)) : ''))
const renderedReverse = computed(() => (props.reverseAnalysis ? DOMPurify.sanitize(md.render(props.reverseAnalysis)) : ''))
const renderedLegacy = computed(() => (props.aiAnalysis ? DOMPurify.sanitize(md.render(props.aiAnalysis)) : ''))
const renderedGradingReport = computed(() => (gradingReport.value ? DOMPurify.sanitize(md.render(gradingReport.value)) : ''))
const busy = computed(() => analyzing.value || grading.value)
const canGradeCoin = computed(() => props.hasObverse || props.hasReverse)
const gradingDisabledTitle = computed(() => {
  if (!aiAvailable.value) return aiMessage.value
  if (!canGradeCoin.value) return 'Add coin photos before requesting grading'
  return ''
})
const jobStatusMessage = computed(() => {
  if (!activeJob.value) return ''
  if (grading.value) {
    const status = activeJob.value.status || 'queued'
    return `Coin grading ${formatStatus(status)}. This will continue in the background; you can leave this page.`
  }
  if (!analyzingSide.value) return ''
  const status = activeJob.value.status || 'queued'
  return `${capitalize(analyzingSide.value)} analysis ${formatStatus(status)}. This will continue in the background; you can leave this page.`
})

onMounted(async () => {
  try {
    const res = await getAIStatus()
    aiAvailable.value = res.data.available
    aiMessage.value = res.data.message
  } catch {
    aiAvailable.value = false
    aiMessage.value = 'Unable to check AI status'
  }
  await resumeAnalysisJob()
})

onUnmounted(() => {
  unmounted = true
  clearPollTimer()
})

async function handleAnalyze(side: 'obverse' | 'reverse') {
  clearPollTimer()
  analyzing.value = true
  analyzingSide.value = side
  grading.value = false
  activeJob.value = null
  try {
    const res = await analyzeCoin(props.coinId, side)
    const job = normalizeStartedJob(res.data, side)
    rememberJob(side, job.id)
    showToast(`${capitalize(side)} analysis queued. You can leave this page; we will notify you when it is done.`, 'info')
    await pollAnalysisJob(job.id, side, job)
  } catch (err) {
    const detail = formatAgentServiceError(err, 'Check the internal agent service configuration and retry.')
    await showAlert(`AI analysis failed for ${side}. ${detail}`, { title: 'Analysis Failed' })
    analyzing.value = false
    analyzingSide.value = null
  }
}

async function handleGradeCoin() {
  clearPollTimer()
  grading.value = true
  gradingError.value = ''
  gradingReport.value = ''
  analyzing.value = false
  analyzingSide.value = null
  activeJob.value = null
  try {
    const res = await gradeCoin(props.coinId)
    const job = normalizeStartedJob(res.data)
    rememberGradingJob(job.id)
    showToast('Coin grading queued. You can leave this page; we will notify you when it is done.', 'info')
    await pollGradingJob(job.id, job)
  } catch (err) {
    const detail = formatAgentServiceError(err, 'Unable to start coin grading. Confirm both obverse and reverse images are available, then retry.')
    gradingError.value = detail
    await showAlert(`Coin grading failed. ${detail}`, { title: 'Grading Failed' })
    grading.value = false
  }
}

async function handleDeleteAnalysis(side: 'obverse' | 'reverse') {
  if (!await showConfirm(`Delete the ${side} analysis?`, { title: 'Delete Analysis', variant: 'danger' })) return
  try {
    await deleteAnalysis(props.coinId, side)
    emit('analysisUpdated')
  } catch {
    await showAlert(`Failed to delete ${side} analysis`, { title: 'Error' })
  }
}

async function resumeAnalysisJob() {
  try {
    const res = await getCoinAIJobs(props.coinId, true)
    const jobs = normalizeJobList(res.data)
    const activeGrading = jobs.find((job) => isGradingJob(job) && !isTerminalStatus(job.status))
    if (activeGrading?.id) {
      grading.value = true
      gradingError.value = ''
      await pollGradingJob(activeGrading.id, activeGrading)
      return
    }
    const activeAnalysis = jobs.find((job) => isAnalysisJob(job) && !isTerminalStatus(job.status))
    if (activeAnalysis?.id) {
      const side = activeAnalysis.side === 'reverse' ? 'reverse' : 'obverse'
      analyzing.value = true
      analyzingSide.value = side
      await pollAnalysisJob(activeAnalysis.id, side, activeAnalysis)
      return
    }
  } catch {
    // Stored job IDs below still give navigation recovery a chance.
  }

  for (const side of ['obverse', 'reverse'] as const) {
    const jobId = sessionStorage.getItem(jobStorageKey(side))
    if (!jobId) continue
    try {
      const res = await getAIJob(jobId)
      if (!isAnalysisJob(res.data)) continue
      if (isTerminalStatus(res.data.status)) {
        await finishAnalysisJob(res.data, side)
      } else {
        analyzing.value = true
        analyzingSide.value = side
        await pollAnalysisJob(jobId, side, res.data)
      }
      return
    } catch {
      sessionStorage.removeItem(jobStorageKey(side))
    }
  }

  const gradingJobId = sessionStorage.getItem(gradingJobStorageKey())
  if (gradingJobId) {
    try {
      const res = await getAIJob(gradingJobId)
      if (isGradingJob(res.data)) {
        if (isTerminalStatus(res.data.status)) {
          await finishGradingJob(res.data)
        } else {
          grading.value = true
          gradingError.value = ''
          await pollGradingJob(gradingJobId, res.data)
        }
        return
      }
    } catch {
      sessionStorage.removeItem(gradingJobStorageKey())
    }
  }

  await recoverCompletedGradingJob()
}

async function pollAnalysisJob(jobId: string, side: 'obverse' | 'reverse', knownJob?: AIJob) {
  if (unmounted) return
  if (knownJob) activeJob.value = knownJob
  try {
    const res = await getAIJob(jobId)
    activeJob.value = res.data
    if (isTerminalStatus(res.data.status)) {
      await finishAnalysisJob(res.data, side)
      return
    }
  } catch {
    // Keep polling; transient network errors should not lose the backend job.
  }
  schedulePoll(jobId, side)
}

function schedulePoll(jobId: string, side: 'obverse' | 'reverse') {
  clearPollTimer()
  pollTimer = setTimeout(() => {
    void pollAnalysisJob(jobId, side)
  }, POLL_INTERVAL_MS)
}

async function pollGradingJob(jobId: string, knownJob?: AIJob) {
  if (unmounted) return
  if (knownJob) activeJob.value = knownJob
  try {
    const res = await getAIJob(jobId)
    activeJob.value = res.data
    if (isTerminalStatus(res.data.status)) {
      await finishGradingJob(res.data)
      return
    }
  } catch {
    // Keep polling; transient network errors should not lose the backend job.
  }
  scheduleGradingPoll(jobId)
}

function scheduleGradingPoll(jobId: string) {
  clearPollTimer()
  pollTimer = setTimeout(() => {
    void pollGradingJob(jobId)
  }, POLL_INTERVAL_MS)
}

async function finishAnalysisJob(job: AIJob, side: 'obverse' | 'reverse') {
  clearPollTimer()
  sessionStorage.removeItem(jobStorageKey(side))
  activeJob.value = job
  analyzing.value = false
  analyzingSide.value = null
  if (isFailedStatus(job.status)) {
    const message = job.errorMessage || 'AI analysis failed. Please retry.'
    showToast(message, 'error')
    await showAlert(message, { title: 'Analysis Failed' })
    return
  }
  activeJob.value = null
  emit('analysisUpdated')
  showToast(`${capitalize(side)} analysis complete.`, 'success')
  await refreshNotifications()
}

async function finishGradingJob(job: AIJob) {
  clearPollTimer()
  sessionStorage.removeItem(gradingJobStorageKey())
  activeJob.value = job
  grading.value = false
  if (isFailedStatus(job.status)) {
    gradingError.value = job.errorMessage || 'Coin grading failed. Please retry.'
    showToast(gradingError.value, 'error')
    await showAlert(gradingError.value, { title: 'Grading Failed' })
    return
  }

  const parsed = parseGradingResult(job.result)
  if (!parsed?.gradingReport) {
    gradingError.value = 'No grading report returned from AI'
    return
  }

  gradingReport.value = parsed.gradingReport
  activeJob.value = null
  showToast('Coin grading report ready.', 'success')
  await refreshNotifications()
}

async function recoverCompletedGradingJob() {
  try {
    const res = await getCoinAIJobs(props.coinId, false)
    const jobs = normalizeJobList(res.data)
    const completedGrading = jobs
      .filter((job) => isGradingJob(job) && isSuccessfulStatus(job.status) && parseGradingResult(job.result)?.gradingReport)
      .sort((a, b) => jobTimestamp(b) - jobTimestamp(a))[0]
    if (!completedGrading) return

    const parsed = parseGradingResult(completedGrading.result)
    if (!parsed?.gradingReport) return

    gradingReport.value = parsed.gradingReport
    gradingError.value = ''
    grading.value = false
    activeJob.value = null
  } catch {
    // Completed-job recovery is best effort; active jobs and explicit actions still work.
  }
}

function clearPollTimer() {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

function normalizeStartedJob(job: AIJobStartResponse, side?: 'obverse' | 'reverse'): AIJob {
  const data = job.job ?? job
  const id = String(('jobId' in data ? data.jobId : data.id) ?? '')
  if (!id) throw new Error('Missing AI job ID')
  return {
    id,
    coinId: data.coinId,
    jobType: data.jobType,
    side: data.side ?? side ?? null,
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

function isAnalysisJob(job: AIJob) {
  return job.coinId === props.coinId && /analy/i.test(job.jobType)
}

function isGradingJob(job: AIJob) {
  return job.coinId === props.coinId && /grading/i.test(job.jobType)
}

function parseGradingResult(result: unknown): CoinGradingResult | null {
  const raw = unwrapGradingResult(result)
  if (!raw || typeof raw !== 'object') return null
  const data = raw as Record<string, unknown>
  const gradingReport = typeof data.gradingReport === 'string'
    ? data.gradingReport
    : typeof data.grading_report === 'string'
      ? data.grading_report
      : ''
  return gradingReport ? { gradingReport } : null
}

function unwrapGradingResult(result: unknown): unknown {
  if (typeof result === 'string') {
    try {
      return unwrapGradingResult(JSON.parse(result))
    } catch {
      return { gradingReport: result }
    }
  }
  if (result && typeof result === 'object') {
    const data = result as Record<string, unknown>
    return data.gradingResult ?? data.result ?? result
  }
  return result
}

function isTerminalStatus(status: string) {
  return ['completed', 'succeeded', 'success', 'failed', 'error', 'cancelled', 'canceled'].includes(status.toLowerCase())
}

function isFailedStatus(status: string) {
  return ['failed', 'error', 'cancelled', 'canceled'].includes(status.toLowerCase())
}

function isSuccessfulStatus(status: string) {
  return ['completed', 'succeeded', 'success'].includes(status.toLowerCase())
}

function jobTimestamp(job: AIJob) {
  return Date.parse(job.completedAt ?? job.updatedAt ?? job.createdAt ?? '') || 0
}

function rememberJob(side: 'obverse' | 'reverse', jobId: string) {
  sessionStorage.setItem(jobStorageKey(side), jobId)
}

function rememberGradingJob(jobId: string) {
  sessionStorage.setItem(gradingJobStorageKey(), jobId)
}

function jobStorageKey(side: 'obverse' | 'reverse') {
  return `aiJob:analysis:${props.coinId}:${side}`
}

function gradingJobStorageKey() {
  return `aiJob:grading:${props.coinId}`
}

function formatStatus(status: string) {
  const normalized = status.toLowerCase()
  if (normalized === 'queued' || normalized === 'pending') return 'queued'
  if (normalized === 'running' || normalized === 'processing') return 'in progress'
  return normalized
}

function capitalize(value: string) {
  return `${value.charAt(0).toUpperCase()}${value.slice(1)}`
}
</script>

<style scoped>
.ai-analysis-section {
  margin-bottom: 1.5rem;
}

.ai-analysis-content {
  padding: 0.75rem 1rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
}

.ai-buttons {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.ai-action-btn {
  min-width: 0;
  min-height: 2.75rem;
  justify-content: center;
  white-space: nowrap;
  line-height: 1.25;
}

.ai-result-section {
  margin-bottom: 1.25rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-subtle);
}

.ai-result-section:last-of-type {
  border-bottom: none;
  margin-bottom: 0;
  padding-bottom: 0;
}

.ai-result-heading {
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--accent-gold);
  margin-bottom: 0.5rem;
}

.ai-result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.ai-result-header .ai-result-heading {
  margin-bottom: 0;
}

.ai-content {
  font-size: 0.85rem;
  line-height: 1.7;
  color: var(--text-secondary);
}

.ai-content :deep(h1),
.ai-content :deep(h2),
.ai-content :deep(h3) {
  color: var(--accent-gold);
  margin-top: 1rem;
  margin-bottom: 0.5rem;
}

.ai-content :deep(strong) {
  color: var(--text-primary);
}

.ai-content :deep(ul),
.ai-content :deep(ol) {
  padding-left: 1.25rem;
}

.ai-empty {
  font-size: 0.85rem;
  color: var(--text-muted);
  font-style: italic;
}

.ai-unavailable {
  font-size: 0.85rem;
  color: var(--accent-bronze);
  font-style: italic;
  margin-bottom: 0.5rem;
}

.ai-job-status {
  font-size: 0.85rem;
  color: var(--accent-gold);
  margin-bottom: 0.5rem;
}

.grading-panel {
  margin-bottom: 1rem;
}

.grading-copy {
  display: grid;
  gap: 0.35rem;
}

.grading-limit,
.grading-hint,
.grading-error {
  font-size: 0.85rem;
  line-height: 1.5;
}

.grading-limit {
  color: var(--text-secondary);
}

.grading-hint {
  color: var(--text-muted);
}

.grading-error {
  color: var(--accent-bronze);
}

.grading-result {
  margin-top: 0.75rem;
}
</style>
