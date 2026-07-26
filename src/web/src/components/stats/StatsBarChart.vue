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
              :style="{
                width: `${(item.count / maxCount) * 100}%`,
                background: `linear-gradient(90deg, ${startColorFor(fillClass(item.label))}, var(--accent-gold))`,
              }"
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
import { colorForLabel } from '@/utils/categoryColor'

export interface BarItem {
  label: string
  count: number
}

const props = defineProps<{
  title: string
  items: BarItem[]
  /**
   * Returns either one of the fixed sentinel keys below (for a chart whose
   * bars all share one static color, e.g. material/grade/era/ruler/price)
   * or, for a per-item-colored chart (categories), the raw label itself -
   * which startColorFor then resolves via the shared color utility so
   * custom categories get their own stable color instead of a fallback.
   */
  fillClass: (label: string) => string
  wide?: boolean
}>()

const STATIC_CHART_COLORS: Record<string, string> = {
  'fill-material': 'var(--accent-bronze)',
  'fill-era': 'var(--accent-bronze)',
  'fill-grade': 'var(--cat-modern)',
  'fill-ruler': 'var(--text-muted)',
  'fill-price': 'var(--color-positive)',
}

function startColorFor(key: string): string {
  return STATIC_CHART_COLORS[key] ?? colorForLabel(key)
}

const maxCount = computed(() =>
  Math.max(...props.items.map((i) => i.count), 1),
)
</script>
