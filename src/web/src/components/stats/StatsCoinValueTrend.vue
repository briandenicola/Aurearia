<template>
  <div class="stats-section card">
    <h2>Coin Value Trend</h2>
    <div class="form-group" style="margin-bottom: 1rem;">
      <select v-model="selectedCoinId" class="form-input" style="max-width: 400px;">
        <option :value="0">Select a coin...</option>
        <option v-for="c in coinsWithValues" :key="c.id" :value="c.id">
          {{ c.name }} {{ c.currentValue ? `(${formatCurrency(c.currentValue)})` : '' }}
        </option>
      </select>
    </div>
    <div v-if="selectedCoinId && coinChartData.length >= 2" class="line-chart-container">
      <div class="line-chart-y-axis">
        <span>{{ formatCurrency(coinChartMax) }}</span>
        <span>{{ formatCurrency(coinChartMax / 2) }}</span>
        <span>$0</span>
      </div>
      <div class="line-chart">
        <svg viewBox="0 0 1000 300" preserveAspectRatio="none" class="line-chart-svg">
          <polyline
            :points="coinChartPoints"
            fill="none"
            stroke="var(--accent-gold)"
            stroke-width="2.5"
          />
          <circle
            v-for="(pt, i) in coinChartPointsList"
            :key="i"
            :cx="pt.x" :cy="pt.y" r="4"
            fill="var(--accent-gold)"
          />
        </svg>
      </div>
    </div>
    <div v-if="selectedCoinId && coinChartData.length >= 2" class="line-chart-dates">
      <span>{{ formatShortDate(coinChartData[0]?.date ?? '') }}</span>
      <span>{{ formatShortDate(coinChartData[coinChartData.length - 1]?.date ?? '') }}</span>
    </div>
    <p v-else-if="selectedCoinId && coinChartData.length < 2" class="chart-empty">
      Not enough data points to chart. Run an AI estimate to start tracking.
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useCoinsStore } from '@/stores/coins'
import { getCoinValueHistory } from '@/api/client'
import type { CoinValueHistory } from '@/types'
import { formatCurrency } from '@/utils/format'

const store = useCoinsStore()

const selectedCoinId = ref(0)
const coinValueEntries = ref<CoinValueHistory[]>([])

const coinsWithValues = computed(() => {
  if (!store.coins.length) return []
  return store.coins
    .filter((c) => !c.isWishlist && !c.isSold && (c.purchasePrice || c.currentValue))
    .sort((a, b) => (a.name || '').localeCompare(b.name || ''))
})

const coinChartData = computed(() => {
  const coin = store.coins.find((c) => c.id === selectedCoinId.value)
  if (!coin) return []
  const points: { date: string; value: number }[] = []
  if (coin.purchasePrice != null && coin.purchaseDate != null) {
    points.push({ date: coin.purchaseDate, value: coin.purchasePrice })
  }
  for (const e of coinValueEntries.value) {
    points.push({ date: e.recordedAt, value: e.value })
  }
  return points.sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime())
})

const coinChartMax = computed(() => {
  if (!coinChartData.value.length) return 1
  return Math.max(...coinChartData.value.map((d) => d.value)) * 1.1 || 1
})

const coinChartPoints = computed(() => {
  const data = coinChartData.value.map((d) => d.value)
  if (!data.length) return ''
  const max = coinChartMax.value
  return data
    .map((v, i) => {
      const x = data.length === 1 ? 500 : (i / (data.length - 1)) * 1000
      const y = 300 - (v / max) * 280 - 10
      return `${x},${y}`
    })
    .join(' ')
})

const coinChartPointsList = computed(() => {
  const data = coinChartData.value.map((d) => d.value)
  const max = coinChartMax.value
  return data.map((v, i) => ({
    x: data.length === 1 ? 500 : (i / (data.length - 1)) * 1000,
    y: 300 - (v / max) * 280 - 10,
  }))
})

watch(selectedCoinId, async (id) => {
  if (!id) {
    coinValueEntries.value = []
    return
  }
  try {
    const res = await getCoinValueHistory(id)
    coinValueEntries.value = res.data || []
  } catch {
    coinValueEntries.value = []
  }
})

function formatShortDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: '2-digit' })
}
</script>

<style scoped>
.stats-section h2 {
  margin-bottom: 1.25rem;
  font-size: 1.1rem;
}

.line-chart-container {
  display: flex;
  gap: 0.5rem;
}

.line-chart-y-axis {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  font-size: 0.7rem;
  color: var(--text-muted);
  text-align: right;
  min-width: 60px;
  padding: 0.25rem 0;
}

.line-chart {
  flex: 1;
  height: 200px;
  background: var(--bg-primary);
  border-radius: var(--radius-sm);
  padding: 0.5rem;
}

.line-chart-svg {
  width: 100%;
  height: 100%;
}

.line-chart-dates {
  display: flex;
  justify-content: space-between;
  font-size: 0.7rem;
  color: var(--text-muted);
  margin-top: 0.25rem;
  padding: 0 0.5rem 0 68px;
}

.chart-empty {
  color: var(--text-muted);
  font-size: 0.85rem;
  font-style: italic;
  padding: 1rem 0;
}
</style>
