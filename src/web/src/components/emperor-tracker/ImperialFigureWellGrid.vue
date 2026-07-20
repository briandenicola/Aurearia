<template>
  <div class="grid grid-cols-[repeat(auto-fill,minmax(84px,1fr))] gap-4">
    <div v-for="slot in slots" :key="wellKey(slot)" class="flex flex-col items-center gap-1 text-center">
      <MuseumTrayWell
        :coin="toTrayCoin(slot)"
        :render-size-px="72"
        :interactive="!!slot.coin"
        @coin-clicked="onWellClicked"
      />
      <span class="max-w-[84px] truncate text-xs text-text-secondary" :title="slot.figure.name">
        {{ slot.figure.name }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import MuseumTrayWell from '@/components/tray/MuseumTrayWell.vue'
import type { ImperialFigureSlot } from '@/types'
import type { TrayCoin } from '@/utils/trayLayout'

defineProps<{
  slots: ImperialFigureSlot[]
}>()

const router = useRouter()

function wellKey(slot: ImperialFigureSlot): string {
  return slot.coin ? `coin-${slot.coin.id}` : `figure-${slot.figure.id}`
}

// Unowned figures render as placeholder wells. Their synthetic id is
// negative so it never collides with a real coin id, and interactive is
// false so clicking one never tries to navigate to a nonexistent coin.
function toTrayCoin(slot: ImperialFigureSlot): TrayCoin {
  if (slot.coin) {
    return {
      id: slot.coin.id,
      name: slot.coin.name,
      diameterMm: slot.coin.diameterMm,
      images: slot.coin.images,
    }
  }
  return {
    id: -slot.figure.id,
    name: slot.figure.name,
    diameterMm: null,
    images: [],
  }
}

function onWellClicked(coinId: number) {
  if (coinId <= 0) return
  router.push({ name: 'coin-detail', params: { id: coinId } })
}
</script>
