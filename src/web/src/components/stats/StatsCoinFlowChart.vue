<template>
  <div class="stats-section card flow-chart-card">
    <div class="flow-chart-header">
      <div>
        <p class="section-label">Acquisition Flow</p>
        <h2>Coins Bought by Period, Ruler, Era &amp; Type</h2>
      </div>
      <span v-if="!isLoading && chartCoins.length" class="flow-total-label">
        {{ chartCoins.length }} coins with purchase date
      </span>
    </div>

    <div v-if="isLoading" class="flow-loading">
      <div class="spinner"></div>
    </div>

    <div v-else-if="chartCoins.length < 3" class="flow-empty">
      <p>
        Not enough coins with a purchase date to draw this chart.
        Set a purchase date on your coins to see their acquisition flow.
      </p>
    </div>

    <div v-else class="flow-chart-body">
      <ZoomableSurface aria-label="Zoomable acquisition flow chart. Use the controls, mouse wheel, pinch, drag, or keyboard shortcuts to inspect dense paths.">
        <!-- Column headers -->
        <div class="flow-col-headers" :style="colHeaderStyle">
          <span class="flow-col-label">Purchase Period</span>
          <span class="flow-col-label">Ruler</span>
          <span class="flow-col-label">Era</span>
          <span class="flow-col-label">Type</span>
        </div>

        <!-- SVG alluvial chart: Purchase Period → Ruler → Era → Type -->
        <svg
          :viewBox="`0 0 ${SVG_W} ${SVG_H}`"
          preserveAspectRatio="xMidYMid meet"
          class="flow-chart-svg"
          role="img"
          aria-label="Alluvial flow chart of acquired coins by purchase period, ruler, era, and type"
        >
          <!-- Period → Ruler flows -->
          <path
            v-for="(band, i) in periodRulerFlows"
            :key="`pr-${i}`"
            class="sankey-flow"
            :d="flowPath(band, COL_X[0], COL_X[1])"
            :fill="band.color"
            opacity="0.45"
          />

          <!-- Ruler → Era flows -->
          <path
            v-for="(band, i) in rulerEraFlows"
            :key="`re-${i}`"
            class="sankey-flow"
            :d="flowPath(band, COL_X[1], COL_X[2])"
            :fill="band.color"
            opacity="0.45"
          />

          <!-- Era → Type flows -->
          <path
            v-for="(band, i) in eraTypeFlows"
            :key="`et-${i}`"
            class="sankey-flow"
            :d="flowPath(band, COL_X[2], COL_X[3])"
            :fill="band.color"
            opacity="0.45"
          />

          <!-- Period nodes (leftmost) -->
          <g v-for="node in periodNodes" :key="`period-${node.key}`">
            <rect
              class="sankey-node"
              :x="COL_X[0]"
              :y="node.y"
              :width="NODE_W"
              :height="node.height"
              :fill="node.color"
              rx="2"
            />
            <text
              class="sankey-label sankey-label-left"
              :x="COL_X[0] - 8"
              :y="node.y + node.height / 2"
              dominant-baseline="middle"
              text-anchor="end"
            >{{ node.label }} ({{ node.count }})</text>
          </g>

          <!-- Ruler nodes -->
          <g v-for="node in rulerNodes" :key="`ruler-${node.key}`">
            <rect
              class="sankey-node"
              :x="COL_X[1]"
              :y="node.y"
              :width="NODE_W"
              :height="node.height"
              :fill="node.color"
              rx="2"
            />
            <text
              class="sankey-label sankey-label-center"
              :x="COL_X[1] + NODE_W / 2"
              :y="node.y + node.height / 2"
              dominant-baseline="middle"
              text-anchor="middle"
            >{{ node.label }}</text>
          </g>

          <!-- Era nodes -->
          <g v-for="node in eraNodes" :key="`era-${node.key}`">
            <rect
              class="sankey-node"
              :x="COL_X[2]"
              :y="node.y"
              :width="NODE_W"
              :height="node.height"
              :fill="node.color"
              rx="2"
            />
            <text
              class="sankey-label sankey-label-center"
              :x="COL_X[2] + NODE_W / 2"
              :y="node.y + node.height / 2"
              dominant-baseline="middle"
              text-anchor="middle"
            >{{ node.label }}</text>
          </g>

          <!-- Type nodes (rightmost) -->
          <g v-for="node in typeNodes" :key="`type-${node.key}`">
            <rect
              class="sankey-node"
              :x="COL_X[3]"
              :y="node.y"
              :width="NODE_W"
              :height="node.height"
              :fill="node.color"
              rx="2"
            />
            <text
              class="sankey-label sankey-label-right"
              :x="COL_X[3] + NODE_W + 8"
              :y="node.y + node.height / 2"
              dominant-baseline="middle"
              text-anchor="start"
            >{{ node.label }} ({{ node.count }})</text>
          </g>
        </svg>
      </ZoomableSurface>

      <p class="flow-footnote">
        Only coins with a recorded purchase date are shown. Ruler and type columns show top
        {{ TOP_N }} by count; remaining items are grouped as "Other Rulers" / "Other Types".
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { getCoins } from '@/api/client'
import ZoomableSurface from '@/components/ZoomableSurface.vue'
import type { Coin } from '@/types'

interface SankeyNode {
  key: string
  label: string
  count: number
  y: number
  height: number
  color: string
}

interface SankeyBand {
  sourceKey: string
  targetKey: string
  count: number
  sourceY: number
  targetY: number
  height: number
  color: string
}

const SVG_W = 760
const SVG_H = 380
const NODE_W = 14
const NODE_GAP = 8
const CHART_TOP = 10
const CHART_H = SVG_H - CHART_TOP
const TOP_N = 8

// Four-column layout: Period | Ruler | Era | Type
const COL_X = [75, 245, 415, 585] as const

const PERIOD_PALETTE = [
  'var(--accent-gold)',
  'var(--accent-bronze)',
  'var(--cat-roman)',
  'var(--cat-greek)',
  'var(--cat-byzantine)',
  'var(--cat-modern)',
]

const ERA_COLORS: Record<string, string> = {
  ancient: 'var(--accent-gold)',
  medieval: 'var(--accent-bronze)',
  modern: 'var(--cat-modern)',
}

function getEraColor(key: string): string {
  return ERA_COLORS[key.toLowerCase()] ?? 'var(--text-secondary)'
}

function getRulerColor(key: string): string {
  return key === 'Other Rulers' ? 'var(--text-muted)' : 'var(--accent-bronze)'
}

function getTypeColor(key: string): string {
  return key === 'Other Types' ? 'var(--text-muted)' : 'var(--accent-gold)'
}

function getScale(total: number, maxNodes: number): number {
  if (!total) return 1
  const usable = CHART_H - (maxNodes - 1) * NODE_GAP
  return usable / total
}

const isLoading = ref(true)
const coins = ref<Coin[]>([])

async function loadCoins() {
  isLoading.value = true
  try {
    const limit = 100
    let page = 1
    const all: Coin[] = []
    while (true) {
      const res = await getCoins({
        wishlist: 'false',
        sold: 'false',
        page,
        limit,
        sort: 'purchase_date',
        order: 'asc',
      })
      all.push(...res.data.coins)
      if (all.length >= res.data.total || !res.data.coins.length) break
      page++
    }
    coins.value = all
  } finally {
    isLoading.value = false
  }
}

onMounted(loadCoins)

// Only coins with a purchaseDate participate in this chart
const chartCoins = computed(() => coins.value.filter((c) => !!c.purchaseDate))

// Return top-N keys by frequency; everything else maps to otherLabel
function applyTopN(rawValues: string[], n: number, otherLabel: string): string[] {
  const counts = new Map<string, number>()
  for (const v of rawValues) counts.set(v, (counts.get(v) ?? 0) + 1)
  const topKeys = new Set(
    [...counts.entries()].sort((a, b) => b[1] - a[1]).slice(0, n).map(([k]) => k),
  )
  return rawValues.map((v) => (topKeys.has(v) ? v : otherLabel))
}

interface MappedRow {
  period: string
  ruler: string
  era: string
  type: string
}

// Build normalized rows with top-N grouping applied to ruler and type
const mappedCoins = computed<MappedRow[]>(() => {
  const base = chartCoins.value
  if (!base.length) return []

  const rulerRaw = base.map((c) => c.ruler?.trim() || 'Unknown Ruler')
  // Type: denomination preferred, then category, then fallback
  const typeRaw = base.map((c) => c.denomination?.trim() || c.category?.trim() || 'Unknown Type')

  const rulerMapped = applyTopN(rulerRaw, TOP_N, 'Other Rulers')
  const typeMapped = applyTopN(typeRaw, TOP_N, 'Other Types')

  return base.map((coin, i) => ({
    period: (coin.purchaseDate ?? '').slice(0, 4),
    ruler: rulerMapped[i]!,
    era: coin.era?.trim() || 'Unknown Era',
    type: typeMapped[i]!,
  }))
})

type RowField = keyof MappedRow

function countByField(field: RowField): Map<string, number> {
  const map = new Map<string, number>()
  for (const row of mappedCoins.value) {
    const key = row[field]
    map.set(key, (map.get(key) ?? 0) + 1)
  }
  return map
}

function crossTabFields(
  fieldA: RowField,
  fieldB: RowField,
): Map<string, Map<string, number>> {
  const map = new Map<string, Map<string, number>>()
  for (const row of mappedCoins.value) {
    const a = row[fieldA]
    const b = row[fieldB]
    if (!map.has(a)) map.set(a, new Map())
    map.get(a)!.set(b, (map.get(a)!.get(b) ?? 0) + 1)
  }
  return map
}

function buildNodes(
  counts: Map<string, number>,
  scale: number,
  colorFn: (k: string, i: number) => string,
): SankeyNode[] {
  const entries = [...counts.entries()].filter(([, c]) => c > 0).sort((a, b) => b[1] - a[1])
  let y = CHART_TOP
  return entries.map(([key, count], i) => {
    const height = Math.max(4, count * scale)
    const node: SankeyNode = { key, label: key, count, y, height, color: colorFn(key, i) }
    y += height + NODE_GAP
    return node
  })
}

function buildFlows(
  sourceNodes: SankeyNode[],
  targetNodes: SankeyNode[],
  crossTabMap: Map<string, Map<string, number>>,
  scale: number,
): SankeyBand[] {
  const srcOffsets = new Map(sourceNodes.map((n) => [n.key, 0]))
  const tgtOffsets = new Map(targetNodes.map((n) => [n.key, 0]))
  const bands: SankeyBand[] = []

  for (const src of sourceNodes) {
    const targets = crossTabMap.get(src.key)
    if (!targets) continue
    for (const tgt of targetNodes) {
      const count = targets.get(tgt.key) ?? 0
      if (!count) continue
      const height = Math.max(1, count * scale)
      const srcOff = srcOffsets.get(src.key) ?? 0
      const tgtOff = tgtOffsets.get(tgt.key) ?? 0
      bands.push({
        sourceKey: src.key,
        targetKey: tgt.key,
        count,
        sourceY: src.y + srcOff,
        targetY: tgt.y + tgtOff,
        height,
        color: src.color,
      })
      srcOffsets.set(src.key, srcOff + height)
      tgtOffsets.set(tgt.key, tgtOff + height)
    }
  }
  return bands
}

function flowPath(band: SankeyBand, x0: number, x1: number): string {
  const sx = x0 + NODE_W
  const tx = x1
  const mid = (sx + tx) / 2
  const t0 = band.sourceY
  const b0 = band.sourceY + band.height
  const t1 = band.targetY
  const b1 = band.targetY + band.height
  return `M ${sx} ${t0} C ${mid} ${t0} ${mid} ${t1} ${tx} ${t1} L ${tx} ${b1} C ${mid} ${b1} ${mid} ${b0} ${sx} ${b0} Z`
}

const periodCounts = computed(() => countByField('period'))
const rulerCounts = computed(() => countByField('ruler'))
const eraCounts = computed(() => countByField('era'))
const typeCounts = computed(() => countByField('type'))

const maxNodes = computed(() =>
  Math.max(
    periodCounts.value.size,
    rulerCounts.value.size,
    eraCounts.value.size,
    typeCounts.value.size,
  ),
)
const scale = computed(() => getScale(chartCoins.value.length, maxNodes.value))

const periodNodes = computed(() =>
  buildNodes(
    periodCounts.value,
    scale.value,
    (_k, i) => PERIOD_PALETTE[i % PERIOD_PALETTE.length]!,
  ),
)
const rulerNodes = computed(() =>
  buildNodes(rulerCounts.value, scale.value, (k) => getRulerColor(k)),
)
const eraNodes = computed(() =>
  buildNodes(eraCounts.value, scale.value, (k) => getEraColor(k)),
)
const typeNodes = computed(() =>
  buildNodes(typeCounts.value, scale.value, (k) => getTypeColor(k)),
)

const periodRulerTab = computed(() => crossTabFields('period', 'ruler'))
const rulerEraTab = computed(() => crossTabFields('ruler', 'era'))
const eraTypeTab = computed(() => crossTabFields('era', 'type'))

const periodRulerFlows = computed(() =>
  buildFlows(periodNodes.value, rulerNodes.value, periodRulerTab.value, scale.value),
)
const rulerEraFlows = computed(() =>
  buildFlows(rulerNodes.value, eraNodes.value, rulerEraTab.value, scale.value),
)
const eraTypeFlows = computed(() =>
  buildFlows(eraNodes.value, typeNodes.value, eraTypeTab.value, scale.value),
)

// Column header grid proportional to SVG column positions
const colHeaderStyle = computed(() => ({
  gridTemplateColumns: `${COL_X[0]}px 1fr 1fr ${SVG_W - COL_X[3] - NODE_W}px`,
}))
</script>

<style scoped>
.flow-chart-card {
  position: relative;
  overflow: hidden;
  border-color: var(--border-subtle);
  box-shadow: var(--shadow-card);
}

.flow-chart-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
}

.flow-chart-header h2 {
  margin: 0.15rem 0 0;
  font-size: 1.4rem;
}

.flow-total-label {
  font-size: 0.8rem;
  color: var(--text-muted);
  align-self: flex-end;
}

.flow-loading {
  display: flex;
  justify-content: center;
  padding: 3rem 0;
}

.flow-empty {
  padding: 2rem 0;
  color: var(--text-muted);
  font-size: 0.9rem;
  text-align: center;
}

.flow-chart-body {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.flow-col-headers {
  display: grid;
  padding: 0 0 0.35rem;
}

.flow-col-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  text-align: center;
}

.flow-col-label:first-child {
  text-align: left;
  padding-left: 0;
}

.flow-col-label:last-child {
  text-align: right;
  padding-right: 0;
}

.flow-chart-svg {
  width: 100%;
  height: auto;
  overflow: visible;
}

/* SVG element styles */
.sankey-node {
  vector-effect: non-scaling-stroke;
}

.sankey-flow {
  vector-effect: non-scaling-stroke;
}

.sankey-label {
  font-size: 0.7rem;
  fill: var(--text-secondary);
  font-weight: 500;
}

.flow-footnote {
  margin-top: 0.5rem;
  font-size: 0.75rem;
  color: var(--text-muted);
  font-style: italic;
}

@media (max-width: 480px) {
  .flow-chart-header {
    flex-direction: column;
  }
}
</style>
