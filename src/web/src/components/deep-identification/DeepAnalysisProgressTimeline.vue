<template>
  <section class="grid min-w-0 gap-4 overflow-hidden" aria-label="Deep Analysis progress">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2" role="status" aria-live="polite">
        <span
          class="inline-flex items-center rounded-full border px-[0.7rem] py-1 text-sm font-semibold uppercase tracking-[0.08em]"
          :class="connectionClasses"
        >{{ connectionLabel }}</span>
        <span v-if="truncated" class="text-sm text-text-secondary">
          Some earlier progress details are no longer available, but the job continues below.
        </span>
      </div>
      <div class="flex items-center gap-2">
        <button
          v-if="showRetry"
          type="button"
          class="btn btn-secondary btn-sm min-h-[44px]"
          aria-label="Retry Deep Analysis connection"
          @click="$emit('retry')"
        >
          Retry
        </button>
        <button
          v-if="showCancel"
          type="button"
          class="btn btn-danger btn-sm min-h-[44px]"
          :disabled="cancelDisabled"
          aria-label="Cancel Deep Analysis"
          @click="$emit('cancel')"
        >
          {{ cancelling ? 'Cancelling…' : 'Cancel' }}
        </button>
      </div>
    </div>

    <DeepAnalysisActivityTimeline
      :events="props.events"
      :terminal-status="props.terminalStatus"
      :ended="props.ended"
    />
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import DeepAnalysisActivityTimeline from './DeepAnalysisActivityTimeline.vue'
import type { DeepStreamEvent } from '@/types'

const props = defineProps<{
  events: DeepStreamEvent[]
  connected: boolean
  streaming: boolean
  /**
   * True only while an actual reconnect attempt is scheduled or in flight
   * (T085/B6). Never derived as a fallback default - a badge reading
   * "Reconnecting…" must only appear when a reconnect is genuinely
   * happening, otherwise it lies to a user watching a dead stream.
   */
  reconnecting?: boolean
  truncated?: boolean
  terminalStatus?: string | null
  cancelling?: boolean
  /** True once the SSE stream has received `event: end` (contract §2). */
  ended?: boolean
}>()

defineEmits<{ (e: 'cancel'): void; (e: 'retry'): void }>()

const showCancel = computed(() => !props.terminalStatus)
const cancelDisabled = computed(() => Boolean(props.cancelling) || Boolean(props.terminalStatus))

// The Retry control is offered whenever the stream is neither live,
// actively (re)connecting, nor finished - i.e. genuinely disconnected with
// no reconnect scheduled, so the user has an explicit recovery action
// instead of a badge that silently lies about reconnecting forever.
const showRetry = computed(() =>
  !props.terminalStatus && !props.connected && !props.streaming && !props.reconnecting,
)

const connectionClasses = computed(() => {
  if (props.terminalStatus) {
    return props.terminalStatus === 'completed' || props.terminalStatus === 'partial'
      ? 'border-gold text-gold'
      : 'border-byzantine text-byzantine'
  }
  if (props.connected) return 'border-gold text-gold'
  if (props.reconnecting || props.streaming) return 'border-border-accent text-text-secondary'
  return 'border-byzantine text-byzantine'
})

const connectionLabel = computed(() => {
  if (props.terminalStatus) return props.terminalStatus
  if (props.connected) return 'Live'
  if (props.reconnecting) return 'Reconnecting…'
  if (props.streaming) return 'Connecting…'
  return 'Disconnected'
})

</script>
