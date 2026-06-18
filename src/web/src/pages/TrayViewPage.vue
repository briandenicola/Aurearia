<template>
  <div class="tray-view-page">
    <!-- Empty state -->
    <div v-if="store.coins.length === 0 && !store.loading" class="empty-state card">
      <Landmark :size="64" :stroke-width="1" class="empty-icon" />
      <h2>No Coins in Tray</h2>
      <p>Your collection is empty or no coins match the current filters.</p>
      <div class="empty-actions">
        <router-link to="/" class="btn btn-secondary">
          <ArrowLeft :size="18" />
          Back to Collection
        </router-link>
        <router-link to="/add" class="btn btn-primary">
          <Plus :size="18" />
          Add Coin
        </router-link>
      </div>
    </div>

    <!-- Tray view -->
    <div v-else-if="!store.loading" class="tray-content">
      <TrayControls
        :drawer-index="drawerIndex"
        :total-drawers="totalDrawers"
        :felt-theme="feltColor"
        @prev="handlePrevDrawer"
        @next="handleNextDrawer"
        @update:felt-theme="feltColor = $event"
      />
      <MuseumTray
        :coins="currentDrawerCoins"
        :felt-theme="feltColor"
        @coin-clicked="handleCoinClicked"
      />
    </div>

    <!-- Loading state -->
    <div v-else class="loading-state">
      <div class="spinner"></div>
      <p>Loading coins...</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useCoinsStore } from '@/stores/coins'
import { useTrayPreference } from '@/composables/useTrayPreference'
import { getDrawerCoins, getTotalDrawers, type TrayCoin } from '@/utils/trayLayout'
import MuseumTray from '@/components/tray/MuseumTray.vue'
import TrayControls from '@/components/tray/TrayControls.vue'
import { Landmark, ArrowLeft, Plus } from 'lucide-vue-next'

const store = useCoinsStore()
const router = useRouter()
const { feltColor } = useTrayPreference()

const drawerIndex = ref(0)
const coinsPerDrawer = 24

const trayCoins = computed((): TrayCoin[] => {
  return store.coins.map(coin => ({
    id: coin.id,
    name: coin.name,
    diameterMm: coin.diameterMm,
    images: coin.images,
  }))
})

const currentDrawerCoins = computed(() => {
  return getDrawerCoins(trayCoins.value, drawerIndex.value, coinsPerDrawer)
})

const totalDrawers = computed(() => {
  return getTotalDrawers(trayCoins.value.length, coinsPerDrawer)
})

function handlePrevDrawer() {
  drawerIndex.value = Math.max(0, drawerIndex.value - 1)
}

function handleNextDrawer() {
  drawerIndex.value = Math.min(totalDrawers.value - 1, drawerIndex.value + 1)
}

function handleCoinClicked(coinId: number) {
  router.push({ name: 'coin-detail', params: { id: coinId } })
}
</script>

<style scoped>
.tray-view-page {
  padding: 1rem;
  max-width: 1400px;
  margin: 0 auto;
}

.tray-content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.empty-state {
  padding: 3rem 2rem;
  text-align: center;
  max-width: 500px;
  margin: 3rem auto;
}

.empty-icon {
  color: var(--text-muted);
  margin-bottom: 1rem;
}

.empty-state h2 {
  font-size: 1.5rem;
  margin-bottom: 0.5rem;
  color: var(--text-heading);
}

.empty-state p {
  color: var(--text-secondary);
  margin-bottom: 1.5rem;
}

.empty-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: center;
  flex-wrap: wrap;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  gap: 1rem;
}

.loading-state p {
  color: var(--text-secondary);
}

.spinner {
  width: 48px;
  height: 48px;
  border: 4px solid var(--border-subtle);
  border-top-color: var(--accent-gold);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .spinner {
    animation: none;
    border-top-color: var(--accent-gold);
  }
}

@media (max-width: 575px) {
  .tray-view-page {
    padding: 0.75rem;
  }
  
  .empty-state {
    padding: 2rem 1.5rem;
    margin: 2rem auto;
  }
}
</style>
