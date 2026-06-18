<template>
  <PullToRefresh :on-refresh="handleRefresh">
    <div class="container stats-value-trends-page">
      <header class="page-header stats-value-trends-header">
        <div>
          <p class="section-label">Collection Insights</p>
          <h1>Value Trends</h1>
          <p class="page-intro">Track your collection's value over time.</p>
        </div>
        <router-link class="btn btn-secondary" to="/stats">Back to Stats</router-link>
      </header>

      <div v-if="isLoading" class="loading-overlay">
        <div class="spinner"></div>
      </div>

      <StatsValueOverTime v-else />
    </div>
  </PullToRefresh>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useCoinsStore } from '@/stores/coins'
import PullToRefresh from '@/components/PullToRefresh.vue'
import StatsValueOverTime from '@/components/stats/StatsValueOverTime.vue'

const store = useCoinsStore()
const isLoading = ref(true)

async function handleRefresh() {
  isLoading.value = true
  await store.fetchValueHistory()
  isLoading.value = false
}

onMounted(async () => {
  isLoading.value = true
  await store.fetchValueHistory()
  isLoading.value = false
})
</script>

<style scoped>
.stats-value-trends-page {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.stats-value-trends-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.page-intro {
  margin: 0.35rem 0 0;
  color: var(--text-secondary);
  font-size: 0.9rem;
}

@media (max-width: 768px) {
  .stats-value-trends-header {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
