<template>
  <section class="stats-section card investment-chart-card">
    <div class="investment-chart-header">
      <div>
        <p class="section-label">{{ eyebrow }}</p>
        <h2>{{ title }}</h2>
        <p v-if="description" class="chart-description">{{ description }}</p>
      </div>
      <span v-if="rows.length" class="coin-count">{{ totalCoinCount }} coins</span>
    </div>

    <div v-if="!rows.length" class="empty-state">
      <p>No investment data is available for this breakdown.</p>
    </div>

    <template v-else>
      <div class="summary-grid" aria-label="Investment summary">
        <div class="summary-pill">
          <span class="summary-label">Invested</span>
          <strong>{{ formatCurrency(totals.invested) }}</strong>
        </div>
        <div class="summary-pill">
          <span class="summary-label">Current Value</span>
          <strong class="gold">{{ formatCurrency(totals.currentValue) }}</strong>
        </div>
        <div class="summary-pill">
          <span class="summary-label">Gain / Loss</span>
          <strong :class="totals.gainLoss >= 0 ? 'positive' : 'negative'">
            {{ totals.gainLoss >= 0 ? '+' : '' }}{{ formatCurrency(totals.gainLoss) }}
          </strong>
        </div>
        <div class="summary-pill">
          <span class="summary-label">Return</span>
          <strong :class="summaryGainLossPct >= 0 ? 'positive' : 'negative'">
            {{ summaryGainLossPct >= 0 ? '+' : '' }}{{ summaryGainLossPct.toFixed(1) }}%
          </strong>
        </div>
      </div>

      <div v-if="hasMissingValues" class="confidence-callout" role="note">
        <span class="section-label">Confidence</span>
        <p>
          {{ missingValueSummary }}. Totals use available prices and values, so returns may change as records are completed.
        </p>
      </div>

      <div class="chart-legend" aria-label="Chart legend">
        <span class="legend-item"><span class="legend-swatch legend-invested"></span> Invested flow</span>
        <span class="legend-item"><span class="legend-swatch legend-current"></span> Current value marker</span>
      </div>

      <ZoomableSurface
        class="flow-layout"
        :aria-label="`Zoomable ${title} investment flow chart. Use controls, wheel, pinch, drag, or keyboard shortcuts to inspect segments.`"
      >
        <svg
          :viewBox="`0 0 ${SVG_W} ${svgHeight}`"
          preserveAspectRatio="xMidYMid meet"
          class="investment-flow-svg"
          role="img"
          :aria-label="`${title} flow chart showing invested amount by segment`"
        >
          <defs>
            <linearGradient :id="gradientId" x1="0" x2="1" y1="0" y2="0">
              <stop offset="0%" stop-color="var(--accent-gold)" stop-opacity="0.45" />
              <stop offset="100%" stop-color="var(--accent-bronze)" stop-opacity="0.25" />
            </linearGradient>
          </defs>

          <rect
            class="total-node"
            :x="TOTAL_X"
            :y="totalNode.y"
            :width="NODE_W"
            :height="totalNode.height"
          />
          <text
            class="node-label node-label-left"
            :x="TOTAL_X - 10"
            :y="totalNode.y + totalNode.height / 2"
            dominant-baseline="middle"
            text-anchor="end"
          >
            Total invested
          </text>

          <path
            v-for="band in flowBands"
            :key="band.key"
            class="flow-band"
            :d="flowPath(band)"
            :fill="`url(#${gradientId})`"
          />

          <g v-for="node in segmentNodes" :key="node.key">
            <rect
              class="segment-node"
              :x="SEGMENT_X"
              :y="node.y"
              :width="NODE_W"
              :height="node.height"
              :fill="node.color"
            />
            <line
              class="current-value-marker"
              :x1="SEGMENT_X + NODE_W + 8"
              :x2="SEGMENT_X + NODE_W + 8 + currentMarkerWidth(node.row)"
              :y1="node.y + node.height / 2"
              :y2="node.y + node.height / 2"
            />
            <text
              class="node-label node-label-right"
              :x="SEGMENT_X + NODE_W + 14"
              :y="node.y + Math.max(node.height / 2 - 7, 0)"
              dominant-baseline="middle"
            >
              {{ node.label }}
            </text>
            <text
              class="node-value"
              :x="SEGMENT_X + NODE_W + 14"
              :y="node.y + node.height / 2 + 9"
              dominant-baseline="middle"
            >
              {{ formatCurrency(node.row.invested) }} invested · {{ formatCurrency(node.row.currentValue) }} current
            </text>
          </g>
        </svg>
      </ZoomableSurface>

      <div class="mobile-aggregate-summary" aria-label="Investment aggregate summary">
        Invested: {{ formatCurrency(totals.invested) }} · Current: {{ formatCurrency(totals.currentValue) }} ·
        <span :class="totals.gainLoss >= 0 ? 'positive' : 'negative'">
          Gain/Loss: {{ totals.gainLoss >= 0 ? '+' : '' }}{{ formatCurrency(totals.gainLoss) }} ({{ summaryGainLossPct >= 0 ? '+' : '' }}{{ summaryGainLossPct.toFixed(1) }}%)
        </span>
      </div>

      <div class="segment-list" aria-label="Investment breakdown segments">
        <article
          v-for="row in displayRows"
          :key="rowKey(row)"
          class="segment-card"
        >
          <div class="segment-card-heading">
            <span class="segment-name">{{ formatSegmentLabel(row) }}</span>
            <span class="chip-sm">{{ row.coinCount }} coins</span>
          </div>
          <div class="segment-values">
            <span><strong>{{ formatCurrency(row.invested) }}</strong> invested</span>
            <span><strong class="gold">{{ formatCurrency(row.currentValue) }}</strong> current</span>
            <span :class="row.gainLoss >= 0 ? 'positive' : 'negative'">
              {{ row.gainLoss >= 0 ? '+' : '' }}{{ formatCurrency(row.gainLoss) }}
              <template v-if="row.gainLossPct !== null">
                ({{ row.gainLossPct >= 0 ? '+' : '' }}{{ row.gainLossPct.toFixed(1) }}%)
              </template>
            </span>
          </div>
        </article>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ZoomableSurface from '@/components/ZoomableSurface.vue'
import { formatCurrency } from '@/utils/format'
import type { InvestmentBreakdownSegment } from '@/types'

interface SegmentNode {
  key: string
  label: string
  row: InvestmentBreakdownSegment
  y: number
  height: number
  sourceY: number
  color: string
}

interface FlowBand {
  key: string
  y0: number
  y1: number
  height: number
}

const props = defineProps<{
  title: string
  eyebrow: string
  description?: string
  rows: InvestmentBreakdownSegment[]
}>()

const SVG_W = 760
const TOTAL_X = 95
const SEGMENT_X = 405
const NODE_W = 16
const CHART_TOP = 24
const NODE_GAP = 10
const MIN_NODE_H = 10
const PALETTE = [
  'var(--accent-gold)',
  'var(--accent-bronze)',
  'var(--cat-roman)',
  'var(--cat-greek)',
  'var(--cat-byzantine)',
  'var(--cat-modern)',
  'var(--mat-silver)',
  'var(--mat-bronze)',
]
const MATERIAL_COLORS: Record<string, string> = {
  gold: 'var(--mat-gold)',
  silver: 'var(--mat-silver)',
  bronze: 'var(--mat-bronze)',
  copper: 'var(--accent-bronze)',
  electrum: 'var(--accent-gold)',
  other: 'var(--text-secondary)',
}
const MONTH_LABELS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'] as const

const gradientId = computed(() => `investmentFlowGradient-${props.title.replace(/\W+/g, '-')}`)
const displayRows = computed(() =>
  [...props.rows]
    .filter((row) => row.coinCount > 0)
    .sort((a, b) => {
      if (a.year == null && b.year != null) return 1
      if (a.year != null && b.year == null) return -1
      if (a.year != null && b.year != null && a.year !== b.year) return a.year - b.year
      if (a.month == null && b.month != null) return 1
      if (a.month != null && b.month == null) return -1
      if (a.month != null && b.month != null && a.month !== b.month) return a.month - b.month
      return b.invested - a.invested
    }),
)
const totalCoinCount = computed(() => displayRows.value.reduce((sum, row) => sum + row.coinCount, 0))
const totals = computed(() =>
  displayRows.value.reduce(
    (sum, row) => ({
      invested: sum.invested + row.invested,
      currentValue: sum.currentValue + row.currentValue,
      gainLoss: sum.gainLoss + row.gainLoss,
      missingCurrentValueCount: sum.missingCurrentValueCount + row.missingCurrentValueCount,
      missingPurchasePriceCount: sum.missingPurchasePriceCount + row.missingPurchasePriceCount,
    }),
    { invested: 0, currentValue: 0, gainLoss: 0, missingCurrentValueCount: 0, missingPurchasePriceCount: 0 },
  ),
)
const summaryGainLossPct = computed(() => {
  if (!totals.value.invested) return 0
  return (totals.value.gainLoss / totals.value.invested) * 100
})
const hasMissingValues = computed(
  () => totals.value.missingCurrentValueCount > 0 || totals.value.missingPurchasePriceCount > 0,
)
const missingValueSummary = computed(() => {
  const parts: string[] = []
  if (totals.value.missingPurchasePriceCount) {
    parts.push(`${totals.value.missingPurchasePriceCount} missing purchase price`)
  }
  if (totals.value.missingCurrentValueCount) {
    parts.push(`${totals.value.missingCurrentValueCount} missing current value`)
  }
  return parts.join(' and ')
})
const maxCurrentValue = computed(() => Math.max(1, ...displayRows.value.map((row) => row.currentValue)))
const svgHeight = computed(() => Math.max(240, CHART_TOP * 2 + displayRows.value.length * (MIN_NODE_H + NODE_GAP) + 90))
const chartHeight = computed(() => svgHeight.value - CHART_TOP * 2)
const investedScale = computed(() => {
  const gapTotal = Math.max(0, displayRows.value.length - 1) * NODE_GAP
  const available = Math.max(80, chartHeight.value - gapTotal)
  return totals.value.invested ? available / totals.value.invested : 1
})
const totalNode = computed(() => ({
  y: CHART_TOP,
  height: Math.max(80, displayRows.value.reduce((sum, row) => sum + Math.max(MIN_NODE_H, row.invested * investedScale.value), 0)),
}))
const segmentNodes = computed<SegmentNode[]>(() => {
  let y = CHART_TOP
  let sourceY = totalNode.value.y
  return displayRows.value.map((row, index) => {
    const height = Math.max(MIN_NODE_H, row.invested * investedScale.value)
    const node: SegmentNode = {
      key: rowKey(row),
      label: formatSegmentLabel(row),
      row,
      y,
      height,
      sourceY,
      color: colorForRow(row, index),
    }
    y += height + NODE_GAP
    sourceY += height
    return node
  })
})
const flowBands = computed<FlowBand[]>(() =>
  segmentNodes.value.map((node) => ({
    key: node.key,
    y0: node.sourceY,
    y1: node.y,
    height: node.height,
  })),
)

function rowKey(row: InvestmentBreakdownSegment): string {
  return `${row.year ?? 'none'}-${row.month ?? 'none'}-${row.label}`
}

function formatSegmentLabel(row: InvestmentBreakdownSegment): string {
  if (row.year != null && row.month != null) {
    const month = MONTH_LABELS[row.month - 1] ?? `${row.month}`
    return `${row.year} ${month}`
  }
  if (row.year != null) return `${row.year}`
  return row.label || 'Unspecified'
}

function colorForRow(row: InvestmentBreakdownSegment, index: number): string {
  return MATERIAL_COLORS[row.label.toLowerCase()] ?? PALETTE[index % PALETTE.length]!
}

function currentMarkerWidth(row: InvestmentBreakdownSegment): number {
  return Math.max(20, (row.currentValue / maxCurrentValue.value) * 170)
}

function flowPath(band: FlowBand): string {
  const sx = TOTAL_X + NODE_W
  const tx = SEGMENT_X
  const mid = (sx + tx) / 2
  const sourceTop = band.y0
  const sourceBottom = band.y0 + band.height
  const targetTop = band.y1
  const targetBottom = band.y1 + band.height
  return `M ${sx} ${sourceTop} C ${mid} ${sourceTop} ${mid} ${targetTop} ${tx} ${targetTop} L ${tx} ${targetBottom} C ${mid} ${targetBottom} ${mid} ${sourceBottom} ${sx} ${sourceBottom} Z`
}
</script>

<style scoped>
.investment-chart-card {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  overflow: hidden;
}

.investment-chart-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}

.investment-chart-header h2 {
  margin: 0.15rem 0 0;
  font-size: 1.4rem;
}

.chart-description {
  margin: 0.35rem 0 0;
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.coin-count {
  color: var(--text-muted);
  font-size: 0.8rem;
  white-space: nowrap;
}

.empty-state {
  padding: 2rem 0;
  color: var(--text-muted);
  text-align: center;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
  gap: 0.75rem;
}

.summary-pill {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  padding: 0.75rem;
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

.gold {
  color: var(--accent-gold);
}

.positive {
  color: var(--color-positive);
}

.negative {
  color: var(--color-negative);
}

.confidence-callout {
  padding: 0.75rem;
  background: var(--accent-gold-glow);
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-sm);
}

.confidence-callout p {
  margin: 0.25rem 0 0;
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.chart-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  color: var(--text-secondary);
  font-size: 0.8rem;
}

.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.legend-swatch {
  width: 1.5rem;
  height: 0.25rem;
  border-radius: var(--radius-full);
}

.legend-invested {
  background: var(--accent-gold);
}

.legend-current {
  background: var(--text-secondary);
}

.investment-flow-svg {
  width: 100%;
  min-width: 42rem;
  height: auto;
}

.total-node,
.segment-node {
  vector-effect: non-scaling-stroke;
}

.total-node {
  fill: var(--accent-gold);
}

.flow-band {
  vector-effect: non-scaling-stroke;
}

.current-value-marker {
  stroke: var(--text-secondary);
  stroke-width: 4;
  stroke-linecap: round;
  vector-effect: non-scaling-stroke;
}

.node-label {
  fill: var(--text-secondary);
  font-size: 0.7rem;
  font-weight: 600;
}

.node-value {
  fill: var(--text-muted);
  font-size: 0.62rem;
}

.segment-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
  gap: 0.75rem;
}

.segment-card {
  padding: 0.75rem;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
}

.segment-card-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.segment-name {
  color: var(--text-primary);
  font-weight: 600;
}

.segment-values {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  color: var(--text-secondary);
  font-size: 0.8rem;
}

.mobile-aggregate-summary {
  display: none;
}

@media (max-width: 768px) {
  .investment-chart-header {
    flex-direction: column;
  }

  .investment-flow-svg {
    min-width: 34rem;
  }

  .segment-list {
    display: none;
  }

  .mobile-aggregate-summary {
    display: block;
    padding: 0.75rem;
    background: var(--bg-input);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    font-size: 0.85rem;
    text-align: center;
  }
}
</style>
