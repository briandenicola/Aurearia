<template>
  <div ref="pullContainer" :style="pullDistance > 0 ? `transform: translateY(${pullDistance}px); transition: none` : ''">
    <div
      class="fixed left-1/2 z-[100] flex -translate-x-1/2 items-center gap-2 rounded-full border border-border-subtle bg-card px-4 py-1.5 shadow-[0_2px_12px_rgba(0,0,0,0.3)]"
      :class="pullDistance > 0 || refreshing ? 'pointer-events-auto' : 'pointer-events-none'"
      :style="`top: ${-50 + pullDistance * 0.6}px; opacity: ${Math.min(pullDistance / 60, 1)}`"
    >
      <div class="h-[18px] w-[18px] rounded-full border-2 border-border-subtle border-t-gold" :class="{ 'animate-spin': refreshing }" :style="refreshing ? '' : `transform: rotate(${pullDistance * 3}deg)`"></div>
      <span class="whitespace-nowrap text-sm text-text-secondary">{{ refreshing ? 'Refreshing...' : pullDistance >= 60 ? 'Release to refresh' : 'Pull to refresh' }}</span>
    </div>
    <slot />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { usePullToRefresh } from '@/composables/usePullToRefresh'

const props = defineProps<{
  onRefresh: () => Promise<void>
}>()

const pullContainer = ref<HTMLElement | null>(null)
const { pullDistance, refreshing } = usePullToRefresh(pullContainer, props.onRefresh)
</script>
