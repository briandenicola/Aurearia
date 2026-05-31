<template>
  <div class="container">
    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
    </div>

    <div v-else-if="coin" class="section-page">
      <!-- Back link -->
      <div class="section-header">
        <button class="btn btn-ghost btn-sm back-link" @click="navigateToOverview">
          <ChevronLeft :size="16" />
          Back to Overview
        </button>
      </div>

      <!-- Coin identity banner -->
      <div class="coin-identity-banner">
        <h1>{{ coin.name }}</h1>
        <p v-if="coin.ruler" class="coin-ruler">{{ coin.ruler }}</p>
      </div>

      <!-- Section title -->
      <h2 class="section-title">{{ sectionTitle }}</h2>

      <!-- Content slot -->
      <div class="section-content">
        <slot :coin="coin" :refresh="refreshCoin"></slot>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ChevronLeft } from 'lucide-vue-next'
import { useCoinDetailContext } from '@/composables/useCoinDetailContext'

defineProps<{
  sectionTitle: string
}>()

const { coin, loading, refreshCoin, navigateToOverview } = useCoinDetailContext()
</script>

<style scoped>
.section-page {
  max-width: 900px;
  margin-left: auto;
  margin-right: auto;
}

.section-header {
  margin-bottom: 1rem;
}

.back-link {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

.coin-identity-banner {
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-subtle);
}

.coin-identity-banner h1 {
  margin-bottom: 0.25rem;
  font-size: 1.5rem;
}

.coin-ruler {
  color: var(--text-secondary);
  font-size: 1rem;
}

.section-title {
  margin-bottom: 1.5rem;
  font-size: 1.2rem;
}

.section-content {
  /* Content area for section-specific content */
}
</style>
