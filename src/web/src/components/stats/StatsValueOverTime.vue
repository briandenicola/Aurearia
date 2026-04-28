<template>
  <div v-if="store.valueHistory.length >= 2" class="stats-section card">
    <h2>Value Over Time</h2>
    <div class="line-chart-container">
      <div class="line-chart-y-axis">
        <span>{{ formatCurrency(chartMaxValue) }}</span>
        <span>{{ formatCurrency(chartMaxValue / 2) }}</span>
        <span>$0</span>
      </div>
      <div class="line-chart">
        <svg viewBox="0 0 1000 300" preserveAspectRatio="none" class="line-chart-svg">
          <polyline
            :points="investedPoints"
            fill="none"
            stroke="var(--text-muted)"
            stroke-width="2"
            stroke-dasharray="6 3"
          />
          <polyline
            :points="valuePoints"
            fill="none"
            stroke="var(--accent-gold)"
            stroke-width="2.5"
          />
          <circle
            v-for="(pt, i) in valuePointsList"
            :key="i"
            :cx="pt.x" :cy="pt.y" r="4"
            fill="var(--accent-gold)"
          />
        </svg>
      </div>
    </div>
    <div class="line-chart-legend">
      <span class="legend-item"><span class="legend-line legend-value"></span> Current Value</span>
      <span class="legend-item"><span class="legend-line legend-invested"></span> Invested</span>
    </div>
    <div class="line-chart-dates">
      <span>{{ formatShortDate(store.valueHistory[0]?.recordedAt ?? '') }}</span>
      <span>{{ formatShortDate(store.valueHistory[store.valueHistory.length - 1]?.recordedAt ?? '') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useCoinsStore } from '@/stores/coins'
import { formatCurrency } from '@/utils/format'

const store = useCoinsStore()

const chartMaxValue = computed(() => {
  if (!store.valueHistory.length) return 1
  const max = Math.max(...store.valueHistory.flatMap((s) => [s.totalValue, s.totalInvested]))
  return max * 1.1 || 1
})

function toSvgPoints(data: number[]): string {
  if (!data.length) return ''
  const max = chartMaxValue.value
  return data
    .map((v, i) => {
      const x = data.length === 1 ? 500 : (i / (data.length - 1)) * 1000
      const y = 300 - (v / max) * 280 - 10
      return `${x},${y}`
    })
    .join(' ')
}

const valuePoints = computed(() => toSvgPoints(store.valueHistory.map((s) => s.totalValue)))
const investedPoints = computed(() => toSvgPoints(store.valueHistory.map((s) => s.totalInvested)))
const valuePointsList = computed(() => {
  const data = store.valueHistory.map((s) => s.totalValue)
  const max = chartMaxValue.value
  return data.map((v, i) => ({
    x: data.length === 1 ? 500 : (i / (data.length - 1)) * 1000,
    y: 300 - (v / max) * 280 - 10,
  }))
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

.line-chart-legend {
  display: flex;
  gap: 1.5rem;
  justify-content: center;
  margin-top: 0.75rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.legend-line {
  display: inline-block;
  width: 20px;
  height: 3px;
  border-radius: 2px;
}

.legend-value {
  background: var(--accent-gold);
}

.legend-invested {
  background: var(--text-muted);
  background-image: repeating-linear-gradient(
    90deg,
    var(--text-muted) 0px,
    var(--text-muted) 6px,
    transparent 6px,
    transparent 9px
  );
}

.line-chart-dates {
  display: flex;
  justify-content: space-between;
  font-size: 0.7rem;
  color: var(--text-muted);
  margin-top: 0.25rem;
  padding: 0 0.5rem 0 68px;
}
</style>
