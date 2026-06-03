<template>
  <CoinDetailSectionPageShell section-title="Value Trend">
    <template #default="{ coin: coinData }">
      <div v-if="coinData.isWishlist || coinData.isSold" class="valuation-empty card">
        <p>
          Value tracking is only available for active coins in your collection.
        </p>
      </div>
      <div v-else>
        <div v-if="coinChartData.length >= 2" class="line-chart-container">
          <div class="line-chart-y-axis">
            <span>{{ formatCurrency(coinChartMax) }}</span>
            <span>{{ formatCurrency(coinChartMax / 2) }}</span>
            <span>$0</span>
          </div>
          <div class="line-chart">
            <svg viewBox="0 0 1000 300" preserveAspectRatio="none" class="line-chart-svg">
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
        <div v-if="coinChartData.length >= 2" class="line-chart-dates">
          <span>{{ formatShortDate(coinChartData[0]?.date ?? '') }}</span>
          <span>{{ formatShortDate(coinChartData[coinChartData.length - 1]?.date ?? '') }}</span>
        </div>
        <div v-else class="valuation-empty card">
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
</script>

<style scoped>
.valuation-empty {
  padding: 1.5rem;
  text-align: center;
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.line-chart-container {
  display: flex;
  gap: 0.5rem;
}

.line-chart-y-axis {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  font-size: 0.7rem;
  color: var(--text-muted);
  text-align: right;
  min-width: 60px;
  padding: 0.25rem 0;
}

.line-chart {
  flex: 1;
  height: 200px;
  background: var(--bg-primary);
  border-radius: var(--radius-sm);
  padding: 0.5rem;
}

.line-chart-svg {
  width: 100%;
  height: 100%;
}

.line-chart-dates {
  display: flex;
  justify-content: space-between;
  font-size: 0.7rem;
  color: var(--text-muted);
  margin-top: 0.25rem;
  padding: 0 0.5rem 0 68px;
}
</style>
