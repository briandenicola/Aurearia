<template>
  <div v-if="loading" class="loading-overlay">
    <div class="spinner"></div>
    <p>Loading collection...</p>
  </div>

  <template v-else-if="coins.length">
    <div v-if="selectMode" class="select-controls">
      <button class="btn btn-sm btn-secondary" @click="$emit('select-all')">Select All</button>
      <button class="btn btn-sm btn-secondary" @click="$emit('deselect-all')">Deselect All</button>
      <span class="select-count">{{ selectedCount }} selected</span>
    </div>
    <SwipeGallery v-if="isPwa && viewMode === 'swipe' && !selectMode" :coins="coins" :total="total" :page="page" :per-page="perPage" @page-change="$emit('page-change', $event)" />
    <div v-else class="coins-grid">
      <CoinCard
        v-for="coin in coins"
        :key="coin.id"
        :coin="coin"
        :image-side="gridSide"
        :selectable="selectMode"
        :selected="selectedCoinIds.has(coin.id)"
        @toggle-select="$emit('toggle-coin-select', $event)"
      />
    </div>
  </template>

  <div v-else class="empty-state">
    <h3>{{ hasFilters ? 'No coins match your search' : 'Your collection is empty' }}</h3>
    <p>{{ hasFilters ? 'Try different filters' : 'Add your first coin to get started' }}</p>
    <router-link v-if="!hasFilters" to="/add" class="btn btn-primary" style="margin-top: 1rem">
      Add Your First Coin
    </router-link>
  </div>
</template>

<script setup lang="ts">
import type { Coin, ImageType } from '@/types'
import CoinCard from '@/components/CoinCard.vue'
import SwipeGallery from '@/components/SwipeGallery.vue'

defineProps<{
  loading: boolean
  coins: Coin[]
  selectMode: boolean
  selectedCoinIds: Set<number>
  selectedCount: number
  isPwa: boolean
  viewMode: 'grid' | 'swipe'
  gridSide: ImageType | null
  hasFilters: boolean
  total: number
  page: number
  perPage: number
}>()

defineEmits<{
  'select-all': []
  'deselect-all': []
  'toggle-coin-select': [coinId: number]
  'page-change': [page: number]
}>()
</script>

<style scoped>
.select-controls {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.select-count {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin-left: 0.5rem;
}
</style>
