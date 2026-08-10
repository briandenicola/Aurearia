<template>
  <section class="stats-section card flex flex-col gap-4">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <p class="section-label">Acquisition Trend</p>
        <h2 class="mt-[0.15rem]">Coins Purchased Per Year</h2>
      </div>
      <span v-if="!isLoading && yearlyCounts.length" class="text-chip text-text-muted">
        {{ totalPurchasedCoins }} purchased coins
      </span>
    </div>

    <div v-if="isLoading" class="flex justify-center py-10">
      <div class="spinner"></div>
    </div>

    <div v-else-if="!yearlyCounts.length" class="py-6 text-center text-base text-text-muted">
      Add purchase dates to your coins to see yearly purchase trends.
    </div>

    <div v-else class="overflow-x-auto pb-1 [scrollbar-width:thin]">
      <div :style="{ minWidth: `${chartWidth}px` }">
        <svg
          :viewBox="`0 0 ${chartWidth} ${SVG_HEIGHT}`"
          class="h-[17rem] w-full"
          preserveAspectRatio="none"
          role="img"
          aria-label="Line chart of coins purchased per year"
        >
          <line
            v-for="tick in yTicks"
            :key="`y-${tick}`"
            class="yearly-grid-line"
            :x1="PADDING_X"
            :x2="chartWidth - PADDING_X"
            :y1="toY(tick)"
            :y2="toY(tick)"
          />

          <line
            class="yearly-axis-line"
            :x1="PADDING_X"
            :x2="chartWidth - PADDING_X"
            :y1="CHART_BOTTOM"
            :y2="CHART_BOTTOM"
          />

          <path class="yearly-area" :d="areaPath" />
          <path class="yearly-line" :d="linePath" />

          <g v-for="point in points" :key="`pt-${point.year}`">
            <circle class="yearly-point" :cx="point.x" :cy="point.y" r="4" />
            <text class="yearly-value-label" :x="point.x" :y="point.y - 10" text-anchor="middle">{{ point.count }}</text>
            <text class="yearly-year-label" :x="point.x" :y="CHART_BOTTOM + 18" text-anchor="middle">{{ point.year }}</text>
          </g>

          <text
            v-for="tick in yTicks"
            :key="`ylbl-${tick}`"
            class="yearly-y-label"
            :x="PADDING_X - 10"
            :y="toY(tick) + 4"
            text-anchor="end"
          >{{ tick }}</text>
        </svg>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { getCoins } from '@/api/client'
import type { Coin } from '@/types'

const isLoading = ref(true)
const coins = ref<Coin[]>([])

const SVG_HEIGHT = 260
const PADDING_X = 28
const CHART_TOP = 20
const CHART_BOTTOM = 220
const YEAR_SPACING = 72

type YearlyCount = { year: number; count: number }
type Point = { year: number; count: number; x: number; y: number }

const yearlyCounts = computed<YearlyCount[]>(() => {
  const byYear = new Map<number, number>()
  for (const coin of coins.value) {
    if (!coin.purchaseDate) continue
    const year = new Date(coin.purchaseDate).getUTCFullYear()
    if (!Number.isFinite(year) || year < 1) continue
    byYear.set(year, (byYear.get(year) ?? 0) + 1)
  }
  return [...byYear.entries()]
    .sort((a, b) => a[0] - b[0])
    .map(([year, count]) => ({ year, count }))
})

const totalPurchasedCoins = computed(() =>
  yearlyCounts.value.reduce((sum, item) => sum + item.count, 0),
)

const maxCount = computed(() =>
  Math.max(1, ...yearlyCounts.value.map((item) => item.count)),
)

const yTicks = computed(() => {
  const max = maxCount.value
  if (max <= 4) return [0, 1, 2, 3, 4]
  const step = Math.max(1, Math.ceil(max / 4))
  const top = step * 4
  return [0, step, step * 2, step * 3, top]
})

const chartWidth = computed(() => {
  const points = yearlyCounts.value.length
  const dynamic = Math.max(320, points * YEAR_SPACING + PADDING_X * 2)
  return dynamic
})

const points = computed<Point[]>(() => {
  const values = yearlyCounts.value
  if (!values.length) return []
  if (values.length === 1) {
    const only = values[0]
    if (!only) return []
    return [{
      year: only.year,
      count: only.count,
      x: chartWidth.value / 2,
      y: toY(only.count),
    }]
  }
  const startX = PADDING_X
  const endX = chartWidth.value - PADDING_X
  const step = (endX - startX) / (values.length - 1)
  return values.map((item, index) => ({
    year: item.year,
    count: item.count,
    x: startX + step * index,
    y: toY(item.count),
  }))
})

const linePath = computed(() => {
  const plot = points.value
  if (!plot.length) return ''
  return plot.map((point, index) => `${index === 0 ? 'M' : 'L'} ${point.x} ${point.y}`).join(' ')
})

const areaPath = computed(() => {
  const plot = points.value
  if (!plot.length) return ''
  const first = plot[0]
  const last = plot[plot.length - 1]
  if (!first || !last) return ''
  return `${linePath.value} L ${last.x} ${CHART_BOTTOM} L ${first.x} ${CHART_BOTTOM} Z`
})

function toY(count: number): number {
  const max = yTicks.value[yTicks.value.length - 1] ?? 1
  if (max <= 0) return CHART_BOTTOM
  const ratio = count / max
  return CHART_BOTTOM - ratio * (CHART_BOTTOM - CHART_TOP)
}

async function loadCoins() {
  isLoading.value = true
  try {
    const limit = 100
    let page = 1
    const allCoins: Coin[] = []
    while (true) {
      const response = await getCoins({
        wishlist: 'false',
        sold: 'false',
        page,
        limit,
        sort: 'purchase_date',
        order: 'asc',
      })
      allCoins.push(...response.data.coins)
      if (allCoins.length >= response.data.total || !response.data.coins.length) break
      page += 1
    }
    coins.value = allCoins
  } finally {
    isLoading.value = false
  }
}

onMounted(loadCoins)
</script>

<style scoped>
.yearly-grid-line {
  stroke: var(--border-subtle);
  stroke-width: 1;
  stroke-dasharray: 4 8;
  opacity: 0.5;
}

.yearly-axis-line {
  stroke: var(--border-accent);
  stroke-width: 1.2;
  opacity: 0.7;
}

.yearly-area {
  fill: color-mix(in srgb, var(--accent-gold) 16%, transparent);
}

.yearly-line {
  fill: none;
  stroke: var(--accent-gold);
  stroke-width: 2.5;
  stroke-linejoin: round;
  stroke-linecap: round;
}

.yearly-point {
  fill: var(--accent-gold);
  stroke: var(--bg-card);
  stroke-width: 2;
}

.yearly-value-label {
  fill: var(--text-primary);
  font-size: 0.63rem;
  font-weight: 600;
}

.yearly-year-label {
  fill: var(--text-secondary);
  font-size: 0.68rem;
  font-weight: 600;
}

.yearly-y-label {
  fill: var(--text-muted);
  font-size: 0.62rem;
  font-weight: 500;
}
</style>
