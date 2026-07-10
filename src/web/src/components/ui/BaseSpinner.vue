<template>
  <div
    :class="[
      'inline-flex items-center justify-center',
      wrapperClass,
    ]"
    role="status"
    :aria-label="label"
  >
    <div
      :class="[
        'rounded-full border-[3px] border-border-subtle border-t-gold',
        'animate-spin',
        sizeClass,
      ]"
    />
    <span v-if="showLabel" class="ml-2 text-text-secondary text-base">{{ label }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  size?: 'sm' | 'md' | 'lg'
  label?: string
  showLabel?: boolean
  overlay?: boolean
}>(), {
  size: 'md',
  label: 'Loading…',
  showLabel: false,
  overlay: false,
})

const sizeClass = computed(() => {
  switch (props.size) {
    case 'sm': return 'w-4 h-4'
    case 'lg': return 'w-10 h-10'
    default:   return 'w-8 h-8'
  }
})

const wrapperClass = computed(() =>
  props.overlay
    ? 'flex-col gap-3 p-16 text-text-secondary'
    : '',
)
</script>
