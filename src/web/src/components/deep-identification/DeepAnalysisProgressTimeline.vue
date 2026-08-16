<template>
  <section class="grid min-w-0 gap-4 overflow-hidden" aria-label="Deep Analysis progress">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2" role="status" aria-live="polite">
        <span
          class="inline-flex items-center rounded-full border px-[0.7rem] py-1 text-sm font-semibold uppercase tracking-[0.05em]"
          :class="connectionClasses"
        >{{ connectionLabel }}</span>
        <span v-if="truncated" class="text-sm text-text-secondary">
          Some earlier progress details are no longer available, but the job continues below.
        </span>
      </div>
      <button
        v-if="showCancel"
        type="button"
        class="rounded-full border border-byzantine px-4 py-2 text-sm font-medium text-byzantine transition-colors duration-150 hover:bg-byzantine hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
        :disabled="cancelDisabled"
        aria-label="Cancel Deep Analysis"
        @click="$emit('cancel')"
      >
        {{ cancelling ? 'Cancelling…' : 'Cancel' }}
      </button>
    </div>

    <ol v-if="events.length" class="m-0 grid gap-2 p-0" style="list-style: none;">
      <li
        v-for="event in events"
        :key="event.seq"
        class="grid min-w-0 grid-cols-1 items-baseline gap-1 rounded-sm border border-border-subtle bg-card p-3 sm:grid-cols-[auto_minmax(0,1fr)] sm:gap-3"
      >
        <span class="text-sm font-semibold text-gold">{{ labelFor(event) }}</span>
        <span class="min-w-0 break-words text-sm text-text-secondary [overflow-wrap:anywhere]">{{ detailFor(event) }}</span>
      </li>
    </ol>
    <p v-else class="text-body text-text-secondary">Waiting for Deep Analysis to begin…</p>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { DeepStreamEvent } from '@/types'

const props = defineProps<{
  events: DeepStreamEvent[]
  connected: boolean
  streaming: boolean
  truncated?: boolean
  terminalStatus?: string | null
  cancelling?: boolean
}>()

defineEmits<{ (e: 'cancel'): void }>()

const showCancel = computed(() => !props.terminalStatus)
const cancelDisabled = computed(() => Boolean(props.cancelling) || Boolean(props.terminalStatus))

const connectionClasses = computed(() => {
  if (props.terminalStatus) {
    return props.terminalStatus === 'completed' || props.terminalStatus === 'partial'
      ? 'border-gold text-gold'
      : 'border-byzantine text-byzantine'
  }
  if (props.connected) return 'border-gold text-gold'
  if (props.streaming) return 'border-border-accent text-text-secondary'
  return 'border-byzantine text-byzantine'
})

const connectionLabel = computed(() => {
  if (props.terminalStatus) return props.terminalStatus
  if (props.connected) return 'Live'
  if (props.streaming) return 'Connecting…'
  return 'Reconnecting…'
})

const eventLabels: Record<string, string> = {
  job_accepted: 'Job accepted',
  status_changed: 'Status changed',
  router_selected: 'Providers selected',
  provider_started: 'Provider started',
  provider_result: 'Provider result',
  evaluation: 'Evaluating results',
  synthesis_started: 'Building report',
  progress: 'Progress',
  terminal: 'Finished',
}

function labelFor(event: DeepStreamEvent): string {
  return eventLabels[event.type] ?? event.type
}

function detailFor(event: DeepStreamEvent): string {
  const payload = event.payload || {}
  switch (event.type) {
    case 'provider_started':
      return typeof payload.provider === 'string' ? String(payload.provider) : ''
    case 'provider_result':
      return [payload.provider, payload.status].filter(Boolean).join(': ')
    case 'router_selected':
      return Array.isArray(payload.selectedProviders) ? payload.selectedProviders.join(', ') : ''
    case 'progress':
      return typeof payload.message === 'string' ? payload.message : ''
    case 'evaluation':
      return `${payload.disagreementCount ?? 0} disagreement(s), ${payload.resolvedCount ?? 0} resolved`
    case 'terminal':
      return typeof payload.status === 'string' ? String(payload.status) : ''
    default:
      return ''
  }
}
</script>
