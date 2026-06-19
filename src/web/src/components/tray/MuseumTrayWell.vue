<template>
  <div
    class="tray-well"
    :style="{ width: `${renderSizePx}px`, height: `${renderSizePx}px` }"
    :aria-label="coin.name"
    tabindex="0"
    role="button"
    @click="handleClick"
    @keydown.enter="handleClick"
  >
    <div class="well-container">
      <img
        v-if="primaryImage"
        :src="primaryImage"
        :alt="coin.name"
        class="well-coin"
        loading="eager"
        decoding="async"
      />
      <div v-else class="well-placeholder">
        <Coins :size="Math.floor(renderSizePx * 0.4)" :stroke-width="1" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Coins } from 'lucide-vue-next'
import type { TrayCoin } from '@/utils/trayLayout'

interface Props {
  coin: TrayCoin
  renderSizePx: number
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'coin-clicked': [coinId: number]
}>()

const primaryImage = computed(() => {
  const path = props.coin.images.find(img => img.isPrimary)?.filePath ?? props.coin.images[0]?.filePath ?? null
  if (!path) return null
  // Preserve absolute URLs; prefix relative paths with /uploads/
  if (path.startsWith('/') || path.startsWith('http://') || path.startsWith('https://')) {
    return path
  }
  return `/uploads/${path}`
})

function handleClick() {
  emit('coin-clicked', props.coin.id)
}
</script>

<style scoped>
.tray-well {
  position: relative;
  cursor: pointer;
  transition: var(--transition-fast);
  border-radius: 50%;
  background: radial-gradient(
    circle at center,
    rgba(0, 0, 0, 0.3) 0%,
    rgba(0, 0, 0, 0.15) 40%,
    transparent 70%
  );
  padding: 8%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tray-well:hover {
  transform: translateY(-2px);
  filter: brightness(1.1);
}

.tray-well:focus-visible {
  outline: 2px solid var(--accent-gold);
  outline-offset: 4px;
}

.well-container {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 
    0 2px 4px rgba(0, 0, 0, 0.3),
    0 4px 8px rgba(0, 0, 0, 0.2),
    inset 0 1px 2px rgba(255, 255, 255, 0.1);
}

.well-coin {
  width: 100%;
  height: 100%;
  object-fit: cover;
  filter: drop-shadow(0 2px 3px rgba(0, 0, 0, 0.4));
}

.well-placeholder {
  color: var(--text-muted);
  opacity: 0.5;
  display: flex;
  align-items: center;
  justify-content: center;
}

@media (prefers-reduced-motion: reduce) {
  .tray-well {
    transition: none;
  }
  .tray-well:hover {
    transform: none;
  }
}
</style>
