<template>
  <PullToRefresh :on-refresh="handleRefresh">
    <div class="container investment-breakdown-page">
      <header class="page-header investment-breakdown-header">
        <div>
          <p class="section-label">Collection Insights</p>
          <h1>Investment Breakdown</h1>
          <p class="page-intro">See where capital is allocated by acquisition timing and material.</p>
        </div>
        <router-link class="back-button" to="/stats" aria-label="Back to Stats">
          <ArrowLeft :size="20" />
        </router-link>
      </header>

      <div v-if="isLoading" class="loading-overlay">
        <div class="spinner"></div>
      </div>

      <div v-else-if="errorMessage" class="error-card card" role="alert">
        <p>{{ errorMessage }}</p>
        <button class="btn btn-sm btn-secondary" type="button" @click="handleRefresh">Try Again</button>
      </div>

      <template v-else>
        <section class="investment-highlights-grid" aria-label="Investment valuation highlights">
          <article class="stats-section card investment-highlight-card">
            <div class="highlight-header">
              <p class="section-label">Top Increases</p>
              <h2>Biggest Value Gains</h2>
            </div>
            <ol v-if="topIncreases.length" class="highlight-list">
              <li v-for="coin in topIncreases" :key="coin.coinId" class="highlight-row">
                <router-link class="coin-link" :to="`/coin/${coin.coinId}`">{{ coin.name }}</router-link>
                <div class="valuation-pair">
                  <span>{{ formatCurrency(coin.initialValue) }}</span>
                  <span class="arrow">to</span>
                  <span class="gold">{{ formatCurrency(coin.currentValue) }}</span>
                </div>
              </li>
            </ol>
            <p v-else class="empty-copy">No valuation increases are available yet.</p>
          </article>

          <article class="stats-section card investment-highlight-card">
            <div class="highlight-header">
              <p class="section-label">Top Drops</p>
              <h2>Biggest Value Declines</h2>
            </div>
            <ol v-if="topDrops.length" class="highlight-list">
              <li v-for="coin in topDrops" :key="coin.coinId" class="highlight-row">
                <router-link class="coin-link" :to="`/coin/${coin.coinId}`">{{ coin.name }}</router-link>
                <div class="valuation-pair">
                  <span>{{ formatCurrency(coin.initialValue) }}</span>
                  <span class="arrow">to</span>
                  <span class="loss">{{ formatCurrency(coin.currentValue) }}</span>
                </div>
              </li>
            </ol>
            <p v-else class="empty-copy">No valuation declines are available yet.</p>
          </article>

          <article class="stats-section card investment-highlight-card stale-card">
            <div class="highlight-header">
              <p class="section-label">Stale Valuations</p>
              <h2>Needs Refresh</h2>
            </div>
            <ol v-if="staleValuations.length" class="highlight-list">
              <li v-for="coin in staleValuations" :key="coin.coinId" class="highlight-row stale-row">
                <router-link class="coin-link" :to="`/coin/${coin.coinId}/actions`">{{ coin.name }}</router-link>
                <span class="last-run">{{ formatLastValuation(coin.lastValuationAt) }}</span>
              </li>
            </ol>
            <p v-else class="empty-copy">No stale valuations are available.</p>
          </article>
        </section>

        <StatsInvestmentBreakdownChart
          title="Purchase Year to Month"
          eyebrow="Acquisition Timing"
          description="Invested capital grouped by purchase year and month."
          :rows="purchaseMonthRows"
        />

        <StatsInvestmentBreakdownChart
          title="Material"
          eyebrow="Portfolio Composition"
          description="Invested capital and current value grouped by coin material."
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
const purchaseMonthRows = ref<InvestmentBreakdownSegment[]>([])
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
    const [purchaseMonthRes, materialRes] = await Promise.all([
      getInvestmentBreakdown('purchase-month'),
      getInvestmentBreakdown('material'),
    ])
    purchaseMonthRows.value = normalizeBreakdown(purchaseMonthRes.data)
    materialRows.value = normalizeBreakdown(materialRes.data)
    applyHighlights(purchaseMonthRes.data)
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

<style scoped>
.investment-breakdown-page {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.investment-breakdown-header {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: nowrap;
}

.page-intro {
  margin: 0.35rem 0 0;
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.back-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.4rem;
  color: var(--text-secondary);
  background: transparent;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  text-decoration: none;
  transition: var(--transition-fast);
  flex-shrink: 0;
}

.back-button:hover {
  color: var(--accent-gold);
  border-color: var(--border-accent);
  background: var(--accent-gold-glow);
}

.error-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  color: var(--text-secondary);
}

.error-card p {
  margin: 0;
}

.investment-highlights-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1rem;
}

.investment-highlight-card {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
}

.highlight-header h2 {
  margin: 0.25rem 0 0;
  font-size: 1.2rem;
}

.highlight-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0;
  margin: 0;
  list-style: none;
}

.highlight-row {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 0.75rem;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
}

.coin-link {
  color: var(--accent-gold);
  font-weight: 600;
  text-decoration: none;
}

.coin-link:hover {
  text-decoration: underline;
}

.valuation-pair,
.stale-row {
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.valuation-pair {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  flex-wrap: wrap;
}

.arrow,
.last-run,
.empty-copy {
  color: var(--text-muted);
}

.gold {
  color: var(--accent-gold);
}

.loss {
  color: var(--cat-byzantine);
}

.empty-copy {
  margin: 0;
  font-size: 0.85rem;
}

@media (max-width: 640px) {
  .investment-highlights-grid {
    grid-template-columns: 1fr;
  }

  .error-card {
    flex-direction: column;
    align-items: stretch;
  }
}

@media (min-width: 641px) and (max-width: 1024px) {
  .investment-highlights-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .stale-card {
    grid-column: 1 / -1;
  }
}
</style>
