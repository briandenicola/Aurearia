<template>
  <div class="stats-section card">
    <h2>{{ title }}</h2>
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
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

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
  transition: width 0.5s ease;
  min-width: 4px;
}

/* Category fills */
.fill-roman { background: linear-gradient(90deg, #7b2d8e, #9b59b6); }
.fill-greek { background: linear-gradient(90deg, #4a6e18, #6b8e23); }
.fill-byzantine { background: linear-gradient(90deg, #8b1a1a, #c0392b); }
.fill-modern { background: linear-gradient(90deg, #2c5f8a, #4682b4); }
.fill-other { background: linear-gradient(90deg, #555, #888); }
.fill-material { background: linear-gradient(90deg, var(--accent-bronze), var(--accent-gold)); }
.fill-grade { background: linear-gradient(90deg, #2c5f8a, #7ab3d4); }
.fill-era { background: linear-gradient(90deg, #6b4c3b, #a67c52); }
.fill-ruler { background: linear-gradient(90deg, #8b6914, var(--accent-gold)); }
.fill-price { background: linear-gradient(90deg, #2e7d32, #66bb6a); }

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
