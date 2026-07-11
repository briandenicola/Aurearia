<template>
  <PullToRefresh :on-refresh="handleRefresh">
    <div class="container flex flex-col gap-6">
      <header class="page-header flex flex-nowrap items-center justify-between gap-4">
        <div>
          <p class="section-label">Collection Insights</p>
          <h1>Health</h1>
          <p class="mt-[0.35rem] text-base text-text-secondary">Track the completeness and quality of your collection data.</p>
        </div>
        <router-link
          class="inline-flex shrink-0 items-center justify-center rounded-sm border border-border-subtle bg-transparent p-[0.4rem] text-text-secondary transition hover:border-border-accent hover:bg-gold-glow hover:text-gold"
          to="/stats"
          aria-label="Back to Stats"
        >
          <ArrowLeft :size="20" />
        </router-link>
      </header>

      <div v-if="healthLoading" class="loading-overlay">
        <div class="spinner"></div>
      </div>

      <section v-else-if="collectionHealth" class="flex flex-col gap-6">
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
