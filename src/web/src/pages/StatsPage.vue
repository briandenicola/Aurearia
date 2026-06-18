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

      <section v-else id="summary" aria-label="Summary metrics">
        <StatsSummaryCards :stats="stats" />
      </section>
    </div>
  </PullToRefresh>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useCoinsStore } from '@/stores/coins'
import PullToRefresh from '@/components/PullToRefresh.vue'
import StatsSummaryCards from '@/components/stats/StatsSummaryCards.vue'

const store = useCoinsStore()
const stats = computed(() => store.stats)

async function handleRefresh() {
  await store.fetchStats()
}

onMounted(() => {
  store.fetchStats()
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
