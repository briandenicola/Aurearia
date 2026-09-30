<script setup lang="ts">
import { computed } from 'vue'

export type StatusTone = 'success' | 'error' | 'warning' | 'info' | 'neutral'

const props = withDefaults(
  defineProps<{
    tone?: StatusTone
  }>(),
  { tone: 'neutral' }
)

const toneStyle = computed(() => ({
  backgroundColor: `var(--status-${props.tone}-bg)`,
  color: `var(--status-${props.tone}-fg)`,
}))
</script>

<template>
  <span
    class="inline-block rounded-full px-2 py-[0.15rem] text-label font-semibold"
    :style="toneStyle"
  >
    <slot />
  </span>
</template>
