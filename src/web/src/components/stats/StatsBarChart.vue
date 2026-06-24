<template>
  <div class="stats-section card">
    <h2>{{ title }}</h2>
    <ZoomableSurface :aria-label="`Zoomable ${title} bar chart. Use controls, wheel, pinch, drag, or keyboard shortcuts to inspect dense rows.`">
      <div class="bar-chart">
        <div
          v-for="item in items"
          :key="String(item.label)"
          class="bar-row"
          :class="{ 'bar-row-wide': wide }"
        >
          <span class="bar-label" :class="{ 'bar-label-wide': wide }">
            <slot name="label" :item="item">
              {{ item.label }}
            </slot>
          </span>
          <div class="bar-track">
            <div
              class="bar-fill"
              :class="fillClass(item.label)"
              :style="{ width: `${(item.count / maxCount) * 100}%` }"
            ></div>
          </div>
          <span class="bar-value">{{ item.count }}</span>
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

<style scoped>
.stats-section h2 {
  margin-bottom: 1.25rem;
  font-size: 1.1rem;
}

.bar-chart {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0.75rem;
}

.bar-row {
  display: grid;
  grid-template-columns: 100px 1fr 40px;
  gap: 0.75rem;
  align-items: center;
}

.bar-label {
  font-size: 0.85rem;
}

.bar-track {
  height: 24px;
  background: var(--bg-primary);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.bar-fill {
  height: 100%;
  border-radius: var(--radius-sm);
  transition: width var(--transition-med);
  min-width: 4px;
}

/* Category fills */
.fill-roman { background: linear-gradient(90deg, var(--cat-roman), var(--accent-gold)); }
.fill-greek { background: linear-gradient(90deg, var(--cat-greek), var(--accent-gold)); }
.fill-byzantine { background: linear-gradient(90deg, var(--cat-byzantine), var(--accent-gold)); }
.fill-modern { background: linear-gradient(90deg, var(--cat-modern), var(--accent-gold)); }
.fill-other { background: linear-gradient(90deg, var(--cat-other), var(--text-secondary)); }
.fill-material { background: linear-gradient(90deg, var(--accent-bronze), var(--accent-gold)); }
.fill-grade { background: linear-gradient(90deg, var(--cat-modern), var(--accent-gold)); }
.fill-era { background: linear-gradient(90deg, var(--accent-bronze), var(--accent-gold)); }
.fill-ruler { background: linear-gradient(90deg, var(--text-muted), var(--accent-gold)); }
.fill-price { background: linear-gradient(90deg, var(--color-positive), var(--accent-gold)); }

.bar-row-wide {
  grid-template-columns: 150px 1fr 40px;
}

.bar-label-wide {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.bar-value {
  font-size: 0.85rem;
  font-weight: 600;
  text-align: right;
  color: var(--text-secondary);
}
</style>
