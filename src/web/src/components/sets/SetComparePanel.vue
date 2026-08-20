<template>
  <section class="mb-6 rounded-md border border-border-subtle bg-card p-6">
    <div class="mb-3 flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
      <div>
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

    <div v-if="sets.length" class="space-y-3">
      <div class="flex flex-col gap-2 md:flex-row md:items-center">
        <select
          id="set-compare-select"
          v-model.number="pendingSetId"
          class="form-select"
          aria-label="Choose a set to compare"
        >
          <option :value="0" disabled>Select set to compare...</option>
          <option v-for="set in sets" :key="set.id" :value="set.id">
            {{ set.name }}
          </option>
        </select>
        <button class="btn btn-secondary btn-sm" type="button" :disabled="pendingSetId === 0" @click="addPendingSet">
          Add
        </button>
      </div>
      <div v-if="selected.length" class="flex flex-wrap gap-[0.35rem]">
        <button
          v-for="setId in selected"
          :key="setId"
          type="button"
          class="chip border-gold bg-gold-dim text-gold"
          @click="removeSelectedSet(setId)"
        >
          {{ setNameById(setId) }} ×
        </button>
      </div>
      <p class="m-0 text-chip text-text-muted">{{ selected.length }} {{ selected.length === 1 ? 'set selected' : 'sets selected' }}</p>
    </div>
    <p v-else class="mb-4 text-body text-text-secondary">Create another set to enable comparisons.</p>

    <p v-if="error" class="mt-4 text-body text-confidence-low" role="alert">{{ error }}</p>

    <div v-if="results.length" class="mt-4 overflow-hidden rounded-sm border border-border-subtle" aria-live="polite">
      <div class="hidden bg-input px-3 py-2 text-label font-semibold uppercase tracking-[0.08em] text-text-muted md:grid md:grid-cols-[minmax(0,1.4fr)_repeat(3,minmax(0,1fr))] md:gap-3">
        <span>Set</span>
        <span>Start</span>
        <span>End</span>
        <span>Change</span>
      </div>
      <div
        v-for="result in results"
        :key="result.setId"
        class="grid grid-cols-1 gap-[0.35rem] border-b border-border-subtle px-3 py-[0.6rem] text-body text-text-secondary last:border-b-0 md:grid-cols-[minmax(0,1.4fr)_repeat(3,minmax(0,1fr))] md:gap-3"
      >
        <span class="text-text-primary">{{ result.name }}</span>
        <span>{{ formatCurrency(result.startValue) }}</span>
        <span>{{ formatCurrency(result.endValue) }}</span>
        <strong
          class="font-semibold text-gold"
          :class="{
            'text-confidence-high': result.valueChange > 0,
            'text-confidence-low': result.valueChange < 0,
          }"
        >
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
const pendingSetId = ref(0)

function addPendingSet() {
  if (pendingSetId.value === 0 || selected.value.includes(pendingSetId.value)) return
  selected.value = [...selected.value, pendingSetId.value]
  pendingSetId.value = 0
}

function removeSelectedSet(setId: number) {
  selected.value = selected.value.filter((id) => id !== setId)
}

function setNameById(setId: number): string {
  return props.sets.find((set) => set.id === setId)?.name ?? 'Set'
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

</script>
