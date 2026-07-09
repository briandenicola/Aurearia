<template>
  <section class="stats-section card investment-table-card">
    <div class="investment-table-header">
      <div>
        <p class="section-label">{{ eyebrow }}</p>
        <h2>{{ title }}</h2>
        <p v-if="description" class="table-description">{{ description }}</p>
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
          <span class="summary-label">ROI</span>
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

      <div class="investment-row-list" aria-label="Investment performance rows">
        <article
          v-for="row in displayRows"
          :key="rowKey(row)"
          class="investment-row"
        >
          <div class="row-main">
            <div>
              <h3>{{ formatSegmentLabel(row) }}</h3>
              <p>{{ row.coinCount }} coins</p>
            </div>
            <span class="roi-pill" :class="row.gainLoss >= 0 ? 'positive-bg' : 'negative-bg'">
              {{ row.gainLoss >= 0 ? '+' : '' }}{{ normalizedPct(row).toFixed(1) }}%
            </span>
          </div>

          <div class="value-grid">
            <div>
              <span class="value-label">Invested</span>
              <strong>{{ formatCurrency(row.invested) }}</strong>
            </div>
            <div>
              <span class="value-label">Current</span>
              <strong class="gold">{{ formatCurrency(row.currentValue) }}</strong>
            </div>
            <div>
              <span class="value-label">Gain / Loss</span>
              <strong :class="row.gainLoss >= 0 ? 'positive' : 'negative'">
                {{ row.gainLoss >= 0 ? '+' : '' }}{{ formatCurrency(row.gainLoss) }}
              </strong>
            </div>
          </div>

          <div class="allocation-bars" aria-hidden="true">
            <div class="bar-row">
              <span>Invested</span>
              <div class="bar-track">
                <span class="bar-fill invested" :style="{ width: `${barWidth(row.invested, maxInvested)}%` }"></span>
              </div>
            </div>
            <div class="bar-row">
              <span>Current</span>
              <div class="bar-track">
                <span class="bar-fill current" :style="{ width: `${barWidth(row.currentValue, maxCurrentValue)}%` }"></span>
              </div>
            </div>
          </div>
        </article>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { formatCurrency } from '@/utils/format'
import type { InvestmentBreakdownSegment } from '@/types'

const props = defineProps<{
  title: string
  eyebrow: string
  description?: string
  rows: InvestmentBreakdownSegment[]
}>()

const displayRows = computed(() => props.rows)

const totalCoinCount = computed(() => props.rows.reduce((sum, row) => sum + row.coinCount, 0))

const totals = computed(() => props.rows.reduce(
  (acc, row) => ({
    invested: acc.invested + row.invested,
    currentValue: acc.currentValue + row.currentValue,
    gainLoss: acc.gainLoss + row.gainLoss,
  }),
  { invested: 0, currentValue: 0, gainLoss: 0 },
))

const summaryGainLossPct = computed(() => {
  if (totals.value.invested === 0) return 0
  return (totals.value.gainLoss / totals.value.invested) * 100
})

const maxInvested = computed(() => Math.max(...props.rows.map((row) => row.invested), 0))
const maxCurrentValue = computed(() => Math.max(...props.rows.map((row) => row.currentValue), 0))

const hasMissingValues = computed(() => props.rows.some(
  (row) => row.missingCurrentValueCount > 0 || row.missingPurchasePriceCount > 0,
))

const missingValueSummary = computed(() => {
  const missingPurchase = props.rows.reduce((sum, row) => sum + row.missingPurchasePriceCount, 0)
  const missingCurrent = props.rows.reduce((sum, row) => sum + row.missingCurrentValueCount, 0)
  const parts: string[] = []
  if (missingPurchase > 0) parts.push(`${missingPurchase} missing purchase price${missingPurchase === 1 ? '' : 's'}`)
  if (missingCurrent > 0) parts.push(`${missingCurrent} missing current value${missingCurrent === 1 ? '' : 's'}`)
  return parts.join(' and ')
})

function rowKey(row: InvestmentBreakdownSegment): string {
  return `${row.label}-${row.year ?? 'none'}-${row.month ?? 'none'}`
}

function formatSegmentLabel(row: InvestmentBreakdownSegment): string {
  if (row.year) return `${row.year}`
  return row.label
}

function normalizedPct(row: InvestmentBreakdownSegment): number {
  if (row.gainLossPct !== null) return row.gainLossPct
  if (row.invested === 0) return 0
  return (row.gainLoss / row.invested) * 100
}

function barWidth(value: number, max: number): number {
  if (max <= 0 || value <= 0) return 0
  return Math.max(6, Math.min(100, (value / max) * 100))
}
</script>

<style scoped>
.investment-table-card {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1rem;
}

.investment-table-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.investment-table-header h2 {
  margin: 0.25rem 0 0;
  font-size: 1.2rem;
}

.table-description,
.empty-state p,
.row-main p {
  margin: 0.35rem 0 0;
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.coin-count,
.roi-pill {
  display: inline-flex;
  align-items: center;
  border-radius: var(--radius-full);
  font-size: 0.75rem;
  font-weight: 600;
  white-space: nowrap;
}

.coin-count {
  padding: 0.2rem 0.7rem;
  color: var(--accent-gold);
  background: var(--accent-gold-dim);
  border: 1px solid var(--border-accent);
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.75rem;
}

.summary-pill {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: 0.75rem;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
}

.summary-label,
.value-label,
.bar-row span {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.confidence-callout {
  padding: 0.75rem;
  background: var(--accent-gold-glow);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
}

.confidence-callout p {
  margin: 0.25rem 0 0;
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.investment-row-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.investment-row {
  display: grid;
  grid-template-columns: minmax(9rem, 1fr) minmax(16rem, 1.5fr) minmax(13rem, 1fr);
  gap: 1rem;
  align-items: center;
  padding: 0.75rem;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
}

.row-main {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  align-items: flex-start;
}

.row-main h3 {
  margin: 0;
  color: var(--text-heading);
  font-size: 1.2rem;
}

.roi-pill {
  padding: 0.15rem 0.5rem;
}

.positive-bg {
  color: var(--accent-gold);
  background: var(--accent-gold-dim);
}

.negative-bg {
  color: var(--cat-byzantine);
  background: color-mix(in srgb, var(--cat-byzantine) 18%, transparent);
}

.value-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
}

.value-grid div {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.allocation-bars {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.bar-row {
  display: grid;
  grid-template-columns: 4.5rem 1fr;
  gap: 0.5rem;
  align-items: center;
}

.bar-track {
  height: 0.45rem;
  overflow: hidden;
  background: var(--bg-card);
  border-radius: var(--radius-full);
}

.bar-fill {
  display: block;
  height: 100%;
  border-radius: var(--radius-full);
}

.bar-fill.invested {
  background: var(--accent-bronze);
}

.bar-fill.current {
  background: var(--accent-gold);
}

.gold,
.positive {
  color: var(--accent-gold);
}

.negative {
  color: var(--cat-byzantine);
}

@media (max-width: 900px) {
  .summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .investment-row {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .investment-table-header {
    flex-direction: column;
  }

  .summary-grid,
  .value-grid {
    grid-template-columns: 1fr;
  }
}
</style>
