<template>
  <PullToRefresh :on-refresh="handleRefresh">
    <div class="container stats-value-details-page">
      <header class="page-header value-details-header">
        <div>
          <p class="section-label">Collection Insights</p>
          <h1>Value Details</h1>
          <p class="page-intro">Your collection's value breakdown and history.</p>
        </div>
        <router-link class="back-button" to="/stats" aria-label="Back to Stats">
          <ArrowLeft :size="20" />
        </router-link>
      </header>

      <div v-if="isLoading" class="loading-overlay">
        <div class="spinner"></div>
      </div>

      <template v-else>
        <!-- Value summary cards -->
        <div v-if="stats" class="value-summary-cards">
          <div class="stat-card">
            <span class="stat-number gold">{{ formatCurrency(stats.values.totalCurrentValue) }}</span>
            <span class="stat-label">Total Value</span>
          </div>
          <div class="stat-card">
            <span class="stat-number">{{ formatCurrency(stats.values.totalPurchasePrice) }}</span>
            <span class="stat-label">Total Invested</span>
          </div>
          <div class="stat-card">
            <span
              class="stat-number"
              :class="netGainLoss >= 0 ? 'positive' : 'negative'"
            >
              {{ netGainLoss >= 0 ? '+' : '' }}{{ formatCurrency(netGainLoss) }}
            </span>
            <span class="stat-label">Net Gain / Loss</span>
          </div>
          <div class="stat-card" v-if="stats.values.totalPurchasePrice">
            <span
              class="stat-number"
              :class="roi >= 0 ? 'positive' : 'negative'"
            >
              {{ roi >= 0 ? '+' : '' }}{{ roi.toFixed(1) }}%
            </span>
            <span class="stat-label">ROI</span>
          </div>
        </div>

        <!-- Average details -->
        <div v-if="stats" class="stats-section card">
          <h2>Averages</h2>
          <div class="value-stats">
            <div class="value-stat">
              <span class="value-stat-label">Average Purchase Price</span>
              <span class="value-stat-amount">{{ formatCurrency(stats.values.avgPurchasePrice) }}</span>
            </div>
            <div class="value-stat">
              <span class="value-stat-label">Average Current Value</span>
              <span class="value-stat-amount gold">{{ formatCurrency(stats.values.avgCurrentValue) }}</span>
            </div>
          </div>
        </div>

        <!-- Timeframe selector + chart -->
        <div class="chart-section">
          <div class="timeframe-row">
            <span class="section-label">Value Over Time</span>
            <div class="timeframe-chips">
              <button
                v-for="tf in timeframes"
                :key="tf.label"
                class="chip chip-sm"
                :class="{ active: selectedDays === tf.days }"
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

<style scoped>
.stats-value-details-page {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.value-details-header {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: nowrap;
}

.page-intro {
  margin: 0.35rem 0 0;
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.back-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.4rem;
  color: var(--text-secondary);
  background: transparent;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  text-decoration: none;
  transition: var(--transition-fast);
  flex-shrink: 0;
}

.back-button:hover {
  color: var(--accent-gold);
  border-color: var(--border-accent);
  background: var(--accent-gold-glow);
}

.value-summary-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 1rem;
}

.stat-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1.5rem;
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.stat-number {
  font-family: 'Cinzel', serif;
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-primary);
}

.stat-number.gold { color: var(--accent-gold); }
.stat-number.positive { color: var(--color-positive); }
.stat-number.negative { color: var(--color-negative); }

.stat-label {
  font-size: 0.8rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.stats-section h2 {
  margin-bottom: 1.25rem;
  font-size: 1.1rem;
}

.value-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.value-stat {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.value-stat-label {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.value-stat-amount {
  font-size: 1.2rem;
  font-weight: 600;
}

.value-stat-amount.gold { color: var(--accent-gold); }

.chart-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.timeframe-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.timeframe-chips {
  display: flex;
  gap: 0.35rem;
}

.chip.active {
  background: var(--accent-gold-dim);
  border-color: var(--accent-gold);
  color: var(--accent-gold);
}
</style>
