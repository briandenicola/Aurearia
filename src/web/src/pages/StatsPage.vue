<template>
  <PullToRefresh :on-refresh="handleRefresh">
    <div class="container stats-landing">
      <header class="page-header stats-landing-header">
        <div>
          <p class="section-label">Collection Insights</p>
          <h1>Stats</h1>
          <p class="page-intro">Your collection summary at a glance.</p>
        </div>
      </header>

      <div v-if="!stats" class="loading-overlay">
        <div class="spinner"></div>
      </div>

      <template v-else>
        <section id="summary" aria-label="Summary metrics">
          <StatsSummaryCards :stats="stats" />
        </section>

        <StatsBarChart
          v-if="categoryItems.length"
          title="By Coin Type"
          :items="categoryItems"
          :fill-class="(label: string) => `fill-${label.toLowerCase()}`"
        >
          <template #label="{ item }">
            <span class="badge" :class="`badge-${item.label.toLowerCase()}`">{{ item.label }}</span>
          </template>
        </StatsBarChart>

        <StatsBarChart
          v-if="eraItems.length"
          title="By Era"
          :items="eraItems"
          :fill-class="() => 'fill-era'"
        />

        <StatsBarChart
          v-if="rulerItems.length"
          title="Top Rulers"
          :items="rulerItems"
          :fill-class="() => 'fill-ruler'"
          :wide="true"
        />

        <StatsHeatMap ref="heatMapRef" />

        <StatsCoinFlowChart />
      </template>
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
import StatsHeatMap from '@/components/stats/StatsHeatMap.vue'
import StatsCoinFlowChart from '@/components/stats/StatsCoinFlowChart.vue'

const store = useCoinsStore()
const stats = computed(() => store.stats)
const heatMapRef = ref<InstanceType<typeof StatsHeatMap>>()

const categoryItems = computed<BarItem[]>(() =>
  stats.value?.byCategory.map((c) => ({ label: c.category, count: c.count })) ?? [],
)
const eraItems = computed<BarItem[]>(() =>
  stats.value?.byEra?.map((e) => ({ label: e.era, count: e.count })) ?? [],
)
const rulerItems = computed<BarItem[]>(() =>
  stats.value?.byRuler?.map((r) => ({ label: r.ruler, count: r.count })) ?? [],
)

async function handleRefresh() {
  await Promise.all([
    store.fetchStats(),
    heatMapRef.value?.fetchDistribution(),
  ])
}

onMounted(() => {
  store.fetchStats()
  heatMapRef.value?.fetchDistribution()
})
</script>

<style scoped>
.stats-landing {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.stats-landing-header {
  margin-bottom: 0;
}

.page-intro {
  margin: 0.35rem 0 0;
  color: var(--text-secondary);
  font-size: 0.9rem;
}
</style>
