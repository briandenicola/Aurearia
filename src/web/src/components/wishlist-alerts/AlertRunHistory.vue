<template>
  <section class="grid gap-3 rounded-md border border-border-subtle bg-card p-4">
    <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
      <div>
        <h3 class="m-0">Run history</h3>
        <p class="m-0 text-body text-text-muted">Manual discovery runs are stored separately from wishlist availability checks.</p>
      </div>
      <button class="btn btn-secondary btn-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" type="button" :disabled="loading" @click="$emit('refresh')">Refresh</button>
    </div>

    <p v-if="error" class="m-0 text-body text-bronze">{{ error }}</p>
    <div v-else-if="loading" class="text-body text-text-muted">Loading run history...</div>
    <div v-else-if="!runs.length" class="text-body text-text-muted">No runs yet. Use Run Now to discover source-backed candidates.</div>
    <div v-else class="grid gap-1.5">
      <button
        v-for="run in runs"
        :key="run.id"
        :class="[
          'grid w-full gap-3 rounded-sm border bg-input p-3 text-left text-text-primary focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)] md:grid-cols-[auto_1fr_auto] md:items-center',
          selectedRunId === run.id ? 'border-gold shadow-glow' : 'border-border-subtle',
        ]"
        type="button"
        @click="$emit('select', run.id)"
      >
        <span
          :class="[
            'badge w-fit border capitalize',
            run.status === 'failed' || run.status === 'rate_limited'
              ? 'border-bronze text-bronze'
              : run.status === 'partial'
                ? 'border-gold text-gold'
                : 'border-border-subtle text-text-secondary',
          ]"
        >
          {{ statusLabel(run.status) }}
        </span>
        <span class="text-body text-text-secondary">{{ formatDate(run.startedAt) }}</span>
        <span class="text-body text-text-secondary">{{ run.resultCount }} results, {{ run.newCount }} new, {{ run.duplicateCount }} duplicates</span>
      </button>
    </div>

    <article v-if="selectedRun" class="grid gap-3 border-t border-border-subtle pt-3">
      <div class="grid gap-3 md:grid-cols-2">
        <div class="grid gap-1 rounded-sm border border-border-subtle bg-input p-3"><span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Status</span><strong>{{ statusLabel(selectedRun.status) }}</strong></div>
        <div class="grid gap-1 rounded-sm border border-border-subtle bg-input p-3"><span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Started</span><strong>{{ formatDate(selectedRun.startedAt) }}</strong></div>
        <div class="grid gap-1 rounded-sm border border-border-subtle bg-input p-3"><span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Completed</span><strong>{{ selectedRun.completedAt ? formatDate(selectedRun.completedAt) : 'Unknown' }}</strong></div>
        <div class="grid gap-1 rounded-sm border border-border-subtle bg-input p-3"><span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Rate limit</span><strong>{{ selectedRun.rateLimitStatus || 'ok' }}</strong></div>
      </div>
      <p v-if="selectedRun.errorMessage" class="m-0 text-body text-bronze">{{ selectedRun.errorMessage }}</p>
      <ul v-if="selectedRun.partialWarnings?.length" class="m-0 grid gap-1 pl-5 text-body text-gold">
        <li v-for="warning in selectedRun.partialWarnings" :key="warning">{{ warning }}</li>
      </ul>
      <details class="grid gap-2">
        <summary class="cursor-pointer text-gold focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]">Criteria snapshot</summary>
        <dl class="mt-3 grid grid-cols-[minmax(120px,_auto)_1fr] gap-x-3 gap-y-1">
          <template v-for="item in snapshotItems" :key="item.label">
            <dt class="text-text-muted">{{ item.label }}</dt>
            <dd class="m-0 text-text-secondary">{{ item.value }}</dd>
          </template>
        </dl>
      </details>
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AlertRun, AlertRunStatus } from '@/types'

const props = defineProps<{
  runs: AlertRun[]
  selectedRun?: AlertRun | null
  selectedRunId?: number | null
  loading?: boolean
  error?: string
}>()

defineEmits<{ select: [runId: number]; refresh: [] }>()

const statusLabel = (status: AlertRunStatus) => status.replace(/_/g, ' ')
const formatDate = (value: string) => new Date(value).toLocaleString()

const snapshotItems = computed(() => {
  const raw = props.selectedRun?.criteriaSnapshot
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>
    return Object.entries(parsed)
      .filter(([, value]) => value !== '' && value !== null && value !== undefined)
      .map(([key, value]) => ({ label: key, value: Array.isArray(value) ? value.join(', ') : String(value) }))
  } catch {
    return [{ label: 'Snapshot', value: raw }]
  }
})
</script>
