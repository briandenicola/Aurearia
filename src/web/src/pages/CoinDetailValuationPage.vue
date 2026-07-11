<template>
  <CoinDetailSectionPageShell section-title="Value Trend">
    <template #default="{ coin: coinData }">
      <div v-if="coinData.isWishlist || coinData.isSold" class="card p-6 text-center text-text-secondary text-base">
        <p>Value tracking is only available for active coins in your collection.</p>
      </div>
      <div v-else>
        <div v-if="coinChartData.length >= 2" class="flex gap-2">
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
        <div v-if="coinChartData.length >= 2" class="flex justify-between text-label text-text-muted mt-1 px-2 pl-[68px]">
          <span>{{ formatShortDate(coinChartData[0]?.date ?? '') }}</span>
          <span>{{ formatShortDate(coinChartData[coinChartData.length - 1]?.date ?? '') }}</span>
        </div>
        <div v-else class="card p-6 text-center text-text-secondary text-base">
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
