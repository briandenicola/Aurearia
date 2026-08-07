<template>
  <div class="museum-tray" :class="`felt-${feltTheme}`">
    <div v-if="hasAnyFaceImage" class="tray-actions">
      <button
        class="tray-face-toggle"
        :class="{ 'is-reverse': activeFace === 'reverse' }"
        type="button"
        :aria-pressed="activeFace === 'reverse'"
        :aria-label="`Show ${activeFace === 'obverse' ? 'reverse' : 'obverse'} side for all coins`"
        :title="`Show ${activeFace === 'obverse' ? 'reverse' : 'obverse'} side`"
        @click="toggleFace"
      >
        <RotateCcw :size="16" />
      </button>
    </div>
    <div class="tray-grid">
      <MuseumTrayWell
        v-for="coin in coins"
        :key="`${coin.placeholder ? 'slot' : 'coin'}-${coin.id}`"
        :coin="coin"
        :render-size-px="getRenderSize(coin)"
        :image-src-resolver="imageSrcResolver"
        :interactive="interactive"
        :show-captions="showCaptions"
        :show-names="showNames"
        :preferred-face="activeFace"
        @coin-clicked="emit('coin-clicked', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { RotateCcw } from 'lucide-vue-next'
import MuseumTrayWell from './MuseumTrayWell.vue'
import { getScaledCoinRenderSizePx, normalizeDiameterMm, type TrayCoin, type TrayCoinFace } from '@/utils/trayLayout'
import type { FeltColor } from '@/composables/useTrayPreference'

interface Props {
  coins: TrayCoin[]
  feltTheme: FeltColor
  imageSrcResolver?: (filePath: string) => string
  interactive?: boolean
  showCaptions?: boolean
  showNames?: boolean
  sizeScale?: number
}

const props = withDefaults(defineProps<Props>(), {
  imageSrcResolver: undefined,
  interactive: true,
  showCaptions: true,
  showNames: false,
  sizeScale: 1,
})
const emit = defineEmits<{
  'coin-clicked': [coinId: number]
}>()

const layoutOptions = {
  minCoinPx: 40,
  maxCoinPx: 120,
  defaultDiameterMm: 20,
}
const activeFace = ref<TrayCoinFace>('obverse')

const allDiameters = computed(() => {
  return props.coins.map(coin => normalizeDiameterMm(coin.diameterMm, layoutOptions.defaultDiameterMm))
})

const hasAnyFaceImage = computed(() => {
  return props.coins.some((coin) =>
    coin.images.some((image) => {
      const imageType = image.imageType?.toLowerCase()
      return imageType === 'obverse' || imageType === 'reverse'
    })
  )
})

function getRenderSize(coin: TrayCoin): number {
  const diameter = normalizeDiameterMm(coin.diameterMm, layoutOptions.defaultDiameterMm)
  return getScaledCoinRenderSizePx(diameter, allDiameters.value, layoutOptions, props.sizeScale)
}

function toggleFace() {
  activeFace.value = activeFace.value === 'obverse' ? 'reverse' : 'obverse'
}
</script>

<style scoped>
.museum-tray {
  position: relative;
  padding: 1.5rem;
  border-radius: var(--radius-md);
  min-height: 400px;
}

.tray-actions {
  position: absolute;
  top: 0.85rem;
  right: 0.85rem;
  z-index: 2;
}

.tray-face-toggle {
  width: 2.2rem;
  height: 2.2rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-full);
  background: color-mix(in srgb, var(--bg-card) 88%, transparent);
  color: var(--text-secondary);
  box-shadow: var(--shadow-card);
  transition: all var(--transition-fast);
}

.tray-face-toggle:hover {
  border-color: var(--accent-gold);
  color: var(--accent-gold);
  transform: translateY(-1px);
}

.tray-face-toggle:focus-visible {
  outline: 2px solid var(--accent-gold);
  outline-offset: 2px;
}

.tray-face-toggle svg {
  transition: transform var(--transition-fast);
}

.tray-face-toggle.is-reverse {
  border-color: var(--accent-gold);
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.tray-face-toggle.is-reverse svg {
  transform: rotate(-180deg);
}

/* Felt texture backgrounds with design tokens */
.felt-red {
  background:
    linear-gradient(135deg, rgba(0,0,0,0.05) 25%, transparent 25%,
                    transparent 75%, rgba(0,0,0,0.05) 75%,
                    rgba(0,0,0,0.05)),
    linear-gradient(45deg, rgba(0,0,0,0.05) 25%, transparent 25%,
                    transparent 75%, rgba(0,0,0,0.05) 75%),
    linear-gradient(to bottom, var(--felt-red-base), var(--felt-red-dark));
  background-size: 4px 4px, 4px 4px, 100% 100%;
  background-position: 0 0, 2px 2px, 0 0;
  box-shadow:
    inset 0 2px 8px rgba(0, 0, 0, 0.3),
    var(--shadow-card);
}

.felt-green {
  background:
    linear-gradient(135deg, rgba(0,0,0,0.05) 25%, transparent 25%,
                    transparent 75%, rgba(0,0,0,0.05) 75%,
                    rgba(0,0,0,0.05)),
    linear-gradient(45deg, rgba(0,0,0,0.05) 25%, transparent 25%,
                    transparent 75%, rgba(0,0,0,0.05) 75%),
    linear-gradient(to bottom, var(--felt-green-base), var(--felt-green-dark));
  background-size: 4px 4px, 4px 4px, 100% 100%;
  background-position: 0 0, 2px 2px, 0 0;
  box-shadow:
    inset 0 2px 8px rgba(0, 0, 0, 0.3),
    var(--shadow-card);
}

.felt-navy {
  background:
    linear-gradient(135deg, rgba(0,0,0,0.05) 25%, transparent 25%,
                    transparent 75%, rgba(0,0,0,0.05) 75%,
                    rgba(0,0,0,0.05)),
    linear-gradient(45deg, rgba(0,0,0,0.05) 25%, transparent 25%,
                    transparent 75%, rgba(0,0,0,0.05) 75%),
    linear-gradient(to bottom, var(--felt-navy-base), var(--felt-navy-dark));
  background-size: 4px 4px, 4px 4px, 100% 100%;
  background-position: 0 0, 2px 2px, 0 0;
  box-shadow:
    inset 0 2px 8px rgba(0, 0, 0, 0.3),
    var(--shadow-card);
}

.tray-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1.5rem;
  justify-items: center;
  align-items: center;
  padding: 1rem;
}

@media (max-width: 575px) {
  .tray-grid {
    gap: 0.5rem;
    padding: 0.5rem;
  }
}

@media (min-width: 576px) and (max-width: 767px) {
  .tray-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 0.75rem;
  }
}

@media (min-width: 768px) {
  .tray-grid {
    grid-template-columns: repeat(6, minmax(0, 1fr));
    gap: 1.5rem;
  }
}
</style>
