<template>
  <div class="detail-actions">
    <div class="section-content-card">
      <div class="mb-6">
        <h3 class="mb-3 text-base font-medium text-text-primary">Upload Images</h3>
        <div>
          <div class="flex gap-2 max-sm:flex-col">
            <select v-model="uploadType" class="form-select flex-1">
              <option value="obverse">Obverse</option>
              <option value="reverse">Reverse</option>
              <option value="detail">Detail</option>
              <option value="other">Other</option>
            </select>
            <label class="btn btn-secondary btn-sm cursor-pointer whitespace-nowrap">
              Choose File
              <input type="file" accept="image/*" hidden @change="handleImageUpload" />
            </label>
            <button
              v-if="isPwa"
              type="button"
              class="btn btn-secondary btn-sm inline-flex items-center gap-1 whitespace-nowrap"
              @click="showCameraModal = true"
            >
              <Camera :size="14" /> Photo
            </button>
          </div>

          <div class="mt-2 flex gap-2 max-sm:flex-col">
            <input
              v-model="imageUrl"
              type="url"
              class="form-input min-w-0 flex-1 text-[0.82rem]"
              placeholder="Or paste an image URL..."
              @keydown.enter="handleUrlUpload"
            />
            <button
              class="btn btn-secondary btn-sm"
              :disabled="!imageUrl || urlLoading"
              @click="handleUrlUpload"
            >
              {{ urlLoading ? 'Fetching...' : 'Fetch' }}
            </button>
          </div>

          <p v-if="uploadStatus" class="mt-2 text-chip" :class="uploadError ? 'text-loss' : 'text-gold'">{{ uploadStatus }}</p>
        </div>
      </div>

      <div class="mb-6">
        <div class="mb-3 flex items-center justify-between gap-3">
          <h3 class="m-0 text-base font-medium text-text-primary">AI Value Estimate</h3>
          <button
            class="btn btn-secondary btn-sm"
            :disabled="estimating"
            @click="handleEstimateValue"
          >
            {{ estimating ? 'Estimating...' : 'Estimate Value' }}
          </button>
        </div>
        <div v-if="estimating" class="flex items-center gap-3 rounded-sm border border-border-subtle bg-card p-4 text-text-secondary">
          <div class="h-5 w-5 animate-spin rounded-full border-2 border-border-subtle border-t-gold" />
          <span>{{ estimateStatusMessage || 'Estimating current market value...' }}</span>
        </div>
        <div v-if="estimateError" class="px-2 py-2 text-base text-loss">{{ estimateError }}</div>
        <div v-if="valueEstimate" class="rounded-sm border border-border-subtle bg-card p-4">
          <div class="mb-3 flex items-center gap-3 max-sm:flex-col max-sm:items-start">
            <span class="text-xl font-bold text-gold">{{ valueEstimate.estimatedValue ? formatCurrency(valueEstimate.estimatedValue) : 'N/A' }}</span>
            <span
              class="rounded-full px-[0.6rem] py-[0.2rem] text-sm font-semibold uppercase tracking-[0.03em]"
              :class="{
                'bg-gold-glow text-gold': valueEstimate.confidence === 'high',
                'bg-gold-dim text-text-primary': valueEstimate.confidence === 'medium',
                'bg-input text-text-secondary': valueEstimate.confidence === 'low',
              }"
            >
              {{ valueEstimate.confidence }} confidence
            </span>
          </div>
          <p class="mb-3 text-base leading-6 text-text-secondary">{{ valueEstimate.reasoning }}</p>
          <div v-if="valueEstimate.comparables?.length" class="mb-3">
            <h4 class="mb-2 text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Comparable Listings</h4>
            <div v-for="(comp, i) in valueEstimate.comparables" :key="i" class="flex items-center justify-between gap-3 border-b border-border-subtle py-1.5 last:border-b-0">
              <SafeExternalLink
                v-if="safeComparableUrl(comp.url)"
                :href="comp.url"
                target="_blank"
                rel="noopener"
                class="text-body text-gold no-underline hover:underline"
              >
                {{ comp.source }}
              </SafeExternalLink>
              <span v-else class="text-body text-gold">{{ comp.source }}</span>
              <span class="text-body font-semibold text-text-primary">{{ comp.price }}</span>
            </div>
          </div>
          <div class="mt-3 flex gap-2 max-sm:flex-col">
            <button class="btn btn-primary btn-sm" @click="handleApplyEstimate">
              Apply as Current Value
            </button>
            <button class="btn btn-ghost btn-sm" @click="valueEstimate = null">
              Dismiss
            </button>
          </div>
        </div>
      </div>

      <CoinNumistaPanel
        :coin-name="coinName"
        :coin-ruler="coinRuler ?? ''"
        :coin-denomination="coinDenomination ?? ''"
      />
    </div>

    <CameraCaptureModal
      :is-open="showCameraModal"
      @close="showCameraModal = false"
      @captured="handleCameraCaptured"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { uploadImage, proxyImage, estimateCoinValue, updateCoin, getAIJob, getCoinAIJobs } from '@/api/client'
import { formatCurrency } from '@/utils/format'
import CoinNumistaPanel from '@/components/coin/CoinNumistaPanel.vue'
import CameraCaptureModal from '@/components/CameraCaptureModal.vue'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import { Camera } from 'lucide-vue-next'
import { useDialog } from '@/composables/useDialog'
import { useNotifications } from '@/composables/useNotifications'
import { sanitizeExternalUrl } from '@/composables/useSafeExternalLink'
import { useToast } from '@/composables/useToast'
import type { AIJob, AIJobStartResponse, ValueEstimate } from '@/types'

const props = defineProps<{
  coinId: number
  coinName: string
  coinRuler?: string | null
  coinDenomination?: string | null
  imageCount: number
  isPwa: boolean
}>()

const emit = defineEmits<{
  imagesChanged: []
  estimateApplied: []
}>()

const { showAlert } = useDialog()
const { refresh: refreshNotifications } = useNotifications()
const { showToast } = useToast()
const POLL_INTERVAL_MS = 3_000

const uploadType = ref('obverse')
const uploadStatus = ref('')
const uploadError = ref(false)
const imageUrl = ref('')
const urlLoading = ref(false)
const estimating = ref(false)
const valueEstimate = ref<ValueEstimate | null>(null)
const estimateError = ref('')
const showCameraModal = ref(false)
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

function safeComparableUrl(url: string | null | undefined): string | null {
  return sanitizeExternalUrl(url)
}

async function handleImageUpload(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return

  uploadStatus.value = 'Uploading...'
  uploadError.value = false

  try {
    await uploadImage(props.coinId, file, uploadType.value, props.imageCount === 0)
    uploadStatus.value = 'Upload complete!'
    emit('imagesChanged')
  } catch {
    uploadStatus.value = 'Upload failed'
    uploadError.value = true
  }
}

async function handleCameraCaptured(file: File) {
  uploadStatus.value = 'Uploading...'
  uploadError.value = false

  try {
    // Pass circleClip=true for obverse/reverse, false for other types
    const shouldCircleClip = uploadType.value === 'obverse' || uploadType.value === 'reverse'
    await uploadImage(props.coinId, file, uploadType.value, props.imageCount === 0, shouldCircleClip)
    uploadStatus.value = 'Upload complete!'
    emit('imagesChanged')
  } catch {
    uploadStatus.value = 'Upload failed'
    uploadError.value = true
  }
}

async function handleUrlUpload() {
  if (!imageUrl.value) return

  urlLoading.value = true
  uploadStatus.value = 'Fetching image...'
  uploadError.value = false

  try {
    const imgRes = await proxyImage(imageUrl.value)
    const blob = imgRes.data as Blob
    if (blob.size === 0) {
      uploadStatus.value = 'No image data received from URL'
      uploadError.value = true
      return
    }
    const ext = blob.type.includes('png') ? '.png' : '.jpg'
    const file = new File([blob], `${uploadType.value}${ext}`, { type: blob.type || 'image/jpeg' })
    await uploadImage(props.coinId, file, uploadType.value, props.imageCount === 0)
    uploadStatus.value = 'Image saved from URL!'
    imageUrl.value = ''
    emit('imagesChanged')
  } catch {
    uploadStatus.value = 'Failed to fetch image from URL'
    uploadError.value = true
  } finally {
    urlLoading.value = false
  }
}

async function handleEstimateValue() {
  clearEstimatePollTimer()
  estimating.value = true
  estimateError.value = ''
  valueEstimate.value = null
  activeEstimateJob.value = null
  try {
    const res = await estimateCoinValue(props.coinId)
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
    await updateCoin(props.coinId, { currentValue: valueEstimate.value.estimatedValue }, { source: 'estimate' })
    valueEstimate.value = null
    emit('estimateApplied')
  } catch {
    await showAlert('Failed to update coin value', { title: 'Error' })
  }
}

async function resumeEstimateJob() {
  try {
    const res = await getCoinAIJobs(props.coinId, true)
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

function isEstimateJob(job: AIJob) {
  return job.coinId === props.coinId && /(estimate|value|valuation)/i.test(job.jobType)
}

function isTerminalStatus(status: string) {
  return ['completed', 'succeeded', 'success', 'failed', 'error', 'cancelled', 'canceled'].includes(status.toLowerCase())
}

function isFailedStatus(status: string) {
  return ['failed', 'error', 'cancelled', 'canceled'].includes(status.toLowerCase())
}

function rememberEstimateJob(jobId: string) {
  sessionStorage.setItem(estimateJobStorageKey(), jobId)
}

function estimateJobStorageKey() {
  return `aiJob:value:${props.coinId}`
}

function formatStatus(status: string) {
  const normalized = status.toLowerCase()
  if (normalized === 'queued' || normalized === 'pending') return 'queued'
  if (normalized === 'running' || normalized === 'processing') return 'in progress'
  return normalized
}
</script>
