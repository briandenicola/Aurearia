<template>
  <section
    v-if="run"
    class="w-full rounded-sm border border-border-subtle bg-card p-3"
    aria-live="polite"
    data-testid="copilot-run-progress"
  >
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <span class="chip-sm border-gold text-gold">Beta</span>
        <span class="section-label mb-0">Coin Copilot</span>
      </div>
      <span class="text-sm text-text-secondary">{{ statusLabel }}</span>
    </div>

    <ol v-if="plan.length" class="mt-3 flex list-none flex-col gap-2 p-0">
      <li v-for="item in plan" :key="item.id" class="flex items-start gap-2 text-sm">
        <CheckCircle2 v-if="item.status === 'completed'" :size="16" class="mt-0.5 shrink-0 text-gold" />
        <CircleX v-else-if="item.status === 'failed'" :size="16" class="mt-0.5 shrink-0 text-[var(--color-negative)]" />
        <LoaderCircle v-else-if="item.status === 'in_progress'" :size="16" class="mt-0.5 shrink-0 animate-spin text-gold" />
        <Circle v-else :size="16" class="mt-0.5 shrink-0 text-text-muted" />
        <span :class="item.status === 'completed' ? 'text-text-secondary' : 'text-text-primary'">{{ item.title }}</span>
      </li>
    </ol>

    <div v-if="tools.length" class="mt-3 flex flex-col gap-2 border-t border-border-subtle pt-3">
      <div v-for="tool in tools" :key="tool.toolCallId" class="flex flex-col gap-2 text-sm text-text-secondary">
        <div class="flex items-start gap-2">
          <LoaderCircle v-if="tool.status === 'running'" :size="15" class="mt-0.5 shrink-0 animate-spin text-gold" />
          <CheckCircle2 v-else-if="tool.status === 'succeeded'" :size="15" class="mt-0.5 shrink-0 text-gold" />
          <CircleX v-else :size="15" class="mt-0.5 shrink-0 text-[var(--color-negative)]" />
          <span>
            <strong class="font-medium text-text-primary">{{ toolLabel(tool.toolName) }}</strong>
            <span v-if="tool.resultSummary"> — {{ tool.resultSummary }}</span>
            <span v-if="tool.truncated" class="text-text-muted"> Result details were shortened.</span>
          </span>
        </div>

        <section
          v-if="tool.specialistResult"
          class="ml-6 flex flex-col gap-2 rounded-sm border border-border-subtle bg-input p-3"
          :aria-label="`${toolLabel(tool.toolName)} evidence`"
          data-testid="copilot-specialist-result"
        >
          <div class="flex flex-wrap items-center justify-between gap-2">
            <span class="section-label mb-0">Source evidence</span>
            <span class="chip-sm">{{ outcomeLabel(tool.specialistResult.outcome) }}</span>
          </div>

          <article
            v-for="item in tool.specialistResult.items"
            :key="item.sourceUrl"
            class="flex flex-col gap-2 border-t border-border-subtle pt-2"
          >
            <a
              :href="item.sourceUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="font-medium text-gold"
            >
              {{ item.title }}
            </a>
            <p class="mb-0 text-xs text-text-muted">
              Observed {{ formatObservedAt(item.observedAt) }} ·
              {{ titleCase(item.confidence) }} confidence ·
              {{ titleCase(item.verificationState) }}
            </p>
            <ul v-if="item.facts.length" class="mb-0 flex list-none flex-col gap-1 p-0 text-xs">
              <li v-for="fact in item.facts" :key="fact">{{ fact }}</li>
            </ul>
          </article>

          <p v-if="tool.specialistResult.outcome === 'no_match'" class="mb-0 text-text-secondary">
            No matching evidence was found in the checked sources.
          </p>
          <p v-else-if="tool.specialistResult.outcome === 'unavailable'" class="mb-0 text-text-secondary">
            Sources unavailable. Try again later.
          </p>
          <ul v-if="tool.specialistResult.warnings.length" class="mb-0 flex list-none flex-col gap-1 p-0 text-xs text-text-muted">
            <li v-for="warning in tool.specialistResult.warnings" :key="warning">{{ warning }}</li>
          </ul>
        </section>
      </div>
    </div>

    <p v-if="truncated" class="mb-0 mt-3 text-sm text-text-muted">
      Earlier progress expired, so this view resumed from the retained event history.
    </p>

    <div v-if="canCancel" class="mt-3 flex justify-end">
      <button
        type="button"
        class="btn btn-xs btn-ghost min-h-[44px]"
        :disabled="cancelling"
        @click="$emit('cancel')"
      >
        <Square :size="15" />
        {{ cancelling ? 'Cancelling...' : 'Cancel run' }}
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { CheckCircle2, Circle, CircleX, LoaderCircle, Square } from 'lucide-vue-next'
import type { CoinCopilotPlanItem, CoinCopilotRun } from '@/types'
import type { CoinCopilotToolProgress } from '@/composables/useCoinCopilot'

const props = defineProps<{
  run: CoinCopilotRun | null
  plan: CoinCopilotPlanItem[]
  tools: CoinCopilotToolProgress[]
  canCancel: boolean
  cancelling: boolean
  truncated: boolean
}>()

defineEmits<{
  cancel: []
}>()

const statusLabel = computed(() => {
  switch (props.run?.status) {
    case 'queued': return 'Queued'
    case 'running': return 'Working'
    case 'paused': return 'Waiting for your answer'
    case 'cancel_requested': return 'Cancelling'
    case 'completed': return 'Complete'
    case 'failed': return 'Stopped'
    case 'cancelled': return 'Cancelled'
    default: return ''
  }
})

function toolLabel(name: string) {
  return name
    .split('_')
    .map(part => part ? part.charAt(0).toUpperCase() + part.slice(1) : part)
    .join(' ')
}

function titleCase(value: string) {
  return value.charAt(0).toUpperCase() + value.slice(1).replaceAll('_', ' ')
}

function outcomeLabel(outcome: 'complete' | 'partial' | 'no_match' | 'unavailable') {
  switch (outcome) {
    case 'complete': return 'Complete'
    case 'partial': return 'Partial'
    case 'no_match': return 'No matching evidence'
    case 'unavailable': return 'Sources unavailable'
  }
}

function formatObservedAt(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(new Date(value))
}
</script>
