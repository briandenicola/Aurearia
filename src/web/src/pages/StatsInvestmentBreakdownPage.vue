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
import type { InvestmentBreakdownResponse, InvestmentBreakdownSegment } from '@/types'

const isLoading = ref(true)
const errorMessage = ref('')
const purchaseMonthRows = ref<InvestmentBreakdownSegment[]>([])
const materialRows = ref<InvestmentBreakdownSegment[]>([])

function normalizeBreakdown(data: InvestmentBreakdownResponse): InvestmentBreakdownSegment[] {
  const rows = Array.isArray(data) ? data : data.segments ?? []
  return rows.map((row) => ({
    ...row,
    year: row.year ?? null,
    month: row.month ?? null,
    gainLossPct: row.gainLossPct ?? null,
  }))
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

@media (max-width: 640px) {
  .error-card {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
