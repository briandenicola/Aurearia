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
        </section>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { Search } from 'lucide-vue-next'
import BaseBadge from '@/components/ui/BaseBadge.vue'
import DeepAnalysisProgressTimeline from '@/components/deep-identification/DeepAnalysisProgressTimeline.vue'
import { useDeepIdentification } from '@/composables/useDeepIdentification'
import { useDeepIdentificationStream } from '@/composables/useDeepIdentificationStream'

const route = useRoute()
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

onMounted(async () => {
  if (jobId.value !== null) {
    await refresh(jobId.value)
    const since = loadStoredSeq(jobId.value)
    stream.connect(jobId.value, { since })
  }
})

onUnmounted(() => {
  stream.disconnect()
})
</script>
