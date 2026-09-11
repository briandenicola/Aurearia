<template>
  <section class="storage-tray" :aria-labelledby="headingId">
    <header class="tray-heading">
      <h2 :id="headingId">{{ tray.name }}</h2>
      <span>{{ tray.occupied }} / {{ tray.capacity }}</span>
    </header>
    <div class="tray-scroller" tabindex="0" :aria-label="`${tray.name}, ${tray.columns} columns`">
      <TraySurface :felt-theme="feltTheme" class="physical-surface">
        <div
          class="physical-grid"
          role="grid"
          :aria-label="gridLabel"
          :aria-rowcount="tray.rows"
          :aria-colcount="tray.columns"
          :style="{ gridTemplateColumns: `repeat(${tray.columns}, var(--storage-well-size))` }"
        >
          <div
            v-for="position in positions"
            :key="position.slot"
            class="physical-cell"
            role="gridcell"
            :aria-label="position.coin ? undefined : `Empty, row ${position.row}, column ${position.column}`"
          >
            <MuseumTrayWell
              :coin="position.trayCoin"
              :render-size-px="STORAGE_WELL_SIZE_PX"
              :interactive="Boolean(position.coin)"
              :show-captions="false"
              :show-names="false"
              :aria-label="position.coin ? `${position.coin.name || 'Untitled'}, row ${position.row}, column ${position.column}` : `Empty, row ${position.row}, column ${position.column}`"
              @coin-clicked="emit('coin-clicked', $event)"
            />
            <span class="coordinate">{{ position.row }},{{ position.column }}</span>
          </div>
        </div>
      </TraySurface>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TrayAggregate, StorageTrayCoin } from '@/types'
import type { TrayCoin } from '@/utils/trayLayout'
import type { FeltColor } from '@/composables/useTrayPreference'
import MuseumTrayWell from './MuseumTrayWell.vue'
import TraySurface from './TraySurface.vue'

const STORAGE_WELL_SIZE_PX = 76
const props = defineProps<{ tray: TrayAggregate; feltTheme: FeltColor }>()
const emit = defineEmits<{ 'coin-clicked': [coinId: number] }>()
const headingId = computed(() => `storage-tray-${props.tray.id}`)
const gridLabel = computed(() =>
  `${props.tray.name}, ${props.tray.rows} rows by ${props.tray.columns} columns, ${props.tray.occupied} of ${props.tray.capacity} occupied`
)
const bySlot = computed(() => new Map(props.tray.coins.map((coin) => [coin.storageSlot, coin])))

function adaptCoin(coin: StorageTrayCoin | undefined, slot: number): TrayCoin {
  if (!coin) {
    return { id: -slot, name: `Empty slot ${slot}`, diameterMm: null, images: [], placeholder: true }
  }
  return {
    id: coin.id,
    name: coin.name || 'Untitled',
    diameterMm: coin.diameterMm ?? null,
    images: coin.image ? [coin.image] : [],
  }
}

const positions = computed(() =>
  Array.from({ length: props.tray.capacity }, (_, index) => {
    const slot = index + 1
    const coin = bySlot.value.get(slot)
    return {
      slot,
      row: Math.floor(index / props.tray.columns) + 1,
      column: (index % props.tray.columns) + 1,
      coin,
      trayCoin: adaptCoin(coin, slot),
    }
  })
)
</script>

<style scoped>
.storage-tray {
  min-width: 0;
}
.tray-heading {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.75rem;
}
.tray-heading h2 {
  margin: 0;
}
.tray-heading span,
.coordinate {
  color: var(--text-muted);
  font-size: 0.75rem;
}
.tray-scroller {
  max-width: 100%;
  overflow-x: auto;
  border-radius: var(--radius-md);
}
.tray-scroller:focus-visible {
  outline: 2px solid var(--accent-gold);
  outline-offset: 2px;
}
.physical-surface {
  width: max-content;
  min-width: 100%;
  padding: 1.5rem;
  box-sizing: border-box;
}
.physical-grid {
  --storage-well-size: 4.75rem;
  display: grid;
  gap: 1rem;
  width: max-content;
  margin-inline: auto;
}
.physical-cell {
  width: var(--storage-well-size);
  min-height: 6rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.35rem;
}
@media (prefers-reduced-motion: reduce) {
  .physical-grid {
    scroll-behavior: auto;
  }
}
</style>
