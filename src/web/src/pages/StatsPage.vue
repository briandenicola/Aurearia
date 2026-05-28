<template>
  <PullToRefresh :on-refresh="handleRefresh">
    <div class="container">
      <div class="page-header">
        <h1>Collection Stats</h1>
      </div>

      <div v-if="!stats" class="loading-overlay">
        <div class="spinner"></div>
      </div>

      <div v-else class="stats-layout">
        <StatsSummaryCards :stats="stats" />

        <StatsBarChart
          title="By Category"
          :items="categoryItems"
          :fill-class="(label: string) => `fill-${label.toLowerCase()}`"
        >
          <template #label="{ item }">
            <span class="badge" :class="`badge-${item.label.toLowerCase()}`">{{ item.label }}</span>
          </template>
        </StatsBarChart>

        <StatsBarChart
          title="By Material"
          :items="materialItems"
          :fill-class="() => 'fill-material'"
        >
          <template #label="{ item }">
            <span :class="`material-${item.label.toLowerCase()}`">{{ item.label }}</span>
          </template>
        </StatsBarChart>

        <StatsBarChart
          v-if="stats.byGrade?.length"
          title="By Grade"
          :items="gradeItems"
          :fill-class="() => 'fill-grade'"
        />

        <StatsBarChart
          v-if="stats.byEra?.length"
          title="By Era"
          :items="eraItems"
          :fill-class="() => 'fill-era'"
        />

        <StatsBarChart
          v-if="stats.byRuler?.length"
          title="Top Rulers"
          :items="rulerItems"
          :fill-class="() => 'fill-ruler'"
          :wide="true"
        />

        <StatsBarChart
          v-if="stats.byPriceRange?.length"
          title="Price Range Distribution"
          :items="priceRangeItems"
          :fill-class="() => 'fill-price'"
        />

        <StatsValueOverTime />
        <StatsCoinValueTrend />
        <StatsHeatMap ref="heatMapRef" />
      </div>
    </div>
  </PullToRefresh>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useCoinsStore } from '@/stores/coins'
import PullToRefresh from '@/components/PullToRefresh.vue'
import StatsSummaryCards from '@/components/stats/StatsSummaryCards.vue'
import StatsBarChart from '@/components/stats/StatsBarChart.vue'
import type { BarItem } from '@/components/stats/StatsBarChart.vue'
import StatsValueOverTime from '@/components/stats/StatsValueOverTime.vue'
import StatsCoinValueTrend from '@/components/stats/StatsCoinValueTrend.vue'
import StatsHeatMap from '@/components/stats/StatsHeatMap.vue'

const store = useCoinsStore()
const stats = computed(() => store.stats)
const heatMapRef = ref<InstanceType<typeof StatsHeatMap>>()

const priceRangeOrder = ['Under $50', '$50 - $200', '$200 - $500', '$500 - $1K', '$1K+']

const categoryItems = computed<BarItem[]>(() =>
  stats.value?.byCategory.map((c) => ({ label: c.category, count: c.count })) ?? [],
)
const materialItems = computed<BarItem[]>(() =>
  stats.value?.byMaterial.map((m) => ({ label: m.material, count: m.count })) ?? [],
)
const gradeItems = computed<BarItem[]>(() =>
  stats.value?.byGrade?.map((g) => ({ label: g.grade, count: g.count })) ?? [],
)
const eraItems = computed<BarItem[]>(() =>
  stats.value?.byEra?.map((e) => ({ label: e.era, count: e.count })) ?? [],
)
const rulerItems = computed<BarItem[]>(() =>
  stats.value?.byRuler?.map((r) => ({ label: r.ruler, count: r.count })) ?? [],
)
const priceRangeItems = computed<BarItem[]>(() => {
  if (!stats.value?.byPriceRange) return []
  return [...stats.value.byPriceRange]
    .sort((a, b) => priceRangeOrder.indexOf(a.range) - priceRangeOrder.indexOf(b.range))
    .map((p) => ({ label: p.range, count: p.count }))
})

async function handleRefresh() {
  await Promise.all([
    store.fetchStats(),
    store.fetchValueHistory(),
    heatMapRef.value?.fetchDistribution(),
  ])
}

onMounted(() => {
  store.fetchStats()
  store.fetchValueHistory()
  heatMapRef.value?.fetchDistribution()
  if (!store.coins.length) store.fetchCoins()
})
</script>

<style scoped>
.stats-layout {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}
</style>