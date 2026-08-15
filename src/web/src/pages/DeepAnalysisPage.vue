<template>
  <div class="container">
    <div class="mx-auto max-w-[900px]">
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

        <section v-else-if="job" class="card grid gap-4">
          <div class="flex items-center justify-between">
            <h2 class="m-0 text-lg text-heading">Job #{{ job.id }}</h2>
            <BaseBadge>{{ job.status }}</BaseBadge>
          </div>
          <p class="m-0 text-body text-text-secondary">
            Deep Analysis runs Nomisma and Numista automatically. NGC results link out only; OCRE and RPC
            remain manual for this job. Progress and results will appear here as the job runs.
          </p>

          <p v-if="streamError" role="alert" class="text-body text-byzantine">{{ streamError }}</p>

          <DeepAnalysisProgressTimeline
            :events="stream.events.value"
            :connected="stream.connected.value"
            :streaming="stream.streaming.value"
            :truncated="stream.truncated.value"
            :terminal-status="terminalStatus"
            :cancelling="deep.cancelling.value"
            @cancel="onCancel"
          />

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
            <DeepReportPanel :report="deep.report.value" />
          </template>

          <template v-if="isTerminal && deep.proposal.value && !job.appliedAt">
            <DeepProposalEditor
              :proposal="deep.proposal.value"
              :applying="deep.applying.value"
              @update-field="onUpdateProposalField"
              @confirm="onApplyProposal"
            />
            <p v-if="applyError" role="alert" class="text-body text-byzantine">{{ applyError }}</p>
          </template>

          <p v-else-if="job.appliedAt" class="text-body text-text-secondary" role="status">
            Applied on {{ new Date(job.appliedAt).toLocaleString() }}.
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
import { useDeepIdentification } from '@/composables/useDeepIdentification'
import { useDeepIdentificationStream } from '@/composables/useDeepIdentificationStream'
import type { DeepApplyTarget, DeepProposalFieldEdit } from '@/types'

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

const isTerminal = computed(() => terminalStatus.value === 'completed' || terminalStatus.value === 'partial')

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
  stream.disconnect()
  if (jobId.value !== null) sessionStorage.removeItem(storageKey(jobId.value))
  await router.push({ name: 'deep-analysis', params: { jobId: String(newJob.id) } })
}

const applyError = ref('')

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
    stream.disconnect()
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
  stream.disconnect()
})
</script>
