<template>
  <PullToRefresh :on-refresh="handleRefresh">
  <div class="container">
    <div class="page-header">
      <h1>Collection Stats</h1>
    </div>

    <div v-if="!stats" class="loading-overlay">
      <div class="spinner"></div>
    </div>

    <div v-else class="stats-layout">
      <!-- Summary Cards -->
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

      <!-- Category Breakdown -->
      <div class="stats-section card">
        <h2>By Category</h2>
        <div class="bar-chart">
          <div v-for="item in stats.byCategory" :key="item.category" class="bar-row">
            <span class="bar-label">
              <span class="badge" :class="`badge-${item.category.toLowerCase()}`">{{ item.category }}</span>
            </span>
            <div class="bar-track">
              <div
                class="bar-fill"
                :class="`fill-${item.category.toLowerCase()}`"
                :style="{ width: `${(item.count / maxCategoryCount) * 100}%` }"
              ></div>
            </div>
            <span class="bar-value">{{ item.count }}</span>
          </div>
        </div>
      </div>

      <!-- Material Breakdown -->
      <div class="stats-section card">
        <h2>By Material</h2>
        <div class="bar-chart">
          <div v-for="item in stats.byMaterial" :key="item.material" class="bar-row">
            <span class="bar-label">
              <span :class="`material-${item.material.toLowerCase()}`">{{ item.material }}</span>
            </span>
            <div class="bar-track">
              <div
                class="bar-fill fill-material"
                :style="{ width: `${(item.count / maxMaterialCount) * 100}%` }"
              ></div>
            </div>
            <span class="bar-value">{{ item.count }}</span>
          </div>
        </div>
      </div>

      <!-- Grade Distribution -->
      <div v-if="stats.byGrade?.length" class="stats-section card">
        <h2>By Grade</h2>
        <div class="bar-chart">
          <div v-for="item in stats.byGrade" :key="item.grade" class="bar-row">
            <span class="bar-label">{{ item.grade }}</span>
            <div class="bar-track">
              <div
                class="bar-fill fill-grade"
                :style="{ width: `${(item.count / maxGradeCount) * 100}%` }"
              ></div>
            </div>
            <span class="bar-value">{{ item.count }}</span>
          </div>
        </div>
      </div>

      <!-- Era Breakdown -->
      <div v-if="stats.byEra?.length" class="stats-section card">
        <h2>By Era</h2>
        <div class="bar-chart">
          <div v-for="item in stats.byEra" :key="item.era" class="bar-row">
            <span class="bar-label">{{ item.era }}</span>
            <div class="bar-track">
              <div
                class="bar-fill fill-era"
                :style="{ width: `${(item.count / maxEraCount) * 100}%` }"
              ></div>
            </div>
            <span class="bar-value">{{ item.count }}</span>
          </div>
        </div>
      </div>

      <!-- Top Rulers -->
      <div v-if="stats.byRuler?.length" class="stats-section card">
        <h2>Top Rulers</h2>
        <div class="bar-chart">
          <div v-for="item in stats.byRuler" :key="item.ruler" class="bar-row bar-row-wide">
            <span class="bar-label bar-label-wide">{{ item.ruler }}</span>
            <div class="bar-track">
              <div
                class="bar-fill fill-ruler"
                :style="{ width: `${(item.count / maxRulerCount) * 100}%` }"
              ></div>
            </div>
            <span class="bar-value">{{ item.count }}</span>
          </div>
        </div>
      </div>

      <!-- Price Range Distribution -->
      <div v-if="stats.byPriceRange?.length" class="stats-section card">
        <h2>Price Range Distribution</h2>
        <div class="bar-chart">
          <div v-for="item in sortedPriceRanges" :key="item.range" class="bar-row">
            <span class="bar-label">{{ item.range }}</span>
            <div class="bar-track">
              <div
                class="bar-fill fill-price"
                :style="{ width: `${(item.count / maxPriceRangeCount) * 100}%` }"
              ></div>
            </div>
            <span class="bar-value">{{ item.count }}</span>
          </div>
        </div>
      </div>

      <!-- Value Over Time -->
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

      <!-- Coin Value Trend -->
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
          <div class="line-chart-dates">
            <span>{{ formatShortDate(coinChartData[0]?.date ?? '') }}</span>
            <span>{{ formatShortDate(coinChartData[coinChartData.length - 1]?.date ?? '') }}</span>
          </div>
        </div>
        <p v-else-if="selectedCoinId && coinChartData.length < 2" class="chart-empty">
          Not enough data points to chart. Run an AI estimate to start tracking.
        </p>
      </div>

      <!-- Collection Distribution Heat Map -->
      <div class="stats-section card">
        <h2>Collection Distribution</h2>
        <div v-if="heatMapEras.length && heatMapCategories.length" class="heatmap-container">
          <div class="heatmap-grid" :style="{ gridTemplateColumns: `minmax(60px, 80px) repeat(${heatMapCategories.length}, 1fr)` }">
            <div class="heatmap-corner"></div>
            <div v-for="cat in heatMapCategories" :key="cat" class="heatmap-col-header">{{ cat }}</div>
            <template v-for="era in heatMapEras" :key="era">
              <div class="heatmap-row-header">{{ era }}</div>
              <div
                v-for="cat in heatMapCategories"
                :key="`${era}-${cat}`"
                class="heatmap-cell"
                :style="{ backgroundColor: cellColor(heatMapData[`${era}|${cat}`] ?? 0) }"
                :title="`${era} / ${cat}: ${heatMapData[`${era}|${cat}`] ?? 0} coins`"
                @click="navigateToFiltered(era, cat)"
              >
                <span v-if="(heatMapData[`${era}|${cat}`] ?? 0) > 0" class="heatmap-count">{{ heatMapData[`${era}|${cat}`] }}</span>
              </div>
            </template>
          </div>
          <div class="heatmap-legend">
            <span class="heatmap-legend-label">0</span>
            <div class="heatmap-legend-bar"></div>
            <span class="heatmap-legend-label">{{ heatMapMax }}</span>
          </div>
        </div>
        <p v-else class="chart-empty">Add coins with era and category to see distribution.</p>
      </div>

    </div>
  </div>
  </PullToRefresh>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useCoinsStore } from '@/stores/coins'
import { useRouter } from 'vue-router'
import { getCoinValueHistory, getDistribution } from '@/api/client'
import type { CoinValueHistory } from '@/types'
import PullToRefresh from '@/components/PullToRefresh.vue'

const store = useCoinsStore()
const router = useRouter()
const stats = computed(() => store.stats)

const maxCategoryCount = computed(() =>
  Math.max(...(stats.value?.byCategory.map((c) => c.count) || [1])),
)
const maxMaterialCount = computed(() =>
  Math.max(...(stats.value?.byMaterial.map((m) => m.count) || [1])),
)
const maxGradeCount = computed(() =>
  Math.max(...(stats.value?.byGrade?.map((g) => g.count) || [1])),
)
const maxEraCount = computed(() =>
  Math.max(...(stats.value?.byEra?.map((e) => e.count) || [1])),
)
const maxRulerCount = computed(() =>
  Math.max(...(stats.value?.byRuler?.map((r) => r.count) || [1])),
)
const maxPriceRangeCount = computed(() =>
  Math.max(...(stats.value?.byPriceRange?.map((p) => p.count) || [1])),
)

const priceRangeOrder = ['Under $50', '$50 - $200', '$200 - $500', '$500 - $1K', '$1K+']
const sortedPriceRanges = computed(() => {
  if (!stats.value?.byPriceRange) return []
  return [...stats.value.byPriceRange].sort(
    (a, b) => priceRangeOrder.indexOf(a.range) - priceRangeOrder.indexOf(b.range),
  )
})

const roi = computed(() => {
  if (!stats.value?.values.totalPurchasePrice) return 0
  return (
    ((stats.value.values.totalCurrentValue - stats.value.values.totalPurchasePrice) /
      stats.value.values.totalPurchasePrice) *
    100
  )
})

// Value over time chart
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

// Coin value trend chart
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

function formatCurrency(value: number) {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 }).format(value)
}

function formatShortDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: '2-digit' })
}

// Heat map distribution
const heatMapData = ref<Record<string, number>>({})
const heatMapEras = ref<string[]>([])
const heatMapCategories = ref<string[]>([])
const heatMapMax = ref(1)

async function fetchDistribution() {
  try {
    const res = await getDistribution()
    const cells = res.data?.cells ?? []
    const eras = new Set<string>()
    const cats = new Set<string>()
    const map: Record<string, number> = {}
    let max = 1
    for (const cell of cells) {
      eras.add(cell.era)
      cats.add(cell.category)
      map[`${cell.era}|${cell.category}`] = cell.count
      if (cell.count > max) max = cell.count
    }
    heatMapEras.value = [...eras].sort()
    heatMapCategories.value = [...cats].sort()
    heatMapData.value = map
    heatMapMax.value = max
  } catch { /* ignore */ }
}

function cellColor(count: number): string {
  if (count === 0) return 'rgba(191, 155, 48, 0.05)'
  const intensity = Math.min(count / heatMapMax.value, 1)
  const alpha = 0.15 + intensity * 0.7
  return `rgba(191, 155, 48, ${alpha.toFixed(2)})`
}

function navigateToFiltered(era: string, category: string) {
  if ((heatMapData.value[`${era}|${category}`] ?? 0) > 0) {
    router.push({ path: '/', query: { category } })
  }
}

async function handleRefresh() {
  await Promise.all([store.fetchStats(), store.fetchValueHistory(), fetchDistribution()])
}

onMounted(() => {
  store.fetchStats()
  store.fetchValueHistory()
  fetchDistribution()
  if (!store.coins.length) store.fetchCoins()
})
</script>

<style scoped>
.stats-layout {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

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

.bar-chart {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.bar-row {
  display: grid;
  grid-template-columns: 100px 1fr 40px;
  gap: 0.75rem;
  align-items: center;
}

.bar-label {
  font-size: 0.85rem;
}

.bar-track {
  height: 24px;
  background: var(--bg-primary);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.bar-fill {
  height: 100%;
  border-radius: var(--radius-sm);
  transition: width 0.5s ease;
  min-width: 4px;
}

.fill-roman { background: linear-gradient(90deg, #7b2d8e, #9b59b6); }
.fill-greek { background: linear-gradient(90deg, #4a6e18, #6b8e23); }
.fill-byzantine { background: linear-gradient(90deg, #8b1a1a, #c0392b); }
.fill-modern { background: linear-gradient(90deg, #2c5f8a, #4682b4); }
.fill-other { background: linear-gradient(90deg, #555, #888); }
.fill-material { background: linear-gradient(90deg, var(--accent-bronze), var(--accent-gold)); }
.fill-grade { background: linear-gradient(90deg, #2c5f8a, #7ab3d4); }
.fill-era { background: linear-gradient(90deg, #6b4c3b, #a67c52); }
.fill-ruler { background: linear-gradient(90deg, #8b6914, var(--accent-gold)); }
.fill-price { background: linear-gradient(90deg, #2e7d32, #66bb6a); }

.bar-row-wide {
  grid-template-columns: 150px 1fr 40px;
}

.bar-label-wide {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.bar-value {
  font-size: 0.85rem;
  font-weight: 600;
  text-align: right;
  color: var(--text-secondary);
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

.chart-empty {
  color: var(--text-muted);
  font-size: 0.85rem;
  font-style: italic;
  padding: 1rem 0;
}

/* Heat map */
.heatmap-container {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.heatmap-grid {
  display: grid;
  gap: 2px;
  min-width: 300px;
}

.heatmap-corner {
  background: transparent;
}

.heatmap-col-header {
  font-size: 0.65rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-align: center;
  padding: 0.25rem 0.15rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.heatmap-row-header {
  font-size: 0.65rem;
  font-weight: 600;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  padding-right: 0.35rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 80px;
}

.heatmap-cell {
  aspect-ratio: 1;
  min-height: 28px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s;
  border: 1px solid rgba(191, 155, 48, 0.1);
}

.heatmap-cell:hover {
  transform: scale(1.1);
  box-shadow: 0 0 8px rgba(191, 155, 48, 0.4);
  z-index: 1;
}

.heatmap-count {
  font-size: 0.65rem;
  font-weight: 700;
  color: var(--text-primary);
}

.heatmap-legend {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.75rem;
  justify-content: center;
}

.heatmap-legend-label {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.heatmap-legend-bar {
  width: 120px;
  height: 10px;
  border-radius: 5px;
  background: linear-gradient(to right, rgba(191, 155, 48, 0.1), rgba(191, 155, 48, 0.85));
}

@media (max-width: 480px) {
  .heatmap-grid {
    gap: 1px;
    min-width: 0;
  }
  .heatmap-col-header {
    font-size: 0.55rem;
    padding: 0.2rem 0.1rem;
  }
  .heatmap-row-header {
    font-size: 0.55rem;
    max-width: 60px;
    padding-right: 0.2rem;
  }
  .heatmap-cell {
    min-height: 24px;
    border-radius: 3px;
  }
  .heatmap-count {
    font-size: 0.55rem;
  }
}
</style>
