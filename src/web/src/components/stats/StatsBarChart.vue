<template>
  <div class="stats-section card">
    <h2 class="mb-5 text-lg">{{ title }}</h2>
    <ZoomableSurface :aria-label="`Zoomable ${title} bar chart. Use controls, wheel, pinch, drag, or keyboard shortcuts to inspect dense rows.`">
      <div class="flex flex-col gap-3 p-3">
        <div
          v-for="item in items"
          :key="String(item.label)"
          class="grid items-center gap-3"
          :class="wide ? 'grid-cols-[150px_minmax(0,1fr)_40px]' : 'grid-cols-[100px_minmax(0,1fr)_40px]'"
        >
          <span class="text-body" :class="wide ? 'truncate whitespace-nowrap' : ''">
            <slot name="label" :item="item">
              {{ item.label }}
            </slot>
          </span>
          <div class="h-6 overflow-hidden rounded-sm bg-surface">
            <div
              class="h-full min-w-1 rounded-sm transition-[width] duration-300"
              :class="fillClass(item.label) === 'fill-roman'
                ? 'bg-[linear-gradient(90deg,var(--cat-roman),var(--accent-gold))]'
                : fillClass(item.label) === 'fill-greek'
                  ? 'bg-[linear-gradient(90deg,var(--cat-greek),var(--accent-gold))]'
                  : fillClass(item.label) === 'fill-byzantine'
                    ? 'bg-[linear-gradient(90deg,var(--cat-byzantine),var(--accent-gold))]'
                    : fillClass(item.label) === 'fill-modern'
                      ? 'bg-[linear-gradient(90deg,var(--cat-modern),var(--accent-gold))]'
                      : fillClass(item.label) === 'fill-other'
                        ? 'bg-[linear-gradient(90deg,var(--cat-other),var(--text-secondary))]'
                        : fillClass(item.label) === 'fill-material'
                          ? 'bg-[linear-gradient(90deg,var(--accent-bronze),var(--accent-gold))]'
                          : fillClass(item.label) === 'fill-grade'
                            ? 'bg-[linear-gradient(90deg,var(--cat-modern),var(--accent-gold))]'
                            : fillClass(item.label) === 'fill-era'
                              ? 'bg-[linear-gradient(90deg,var(--accent-bronze),var(--accent-gold))]'
                              : fillClass(item.label) === 'fill-ruler'
                                ? 'bg-[linear-gradient(90deg,var(--text-muted),var(--accent-gold))]'
                                : 'bg-[linear-gradient(90deg,var(--color-positive),var(--accent-gold))]'"
              :style="{ width: `${(item.count / maxCount) * 100}%` }"
            ></div>
          </div>
          <span class="text-right text-body font-semibold text-text-secondary">{{ item.count }}</span>
        </div>
      </div>
    </ZoomableSurface>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ZoomableSurface from '@/components/ZoomableSurface.vue'

export interface BarItem {
  label: string
  count: number
}

const props = defineProps<{
  title: string
  items: BarItem[]
  fillClass: (label: string) => string
  wide?: boolean
}>()

const maxCount = computed(() =>
  Math.max(...props.items.map((i) => i.count), 1),
)
</script>
