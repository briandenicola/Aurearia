<template>
  <div class="stats-summary">
    <div class="stat-card">
      <span class="stat-number">{{ stats.totalCoins }}</span>
      <span class="stat-label">Coins Owned</span>
    </div>
    <div class="stat-card">
      <span class="stat-number">{{ stats.totalWishlist }}</span>
      <span class="stat-label">On Wishlist</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { StatsResponse } from '@/types'

defineProps<{
  stats: StatsResponse
}>()
</script>

<style scoped>
.stats-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.stat-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1.5rem;
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.stat-number {
  font-family: 'Cinzel', serif;
  font-size: 2rem;
  font-weight: 600;
  color: var(--text-primary);
}

.stat-label {
  font-size: 0.8rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
</style>
