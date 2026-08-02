<template>
  <div class="relative">
    <slot name="trigger" :open="open" :toggle="toggle" :close="close" />
    <div
      v-if="open"
      class="absolute right-0 top-[calc(100%+0.45rem)] z-30 rounded-md border border-border-subtle bg-card p-2 shadow-[0_10px_26px_rgba(0,0,0,0.45)]"
      :class="panelWidthClass"
    >
      <slot :close="close" />
    </div>
    <button v-if="open" class="fixed inset-0 z-20 bg-transparent" :aria-label="backdropAriaLabel" @click="close"></button>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

withDefaults(defineProps<{
  panelWidthClass?: string
  backdropAriaLabel?: string
}>(), {
  panelWidthClass: 'w-[260px]',
  backdropAriaLabel: 'Close overflow menu',
})

const open = ref(false)

function close() {
  open.value = false
}

function toggle() {
  open.value = !open.value
}
</script>
