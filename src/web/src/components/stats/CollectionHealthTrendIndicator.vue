<template>
  <div class="card !p-4">
    <div class="mb-4 flex items-center justify-between gap-3">
      <h4 class="m-0 text-base text-heading">30-Day Trend</h4>
      <div
        v-if="trend.direction === 'unavailable'"
        class="inline-flex items-center gap-[0.35rem] rounded-full border border-border-subtle bg-transparent px-[0.7rem] py-[0.2rem] text-sm font-semibold uppercase tracking-[0.08em] text-text-muted"
      >
        Insufficient Data
      </div>
      <div
        v-else
        class="inline-flex items-center gap-[0.35rem] rounded-full border px-[0.7rem] py-[0.2rem] text-sm font-semibold uppercase tracking-[0.08em]"
        :class="trend.direction === 'up'
          ? 'border-[rgba(39,174,96,0.3)] bg-[rgba(39,174,96,0.15)] text-green-400'
          : trend.direction === 'down'
            ? 'border-[rgba(231,76,60,0.3)] bg-[rgba(231,76,60,0.15)] text-red-400'
            : 'border-border-subtle bg-[var(--accent-gold-glow)] text-text-secondary'"
      >
        <component :is="trendIcon" :size="14" />
        {{ formatTrend() }}
      </div>
    </div>
    <div v-if="trend.direction !== 'unavailable'" class="flex items-center justify-center gap-3 rounded-sm bg-input p-3">
      <div class="flex flex-col gap-1">
        <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Change</span>
        <span class="font-display text-xl font-bold text-gold max-md:text-base">{{ formatDelta() }}</span>
      </div>
    </div>
    <p v-else class="m-0 text-body leading-[1.5] text-text-secondary">
      Trend data will be available after 30 days of collection tracking.
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { TrendingUp, TrendingDown, Minus } from 'lucide-vue-next'
import type { CollectionHealthTrend } from '@/types'

const props = defineProps<{
  trend: CollectionHealthTrend
}>()

const trendIcon = computed(() => {
  switch (props.trend.direction) {
    case 'up':
      return TrendingUp
    case 'down':
      return TrendingDown
    default:
      return Minus
  }
})

function formatTrend(): string {
  switch (props.trend.direction) {
    case 'up':
      return 'Improving'
    case 'down':
      return 'Declining'
    case 'flat':
      return 'Stable'
    default:
      return 'No Data'
  }
}

function formatDelta(): string {
  if (props.trend.delta === null) return 'N/A'
  const sign = props.trend.delta > 0 ? '+' : ''
  return `${sign}${props.trend.delta}`
}
</script>
