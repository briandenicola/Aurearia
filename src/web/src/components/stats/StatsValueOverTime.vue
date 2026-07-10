<template>
  <div v-if="displayHistory.length >= 2" class="stats-section card relative overflow-hidden border-border-accent shadow-card before:pointer-events-none before:absolute before:inset-0 before:bg-[radial-gradient(circle_at_top_right,var(--color-gold-glow),transparent_35%),linear-gradient(180deg,var(--color-card-hover),var(--color-card))] before:opacity-80 before:content-[''] [&>*]:relative [&>*]:z-[1]">
    <div class="mb-5">
      <div>
        <p class="section-label">Portfolio Trajectory</p>
        <h2 class="mt-[0.15rem]">Value Over Time</h2>
        <p class="mt-[0.35rem] text-body text-text-secondary">
          Active collection value movement between the first and latest snapshot in this timeframe.
        </p>
      </div>
    </div>

    <div class="flex flex-col items-start gap-5 md:flex-row">
      <div class="min-w-0 flex-1">
        <div class="flex min-w-0 gap-3">
          <div class="flex min-w-[3.75rem] flex-col justify-between py-[0.35rem] text-right text-label text-text-muted max-[480px]:min-w-[3.25rem]" aria-hidden="true">
            <span>{{ formatCurrency(chartMaxValue) }}</span>
            <span>{{ formatCurrency(chartMaxValue / 2) }}</span>
            <span>$0</span>
          </div>
          <div class="h-[15rem] min-w-0 flex-1 rounded-md border border-border-subtle bg-input shadow-[inset_0_1px_0_var(--color-border-subtle)] max-md:h-[13rem]" role="img" aria-label="Current value and invested amount over time">
            <svg viewBox="0 0 1000 320" preserveAspectRatio="none" class="h-full w-full overflow-visible">
              <defs>
                <linearGradient id="valueAreaGradient" x1="0" x2="0" y1="0" y2="1">
                  <stop offset="0%" stop-color="var(--accent-gold)" stop-opacity="0.20" />
                  <stop offset="100%" stop-color="var(--accent-gold)" stop-opacity="0.01" />
                </linearGradient>
                <filter id="valueLineGlow" x="-10%" y="-20%" width="120%" height="140%">
                  <feGaussianBlur stdDeviation="4" result="blur" />
                  <feMerge>
                    <feMergeNode in="blur" />
                    <feMergeNode in="SourceGraphic" />
                  </feMerge>
                </filter>
              </defs>

              <line
                v-for="tick in verticalGridTicks"
                :key="`v-${tick}`"
                class="chart-grid-line chart-grid-line-vertical"
                :x1="tick"
                :x2="tick"
                y1="20"
                y2="300"
              />
              <line class="chart-axis-line" x1="0" x2="1000" y1="300" y2="300" />

              <path class="chart-area-fill" :d="valueAreaPath" />
              <path class="chart-line chart-line-invested" :d="investedPath" />
              <path class="chart-line chart-line-value chart-line-glow" :d="valuePath" />
              <path class="chart-line chart-line-value" :d="valuePath" />

              <circle
                v-for="(pt, i) in valuePointsList.slice(0, -1)"
                :key="`vp-${i}`"
                class="chart-point chart-point-value"
                :cx="pt.x"
                :cy="pt.y"
                r="4"
              />
              <circle
                v-for="(pt, i) in investedPointsList.slice(0, -1)"
                :key="`ip-${i}`"
                class="chart-point chart-point-invested"
                :cx="pt.x"
                :cy="pt.y"
                r="3"
              />

              <text
                v-for="(pt, i) in sparseLabelPoints"
                :key="`lbl-${i}`"
                class="chart-point-label"
                :x="pt.x"
                :y="pt.y - 14"
                dominant-baseline="auto"
                text-anchor="middle"
              >{{ formatShortAmount(pt.value) }}</text>

              <g v-if="latestValuePoint">
                <circle
                  class="endpoint-dot endpoint-dot-value"
                  :cx="latestValuePoint.x"
                  :cy="latestValuePoint.y"
                  r="30"
                />
                <text
                  class="endpoint-callout-text"
                  :x="latestValuePoint.x"
                  :y="latestValuePoint.y"
                  dominant-baseline="middle"
                  text-anchor="middle"
                >{{ formatShortAmount(latestValue) }}</text>
              </g>

              <circle
                v-if="latestInvestedPoint"
                class="endpoint-dot endpoint-dot-invested"
                :cx="latestInvestedPoint.x"
                :cy="latestInvestedPoint.y"
                r="5"
              />
            </svg>
          </div>
        </div>

        <div class="mt-[0.6rem] flex justify-between gap-4 pl-[4.5rem] max-[480px]:flex-col max-[480px]:pl-0">
          <div class="flex flex-wrap gap-3 text-chip text-text-secondary">
            <span class="flex items-center gap-[0.4rem]">
              <span class="inline-block h-[0.2rem] w-6 rounded-full bg-gold shadow-glow"></span>
              Current Value
            </span>
            <span class="flex items-center gap-[0.4rem]">
              <span class="inline-block h-[0.2rem] w-6 rounded-full bg-[repeating-linear-gradient(90deg,var(--color-text-secondary)_0,var(--color-text-secondary)_0.4rem,transparent_0.4rem,transparent_0.65rem)]"></span>
              Invested
            </span>
          </div>
          <div class="flex gap-3 whitespace-nowrap text-label text-text-muted max-[480px]:justify-between">
            <span>{{ formatShortDate(firstSnapshot?.recordedAt ?? '') }}</span>
            <span>{{ formatShortDate(lastSnapshot?.recordedAt ?? '') }}</span>
          </div>
        </div>
      </div>

      <div class="flex w-full flex-wrap items-start gap-3 md:w-[10.5rem] md:flex-shrink-0 md:flex-col md:gap-4">
        <div class="flex min-w-[8rem] flex-1 flex-col gap-[0.2rem] md:flex-none">
          <p class="section-label">Period Value Change</p>
          <span class="block font-display text-[1.9rem] font-semibold leading-[1.1]" :class="changeAmount >= 0 ? 'text-[var(--color-positive)]' : 'text-[var(--color-negative)]'">
            <template v-if="changePercent !== null">
              {{ changePercent >= 0 ? '+' : '' }}{{ changePercent.toFixed(1) }}%
            </template>
            <template v-else>
              {{ changeAmount >= 0 ? '+' : '' }}{{ formatCurrency(changeAmount) }}
            </template>
          </span>
          <span class="text-label text-text-muted">
            {{ formatShortDate(firstSnapshot?.recordedAt ?? '') }} —
            {{ formatShortDate(lastSnapshot?.recordedAt ?? '') }}
          </span>
        </div>

        <div class="flex grow basis-[14rem] flex-wrap gap-2 md:basis-auto md:flex-col">
          <div class="flex min-w-[6rem] flex-1 flex-col gap-[0.15rem] rounded-sm border border-border-subtle bg-input px-3 py-[0.6rem] md:min-w-0 md:flex-none">
            <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Latest Snapshot</span>
            <strong class="overflow-hidden text-body text-text-primary text-ellipsis whitespace-nowrap">{{ formatCurrency(latestValue) }}</strong>
          </div>
          <div class="flex min-w-[6rem] flex-1 flex-col gap-[0.15rem] rounded-sm border border-border-subtle bg-input px-3 py-[0.6rem] md:min-w-0 md:flex-none">
            <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Invested</span>
            <strong class="overflow-hidden text-body text-text-primary text-ellipsis whitespace-nowrap">{{ formatCurrency(latestInvested) }}</strong>
          </div>
          <div class="flex min-w-[6rem] flex-1 flex-col gap-[0.15rem] rounded-sm border border-border-subtle bg-input px-3 py-[0.6rem] md:min-w-0 md:flex-none">
            <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Period Change</span>
            <strong class="overflow-hidden text-body text-ellipsis whitespace-nowrap" :class="changeAmount >= 0 ? 'text-[var(--color-positive)]' : 'text-[var(--color-negative)]'">
              {{ changeAmount >= 0 ? '+' : '' }}{{ formatCurrency(changeAmount) }}
            </strong>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useCoinsStore } from '@/stores/coins'
import type { ValueSnapshot } from '@/types'
import { formatCurrency } from '@/utils/format'

type ChartPoint = { x: number; y: number }
type LabelPoint = ChartPoint & { value: number }

const CHART_WIDTH = 1000
const CHART_TOP = 20
const CHART_BOTTOM = 300
const CHART_RANGE = CHART_BOTTOM - CHART_TOP
const verticalGridTicks = [200, 400, 600, 800]

const props = withDefaults(
  defineProps<{ history?: ValueSnapshot[] }>(),
  { history: undefined },
)

const store = useCoinsStore()

const displayHistory = computed(() => props.history ?? store.valueHistory)
const firstSnapshot = computed(() => displayHistory.value[0])
const lastSnapshot = computed(() => displayHistory.value[displayHistory.value.length - 1])
const latestValue = computed(() => lastSnapshot.value?.totalValue ?? 0)
const latestInvested = computed(() => lastSnapshot.value?.totalInvested ?? 0)
const changeAmount = computed(
  () => latestValue.value - (firstSnapshot.value?.totalValue ?? latestValue.value),
)
const changePercent = computed(() => {
  const startingValue = firstSnapshot.value?.totalValue ?? 0
  if (!startingValue) return null
  return (changeAmount.value / startingValue) * 100
})

const chartMaxValue = computed(() => {
  if (!displayHistory.value.length) return 1
  const max = Math.max(...displayHistory.value.flatMap((s) => [s.totalValue, s.totalInvested]))
  return max * 1.12 || 1
})

function toChartPoints(data: number[]): ChartPoint[] {
  const max = chartMaxValue.value
  return data.map((value, index) => ({
    x: data.length === 1 ? CHART_WIDTH / 2 : (index / (data.length - 1)) * CHART_WIDTH,
    y: CHART_BOTTOM - (value / max) * CHART_RANGE,
  }))
}

function toSmoothPath(points: ChartPoint[]): string {
  if (!points.length) return ''
  if (points.length === 1) return `M ${points[0]?.x ?? 0} ${points[0]?.y ?? CHART_BOTTOM}`
  return points.reduce((path, point, index) => {
    if (index === 0) return `M ${point.x} ${point.y}`
    const previous = points[index - 1]
    if (!previous) return path
    const midpointX = (previous.x + point.x) / 2
    return `${path} Q ${previous.x} ${previous.y} ${midpointX} ${(previous.y + point.y) / 2} T ${point.x} ${point.y}`
  }, '')
}

const valuePointsList = computed(() =>
  toChartPoints(displayHistory.value.map((s) => s.totalValue)),
)
const investedPointsList = computed(() =>
  toChartPoints(displayHistory.value.map((s) => s.totalInvested)),
)
const valuePath = computed(() => toSmoothPath(valuePointsList.value))
const investedPath = computed(() => toSmoothPath(investedPointsList.value))
const valueAreaPath = computed(() => {
  const points = valuePointsList.value
  if (!points.length) return ''
  const first = points[0]
  const last = points[points.length - 1]
  if (!first || !last) return ''
  return `${valuePath.value} L ${last.x} ${CHART_BOTTOM} L ${first.x} ${CHART_BOTTOM} Z`
})
const latestValuePoint = computed(() => valuePointsList.value[valuePointsList.value.length - 1])
const latestInvestedPoint = computed(
  () => investedPointsList.value[investedPointsList.value.length - 1],
)

const labelInterval = computed(() => {
  const n = displayHistory.value.length
  if (n <= 6) return 1
  if (n <= 12) return 2
  return Math.ceil(n / 6)
})

const sparseLabelPoints = computed((): LabelPoint[] => {
  return valuePointsList.value
    .map((pt, i) => ({ ...pt, value: displayHistory.value[i]?.totalValue ?? 0 }))
    .filter((_, i, arr) => {
      if (i === arr.length - 1) return false
      return i === 0 || i % labelInterval.value === 0
    })
})

function formatShortAmount(v: number): string {
  if (v >= 10000) return `$${(v / 1000).toFixed(0)}K`
  if (v >= 1000) return `$${(v / 1000).toFixed(1)}K`
  return `$${Math.round(v)}`
}

function formatShortDate(dateStr: string) {
  if (!dateStr) return '—'
  return new Date(dateStr).toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    year: '2-digit',
  })
}
</script>

<style scoped>
/* kept: SVG chart geometry */
.chart-grid-line {
  stroke: var(--border-subtle);
  stroke-width: 1;
  stroke-dasharray: 4 12;
  vector-effect: non-scaling-stroke;
  opacity: 0.4;
}

.chart-grid-line-vertical {
  opacity: 0.3;
}

.chart-axis-line {
  stroke: var(--border-accent);
  stroke-width: 1;
  opacity: 0.6;
  vector-effect: non-scaling-stroke;
}

.chart-area-fill {
  fill: url('#valueAreaGradient');
}

.chart-line {
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
  vector-effect: non-scaling-stroke;
}

.chart-line-value {
  stroke: var(--accent-gold);
  stroke-width: 3;
}

.chart-line-glow {
  opacity: 0.3;
  stroke-width: 9;
  filter: url('#valueLineGlow');
}

.chart-line-invested {
  stroke: var(--text-secondary);
  stroke-width: 2;
  stroke-dasharray: 6 6;
  opacity: 0.8;
}

.chart-point {
  vector-effect: non-scaling-stroke;
}

.chart-point-value {
  fill: var(--accent-gold);
  stroke: var(--bg-card);
  stroke-width: 2;
}

.chart-point-invested {
  fill: var(--text-secondary);
  opacity: 0.65;
}

.chart-point-label {
  fill: var(--text-secondary);
  font-size: 0.6rem;
  font-weight: 600;
}

.endpoint-dot {
  fill: var(--bg-card);
  vector-effect: non-scaling-stroke;
}

.endpoint-dot-value {
  stroke: var(--accent-gold);
  stroke-width: 2.5;
}

.endpoint-dot-invested {
  stroke: var(--text-secondary);
  stroke-width: 2;
}

.endpoint-callout-text {
  fill: var(--accent-gold);
  font-size: 0.65rem;
  font-weight: 700;
}
</style>
