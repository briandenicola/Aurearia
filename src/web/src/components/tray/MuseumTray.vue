<template>
  <div class="museum-tray" :class="`felt-${feltTheme}`">
    <div class="tray-grid">
      <MuseumTrayWell
        v-for="coin in coins"
        :key="coin.id"
        :coin="coin"
        :render-size-px="getRenderSize(coin)"
        :image-src-resolver="imageSrcResolver"
        :interactive="interactive"
        @coin-clicked="emit('coin-clicked', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import MuseumTrayWell from './MuseumTrayWell.vue'
import { getCoinRenderSizePx, normalizeDiameterMm, type TrayCoin } from '@/utils/trayLayout'
import type { FeltColor } from '@/composables/useTrayPreference'

interface Props {
  coins: TrayCoin[]
  feltTheme: FeltColor
  imageSrcResolver?: (filePath: string) => string
  interactive?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  imageSrcResolver: undefined,
  interactive: true,
})
const emit = defineEmits<{
  'coin-clicked': [coinId: number]
}>()

const layoutOptions = {
  minCoinPx: 40,
  maxCoinPx: 120,
  defaultDiameterMm: 20,
}

const allDiameters = computed(() => {
  return props.coins.map(coin => normalizeDiameterMm(coin.diameterMm, layoutOptions.defaultDiameterMm))
})

function getRenderSize(coin: TrayCoin): number {
  const diameter = normalizeDiameterMm(coin.diameterMm, layoutOptions.defaultDiameterMm)
  return getCoinRenderSizePx(diameter, allDiameters.value, layoutOptions)
}
</script>

<style scoped>
.museum-tray {
  position: relative;
  padding: 1.5rem;
  border-radius: var(--radius-md);
  min-height: 400px;
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
