<template>
  <section class="run-history">
    <div class="section-header">
      <div>
        <h3>Run history</h3>
        <p class="muted">Manual discovery runs are stored separately from wishlist availability checks.</p>
      </div>
      <button class="btn btn-secondary btn-sm" type="button" :disabled="loading" @click="$emit('refresh')">Refresh</button>
    </div>

    <p v-if="error" class="message error-text">{{ error }}</p>
    <div v-else-if="loading" class="message">Loading run history...</div>
    <div v-else-if="!runs.length" class="message">No runs yet. Use Run Now to discover source-backed candidates.</div>
    <div v-else class="runs-list">
      <button
        v-for="run in runs"
        :key="run.id"
        class="run-row"
        :class="{ selected: selectedRunId === run.id }"
        type="button"
        @click="$emit('select', run.id)"
      >
        <span class="status badge" :class="run.status">{{ statusLabel(run.status) }}</span>
        <span class="run-date">{{ formatDate(run.startedAt) }}</span>
        <span class="counts">{{ run.resultCount }} results, {{ run.newCount }} new, {{ run.duplicateCount }} duplicates</span>
      </button>
    </div>

    <article v-if="selectedRun" class="run-detail">
      <div class="detail-grid">
        <div><span class="info-label">Status</span><strong>{{ statusLabel(selectedRun.status) }}</strong></div>
        <div><span class="info-label">Started</span><strong>{{ formatDate(selectedRun.startedAt) }}</strong></div>
        <div><span class="info-label">Completed</span><strong>{{ selectedRun.completedAt ? formatDate(selectedRun.completedAt) : 'Unknown' }}</strong></div>
        <div><span class="info-label">Rate limit</span><strong>{{ selectedRun.rateLimitStatus || 'ok' }}</strong></div>
      </div>
      <p v-if="selectedRun.errorMessage" class="message error-text">{{ selectedRun.errorMessage }}</p>
      <ul v-if="selectedRun.partialWarnings?.length" class="warnings">
        <li v-for="warning in selectedRun.partialWarnings" :key="warning">{{ warning }}</li>
      </ul>
      <details class="snapshot">
        <summary>Criteria snapshot</summary>
        <dl>
          <template v-for="item in snapshotItems" :key="item.label">
            <dt>{{ item.label }}</dt>
            <dd>{{ item.value }}</dd>
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

<style scoped>
.run-history { border: 1px solid var(--border-subtle); border-radius: var(--radius-md); background: var(--bg-card); padding: 1rem; display: grid; gap: 0.75rem; }
.section-header { display: flex; justify-content: space-between; gap: 1rem; align-items: flex-start; }
h3 { margin: 0; }
.muted, .message { color: var(--text-muted); margin: 0; }
.error-text { color: var(--accent-bronze); }
.runs-list { display: grid; gap: 0.35rem; }
.run-row { width: 100%; border: 1px solid var(--border-subtle); border-radius: var(--radius-sm); background: var(--bg-input); color: var(--text-primary); padding: 0.75rem; display: grid; grid-template-columns: auto 1fr auto; gap: 0.75rem; align-items: center; text-align: left; cursor: pointer; }
.run-row.selected { border-color: var(--accent-gold); box-shadow: var(--shadow-glow); }
.status { text-transform: capitalize; }
.status.failed, .status.rate_limited { border-color: var(--accent-bronze); color: var(--accent-bronze); }
.status.partial { border-color: var(--accent-gold); color: var(--accent-gold); }
.run-date, .counts { color: var(--text-secondary); font-size: 0.85rem; }
.run-detail { border-top: 1px solid var(--border-subtle); padding-top: 0.75rem; display: grid; gap: 0.75rem; }
.detail-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 0.75rem; }
.detail-grid div { border: 1px solid var(--border-subtle); border-radius: var(--radius-sm); padding: 0.75rem; background: var(--bg-input); display: grid; gap: 0.25rem; }
.info-label { font-size: 0.7rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.08em; color: var(--text-muted); }
.warnings { color: var(--accent-gold); margin: 0; padding-left: 1.25rem; }
.snapshot summary { color: var(--accent-gold); cursor: pointer; }
dl { display: grid; grid-template-columns: minmax(120px, auto) 1fr; gap: 0.35rem 0.75rem; margin: 0.75rem 0 0; }
dt { color: var(--text-muted); }
dd { margin: 0; color: var(--text-secondary); }
@media (max-width: 640px) { .section-header, .run-row { grid-template-columns: 1fr; display: grid; } }
</style>
