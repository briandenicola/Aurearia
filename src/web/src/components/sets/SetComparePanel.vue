<template>
  <section class="compare-card">
    <div class="compare-heading">
      <div>
        <p class="section-label">Set performance</p>
        <h2>Compare Sets</h2>
      </div>
      <button
        class="btn btn-secondary btn-sm"
        :disabled="selected.length === 0 || loading"
        @click="runCompare"
      >
        {{ loading ? 'Comparing...' : 'Compare' }}
      </button>
    </div>

    <p class="compare-copy">
      Choose one or more other sets to compare against this set over the active trend range.
    </p>

    <div v-if="sets.length" class="compare-options" aria-label="Sets available for comparison">
      <button
        v-for="set in sets"
        :key="set.id"
        type="button"
        class="chip compare-chip"
        :class="{ active: selected.includes(set.id) }"
        :aria-pressed="selected.includes(set.id)"
        @click="toggleSet(set.id)"
      >
        {{ set.name }}
      </button>
    </div>
    <p v-else class="compare-empty">Create another set to enable comparisons.</p>

    <p v-if="error" class="compare-error" role="alert">{{ error }}</p>

    <div v-if="results.length" class="compare-results" aria-live="polite">
      <div class="compare-row compare-row-header">
        <span>Set</span>
        <span>Start</span>
        <span>End</span>
        <span>Change</span>
      </div>
      <div v-for="result in results" :key="result.setId" class="compare-row">
        <span class="compare-name">{{ result.name }}</span>
        <span>{{ formatCurrency(result.startValue) }}</span>
        <span>{{ formatCurrency(result.endValue) }}</span>
        <strong :class="changeClass(result.valueChange)">
          {{ formatChange(result.valueChangePercent) }}
        </strong>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { CoinSetComparison, CoinSetSummary } from '@/types'

const props = defineProps<{
  sets: CoinSetSummary[]
  results: CoinSetComparison[]
  loading?: boolean
  error?: string | null
}>()

const emit = defineEmits<{
  compare: [setIds: number[]]
}>()

const selected = ref<number[]>([])

function toggleSet(setId: number) {
  if (selected.value.includes(setId)) {
    selected.value = selected.value.filter((id) => id !== setId)
    return
  }
  selected.value = [...selected.value, setId]
}

function runCompare() {
  if (selected.value.length === 0 || props.loading) return
  emit('compare', [...selected.value])
}

function formatCurrency(value: number): string {
  return `$${value.toFixed(2)}`
}

function formatChange(value: number): string {
  const prefix = value > 0 ? '+' : ''
  return `${prefix}${value.toFixed(1)}%`
}

function changeClass(value: number): string {
  if (value > 0) return 'positive'
  if (value < 0) return 'negative'
  return ''
}
</script>

<style scoped>
.compare-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1.5rem;
  margin-bottom: 1.5rem;
}

.compare-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.75rem;
}

.compare-heading h2 {
  margin: 0;
}

.compare-copy,
.compare-empty {
  color: var(--text-secondary);
  margin-bottom: 1rem;
}

.compare-options {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.compare-chip {
  font-family: inherit;
}

.compare-error {
  margin-top: 1rem;
  color: var(--confidence-low);
}

.compare-results {
  margin-top: 1rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.compare-row {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
  padding: 0.6rem 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
}

.compare-row:last-child {
  border-bottom: none;
}

.compare-row-header {
  color: var(--text-muted);
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  background: var(--bg-input);
}

.compare-name {
  color: var(--text-primary);
}

.compare-row strong {
  color: var(--accent-gold);
}

.compare-row strong.positive {
  color: var(--confidence-high);
}

.compare-row strong.negative {
  color: var(--confidence-low);
}

@media (max-width: 768px) {
  .compare-heading {
    flex-direction: column;
  }

  .compare-row {
    grid-template-columns: 1fr;
    gap: 0.35rem;
  }

  .compare-row-header {
    display: none;
  }
}
</style>
