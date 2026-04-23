<template>
  <PullToRefresh :on-refresh="handleRefresh">
  <div class="container">
    <div class="page-header">
      <h1>Sold Coins</h1>
    </div>

    <div v-if="store.loading" class="loading-overlay">
      <div class="spinner"></div>
    </div>

    <div v-else-if="store.coins.length" class="coins-grid">
      <CoinCard v-for="coin in store.coins" :key="coin.id" :coin="coin" sold />
    </div>

    <div v-if="store.total > pageSize" class="pagination">
      <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="page--">← Previous</button>
      <span class="page-info">Page {{ page }} of {{ Math.ceil(store.total / pageSize) }}</span>
      <button class="btn btn-secondary btn-sm" :disabled="page * pageSize >= store.total" @click="page++">Next →</button>
    </div>

    <div v-else-if="!store.loading && !store.coins.length" class="empty-state">
      <h3>No sold coins yet</h3>
      <p>When you sell coins from your collection, they'll appear here with their sale history.</p>
    </div>
  </div>
  </PullToRefresh>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useCoinsStore } from '@/stores/coins'
import CoinCard from '@/components/CoinCard.vue'
import PullToRefresh from '@/components/PullToRefresh.vue'

const store = useCoinsStore()
const page = ref(1)
const pageSize = 50

function loadCoins() {
  store.fetchCoins({ sold: 'true', sort: 'updated_at', order: 'desc', page: page.value })
}

watch(page, loadCoins)
loadCoins()

async function handleRefresh() {
  page.value = 1
  await store.fetchCoins({ sold: 'true', sort: 'updated_at', order: 'desc', page: 1 })
}
</script>

<style scoped>
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 1rem;
  margin-top: 1.5rem;
  padding: 1rem 0;
}

.page-info {
  font-size: 0.85rem;
  color: var(--text-secondary);
}
</style>
