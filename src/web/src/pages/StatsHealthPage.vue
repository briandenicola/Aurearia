<template>
  <PullToRefresh :on-refresh="handleRefresh">
    <div class="container stats-health-page">
      <header class="page-header stats-health-header">
        <div>
          <p class="section-label">Collection Insights</p>
          <h1>Health</h1>
          <p class="page-intro">Track the completeness and quality of your collection data.</p>
        </div>
        <router-link class="back-button" to="/stats" aria-label="Back to Stats">
          <ArrowLeft :size="20" />
        </router-link>
      </header>

      <div v-if="healthLoading" class="loading-overlay">
        <div class="spinner"></div>
      </div>

      <section v-else-if="collectionHealth" class="health-content">
        <CollectionHealthScorecard :summary="collectionHealth" />
        <CollectionHealthTrendIndicator :trend="collectionHealth.trend30d" />
      </section>

      <CollectionHealthEmptyState v-else />
    </div>
  </PullToRefresh>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { ArrowLeft } from 'lucide-vue-next'
import { useCoinsStore } from '@/stores/coins'
import PullToRefresh from '@/components/PullToRefresh.vue'
import CollectionHealthScorecard from '@/components/stats/CollectionHealthScorecard.vue'
import CollectionHealthTrendIndicator from '@/components/stats/CollectionHealthTrendIndicator.vue'
import CollectionHealthEmptyState from '@/components/stats/CollectionHealthEmptyState.vue'

const store = useCoinsStore()
const collectionHealth = computed(() => store.collectionHealth)
const healthLoading = computed(() => store.healthLoading)

async function handleRefresh() {
  await store.fetchCollectionHealth().catch(() => {})
}

onMounted(() => {
  store.fetchCollectionHealth().catch(() => {})
})
</script>

<style scoped>
.stats-health-page {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.stats-health-header {
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

.health-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}
</style>
