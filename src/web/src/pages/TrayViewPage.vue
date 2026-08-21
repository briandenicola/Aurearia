<template>
  <div class="mx-auto max-w-[1400px] px-4 pb-24 pt-4 max-sm:px-3 max-sm:pt-3">
    <!-- Empty state -->
    <div
      v-if="trayCoins.length === 0 && !loading"
      class="card mx-auto mt-12 max-w-[500px] px-8 py-12 text-center max-sm:mt-8 max-sm:px-6 max-sm:py-8"
    >
      <Landmark :size="64" :stroke-width="1" class="mx-auto mb-4 text-text-muted" />
      <h2 class="mb-2 text-xl text-heading">No Coins in Tray</h2>
      <p class="mb-6 text-text-secondary">{{ trayEmptyMessage }}</p>
      <div class="flex flex-wrap justify-center gap-3">
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
    <div v-else-if="!loading" class="flex flex-col gap-4">
      <div class="tray-controls-row">
        <label class="tray-size-control" for="tray-size-slider">
          <span>Coin size</span>
          <input
            id="tray-size-slider"
            v-model.number="traySizeScale"
            class="tray-size-slider"
            type="range"
            min="0.75"
            max="2.5"
            step="0.05"
          />
          <span class="tray-size-value">{{ traySizeScale.toFixed(2) }}x</span>
        </label>
      </div>
      <div ref="trayGestureRef" class="tray-gesture-surface select-none">
        <MuseumTray
          :coins="currentDrawerCoins"
          :felt-theme="feltColor"
          :show-captions="false"
          :size-scale="traySizeScale"
          :style="traySwipeStyle"
          @coin-clicked="handleCoinClicked"
        />
      </div>
      <TrayControls
        :drawer-index="drawerIndex"
        :total-drawers="totalDrawers"
        @prev="handlePrevDrawer"
        @next="handleNextDrawer"
      />
    </div>

    <!-- Loading state -->
    <div v-else class="loading-overlay px-8 py-16">
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
import { useSwipeGesture } from '@/composables/useSwipeGesture'

const STORAGE_KEY = 'tray:sizeScale'
const DEFAULT_SIZE_SCALE = 1
const MIN_SIZE_SCALE = 0.75
const MAX_SIZE_SCALE = 2.5

const router = useRouter()
const { feltColor } = useTrayPreference()

const loading = ref(true)
const errorMessage = ref('')
const loadedCoins = ref<Coin[]>([])
const drawerIndex = ref(0)
const coinsPerDrawer = 12
const trayPageLimit = 100

// Consumer-owned drag/animation state; updated by useSwipeGesture callbacks
const trayDragX = ref(0)
const trayIsAnimating = ref(false)
const suppressCoinClick = ref(false)
const traySizeScale = ref(DEFAULT_SIZE_SCALE)
const trayAnimationTimers: ReturnType<typeof setTimeout>[] = []

const FLY_DISTANCE = 600

// Template ref for the gesture surface wrapper div
const trayGestureRef = ref<HTMLElement | null>(null)

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

watch(traySizeScale, (value) => {
  const normalizedValue = Math.min(MAX_SIZE_SCALE, Math.max(MIN_SIZE_SCALE, Number(value) || DEFAULT_SIZE_SCALE))
  traySizeScale.value = normalizedValue
  localStorage.setItem(STORAGE_KEY, normalizedValue.toString())
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

function traySpringBack() {
  trayIsAnimating.value = true
  // trayDragX holds the last live value from onMove; setting to 0 in the same
  // sync block lets Vue batch both and CSS transition fires from current → 0.
  trayDragX.value = 0
  const timer = setTimeout(() => {
    trayIsAnimating.value = false
    suppressCoinClick.value = false
  }, 300)
  trayAnimationTimers.push(timer)
}

/**
 * Fly the tray off-screen then navigate.
 *
 * direction -1 = leftward swipe → next drawer   (swipe left = next, iOS convention)
 * direction  1 = rightward swipe → previous drawer
 */
function flyTray(direction: -1 | 1) {
  trayIsAnimating.value = true
  trayDragX.value = direction * FLY_DISTANCE   // negative = flies left, positive = flies right

  const timer = setTimeout(() => {
    if (direction < 0) {
      handleNextDrawer()    // left swipe = next drawer
    } else {
      handlePrevDrawer()    // right swipe = previous drawer
    }
    trayDragX.value = 0
    trayIsAnimating.value = false
    suppressCoinClick.value = false
  }, 300)
  trayAnimationTimers.push(timer)
}

// Gate: block new gestures during animation, and when only one drawer exists
const trayGestureEnabled = computed(() => !trayIsAnimating.value && totalDrawers.value > 1)

// Shared pointer primitive: owns capture, touch-action, pointer lifecycle, and threshold.
// Consumer owns visual state (trayDragX), animation timers, navigation, and click guard.
const { isDragging: trayIsDragging } = useSwipeGesture(trayGestureRef, {
  threshold: 101,        // strict > 100: primitive commits at >= threshold, so 101 → > 100
  lockSlop: 10,          // axis-lock: vertical-dominant drags yield to the browser's pan-y handler
  touchAction: 'pan-y',  // UA handles vertical scroll; primitive only claims horizontal drags
  capture: true,
  enabled: trayGestureEnabled,
  onStart() {
    suppressCoinClick.value = false
  },
  onMove(dx, dy) {
    trayDragX.value = dx
    if (Math.abs(dx) > 8 && Math.abs(dx) > Math.abs(dy)) {
      suppressCoinClick.value = true
    }
  },
  onCommit(direction) {
    // direction -1 = leftward = next drawer; 1 = rightward = previous drawer
    flyTray(direction)
  },
  onCancel() {
    traySpringBack()
  },
})

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

onMounted(() => {
  const storedScale = localStorage.getItem(STORAGE_KEY)
  if (storedScale !== null) {
    const parsedScale = Number(storedScale)
    if (Number.isFinite(parsedScale)) {
      traySizeScale.value = Math.min(MAX_SIZE_SCALE, Math.max(MIN_SIZE_SCALE, parsedScale))
    }
  }
  loadTrayCoins()
})

onBeforeUnmount(() => {
  trayAnimationTimers.forEach(clearTimeout)
})
</script>

<style scoped>
.tray-controls-row {
  display: flex;
  justify-content: flex-end;
}

.tray-size-control {
  display: inline-flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.55rem 0.9rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  background: rgba(255, 255, 255, 0.04);
  color: var(--text-secondary);
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.tray-size-slider {
  width: 140px;
  accent-color: var(--accent-gold);
}

.tray-size-value {
  min-width: 3.2rem;
  text-align: right;
  color: var(--text-primary);
}

@media (max-width: 575px) {
  .tray-controls-row {
    justify-content: center;
  }

  .tray-size-control {
    width: 100%;
    justify-content: space-between;
  }
}
</style>
