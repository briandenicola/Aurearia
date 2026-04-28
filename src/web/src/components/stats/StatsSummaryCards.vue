<template>
  <div class="stats-summary">
    <div class="stat-card">
      <span class="stat-number">{{ stats.totalCoins }}</span>
      <span class="stat-label">Coins Owned</span>
    </div>
    <div class="stat-card">
      <span class="stat-number">{{ stats.totalWishlist }}</span>
      <span class="stat-label">On Wishlist</span>
    </div>
    <div class="stat-card">
      <span class="stat-number gold">{{ formatCurrency(stats.values.totalCurrentValue) }}</span>
      <span class="stat-label">Total Value</span>
    </div>
    <div class="stat-card">
      <span class="stat-number">{{ formatCurrency(stats.values.totalPurchasePrice) }}</span>
      <span class="stat-label">Total Invested</span>
    </div>
  </div>

  <!-- Value Summary -->
  <div class="stats-section card">
    <h2>Value Summary</h2>
    <div class="value-stats">
      <div class="value-stat">
        <span class="value-stat-label">Average Purchase Price</span>
        <span class="value-stat-amount">{{ formatCurrency(stats.values.avgPurchasePrice) }}</span>
      </div>
      <div class="value-stat">
        <span class="value-stat-label">Average Current Value</span>
        <span class="value-stat-amount gold">{{ formatCurrency(stats.values.avgCurrentValue) }}</span>
      </div>
      <div class="value-stat" v-if="stats.values.totalCurrentValue && stats.values.totalPurchasePrice">
        <span class="value-stat-label">Return on Investment</span>
        <span
          class="value-stat-amount"
          :class="roi >= 0 ? 'positive' : 'negative'"
        >
          {{ roi >= 0 ? '+' : '' }}{{ roi.toFixed(1) }}%
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { StatsResponse } from '@/types'
import { formatCurrency } from '@/utils/format'

const props = defineProps<{
  stats: StatsResponse
}>()

const roi = computed(() => {
  if (!props.stats.values.totalPurchasePrice) return 0
  return (
    ((props.stats.values.totalCurrentValue - props.stats.values.totalPurchasePrice) /
      props.stats.values.totalPurchasePrice) *
    100
  )
})
</script>

<style scoped>
.stats-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
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
  font-size: 2rem;
  font-weight: 600;
  color: var(--text-primary);
}

.stat-number.gold {
  color: var(--accent-gold);
}

.stat-label {
  font-size: 0.8rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
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
  font-size: 1.3rem;
  font-weight: 600;
}

.value-stat-amount.gold { color: var(--accent-gold); }
.value-stat-amount.positive { color: #2ecc71; }
.value-stat-amount.negative { color: #e74c3c; }
</style>
