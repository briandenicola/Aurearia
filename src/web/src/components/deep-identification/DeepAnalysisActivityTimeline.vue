<template>
  <details
    class="grid min-w-0 gap-2 rounded-sm border border-border-subtle bg-card p-3"
    :open="open"
    @toggle="onToggle"
  >
    <summary class="flex min-w-0 cursor-pointer list-none items-center justify-between gap-2 text-sm font-semibold text-text-primary focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]">
      <span class="section-label m-0">Activity timeline</span>
      <ChevronDown :size="16" class="shrink-0 transition-transform" :class="{ 'rotate-180': open }" aria-hidden="true" />
    </summary>

    <ol
      v-if="steps.length"
      class="m-0 grid gap-2 p-0"
      style="list-style: none;"
      role="log"
      aria-live="polite"
      aria-label="Deep Analysis activity"
    >
      <li
        v-for="step in steps"
        :key="step.key"
        class="grid min-w-0 grid-cols-[auto_minmax(0,1fr)] items-start gap-2 rounded-sm border border-border-subtle bg-input p-2 sm:gap-3"
      >
        <component :is="stateIcon(step.state)" :size="18" class="mt-0.5 shrink-0" :class="stateIconClasses(step.state)" aria-hidden="true" />

        <div class="grid min-w-0 gap-1">
          <div class="flex min-w-0 flex-wrap items-baseline gap-x-2 gap-y-0.5">
            <span class="text-sm font-semibold text-text-primary">{{ step.label }}</span>
            <span class="chip-sm" :class="stateChipClasses(step.state)">{{ stateLabel(step.state) }}</span>
            <span v-if="step.elapsedLabel" class="text-xs text-text-muted">{{ step.elapsedLabel }}</span>
          </div>
          <p v-if="step.detail" class="m-0 min-w-0 break-words text-sm text-text-secondary [overflow-wrap:anywhere]">
            {{ step.detail }}
          </p>

          <ul v-if="step.providers?.length" class="m-0 mt-1 flex flex-wrap gap-1.5 p-0" style="list-style: none;">
            <li
              v-for="row in step.providers"
              :key="row.provider"
              class="inline-flex min-h-[28px] items-center gap-1.5 rounded-full border border-border-subtle px-[0.6rem] py-0.5 text-xs"
            >
              <component :is="stateIcon(row.state)" :size="12" class="shrink-0" :class="stateIconClasses(row.state)" aria-hidden="true" />
              <span class="font-semibold uppercase tracking-[0.05em] text-text-primary">{{ row.provider }}</span>
              <span :class="stateTextClasses(row.state)">{{ row.statusLabel }}</span>
              <span v-if="row.elapsedLabel" class="text-text-muted">{{ row.elapsedLabel }}</span>
            </li>
          </ul>
        </div>
      </li>
    </ol>
    <p v-else class="m-0 text-body text-text-secondary">Waiting for Deep Analysis to begin…</p>
  </details>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ChevronDown, Clock, Loader2, CheckCircle2, CircleSlash, XCircle } from 'lucide-vue-next'
import type { DeepProviderId, DeepProviderStatus, DeepStreamEvent } from '@/types'

const props = defineProps<{
  events: DeepStreamEvent[]
  terminalStatus?: string | null
  ended?: boolean
}>()

type StepState = 'pending' | 'active' | 'done' | 'skipped' | 'failed'

interface ProviderRow {
  provider: string
  state: StepState
  statusLabel: string
  elapsedLabel?: string
}

interface ActivityStep {
  key: string
  label: string
  detail: string
  state: StepState
  elapsedLabel?: string
  providers?: ProviderRow[]
}

// Collapsed automatically the first time the job settles so the finished
// report isn't crowded, but a manual toggle (native <details> @toggle)
// always wins afterward - we never fight the user's own click.
const open = ref(true)
let userToggled = false

function onToggle(event: Event) {
  userToggled = true
  open.value = (event.target as HTMLDetailsElement).open
}

watch(() => props.terminalStatus, (status, previous) => {
  if (status && !previous && !userToggled) {
    open.value = false
  }
})

const providerStatusLabels: Record<DeepProviderStatus, string> = {
  pending: 'Pending',
  running: 'Running',
  contributed: 'Contributed',
  no_match: 'No match',
  failed: 'Failed',
  timed_out: 'Timed out',
  skipped: 'Skipped',
  not_automated: 'Manual verification',
  unavailable: 'Unavailable',
}

function providerState(status: DeepProviderStatus): StepState {
  if (status === 'contributed') return 'done'
  if (status === 'failed' || status === 'timed_out') return 'failed'
  if (status === 'running') return 'active'
  if (status === 'pending') return 'pending'
  // no_match / skipped / not_automated / unavailable: the provider ran (or
  // was deliberately not run) and produced nothing to contribute - distinct
  // from both success and failure, and must never look like a silent win.
  return 'skipped'
}

const knownEventLabels: Record<string, string> = {
  job_accepted: 'Job accepted',
  status_changed: 'Status changed',
  router_selected: 'Providers selected',
  evaluation: 'Evaluating results',
  synthesis_started: 'Building report',
}

const knownPhaseLabels: Record<string, string> = {
  image_evidence_ready: 'Image evidence ready',
  provider_fanout_started: 'Provider fan-out started',
  evaluation_started: 'Evaluating results',
  quick_lookup: 'Quick lookup',
}

function titleCase(key: string): string {
  return key
    .replace(/[_-]+/g, ' ')
    .trim()
    .split(' ')
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ') || 'Activity'
}

function parseTs(ts: string): number | null {
  const value = Date.parse(ts)
  return Number.isFinite(value) ? value : null
}

function formatElapsed(ms: number): string {
  if (ms < 0) return ''
  if (ms < 1000) return '<1s'
  const totalSeconds = Math.round(ms / 1000)
  if (totalSeconds < 60) return `${totalSeconds}s`
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return seconds ? `${minutes}m ${seconds}s` : `${minutes}m`
}

const steps = computed<ActivityStep[]>(() => {
  const events = props.events ?? []
  const result: ActivityStep[] = []

  let firstTs: number | null = null
  let previousTs: number | null = null

  const providerOrder: string[] = []
  const providerMeta = new Map<string, { status: DeepProviderStatus; startedAt: number | null; resultAt: number | null }>()
  let providerStepIndex = -1

  function ensureProviderStep(ts: number | null) {
    if (providerStepIndex !== -1) return
    providerStepIndex = result.length
    result.push({
      key: 'provider-fanout',
      label: 'Provider fan-out',
      detail: '',
      state: 'active',
      providers: [],
    })
    if (ts !== null && firstTs === null) firstTs = ts
  }

  function rebuildProviderStep() {
    if (providerStepIndex === -1) return
    const rows: ProviderRow[] = providerOrder.map((provider) => {
      const meta = providerMeta.get(provider)
      const status = meta?.status ?? 'pending'
      const state = providerState(status)
      let elapsedLabel: string | undefined
      if (meta?.startedAt !== null && meta?.startedAt !== undefined) {
        const end = meta.resultAt ?? Date.now()
        elapsedLabel = formatElapsed(end - meta.startedAt)
      }
      return {
        provider,
        state,
        statusLabel: providerStatusLabels[status] ?? status,
        elapsedLabel,
      }
    })
    const anyActive = rows.some((row) => row.state === 'active' || row.state === 'pending')
    const anyDone = rows.some((row) => row.state === 'done')
    const anyFailed = rows.some((row) => row.state === 'failed')
    const jobRunning = !props.terminalStatus && !props.ended
    let groupState: StepState = 'done'
    if (anyActive && jobRunning) groupState = 'active'
    else if (anyDone) groupState = 'done'
    else if (anyFailed) groupState = 'failed'
    else groupState = 'skipped'

    const contributedCount = rows.filter((row) => row.state === 'done').length
    const existing = result[providerStepIndex]
    if (!existing) return
    result[providerStepIndex] = {
      ...existing,
      state: groupState,
      detail: `${contributedCount} of ${rows.length} provider(s) contributed.`,
      providers: rows,
    }
  }

  for (const event of events) {
    const ts = parseTs(event.ts)
    const payload = event.payload || {}

    if (event.type === 'provider_started' || event.type === 'provider_result') {
      const provider = typeof payload.provider === 'string' ? payload.provider : ''
      if (!provider) continue
      ensureProviderStep(ts)
      if (!providerMeta.has(provider)) {
        providerOrder.push(provider)
        providerMeta.set(provider, { status: 'pending', startedAt: null, resultAt: null })
      }
      const meta = providerMeta.get(provider)!
      if (event.type === 'provider_started') {
        meta.status = 'running'
        meta.startedAt = ts ?? meta.startedAt
      } else {
        const status = typeof payload.status === 'string' ? payload.status as DeepProviderStatus : 'failed'
        meta.status = status
        meta.resultAt = ts ?? meta.resultAt
      }
      rebuildProviderStep()
      if (ts !== null) previousTs = ts
      continue
    }

    let label: string
    let detail: string
    if (event.type === 'progress') {
      const phase = typeof payload.phase === 'string' ? payload.phase : ''
      label = knownPhaseLabels[phase] ?? (phase ? titleCase(phase) : 'Progress')
      detail = typeof payload.message === 'string' ? payload.message : ''
    } else if (event.type === 'terminal') {
      const status = typeof payload.status === 'string' ? payload.status : ''
      label = 'Finished'
      detail = status ? `Job ${status}.` : ''
    } else if (event.type === 'router_selected') {
      label = knownEventLabels.router_selected ?? 'Providers selected'
      detail = Array.isArray(payload.selectedProviders) ? payload.selectedProviders.join(', ') : ''
    } else if (event.type === 'evaluation') {
      label = knownEventLabels.evaluation ?? 'Evaluating results'
      detail = `${payload.disagreementCount ?? 0} disagreement(s), ${payload.resolvedCount ?? 0} resolved`
    } else if (knownEventLabels[event.type]) {
      label = knownEventLabels[event.type] ?? titleCase(event.type)
      detail = typeof payload.message === 'string' ? payload.message : ''
    } else {
      // Unrecognized event type: never drop it, render it generically so a
      // future phase/type still shows up instead of silently vanishing.
      label = titleCase(event.type)
      detail = typeof payload.message === 'string'
        ? payload.message
        : (Object.values(payload).find((v) => typeof v === 'string') as string | undefined) ?? ''
    }

    let state: StepState = 'done'
    if (event.type === 'terminal') {
      const status = typeof payload.status === 'string' ? payload.status : ''
      if (status === 'failed') state = 'failed'
      else if (status === 'cancelled') state = 'skipped'
      else state = 'done'
    }

    let elapsedLabel: string | undefined
    if (ts !== null) {
      if (firstTs === null) firstTs = ts
      else if (previousTs !== null) {
        const delta = ts - previousTs
        elapsedLabel = delta > 0 ? `+${formatElapsed(delta)}` : undefined
      }
      previousTs = ts
    }

    result.push({ key: `${event.type}-${event.seq}`, label, detail, state, elapsedLabel })
  }

  // The most recent narration step (excluding a resolved provider-fanout
  // group) reflects "what's happening right now" while the job is still
  // running, so it reads as active rather than a flat, silent "done".
  if (!props.terminalStatus && !props.ended && result.length) {
    const last = result[result.length - 1]
    if (last && last.state === 'done' && last.key !== 'provider-fanout') {
      last.state = 'active'
    }
  }

  return result
})

function stateIcon(state: StepState) {
  switch (state) {
    case 'pending': return Clock
    case 'active': return Loader2
    case 'done': return CheckCircle2
    case 'failed': return XCircle
    default: return CircleSlash
  }
}

function stateLabel(state: StepState): string {
  switch (state) {
    case 'pending': return 'Pending'
    case 'active': return 'In progress'
    case 'done': return 'Done'
    case 'failed': return 'Failed'
    default: return 'No result'
  }
}

function stateIconClasses(state: StepState): string {
  switch (state) {
    case 'done': return 'text-gold'
    case 'failed': return 'text-byzantine'
    case 'active': return 'animate-spin text-text-secondary'
    case 'pending': return 'text-text-muted'
    default: return 'text-text-muted'
  }
}

function stateTextClasses(state: StepState): string {
  switch (state) {
    case 'done': return 'text-gold'
    case 'failed': return 'text-byzantine'
    default: return 'text-text-secondary'
  }
}

function stateChipClasses(state: StepState): string {
  switch (state) {
    case 'done': return 'border-gold text-gold'
    case 'failed': return 'border-byzantine text-byzantine'
    case 'active': return 'border-border-accent text-text-secondary'
    default: return 'border-border-subtle text-text-muted'
  }
}
</script>
