<template>
  <div class="ai-analysis-section">
    <div class="ai-analysis-content">
      <div class="ai-buttons">
        <button
          class="btn btn-primary btn-sm"
          :disabled="analyzing || !hasObverse || !aiAvailable"
          :title="!aiAvailable ? aiMessage : !hasObverse ? 'No obverse image' : ''"
          @click="handleAnalyze('obverse')"
        >
          {{ analyzingSide === 'obverse' ? 'Analyzing...' : 'Analyze Obverse' }}
        </button>
        <button
          class="btn btn-primary btn-sm"
          :disabled="analyzing || !hasReverse || !aiAvailable"
          :title="!aiAvailable ? aiMessage : !hasReverse ? 'No reverse image' : ''"
          @click="handleAnalyze('reverse')"
        >
          {{ analyzingSide === 'reverse' ? 'Analyzing...' : 'Analyze Reverse' }}
        </button>
      </div>
      <p v-if="!aiAvailable" class="ai-unavailable">{{ aiMessage || 'AI unavailable — configure a provider in Admin → AI Configuration' }}</p>
      <p v-if="jobStatusMessage" class="ai-job-status">{{ jobStatusMessage }}</p>

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
import { analyzeCoin, deleteAnalysis, formatAgentServiceError, getAIJob, getAIStatus, getCoinAIJobs } from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import { useNotifications } from '@/composables/useNotifications'
import { useToast } from '@/composables/useToast'
import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'
import type { AIJob, AIJobStartResponse } from '@/types'

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
const aiAvailable = ref(true)
const aiMessage = ref('')
const activeJob = ref<AIJob | null>(null)
let pollTimer: ReturnType<typeof setTimeout> | null = null
let unmounted = false

const renderedObverse = computed(() => (props.obverseAnalysis ? DOMPurify.sanitize(md.render(props.obverseAnalysis)) : ''))
const renderedReverse = computed(() => (props.reverseAnalysis ? DOMPurify.sanitize(md.render(props.reverseAnalysis)) : ''))
const renderedLegacy = computed(() => (props.aiAnalysis ? DOMPurify.sanitize(md.render(props.aiAnalysis)) : ''))
const jobStatusMessage = computed(() => {
  if (!activeJob.value || !analyzingSide.value) return ''
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

function clearPollTimer() {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

function normalizeStartedJob(job: AIJobStartResponse, side: 'obverse' | 'reverse'): AIJob {
  const data = job.job ?? job
  const id = String(('jobId' in data ? data.jobId : data.id) ?? '')
  if (!id) throw new Error('Missing AI job ID')
  return {
    id,
    coinId: data.coinId,
    jobType: data.jobType,
    side: data.side ?? side,
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

function isTerminalStatus(status: string) {
  return ['completed', 'succeeded', 'success', 'failed', 'error', 'cancelled', 'canceled'].includes(status.toLowerCase())
}

function isFailedStatus(status: string) {
  return ['failed', 'error', 'cancelled', 'canceled'].includes(status.toLowerCase())
}

function rememberJob(side: 'obverse' | 'reverse', jobId: string) {
  sessionStorage.setItem(jobStorageKey(side), jobId)
}

function jobStorageKey(side: 'obverse' | 'reverse') {
  return `aiJob:analysis:${props.coinId}:${side}`
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
  display: flex;
  gap: 0.4rem;
  margin-bottom: 0.75rem;
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
</style>
