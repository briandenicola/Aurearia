<template>
  <div v-if="displayHistory.length >= 2" class="stats-section card value-chart-card">
    <div class="chart-card-header">
      <div>
        <p class="section-label">Portfolio Trajectory</p>
        <h2>Value Over Time</h2>
      </div>
    </div>

    <div class="chart-main-layout">
      <!-- Chart area -->
      <div class="chart-area">
        <div class="line-chart-container">
          <div class="line-chart-y-axis" aria-hidden="true">
            <span>{{ formatCurrency(chartMaxValue) }}</span>
            <span>{{ formatCurrency(chartMaxValue / 2) }}</span>
            <span>$0</span>
          </div>
          <div class="line-chart" role="img" aria-label="Current value and invested amount over time">
            <svg viewBox="0 0 1000 320" preserveAspectRatio="none" class="line-chart-svg">
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

              <!-- Sparse vertical grid lines only -->
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

              <!-- Interior data points (last gets the callout ring) -->
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

              <!-- Sparse value labels above interior points -->
              <text
                v-for="(pt, i) in sparseLabelPoints"
                :key="`lbl-${i}`"
                class="chart-point-label"
                :x="pt.x"
                :y="pt.y - 14"
                dominant-baseline="auto"
                text-anchor="middle"
              >{{ formatShortAmount(pt.value) }}</text>

              <!-- Circled endpoint callout for final value -->
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

              <!-- Invested endpoint dot -->
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

        <div class="line-chart-footer">
          <div class="line-chart-legend">
            <span class="legend-item">
              <span class="legend-line legend-value"></span>
              Current Value
            </span>
            <span class="legend-item">
              <span class="legend-line legend-invested"></span>
              Invested
            </span>
          </div>
          <div class="line-chart-dates">
            <span>{{ formatShortDate(firstSnapshot?.recordedAt ?? '') }}</span>
            <span>{{ formatShortDate(lastSnapshot?.recordedAt ?? '') }}</span>
          </div>
        </div>
      </div>

      <!-- Side panel -->
      <div class="chart-side-panel">
        <div class="panel-roi">
          <p class="section-label">Portfolio Change</p>
          <span class="panel-roi-number" :class="changeAmount >= 0 ? 'positive' : 'negative'">
            <template v-if="changePercent !== null">
              {{ changePercent >= 0 ? '+' : '' }}{{ changePercent.toFixed(1) }}%
            </template>
            <template v-else>
              {{ changeAmount >= 0 ? '+' : '' }}{{ formatCurrency(changeAmount) }}
            </template>
          </span>
          <span class="panel-roi-period">
            {{ formatShortDate(firstSnapshot?.recordedAt ?? '') }} —
            {{ formatShortDate(lastSnapshot?.recordedAt ?? '') }}
          </span>
        </div>

        <div class="chart-summary-strip" aria-label="Value history summary">
          <div class="summary-pill">
            <span class="summary-label">Latest Value</span>
            <strong>{{ formatCurrency(latestValue) }}</strong>
          </div>
          <div class="summary-pill">
            <span class="summary-label">Invested</span>
            <strong>{{ formatCurrency(latestInvested) }}</strong>
          </div>
          <div class="summary-pill">
            <span class="summary-label">Change</span>
            <strong :class="changeAmount >= 0 ? 'positive' : 'negative'">
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
.value-chart-card {
  position: relative;
  overflow: hidden;
  border-color: var(--border-accent);
  box-shadow: var(--shadow-card);
}

.value-chart-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at top right, var(--accent-gold-glow), transparent 35%),
    linear-gradient(180deg, var(--bg-card-hover), var(--bg-card));
  opacity: 0.8;
  pointer-events: none;
}

.value-chart-card > * {
  position: relative;
  z-index: 1;
}

.chart-card-header {
  margin-bottom: 1.25rem;
}

.chart-card-header h2 {
  margin: 0.15rem 0 0;
  font-size: 1.5rem;
}

/* Main two-column layout: chart left, side panel right */
.chart-main-layout {
  display: flex;
  gap: 1.25rem;
  align-items: flex-start;
}

.chart-area {
  flex: 1;
  min-width: 0;
}

.line-chart-container {
  display: flex;
  gap: 0.75rem;
  min-width: 0;
}

.line-chart-y-axis {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  min-width: 3.75rem;
  padding: 0.35rem 0;
  color: var(--text-muted);
  font-size: 0.7rem;
  text-align: right;
}

.line-chart {
  flex: 1;
  min-width: 0;
  height: 15rem;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  box-shadow: inset 0 1px 0 var(--border-subtle);
}

.line-chart-svg {
  width: 100%;
  height: 100%;
  overflow: visible;
}

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

/* Sparse value labels at data points */
.chart-point-label {
  fill: var(--text-secondary);
  font-size: 0.6rem;
  font-weight: 600;
}

/* Endpoint callout (large circled final value) */
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

.line-chart-footer {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  margin-top: 0.6rem;
  padding-left: 4.5rem;
}

.line-chart-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  color: var(--text-secondary);
  font-size: 0.8rem;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.legend-line {
  display: inline-block;
  width: 1.5rem;
  height: 0.2rem;
  border-radius: var(--radius-full);
}

.legend-value {
  background: var(--accent-gold);
  box-shadow: var(--shadow-glow);
}

.legend-invested {
  background-image: repeating-linear-gradient(
    90deg,
    var(--text-secondary) 0,
    var(--text-secondary) 0.4rem,
    transparent 0.4rem,
    transparent 0.65rem
  );
}

.line-chart-dates {
  display: flex;
  gap: 0.75rem;
  color: var(--text-muted);
  font-size: 0.7rem;
  white-space: nowrap;
}

/* Side panel */
.chart-side-panel {
  width: 10.5rem;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.panel-roi {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.panel-roi-number {
  display: block;
  font-family: 'Cinzel', serif;
  font-size: 1.9rem;
  font-weight: 600;
  line-height: 1.1;
}

.panel-roi-number.positive {
  color: var(--color-positive);
}

.panel-roi-number.negative {
  color: var(--color-negative);
}

.panel-roi-period {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.chart-summary-strip {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.summary-pill {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  min-width: 0;
  padding: 0.6rem 0.75rem;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
}

.summary-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
}

.summary-pill strong {
  overflow: hidden;
  color: var(--text-primary);
  font-size: 0.85rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.summary-pill strong.positive {
  color: var(--color-positive);
}

.summary-pill strong.negative {
  color: var(--color-negative);
}

@media (max-width: 768px) {
  .chart-main-layout {
    flex-direction: column;
  }

  .chart-side-panel {
    width: 100%;
    flex-direction: row;
    flex-wrap: wrap;
    align-items: flex-start;
    gap: 0.75rem;
  }

  .panel-roi {
    flex: 1;
    min-width: 8rem;
  }

  .chart-summary-strip {
    flex: 2;
    min-width: 14rem;
    flex-direction: row;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .summary-pill {
    flex: 1;
    min-width: 6rem;
  }

  .line-chart {
    height: 13rem;
  }
}

@media (max-width: 480px) {
  .line-chart-footer {
    flex-direction: column;
    padding-left: 0;
  }

  .line-chart-dates {
    justify-content: space-between;
  }

  .line-chart-y-axis {
    min-width: 3.25rem;
  }
}
</style>
