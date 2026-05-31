<template>
  <div class="era-filter">
    <button
      class="chip"
      :class="{ active: !modelValue }"
      @click="$emit('update:modelValue', '')"
    >
      All Eras
    </button>
    <button
      v-for="era in COIN_ERAS"
      :key="era"
      class="chip"
      :class="{ active: modelValue === era }"
      @click="$emit('update:modelValue', era)"
    >
      {{ formatEra(era) }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { COIN_ERAS } from '@/types'
import type { CoinEra } from '@/types'

defineProps<{ modelValue: string }>()
defineEmits<{ 'update:modelValue': [value: '' | CoinEra] }>()

function formatEra(era: CoinEra): string {
  return era.charAt(0).toUpperCase() + era.slice(1)
}
</script>

<style scoped>
.era-filter {
  display: flex;
  gap: 0.4rem;
  flex-wrap: wrap;
}

.chip {
  text-transform: capitalize;
}
</style>
