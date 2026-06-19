<template>
  <div class="swipe-gallery">
    <!-- Card counter -->
    <div v-if="coins.length" class="card-counter">
      {{ absoluteIndex + 1 }} / {{ total }}
    </div>

    <!-- Card stack -->
    <div v-if="coins.length" class="card-stack" ref="stackRef">
      <!-- Next card (underneath) -->
      <div v-if="nextCoin" class="swipe-card next-card">
        <div class="swipe-card-image">
          <AuthenticatedImage v-if="getImage(nextCoin)" :media-path="getImage(nextCoin)!" :alt="nextCoin.name" />
          <div v-else class="swipe-card-placeholder"><Coins :size="64" :stroke-width="1" /></div>
        </div>
        <div class="swipe-card-name">{{ nextCoin.name }}</div>
      </div>

      <!-- Active card (on top, draggable) -->
      <div
        v-if="currentCoin"
        class="swipe-card active-card"
        :class="{ animating: isAnimating }"
        :style="cardStyle"
        @pointerdown="onPointerDown"
        @click="onCardTap"
      >
        <div class="swipe-card-image" :class="{ 'coin-flipping': isFlipping }">
          <AuthenticatedImage v-if="getImage(currentCoin)" :media-path="getImage(currentCoin)!" :alt="currentCoin.name" />
          <div v-else class="swipe-card-placeholder"><Coins :size="64" :stroke-width="1" /></div>
          <button
            class="flip-btn"
            :class="{ spinning: isFlipping }"
            @pointerdown.stop
            @click.stop="flipCoin"
            :disabled="isFlipping"
            title="Flip coin"
          >
            ⟳
          </button>
        </div>
        <div class="swipe-card-name">{{ currentCoin.name }}</div>
        <div class="swipe-hint left-hint" :style="{ opacity: leftHintOpacity }"><ChevronLeft :size="32" /></div>
        <div class="swipe-hint right-hint" :style="{ opacity: rightHintOpacity }"><ChevronRight :size="32" /></div>
      </div>
    </div>

    <div v-else class="swipe-empty">
      <p>No coins to display</p>
    </div>

    <!-- Arrow navigation -->
    <div v-if="total > 1" class="swipe-nav">
      <button class="nav-btn" @click="goPrev">
        <ChevronLeft :size="16" /> Prev
      </button>
      <button class="nav-btn" @click="goNext">
        Next <ChevronRight :size="16" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import type { Coin, ImageType } from '@/types'
import { useCoinsStore } from '@/stores/coins'
import { Coins, ChevronLeft, ChevronRight } from 'lucide-vue-next'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'

const props = defineProps<{
  coins: Coin[]
  total: number
  page: number
  perPage: number
}>()
const emit = defineEmits<{ 'page-change': [page: number] }>()
const router = useRouter()
const store = useCoinsStore()

const activeSide = ref<'obverse' | 'reverse'>('obverse')
const currentIndex = computed({
  get: () => store.galleryIndex,
  set: (val) => { store.galleryIndex = val },
})
const isAnimating = ref(false)
const isFlipping = ref(false)

const absoluteIndex = computed(() => (props.page - 1) * props.perPage + currentIndex.value)

const FLIP_DURATION = 200 // ms for each half of the flip

function flipCoin() {
  if (isFlipping.value || isAnimating.value) return
  isFlipping.value = true
  setTimeout(() => {
    activeSide.value = activeSide.value === 'obverse' ? 'reverse' : 'obverse'
    setTimeout(() => {
      isFlipping.value = false
    }, FLIP_DURATION)
  }, FLIP_DURATION)
}
const stackRef = ref<HTMLElement | null>(null)
const animationTimers: ReturnType<typeof setTimeout>[] = []

// Drag state
const dragX = ref(0)
const dragY = ref(0)
const isDragging = ref(false)
let startX = 0
let startY = 0
let hasDragged = false
let pointerId: number | null = null

const SWIPE_THRESHOLD = 100
const FLY_DISTANCE = 600

// Clamp index if coins list shrinks (e.g., after filter change)
watch(() => props.coins.length, (len) => {
  if (len > 0 && currentIndex.value >= len) {
    store.galleryIndex = len - 1
  }
})

const currentCoin = computed(() => props.coins[currentIndex.value] ?? null)
const nextCoin = computed(() => {
  if (!props.coins.length) return null
  const nextIdx = (currentIndex.value + 1) % props.coins.length
  return props.coins[nextIdx] ?? null
})

const leftHintOpacity = computed(() => Math.min(1, Math.max(0, -dragX.value / SWIPE_THRESHOLD)))
const rightHintOpacity = computed(() => Math.min(1, Math.max(0, dragX.value / SWIPE_THRESHOLD)))

const cardStyle = computed(() => {
  if (!isDragging.value && !isAnimating.value) return {}
  const rotate = dragX.value * 0.05
  return {
    transform: `translate(${dragX.value}px, ${dragY.value}px) rotate(${rotate}deg)`,
    transition: isAnimating.value ? 'transform 0.3s ease' : 'none',
  }
})

function getImage(coin: Coin): string | null {
  const targetType: ImageType = activeSide.value
  const byType = coin.images?.find((img) => img.imageType === targetType)
  if (byType) return byType.filePath
  const primary = coin.images?.find((img) => img.isPrimary)
  const first = coin.images?.[0]
  const img = primary || first
  return img ? img.filePath : null
}

function onPointerDown(e: PointerEvent) {
  if (isAnimating.value || isFlipping.value) return
  const target = e.target as HTMLElement
  target.setPointerCapture(e.pointerId)
  pointerId = e.pointerId
  startX = e.clientX
  startY = e.clientY
  hasDragged = false
  isDragging.value = true
  dragX.value = 0
  dragY.value = 0

  target.addEventListener('pointermove', onPointerMove)
  target.addEventListener('pointerup', onPointerUp)
  target.addEventListener('pointercancel', onPointerUp)
}

function onPointerMove(e: PointerEvent) {
  if (!isDragging.value) return
  dragX.value = e.clientX - startX
  dragY.value = (e.clientY - startY) * 0.3
  if (Math.abs(dragX.value) > 5) hasDragged = true
}

function onPointerUp(e: PointerEvent) {
  if (!isDragging.value) return
  const target = e.target as HTMLElement
  target.removeEventListener('pointermove', onPointerMove)
  target.removeEventListener('pointerup', onPointerUp)
  target.removeEventListener('pointercancel', onPointerUp)
  if (pointerId !== null) {
    target.releasePointerCapture(pointerId)
    pointerId = null
  }
  isDragging.value = false

  if (Math.abs(dragX.value) > SWIPE_THRESHOLD) {
    flyAway(dragX.value > 0 ? 1 : -1)
  } else {
    // Spring back
    isAnimating.value = true
    dragX.value = 0
    dragY.value = 0
    const tid = setTimeout(() => {
      isAnimating.value = false
    }, 300)
    animationTimers.push(tid)
  }
}

function flyAway(direction: 1 | -1) {
  isAnimating.value = true
  dragX.value = direction * FLY_DISTANCE
  dragY.value = direction * -50

  const tid = setTimeout(() => {
    isAnimating.value = false
    dragX.value = 0
    dragY.value = 0

    const len = props.coins.length
    if (len === 0) return

    if (direction > 0) {
      if (currentIndex.value < len - 1) {
        currentIndex.value = currentIndex.value + 1
      } else if (absoluteIndex.value < props.total - 1) {
        currentIndex.value = 0
        emit('page-change', props.page + 1)
      } else {
        currentIndex.value = 0
        if (props.page > 1) {
          emit('page-change', 1)
        }
      }
    } else {
      if (currentIndex.value > 0) {
        currentIndex.value = currentIndex.value - 1
      } else if (props.page > 1) {
        const maxPages = Math.ceil(props.total / props.perPage)
        const prevPage = props.page - 1
        const prevPageSize = prevPage === maxPages ? (props.total % props.perPage || props.perPage) : props.perPage
        currentIndex.value = prevPageSize - 1
        emit('page-change', prevPage)
      } else {
        const lastPage = Math.ceil(props.total / props.perPage)
        const lastPageSize = props.total % props.perPage || props.perPage
        currentIndex.value = lastPageSize - 1
        if (lastPage > 1) {
          emit('page-change', lastPage)
        }
      }
    }
  }, 300)
  animationTimers.push(tid)
}

function onCardTap() {
  if (hasDragged || !currentCoin.value) return
  router.push(`/coin/${currentCoin.value.id}`)
}

function goNext() {
  if (props.coins.length > 0 && props.total > 1 && !isAnimating.value) {
    flyAway(1)
  }
}

function goPrev() {
  if (props.coins.length > 0 && props.total > 1 && !isAnimating.value) {
    flyAway(-1)
  }
}

onUnmounted(() => {
  animationTimers.forEach(clearTimeout)
})
</script>

<style scoped>
.swipe-gallery {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  padding: 1rem 0;
  user-select: none;
  -webkit-user-select: none;
}

.card-counter {
  font-size: 0.8rem;
  color: var(--text-muted);
  letter-spacing: 0.05em;
}

.card-stack {
  position: relative;
  width: 315px;
  height: 399px;
  touch-action: none;
}

@media (display-mode: standalone) {
  .card-stack {
    width: 340px;
    height: 430px;
  }
}

@media (min-width: 480px) {
  .card-stack {
    width: 340px;
    height: 420px;
  }
}

@media (display-mode: standalone) and (min-width: 480px) {
  .card-stack {
    width: 380px;
    height: 470px;
  }
}

.swipe-card {
  position: absolute;
  inset: 0;
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  box-shadow: var(--shadow-card);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.next-card {
  transform: scale(0.95);
  opacity: 0.6;
}

.active-card {
  z-index: 2;
  cursor: grab;
}

.active-card:active {
  cursor: grabbing;
}

.swipe-card-image {
  flex: 1;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background: radial-gradient(ellipse at center, var(--bg-secondary) 0%, var(--bg-primary) 100%);
  position: relative;
}

/* Vignette overlay */
.swipe-card-image::after {
  content: '';
  position: absolute;
  inset: 0;
  box-shadow: inset 0 0 50px rgba(0, 0, 0, 0.3);
  border-bottom: 1px solid var(--accent-gold-dim);
  pointer-events: none;
  z-index: 1;
}

.swipe-card-image img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  transform: scale(1.05);
  pointer-events: none;
}

.swipe-card-placeholder {
  font-size: 5rem;
  opacity: 0.2;
}

@keyframes coin-spin {
  0%   { transform: scaleX(1); }
  50%  { transform: scaleX(0); }
  100% { transform: scaleX(1); }
}

@keyframes flip-btn-spin {
  from { transform: rotate(0deg); }
  to   { transform: rotate(360deg); }
}

.coin-flipping {
  animation: coin-spin 0.4s ease-in-out;
}

.flip-btn {
  position: absolute;
  bottom: 0.75rem;
  right: 0.75rem;
  width: 2.25rem;
  height: 2.25rem;
  border-radius: 50%;
  border: 1px solid var(--border-accent);
  background: var(--bg-card);
  color: var(--accent-gold);
  font-size: 1.25rem;
  line-height: 1;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 3;
  opacity: 0.8;
  transition: opacity var(--transition-fast);
  touch-action: manipulation;
}

.flip-btn:hover:not(:disabled) {
  opacity: 1;
}

.flip-btn:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.flip-btn.spinning {
  animation: flip-btn-spin 0.4s linear;
}

.swipe-card-name {
  padding: 0.75rem 1rem;
  text-align: center;
  font-family: 'Cinzel', serif;
  font-size: 1rem;
  color: var(--text-heading);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  border-top: 1px solid var(--border-subtle);
}

.swipe-hint {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  font-size: 2rem;
  color: var(--accent-gold);
  pointer-events: none;
  opacity: 0;
  transition: opacity 0.1s;
}

.left-hint {
  left: 1rem;
}

.right-hint {
  right: 1rem;
}

.swipe-nav {
  display: flex;
  gap: 1rem;
}

.nav-btn {
  padding: 0.5rem 1.25rem;
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-full);
  background: var(--bg-card);
  color: var(--text-secondary);
  font-size: 0.85rem;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.nav-btn:hover:not(:disabled) {
  background: var(--accent-gold-dim);
  color: var(--text-primary);
}

.nav-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.swipe-empty {
  padding: 3rem;
  color: var(--text-muted);
  text-align: center;
}
</style>
