<template>
  <PullToRefresh :on-refresh="handleRefresh">
    <div class="container flex flex-col gap-6">
      <header class="page-header flex flex-nowrap items-center justify-between gap-4">
        <div>
          <p class="section-label">Collection Insights</p>
          <h1>Value Details</h1>
          <p class="mt-[0.35rem] text-base text-text-secondary">Your collection's value breakdown and history.</p>
        </div>
        <router-link
          class="inline-flex shrink-0 items-center justify-center rounded-sm border border-border-subtle bg-transparent p-[0.4rem] text-text-secondary transition hover:border-border-accent hover:bg-gold-glow hover:text-gold"
          to="/stats"
          aria-label="Back to Stats"
        >
          <ArrowLeft :size="20" />
        </router-link>
      </header>

      <div v-if="isLoading" class="loading-overlay">
        <div class="spinner"></div>
      </div>

      <template v-else>
        <div v-if="stats" class="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(160px,1fr))]">
          <div class="flex flex-col gap-1 rounded-md border border-border-subtle bg-card p-6 text-center">
            <span class="font-display text-xl font-semibold text-gold">{{ formatCurrency(stats.values.totalCurrentValue) }}</span>
            <span class="text-sm uppercase tracking-[0.08em] text-text-muted">Total Value</span>
          </div>
          <div class="flex flex-col gap-1 rounded-md border border-border-subtle bg-card p-6 text-center">
            <span class="font-display text-xl font-semibold text-text-primary">{{ formatCurrency(stats.values.totalPurchasePrice) }}</span>
            <span class="text-sm uppercase tracking-[0.08em] text-text-muted">Total Invested</span>
          </div>
          <div class="flex flex-col gap-1 rounded-md border border-border-subtle bg-card p-6 text-center">
            <span
              class="font-display text-xl font-semibold"
              :class="netGainLoss >= 0 ? 'text-[var(--color-positive)]' : 'text-[var(--color-negative)]'"
            >
              {{ netGainLoss >= 0 ? '+' : '' }}{{ formatCurrency(netGainLoss) }}
            </span>
            <span class="text-sm uppercase tracking-[0.08em] text-text-muted">Net Gain / Loss</span>
          </div>
          <div v-if="stats.values.totalPurchasePrice" class="flex flex-col gap-1 rounded-md border border-border-subtle bg-card p-6 text-center">
            <span
              class="font-display text-xl font-semibold"
              :class="roi >= 0 ? 'text-[var(--color-positive)]' : 'text-[var(--color-negative)]'"
            >
              {{ roi >= 0 ? '+' : '' }}{{ roi.toFixed(1) }}%
            </span>
            <span class="text-sm uppercase tracking-[0.08em] text-text-muted">ROI</span>
          </div>
        </div>

        <div v-if="stats" class="card flex flex-col gap-5">
          <h2>Averages</h2>
          <div class="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))]">
            <div class="flex flex-col gap-1">
              <span class="text-sm text-text-muted">Average Purchase Price</span>
              <span class="text-lg font-semibold text-text-primary">{{ formatCurrency(stats.values.avgPurchasePrice) }}</span>
            </div>
            <div class="flex flex-col gap-1">
              <span class="text-sm text-text-muted">Average Current Value</span>
              <span class="text-lg font-semibold text-gold">{{ formatCurrency(stats.values.avgCurrentValue) }}</span>
            </div>
          </div>
        </div>

        <div class="flex flex-col gap-3">
          <div class="flex flex-wrap items-center justify-between gap-4">
            <span class="section-label">Value Over Time</span>
            <div class="flex gap-[0.35rem]">
              <button
                v-for="tf in timeframes"
                :key="tf.label"
                class="chip chip-sm"
                :class="{ 'border-gold bg-gold-dim text-gold': selectedDays === tf.days }"
                type="button"
                @click="selectedDays = tf.days"
              >
                {{ tf.label }}
              </button>
            </div>
          </div>
          <StatsValueOverTime :history="filteredHistory" />
        </div>
      </template>
    </div>
  </PullToRefresh>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ArrowLeft } from 'lucide-vue-next'
import { useCoinsStore } from '@/stores/coins'
import PullToRefresh from '@/components/PullToRefresh.vue'
import StatsValueOverTime from '@/components/stats/StatsValueOverTime.vue'
import { formatCurrency } from '@/utils/format'

const store = useCoinsStore()
const isLoading = ref(true)
const stats = computed(() => store.stats)

const timeframes = [
  { label: 'All', days: 0 },
  { label: '1Y', days: 365 },
  { label: '6M', days: 180 },
  { label: '3M', days: 90 },
]
const selectedDays = ref(0)

const filteredHistory = computed(() => {
  if (!selectedDays.value) return store.valueHistory
  const cutoff = new Date()
  cutoff.setDate(cutoff.getDate() - selectedDays.value)
  return store.valueHistory.filter((s) => new Date(s.recordedAt) >= cutoff)
})

const roi = computed(() => {
  if (!stats.value?.values.totalPurchasePrice) return 0
  return (
    ((stats.value.values.totalCurrentValue - stats.value.values.totalPurchasePrice) /
      stats.value.values.totalPurchasePrice) *
    100
  )
})

const netGainLoss = computed(() => {
  if (!stats.value) return 0
  return stats.value.values.totalCurrentValue - stats.value.values.totalPurchasePrice
})

async function handleRefresh() {
  isLoading.value = true
  await Promise.all([store.fetchValueHistory(), store.fetchStats()])
  isLoading.value = false
}

onMounted(async () => {
  isLoading.value = true
  await Promise.all([store.fetchValueHistory(), store.fetchStats()])
  isLoading.value = false
})
</script>
