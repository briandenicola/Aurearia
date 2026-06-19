<template>
  <div class="tray-view-page">
    <!-- Empty state -->
    <div v-if="trayCoins.length === 0 && !loading" class="empty-state card">
      <Landmark :size="64" :stroke-width="1" class="empty-icon" />
      <h2>No Coins in Tray</h2>
      <p>{{ trayEmptyMessage }}</p>
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
    <div v-else-if="!loading" class="tray-content">
      <MuseumTray
        class="tray-swipe-surface"
        :coins="currentDrawerCoins"
        :felt-theme="feltColor"
        :style="traySwipeStyle"
        @pointerdown="onTrayPointerDown"
        @coin-clicked="handleCoinClicked"
      />
      <TrayControls
        :drawer-index="drawerIndex"
        :total-drawers="totalDrawers"
        @prev="handlePrevDrawer"
        @next="handleNextDrawer"
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
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getCoins } from '@/api/client'
import { useTrayPreference } from '@/composables/useTrayPreference'
import { getDrawerCoins, getTotalDrawers, hasKnownDiameterMm, type TrayCoin } from '@/utils/trayLayout'
import type { Coin } from '@/types'
import MuseumTray from '@/components/tray/MuseumTray.vue'
import TrayControls from '@/components/tray/TrayControls.vue'
import { Landmark, ArrowLeft, Plus } from 'lucide-vue-next'

const router = useRouter()
const { feltColor } = useTrayPreference()

const loading = ref(true)
const errorMessage = ref('')
const loadedCoins = ref<Coin[]>([])
const drawerIndex = ref(0)
const coinsPerDrawer = 12
const trayPageLimit = 100
const trayDragX = ref(0)
const trayDragY = ref(0)
const trayIsDragging = ref(false)
const trayIsAnimating = ref(false)
const suppressCoinClick = ref(false)
let trayStartX = 0
let trayStartY = 0
let trayPointerId: number | null = null
const trayAnimationTimers: ReturnType<typeof setTimeout>[] = []

const SWIPE_THRESHOLD = 100
const FLY_DISTANCE = 600

const trayCoins = computed((): TrayCoin[] => {
  return loadedCoins.value
    .filter(coin => hasKnownDiameterMm(coin.diameterMm))
    .map(coin => ({
      id: coin.id,
      name: coin.name,
      diameterMm: coin.diameterMm,
      images: coin.images,
    }))
})

const trayEmptyMessage = computed(() => {
  if (errorMessage.value) return errorMessage.value
  if (loadedCoins.value.length > 0 && trayCoins.value.length === 0) {
    return 'No active collection coins have known diameter values. Add coin size details to display them in the tray.'
  }
  return 'Your collection is empty or no coins match the current filters.'
})

const currentDrawerCoins = computed(() => {
  return getDrawerCoins(trayCoins.value, drawerIndex.value, coinsPerDrawer)
})

const totalDrawers = computed(() => {
  return getTotalDrawers(trayCoins.value.length, coinsPerDrawer)
})

watch(totalDrawers, (drawers) => {
  if (drawers === 0) {
    drawerIndex.value = 0
    return
  }
  drawerIndex.value = Math.min(drawerIndex.value, drawers - 1)
})

const traySwipeStyle = computed(() => {
  if (!trayIsDragging.value && !trayIsAnimating.value) return {}
  return {
    transform: `translateX(${trayDragX.value}px)`,
    transition: trayIsAnimating.value ? 'transform 0.3s ease' : 'none',
  }
})

function handlePrevDrawer() {
  drawerIndex.value = Math.max(0, drawerIndex.value - 1)
}

function handleNextDrawer() {
  drawerIndex.value = Math.min(totalDrawers.value - 1, drawerIndex.value + 1)
}

function handleCoinClicked(coinId: number) {
  if (suppressCoinClick.value) return
  router.push({ name: 'coin-detail', params: { id: coinId } })
}

function onTrayPointerDown(event: PointerEvent) {
  if (trayIsAnimating.value || totalDrawers.value <= 1) return
  const target = event.currentTarget as HTMLElement
  target.setPointerCapture(event.pointerId)
  trayPointerId = event.pointerId
  trayStartX = event.clientX
  trayStartY = event.clientY
  trayDragX.value = 0
  trayDragY.value = 0
  trayIsDragging.value = true
  suppressCoinClick.value = false

  target.addEventListener('pointermove', onTrayPointerMove)
  target.addEventListener('pointerup', onTrayPointerUp)
  target.addEventListener('pointercancel', onTrayPointerUp)
}

function onTrayPointerMove(event: PointerEvent) {
  if (!trayIsDragging.value) return
  trayDragX.value = event.clientX - trayStartX
  trayDragY.value = event.clientY - trayStartY
  if (Math.abs(trayDragX.value) > 8 && Math.abs(trayDragX.value) > Math.abs(trayDragY.value)) {
    suppressCoinClick.value = true
  }
}

function onTrayPointerUp(event: PointerEvent) {
  if (!trayIsDragging.value) return
  const target = event.currentTarget as HTMLElement
  target.removeEventListener('pointermove', onTrayPointerMove)
  target.removeEventListener('pointerup', onTrayPointerUp)
  target.removeEventListener('pointercancel', onTrayPointerUp)
  if (trayPointerId !== null) {
    target.releasePointerCapture(trayPointerId)
    trayPointerId = null
  }
  trayIsDragging.value = false

  if (Math.abs(trayDragX.value) > SWIPE_THRESHOLD && Math.abs(trayDragX.value) > Math.abs(trayDragY.value)) {
    flyTray(trayDragX.value > 0 ? -1 : 1)
    return
  }

  trayIsAnimating.value = true
  trayDragX.value = 0
  const timer = setTimeout(() => {
    trayIsAnimating.value = false
    suppressCoinClick.value = false
  }, 300)
  trayAnimationTimers.push(timer)
}

function flyTray(direction: 1 | -1) {
  trayIsAnimating.value = true
  trayDragX.value = direction * -FLY_DISTANCE

  const timer = setTimeout(() => {
    if (direction > 0) {
      handleNextDrawer()
    } else {
      handlePrevDrawer()
    }
    trayDragX.value = 0
    trayDragY.value = 0
    trayIsAnimating.value = false
    suppressCoinClick.value = false
  }, 300)
  trayAnimationTimers.push(timer)
}

async function loadTrayCoins() {
  loading.value = true
  errorMessage.value = ''
  drawerIndex.value = 0
  try {
    const allCoins: Coin[] = []
    let page = 1

    while (true) {
      const res = await getCoins({
        wishlist: 'false',
        sold: 'false',
        page,
        limit: trayPageLimit,
        sort: 'name',
        order: 'asc',
      })
      const pageCoins = res.data.coins ?? []
      allCoins.push(...pageCoins)
      const total = res.data.total ?? allCoins.length

      if (!pageCoins.length || allCoins.length >= total) break
      page += 1
    }

    loadedCoins.value = allCoins
  } catch {
    loadedCoins.value = []
    errorMessage.value = 'Tray coins could not be loaded. Check your connection and try again.'
  } finally {
    loading.value = false
  }
}

onMounted(loadTrayCoins)

onBeforeUnmount(() => {
  trayAnimationTimers.forEach(clearTimeout)
})
</script>

<style scoped>
.tray-view-page {
  padding: 1rem;
  padding-bottom: 6rem;
  max-width: 1400px;
  margin: 0 auto;
}

.tray-content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.tray-swipe-surface {
  touch-action: pan-y;
  user-select: none;
  -webkit-user-select: none;
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
  border-radius: var(--radius-full);
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
