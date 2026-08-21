<template>
  <CoinDetailSectionPageShell section-title="Value Trend">
    <template #default="{ coin: coinData }">
      <div v-if="coinData.isWishlist || coinData.isSold" class="card p-6 text-center text-text-secondary text-base">
        <p>Value tracking is only available for active coins in your collection.</p>
      </div>
      <div v-else class="flex flex-col gap-5">
        <!-- Chart: requires >= 2 data points (oldest-first, left-to-right) -->
        <div v-if="coinChartData.length >= 2">
          <div class="flex gap-2">
            <div class="flex flex-col justify-between text-label text-text-muted text-right min-w-[60px] py-1">
              <span>{{ formatCurrency(coinChartMax) }}</span>
              <span>{{ formatCurrency(coinChartMax / 2) }}</span>
              <span>$0</span>
            </div>
            <div class="flex-1 h-[200px] bg-surface rounded-sm p-2">
              <svg viewBox="0 0 1000 300" preserveAspectRatio="none" class="w-full h-full">
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
          <div class="flex justify-between text-label text-text-muted mt-1 px-2 pl-[68px]">
            <span>{{ formatShortDate(coinChartData[0]?.date ?? '') }}</span>
            <span>{{ formatShortDate(coinChartData[coinChartData.length - 1]?.date ?? '') }}</span>
          </div>
        </div>

        <!-- Value history table: renders at >= 1 entry regardless of chart gate -->
        <div v-if="valueHistoryTableRows.length > 0" class="flex flex-col gap-2">
          <p class="section-label">Value History</p>
          <div class="rounded-sm border border-border-subtle overflow-hidden">
            <div :class="['overflow-y-auto overflow-x-auto', { 'max-h-[16.5rem]': valueHistoryTableRows.length > 4 }]">
              <table class="w-full text-sm">
                <thead class="sticky top-0 z-10 bg-card">
                  <tr class="border-b border-border-subtle">
                    <th class="px-3 py-2 text-left text-xs font-semibold uppercase tracking-[0.08em] text-text-muted">Date</th>
                    <th class="px-3 py-2 text-right text-xs font-semibold uppercase tracking-[0.08em] text-text-muted">Value</th>
                    <th class="px-3 py-2 text-right text-xs font-semibold uppercase tracking-[0.08em] text-text-muted">Change</th>
                    <th class="px-3 py-2 text-right text-xs font-semibold uppercase tracking-[0.08em] text-text-muted">Source</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="row in valueHistoryTableRows"
                    :key="row.entry.id"
                    class="border-b border-border-subtle last:border-0"
                  >
                    <td class="px-3 py-2 text-text-secondary">{{ formatShortDate(row.entry.recordedAt) }}</td>
                    <td class="px-3 py-2 text-right font-medium text-text-primary">{{ formatCurrency(row.entry.value) }}</td>
                    <td
                      class="px-3 py-2 text-right font-medium"
                      :class="
                        row.change === null
                          ? 'text-text-muted'
                          : row.change >= 0
                            ? 'text-[var(--color-positive)]'
                            : 'text-[var(--color-negative)]'"
                    >
                      {{ row.change === null ? '—' : (row.change >= 0 ? '+' : '') + formatCurrency(row.change) }}
                    </td>
                    <td class="px-3 py-2 text-right">
                      <span class="chip-sm whitespace-nowrap">{{ formatEntrySource(row.entry) }}</span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <!-- No data: neither chart nor table -->
        <div v-if="coinChartData.length < 2 && valueHistoryTableRows.length === 0" class="card p-6 text-center text-text-secondary text-base">
          <p>Not enough data points to chart. Run an AI estimate to start tracking.</p>
        </div>
      </div>
    </template>
  </CoinDetailSectionPageShell>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import CoinDetailSectionPageShell from '@/components/coin/CoinDetailSectionPageShell.vue'
import { getCoinValueHistory } from '@/api/client'
import type { CoinValueHistory } from '@/types'
import { formatCurrency } from '@/utils/format'
import { useCoinDetailContext } from '@/composables/useCoinDetailContext'

const route = useRoute()
const { coin } = useCoinDetailContext()
const coinValueEntries = ref<CoinValueHistory[]>([])

const coinChartData = computed(() => {
  if (!coin.value) return []
  const points: { date: string; value: number }[] = []
  if (coin.value.purchasePrice != null && coin.value.purchaseDate != null) {
    points.push({ date: coin.value.purchaseDate, value: coin.value.purchasePrice })
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

// Value history table rows, newest-first, with delta vs previous chronological entry
const valueHistoryTableRows = computed(() => {
  const sorted = [...coinValueEntries.value].sort(
    (a, b) => new Date(a.recordedAt).getTime() - new Date(b.recordedAt).getTime(),
  )
  return sorted
    .map((entry, i) => ({
      entry,
      change: i === 0 ? null : entry.value - sorted[i - 1]!.value,
    }))
    .reverse()
})

onMounted(async () => {
  const coinId = Number(route.params.id)
  try {
    const res = await getCoinValueHistory(coinId)
    coinValueEntries.value = res.data || []
  } catch {
    coinValueEntries.value = []
  }
})

function formatShortDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: '2-digit' })
}

// Resolve human-readable source label. `source` is additive; fall back to
// confidence for legacy rows that predate the column (D2 inference rule).
function formatEntrySource(entry: CoinValueHistory): string {
  const src = entry.source ?? (entry.confidence === 'manual' || !entry.confidence ? 'manual' : 'ai_scheduled')
  if (src === 'ai_scheduled') return 'AI Scheduled'
  if (src === 'ai_estimate') return 'AI Estimate'
  return 'Manual'
}
</script>
