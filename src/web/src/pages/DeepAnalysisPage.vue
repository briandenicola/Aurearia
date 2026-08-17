<template>
  <div class="container min-w-0 overflow-x-hidden">
    <div class="mx-auto min-w-0 max-w-[900px]">
      <div class="page-header">
        <h1>Deep Analysis</h1>
        <div class="pwa-actions">
          <RouterLink class="pwa-icon-btn" to="/lookup" title="Identify Coin" aria-label="Identify Coin">
            <Search :size="22" />
          </RouterLink>
        </div>
      </div>

      <p v-if="!jobId" class="rounded-md border border-border-subtle bg-card p-6 text-base text-text-secondary shadow-[var(--shadow-card)]">
        Start a Deep Analysis from Identify Coin or an existing saved coin to see progress here.
      </p>

      <template v-else>
        <p v-if="loading" class="text-body text-text-secondary">Loading Deep Analysis job...</p>
        <p v-else-if="loadError" role="alert" class="text-body text-byzantine">{{ loadError }}</p>

        <section v-else-if="job" class="card grid min-w-0 gap-4 overflow-hidden">
          <div class="flex min-w-0 flex-wrap items-center justify-between gap-2">
            <h2 class="m-0 text-lg text-heading">Job #{{ job.id }}</h2>
            <BaseBadge>{{ job.status }}</BaseBadge>
          </div>
          <p class="m-0 text-body text-text-secondary">
            Deep Analysis routes eligible reference providers for this coin. NGC remains link-out only and
            RPC is unavailable. Progress, provider selection, and exact outcomes appear here as the job runs.
          </p>

          <p v-if="streamError" role="alert" class="text-body text-byzantine">{{ streamError }}</p>

          <DeepAnalysisProgressTimeline
            :events="stream.events.value"
            :connected="stream.connected.value"
            :streaming="stream.streaming.value"
            :reconnecting="reconnecting"
            :truncated="stream.truncated.value"
            :terminal-status="terminalStatus"
            :ended="stream.ended.value"
            :cancelling="deep.cancelling.value"
            @cancel="onCancel"
            @retry="onManualStreamRetry"
          />

          <DeepProviderCoverageList
            v-if="!isTerminal && liveProviderCoverage.length"
            :coverage="liveProviderCoverage"
            title="Live provider coverage"
          />
          <p v-if="routerRationale" class="m-0 break-words text-sm text-text-secondary [overflow-wrap:anywhere]">
            <strong class="text-text-primary">Provider selection:</strong> {{ routerRationale }}
          </p>

          <div v-if="canRetry" class="flex flex-wrap items-center gap-3">
            <BaseButton
              variant="secondary"
              size="sm"
              class="min-h-[44px]"
              :loading="deep.retrying.value"
              :disabled="deep.retrying.value"
              :aria-busy="deep.retrying.value"
              aria-label="Retry Deep Analysis"
              @click="onRetry"
            >
              <RefreshCw v-if="!deep.retrying.value" :size="16" aria-hidden="true" />
              {{ deep.retrying.value ? 'Retrying…' : 'Retry' }}
            </BaseButton>
            <p v-if="retryError" role="alert" class="m-0 text-body text-byzantine">{{ retryError }}</p>
          </div>

          <template v-if="isTerminal && deep.report.value">
            <div class="grid gap-1">
              <h2 class="m-0 text-lg text-heading">Results</h2>
              <p class="m-0 text-body text-text-secondary">
                Review the report, choose the details you want to keep, then save the result.
              </p>
            </div>
            <DeepReportPanel :report="deep.report.value" />
          </template>

          <template v-if="isTerminal && deep.proposal.value && !job.appliedAt">
            <DeepProposalEditor
              :proposal="deep.proposal.value"
              :applying="deep.applying.value"
              :action-label="proposalActionLabel"
              :applying-label="proposalApplyingLabel"
              @update-field="onUpdateProposalField"
              @confirm="onApplyProposal"
            />
            <p v-if="applyError" role="alert" class="text-body text-byzantine">{{ applyError }}</p>
          </template>

          <div
            v-else-if="isTerminal && deep.report.value && !job.appliedAt"
            class="grid gap-2 rounded-sm border border-border-subtle bg-card p-3"
          >
            <h3 class="m-0 text-lg text-text-primary">No draft fields were proposed</h3>
            <p class="m-0 text-body text-text-secondary">
              The report is still available above. Retry the analysis to gather a new result, or return to Identify Coin.
            </p>
            <RouterLink class="btn btn-secondary justify-self-start" to="/lookup">Identify Coin</RouterLink>
          </div>

          <p v-else-if="job.appliedAt" class="text-body text-text-secondary" role="status">
            {{ appliedStatusText }}
          </p>
        </section>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { Search, RefreshCw } from 'lucide-vue-next'
import BaseBadge from '@/components/ui/BaseBadge.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import DeepAnalysisProgressTimeline from '@/components/deep-identification/DeepAnalysisProgressTimeline.vue'
import DeepReportPanel from '@/components/deep-identification/DeepReportPanel.vue'
import DeepProposalEditor from '@/components/deep-identification/DeepProposalEditor.vue'
import DeepProviderCoverageList from '@/components/deep-identification/DeepProviderCoverageList.vue'
import { useDeepIdentification } from '@/composables/useDeepIdentification'
import { useDeepIdentificationStream } from '@/composables/useDeepIdentificationStream'
import type {
  DeepApplyTarget,
  DeepProposalFieldEdit,
  DeepProviderId,
  DeepProviderStatus,
  DeepReportCoverage,
} from '@/types'

const route = useRoute()
const router = useRouter()
const jobId = computed(() => {
  const raw = route.params.jobId
  const value = Array.isArray(raw) ? raw[0] : raw
  const parsed = value ? Number(value) : NaN
  return Number.isFinite(parsed) ? parsed : null
})

const deep = useDeepIdentification()
const { job, loading, error: loadError, refresh } = deep
const stream = useDeepIdentificationStream()
const streamError = stream.error

// The synthesized `terminal` frame's payload.status is the authoritative
// terminal state (contract §2); fall back to the last-known job snapshot
// so the UI doesn't flicker to a non-terminal label while the stream is
// still connecting on an already-finished job.
const terminalStatus = computed(() => {
  if (stream.terminal.value) return stream.terminal.value.status
  if (job.value && ['completed', 'partial', 'failed', 'cancelled'].includes(job.value.status)) {
    return job.value.status
  }
  return null
})

async function onCancel() {
  if (jobId.value === null) return
  await deep.cancel(jobId.value)
}

// Real reconnect (T085/B6, contract §3): the composable's own `finally`
// clears `connected`/`streaming` on any exit path, including an
// unexpected mid-job drop. Previously the Timeline's fallback branch
// then displayed "Reconnecting…" forever, even though nothing ever
// reconnected. This resumes the stream from the last durably-seen `seq`
// (the same resume mechanism used on mount/T101) with capped, backed-off
// automatic retries, plus an explicit manual Retry control for when
// those attempts are exhausted or the drop happened before any event
// was ever seen.
const RECONNECT_DELAYS_MS = [1000, 2000, 5000, 10000]
const reconnecting = ref(false)
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let reconnectAttempt = 0

function clearReconnectTimer() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  reconnecting.value = false
}

function resetReconnectState() {
  clearReconnectTimer()
  reconnectAttempt = 0
}

// Set right before any *intentional* stream teardown (navigating to a new
// job, unmounting) so the `connected` watcher below can tell that drop
// apart from a genuine unexpected disconnect and never auto-reconnects a
// stream we meant to close.
let suppressAutoReconnect = false

function stopStreamIntentionally() {
  suppressAutoReconnect = true
  resetReconnectState()
  stream.disconnect()
}

function scheduleReconnect() {
  // Never auto-reconnect once the server sent `event: end` (contract §2)
  // or the job has already settled - GET is the source of truth then.
  if (jobId.value === null || stream.ended.value || terminalStatus.value !== null) return
  if (reconnectAttempt >= RECONNECT_DELAYS_MS.length) return
  const id = jobId.value
  const delay = RECONNECT_DELAYS_MS[reconnectAttempt] ?? RECONNECT_DELAYS_MS[RECONNECT_DELAYS_MS.length - 1]
  reconnectAttempt += 1
  reconnecting.value = true
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null
    void stream.connect(id, { since: stream.lastSeq.value })
  }, delay)
}

// Only a stream that was actually live and then unexpectedly dropped
// triggers an automatic reconnect - an initial connect failure surfaces
// its error via `streamError` and the manual Retry control instead.
watch(() => stream.connected.value, (isConnected, wasConnected) => {
  if (isConnected) {
    resetReconnectState()
    suppressAutoReconnect = false
    return
  }
  if (!wasConnected) return
  if (suppressAutoReconnect) {
    suppressAutoReconnect = false
    return
  }
  scheduleReconnect()
})

function onManualStreamRetry() {
  suppressAutoReconnect = false
  resetReconnectState()
  if (jobId.value === null) return
  void stream.connect(jobId.value, { since: stream.lastSeq.value })
}

const isTerminal = computed(() => terminalStatus.value === 'completed' || terminalStatus.value === 'partial')
const routerRationale = computed(() => {
  if (job.value?.routerRationale) return job.value.routerRationale
  const event = [...stream.events.value].reverse().find((entry) => entry.type === 'router_selected')
  return typeof event?.payload.rationale === 'string' ? event.payload.rationale : ''
})
const liveProviderCoverage = computed<DeepReportCoverage[]>(() => {
  const statuses = new Map<DeepProviderId, DeepReportCoverage>()
  for (const event of stream.events.value) {
    const provider = event.payload.provider
    if (typeof provider !== 'string') continue
    const providerId = provider as DeepProviderId
    if (event.type === 'provider_started') {
      statuses.set(providerId, { provider: providerId, status: 'running' })
    } else if (event.type === 'provider_result') {
      const status = typeof event.payload.status === 'string'
        ? event.payload.status as DeepProviderStatus
        : 'failed'
      statuses.set(providerId, {
        provider: providerId,
        status,
        linkOut: typeof event.payload.linkOut === 'string' ? event.payload.linkOut : undefined,
      })
    }
  }
  return [...statuses.values()]
})

// Retry is offered for any contract-eligible terminal state (completed,
// partial, failed, cancelled) and never while the job is still active
// (queued/running) — the backend remains authoritative and rejects an
// ineligible or depth-exceeded retry. The button is disabled while a retry
// request is in flight to prevent duplicate submissions.
const canRetry = computed(() =>
  ['completed', 'partial', 'failed', 'cancelled'].includes(terminalStatus.value ?? ''),
)

const retryError = ref('')

async function onRetry() {
  if (jobId.value === null || deep.retrying.value) return
  retryError.value = ''
  const newJob = await deep.retry(jobId.value)
  if (!newJob) {
    retryError.value = deep.error.value || 'Unable to retry Deep Analysis.'
    // deep.error is the same ref backing the page-level load error; clear it
    // after capturing so a failed retry surfaces beside the Retry button
    // instead of replacing the whole job view.
    deep.error.value = ''
    return
  }
  // The retry is a brand-new job row: tear down the current stream and its
  // resume key, then navigate to the new job's route. The jobId watcher
  // re-initializes the page (refresh + reconnect) for the new id.
  stopStreamIntentionally()
  if (jobId.value !== null) sessionStorage.removeItem(storageKey(jobId.value))
  await router.push({ name: 'deep-analysis', params: { jobId: String(newJob.id) } })
}

const applyError = ref('')
const isIntakeJob = computed(() => job.value?.source !== 'saved_coin')
const proposalActionLabel = computed(() => isIntakeJob.value ? 'Save as Draft' : 'Apply to Coin')
const proposalApplyingLabel = computed(() => isIntakeJob.value ? 'Saving...' : 'Applying...')
const appliedStatusText = computed(() => {
  if (!job.value?.appliedAt) return ''
  const action = job.value.source === 'saved_coin' ? 'Applied to coin' : 'Saved as draft'
  return `${action} on ${new Date(job.value.appliedAt).toLocaleString()}.`
})

async function onUpdateProposalField(name: string, edit: DeepProposalFieldEdit) {
  if (jobId.value === null) return
  await deep.updateProposalField(jobId.value, name, edit)
}

async function onApplyProposal() {
  if (jobId.value === null || !job.value) return
  applyError.value = ''
  const target: DeepApplyTarget = job.value.source === 'saved_coin' ? 'coin' : 'draft'
  const result = await deep.applyProposal(jobId.value, { target })
  if (!result) {
    applyError.value = deep.error.value || 'Unable to apply the Deep Analysis proposal.'
    return
  }
  if (target === 'draft' && result.draftId) {
    await router.push({ name: 'quick-capture-draft', params: { id: String(result.draftId) } })
  } else {
    await refresh(jobId.value)
  }
}

// Resume-on-mount (T101): reconnect from the last durably-seen seq for
// this jobId so a page reload or remount never re-fetches events already
// processed. Storage is keyed per-jobId and cleared once the job reaches
// a terminal state (a retried job gets a new id, so nothing to resume).
function storageKey(id: number) {
  return `deep-identification-last-seq:${id}`
}

function loadStoredSeq(id: number): number {
  const raw = sessionStorage.getItem(storageKey(id))
  const parsed = raw ? Number(raw) : 0
  return Number.isFinite(parsed) ? parsed : 0
}

watch(() => stream.lastSeq.value, (seq) => {
  if (jobId.value !== null && seq > 0) {
    sessionStorage.setItem(storageKey(jobId.value), String(seq))
  }
})

watch(() => stream.ended.value, (ended) => {
  if (ended && jobId.value !== null) {
    sessionStorage.removeItem(storageKey(jobId.value))
  }
})

// Once the SSE stream reports a terminal frame, re-fetch the job snapshot
// (T121): only the plain GET response carries the synthesized report and
// proposal, so the terminal frame alone is not enough to render them.
watch(() => stream.terminal.value, async (terminal) => {
  if (terminal && jobId.value !== null) {
    await refresh(jobId.value)
  }
})

async function activateJob(id: number) {
  await refresh(id)
  const since = loadStoredSeq(id)
  stream.connect(id, { since })
}

// A retry navigates to a new jobId under the same route, so the component
// is reused rather than remounted: re-initialize (disconnect the old
// stream, refresh, reconnect) whenever the id actually changes.
watch(jobId, async (newId, oldId) => {
  if (oldId !== null && newId !== oldId) {
    stopStreamIntentionally()
    retryError.value = ''
    applyError.value = ''
  }
  if (newId !== null && newId !== oldId) {
    await activateJob(newId)
  }
})

onMounted(async () => {
  if (jobId.value !== null) {
    await activateJob(jobId.value)
  }
})

onUnmounted(() => {
  stopStreamIntentionally()
})
</script>
