<template>
  <button
    class="grid w-full grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 rounded-md border border-border-accent bg-card px-[0.85rem] py-[0.65rem] text-left transition-[border-color,background,box-shadow] duration-200 hover:bg-card-hover hover:shadow-card min-[561px]:min-h-20 min-[561px]:gap-[1.15rem] min-[561px]:px-[1.1rem] min-[561px]:py-3"
    type="button"
    @click="$emit('click')"
  >
    <span
      class="h-14 w-[0.45rem] rounded-full shadow-[0_0_16px_var(--accent-gold-glow)] min-[561px]:h-16"
      :style="{ backgroundColor: set.color }"
      aria-hidden="true"
    ></span>
    <span class="flex min-w-0 flex-col gap-1">
      <span class="truncate font-display text-[1.1rem] font-medium tracking-[0.02em] text-heading min-[561px]:text-[1.35rem]">{{ set.name }}</span>
      <span v-if="setDescription" class="text-body text-text-muted">{{ setDescription }}</span>
    </span>
    <span class="flex items-center justify-end gap-[0.85rem]">
      <span class="flex min-w-14 flex-col items-end whitespace-nowrap">
        <span class="font-sans text-[2.25rem] leading-[0.85] font-bold text-gold min-[561px]:text-[2.75rem]">{{ set.coinCount }}</span>
        <span class="text-sm font-semibold text-gold min-[561px]:text-base">{{ set.coinCount === 1 ? 'Coin' : 'Coins' }}</span>
      </span>
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CoinSetSummary } from '@/types'

const props = defineProps<{
  set: CoinSetSummary
}>()

defineEmits<{
  (e: 'click'): void
}>()

const setDescription = computed(() => {
  return props.set.coinCount > 0 ? 'Curated group' : 'Ready for coins'
})
</script>
