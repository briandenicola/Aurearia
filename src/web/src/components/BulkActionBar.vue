<template>
  <Transition
    enter-active-class="transition-all duration-300 ease-out"
    enter-from-class="translate-y-5 opacity-0"
    leave-active-class="transition-all duration-300 ease-in"
    leave-to-class="translate-y-5 opacity-0"
  >
    <div v-if="visible" class="fixed bottom-[calc(1rem+env(safe-area-inset-bottom))] left-1/2 z-[200] flex w-[calc(100vw-2rem)] max-w-[calc(100vw-2rem)] -translate-x-1/2 flex-col items-stretch gap-[0.6rem] rounded-md border border-gold-dim bg-card px-3 py-3 whitespace-normal shadow-[0_8px_30px_rgba(0,0,0,0.5)] min-[641px]:bottom-6 min-[641px]:w-auto min-[641px]:max-w-[calc(100vw-2rem)] min-[641px]:flex-row min-[641px]:items-center min-[641px]:gap-4 min-[641px]:px-5 min-[641px]:whitespace-nowrap">
      <span class="text-center text-body font-medium text-text-secondary min-[641px]:text-left">{{ selectedCount }} coin{{ selectedCount === 1 ? '' : 's' }} selected</span>
      <div class="grid grid-cols-2 gap-2 min-[641px]:flex min-[641px]:flex-wrap min-[641px]:justify-center min-[641px]:gap-2">
        <button class="btn btn-ghost btn-sm min-w-0 justify-center" @click="$emit('tag')">
          <TagIcon :size="16" /> Tag
        </button>
        <button class="btn btn-ghost btn-sm min-w-0 justify-center" @click="$emit('location')">
          <MapPin :size="16" /> Assign Location
        </button>
        <button class="btn btn-ghost btn-sm min-w-0 justify-center hover:border-gain hover:text-gain" @click="$emit('sell')">
          <DollarSign :size="16" /> Mark Sold
        </button>
        <button class="btn btn-ghost btn-sm min-w-0 justify-center hover:border-loss hover:text-loss" @click="$emit('delete')">
          <Trash2 :size="16" /> Delete
        </button>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { Tag as TagIcon, DollarSign, Trash2, MapPin } from 'lucide-vue-next'

defineProps<{
  visible: boolean
  selectedCount: number
}>()

defineEmits<{
  tag: []
  location: []
  sell: []
  delete: []
}>()
</script>
