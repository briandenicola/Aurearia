<template>
  <section class="stats-section card flex flex-col gap-4 p-4">
    <div class="flex flex-col gap-4 min-[641px]:flex-row min-[641px]:items-start min-[641px]:justify-between">
      <div>
        <p class="section-label">{{ eyebrow }}</p>
        <h2 class="mt-1">{{ title }}</h2>
        <p v-if="description" class="mt-[0.35rem] text-body text-text-secondary">{{ description }}</p>
      </div>
      <span v-if="rows.length" class="inline-flex whitespace-nowrap rounded-full border border-border-accent bg-gold-dim px-[0.7rem] py-[0.2rem] text-sm font-semibold text-gold">{{ totalCoinCount }} coins</span>
    </div>

    <div v-if="!rows.length" class="empty-state">
      <p>No investment data is available for this breakdown.</p>
    </div>

    <template v-else>
      <div class="grid gap-3 min-[641px]:grid-cols-2 xl:grid-cols-4" aria-label="Investment summary">
        <div class="flex flex-col gap-1 rounded-sm border border-border-subtle bg-input p-3">
          <span class="text-sm text-text-muted">Invested</span>
          <strong class="text-text-primary">{{ formatCurrency(totals.invested) }}</strong>
        </div>
        <div class="flex flex-col gap-1 rounded-sm border border-border-subtle bg-input p-3">
          <span class="text-sm text-text-muted">Current Value</span>
          <strong class="text-gold">{{ formatCurrency(totals.currentValue) }}</strong>
        </div>
        <div class="flex flex-col gap-1 rounded-sm border border-border-subtle bg-input p-3">
          <span class="text-sm text-text-muted">Gain / Loss</span>
          <strong :class="totals.gainLoss >= 0 ? 'text-gold' : 'text-byzantine'">
            {{ totals.gainLoss >= 0 ? '+' : '' }}{{ formatCurrency(totals.gainLoss) }}
          </strong>
        </div>
        <div class="flex flex-col gap-1 rounded-sm border border-border-subtle bg-input p-3">
          <span class="text-sm text-text-muted">ROI</span>
          <strong :class="summaryGainLossPct >= 0 ? 'text-gold' : 'text-byzantine'">
            {{ summaryGainLossPct >= 0 ? '+' : '' }}{{ summaryGainLossPct.toFixed(1) }}%
          </strong>
        </div>
      </div>

      <div v-if="hasMissingValues" class="rounded-sm border border-border-subtle bg-gold-glow p-3" role="note">
        <span class="section-label">Confidence</span>
        <p class="mt-1 text-body text-text-secondary">
          {{ missingValueSummary }}. Totals use available prices and values, so returns may change as records are completed.
        </p>
      </div>

      <div class="flex flex-col gap-3" aria-label="Investment performance rows">
        <article
          v-for="row in displayRows"
          :key="rowKey(row)"
          class="grid gap-4 rounded-sm border border-border-subtle bg-input p-3 min-[641px]:[grid-template-columns:minmax(9rem,1fr)_minmax(16rem,1.5fr)_minmax(13rem,1fr)] min-[641px]:items-center"
        >
          <div class="flex items-start justify-between gap-3">
            <div>
              <h3 class="m-0 text-lg text-heading">{{ formatSegmentLabel(row) }}</h3>
              <p class="mt-[0.35rem] text-body text-text-secondary">{{ row.coinCount }} coins</p>
            </div>
            <span class="inline-flex items-center rounded-full px-2 py-[0.15rem] text-sm font-semibold whitespace-nowrap" :class="row.gainLoss >= 0 ? 'bg-gold-dim text-gold' : 'bg-[color-mix(in_srgb,var(--color-byzantine)_18%,transparent)] text-byzantine'">
              {{ row.gainLoss >= 0 ? '+' : '' }}{{ normalizedPct(row).toFixed(1) }}%
            </span>
          </div>

          <div class="grid gap-3 max-[640px]:grid-cols-1 min-[641px]:grid-cols-3">
            <div class="flex flex-col gap-1">
              <span class="text-sm text-text-muted">Invested</span>
              <strong class="text-text-primary">{{ formatCurrency(row.invested) }}</strong>
            </div>
            <div class="flex flex-col gap-1">
              <span class="text-sm text-text-muted">Current</span>
              <strong class="text-gold">{{ formatCurrency(row.currentValue) }}</strong>
            </div>
            <div class="flex flex-col gap-1">
              <span class="text-sm text-text-muted">Gain / Loss</span>
              <strong :class="row.gainLoss >= 0 ? 'text-gold' : 'text-byzantine'">
                {{ row.gainLoss >= 0 ? '+' : '' }}{{ formatCurrency(row.gainLoss) }}
              </strong>
            </div>
          </div>

          <div class="flex flex-col gap-2" aria-hidden="true">
            <div class="grid items-center gap-2 [grid-template-columns:4.5rem_1fr]">
              <span class="text-sm text-text-muted">Invested</span>
              <div class="h-[0.45rem] overflow-hidden rounded-full bg-card">
                <span class="block h-full rounded-full bg-bronze" :style="{ width: `${barWidth(row.invested, maxInvested)}%` }"></span>
              </div>
            </div>
            <div class="grid items-center gap-2 [grid-template-columns:4.5rem_1fr]">
              <span class="text-sm text-text-muted">Current</span>
              <div class="h-[0.45rem] overflow-hidden rounded-full bg-card">
                <span class="block h-full rounded-full bg-gold" :style="{ width: `${barWidth(row.currentValue, maxCurrentValue)}%` }"></span>
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
