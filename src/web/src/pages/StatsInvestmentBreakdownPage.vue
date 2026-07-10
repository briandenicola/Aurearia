<template>
  <PullToRefresh :on-refresh="handleRefresh">
    <div class="container flex flex-col gap-6">
      <header class="page-header flex flex-nowrap items-center justify-between gap-4">
        <div>
          <p class="section-label">Collection Insights</p>
          <h1>Investment Breakdown</h1>
          <p class="mt-[0.35rem] text-base text-text-secondary">See valuation movement, acquisition-year performance, and material allocation.</p>
        </div>
        <router-link
          class="inline-flex shrink-0 items-center justify-center rounded-sm border border-border-subtle bg-transparent p-[0.4rem] text-text-secondary transition hover:border-border-accent hover:bg-gold-glow hover:text-gold"
          to="/stats"
          aria-label="Back to Stats"
        >
          <ArrowLeft :size="20" />
        </router-link>
      </header>

      <div v-if="isLoading" class="loading-overlay">
        <div class="spinner"></div>
      </div>

      <div v-else-if="errorMessage" class="card flex flex-col gap-4 text-text-secondary min-[641px]:flex-row min-[641px]:items-center min-[641px]:justify-between" role="alert">
        <p class="m-0">{{ errorMessage }}</p>
        <button class="btn btn-sm btn-secondary" type="button" @click="handleRefresh">Try Again</button>
      </div>

      <template v-else>
        <section class="grid gap-4 min-[641px]:grid-cols-3" aria-label="Investment valuation highlights">
          <article class="card flex flex-col gap-3 p-4">
            <div>
              <p class="section-label">Top Increases</p>
              <h2 class="mt-1">Biggest Value Gains</h2>
            </div>
            <ol v-if="topIncreases.length" class="m-0 flex list-none flex-col gap-3 p-0">
              <li v-for="coin in topIncreases" :key="coin.coinId" class="flex flex-col gap-[0.35rem] rounded-sm border border-border-subtle bg-input p-3">
                <router-link class="font-semibold text-gold hover:underline" :to="`/coin/${coin.coinId}`">{{ coin.name }}</router-link>
                <div class="flex flex-wrap items-center gap-[0.35rem] text-body text-text-secondary">
                  <span>{{ formatCurrency(coin.initialValue) }}</span>
                  <span class="text-text-muted">to</span>
                  <span class="text-gold">{{ formatCurrency(coin.currentValue) }}</span>
                </div>
                <p v-if="coin.changeExplanation" class="m-0 text-chip leading-[1.4] text-text-secondary">{{ coin.changeExplanation }}</p>
              </li>
            </ol>
            <p v-else class="m-0 text-body text-text-muted">No valuation increases are available yet.</p>
          </article>

          <article class="card flex flex-col gap-3 p-4">
            <div>
              <p class="section-label">Top Drops</p>
              <h2 class="mt-1">Biggest Value Declines</h2>
            </div>
            <ol v-if="topDrops.length" class="m-0 flex list-none flex-col gap-3 p-0">
              <li v-for="coin in topDrops" :key="coin.coinId" class="flex flex-col gap-[0.35rem] rounded-sm border border-border-subtle bg-input p-3">
                <router-link class="font-semibold text-gold hover:underline" :to="`/coin/${coin.coinId}`">{{ coin.name }}</router-link>
                <div class="flex flex-wrap items-center gap-[0.35rem] text-body text-text-secondary">
                  <span>{{ formatCurrency(coin.initialValue) }}</span>
                  <span class="text-text-muted">to</span>
                  <span class="text-byzantine">{{ formatCurrency(coin.currentValue) }}</span>
                </div>
                <p v-if="coin.changeExplanation" class="m-0 text-chip leading-[1.4] text-text-secondary">{{ coin.changeExplanation }}</p>
              </li>
            </ol>
            <p v-else class="m-0 text-body text-text-muted">No valuation declines are available yet.</p>
          </article>

          <article class="card flex flex-col gap-3 p-4">
            <div>
              <p class="section-label">Stale Valuations</p>
              <h2 class="mt-1">Needs Refresh</h2>
            </div>
            <ol v-if="staleValuations.length" class="m-0 flex list-none flex-col gap-3 p-0">
              <li v-for="coin in staleValuations" :key="coin.coinId" class="flex flex-col gap-[0.35rem] rounded-sm border border-border-subtle bg-input p-3 text-body text-text-secondary">
                <router-link class="font-semibold text-gold hover:underline" :to="`/coin/${coin.coinId}/actions`">{{ coin.name }}</router-link>
                <span class="text-text-muted">{{ formatLastValuation(coin.lastValuationAt) }}</span>
              </li>
            </ol>
            <p v-else class="m-0 text-body text-text-muted">No stale valuations are available.</p>
          </article>
        </section>

        <StatsInvestmentBreakdownChart
          title="Acquisition Performance by Year"
          eyebrow="Purchase Timing"
          description="Compare each purchase year's invested capital, current value, gain/loss, and ROI."
          :rows="purchaseYearRows"
        />

        <StatsInvestmentBreakdownChart
          title="Material"
          eyebrow="Material Allocation"
          description="Compare invested capital, current value, gain/loss, and ROI by material."
          :rows="materialRows"
        />
      </template>
    </div>
  </PullToRefresh>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ArrowLeft } from 'lucide-vue-next'
import { getInvestmentBreakdown } from '@/api/client'
import PullToRefresh from '@/components/PullToRefresh.vue'
import StatsInvestmentBreakdownChart from '@/components/stats/StatsInvestmentBreakdownChart.vue'
import { formatCurrency } from '@/utils/format'
import type {
  InvestmentBreakdownResponse,
  InvestmentBreakdownSegment,
  InvestmentMovementCoin,
  StaleValuationCoin,
} from '@/types'

const isLoading = ref(true)
const errorMessage = ref('')
const purchaseYearRows = ref<InvestmentBreakdownSegment[]>([])
const materialRows = ref<InvestmentBreakdownSegment[]>([])
const topIncreases = ref<InvestmentMovementCoin[]>([])
const topDrops = ref<InvestmentMovementCoin[]>([])
const staleValuations = ref<StaleValuationCoin[]>([])

function normalizeBreakdown(data: InvestmentBreakdownResponse): InvestmentBreakdownSegment[] {
  const rows = Array.isArray(data) ? data : data.segments ?? []
  return rows.map((row) => ({
    ...row,
    year: row.year ?? null,
    month: row.month ?? null,
    gainLossPct: row.gainLossPct ?? null,
  }))
}

function applyHighlights(data: InvestmentBreakdownResponse) {
  if (Array.isArray(data)) {
    topIncreases.value = []
    topDrops.value = []
    staleValuations.value = []
    return
  }
  topIncreases.value = data.topIncreases ?? []
  topDrops.value = data.topDrops ?? []
  staleValuations.value = data.staleValuations ?? []
}

function formatLastValuation(value: string | null): string {
  if (!value) return 'Never valued'
  return new Date(value).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}

async function loadBreakdowns() {
  isLoading.value = true
  errorMessage.value = ''
  try {
    const [purchaseYearRes, materialRes] = await Promise.all([
      getInvestmentBreakdown('purchase-year'),
      getInvestmentBreakdown('material'),
    ])
    purchaseYearRows.value = normalizeBreakdown(purchaseYearRes.data)
    materialRows.value = normalizeBreakdown(materialRes.data)
    applyHighlights(purchaseYearRes.data)
  } catch {
    errorMessage.value = 'Investment breakdown data could not be loaded.'
  } finally {
    isLoading.value = false
  }
}

async function handleRefresh() {
  await loadBreakdowns()
}

onMounted(loadBreakdowns)
</script>
