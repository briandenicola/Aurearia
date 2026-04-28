<template>
  <div class="stats-section card">
    <h2>Collection Distribution</h2>
    <div v-if="heatMapEras.length && heatMapCategories.length" class="heatmap-container">
      <div class="heatmap-grid" :style="{ gridTemplateColumns: `minmax(60px, 80px) repeat(${heatMapCategories.length}, 1fr)` }">
        <div class="heatmap-corner"></div>
        <div v-for="cat in heatMapCategories" :key="cat" class="heatmap-col-header">{{ cat }}</div>
        <template v-for="era in heatMapEras" :key="era">
          <div class="heatmap-row-header">{{ era }}</div>
          <div
            v-for="cat in heatMapCategories"
            :key="`${era}-${cat}`"
            class="heatmap-cell"
            :style="{ backgroundColor: cellColor(heatMapData[`${era}|${cat}`] ?? 0) }"
            :title="`${era} / ${cat}: ${heatMapData[`${era}|${cat}`] ?? 0} coins`"
            @click="navigateToFiltered(era, cat)"
          >
            <span v-if="(heatMapData[`${era}|${cat}`] ?? 0) > 0" class="heatmap-count">{{ heatMapData[`${era}|${cat}`] }}</span>
          </div>
        </template>
      </div>
      <div class="heatmap-legend">
        <span class="heatmap-legend-label">0</span>
        <div class="heatmap-legend-bar"></div>
        <span class="heatmap-legend-label">{{ heatMapMax }}</span>
      </div>
    </div>
    <p v-else class="chart-empty">Add coins with era and category to see distribution.</p>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { getDistribution } from '@/api/client'

const router = useRouter()

const heatMapData = ref<Record<string, number>>({})
const heatMapEras = ref<string[]>([])
const heatMapCategories = ref<string[]>([])
const heatMapMax = ref(1)

async function fetchDistribution() {
  try {
    const res = await getDistribution()
    const cells = res.data?.cells ?? []
    const eras = new Set<string>()
    const cats = new Set<string>()
    const map: Record<string, number> = {}
    let max = 1
    for (const cell of cells) {
      eras.add(cell.era)
      cats.add(cell.category)
      map[`${cell.era}|${cell.category}`] = cell.count
      if (cell.count > max) max = cell.count
    }
    heatMapEras.value = [...eras].sort()
    heatMapCategories.value = [...cats].sort()
    heatMapData.value = map
    heatMapMax.value = max
  } catch { /* ignore */ }
}

function cellColor(count: number): string {
  if (count === 0) return 'rgba(191, 155, 48, 0.05)'
  const intensity = Math.min(count / heatMapMax.value, 1)
  const alpha = 0.15 + intensity * 0.7
  return `rgba(191, 155, 48, ${alpha.toFixed(2)})`
}

function navigateToFiltered(era: string, category: string) {
  if ((heatMapData.value[`${era}|${category}`] ?? 0) > 0) {
    router.push({ path: '/', query: { category } })
  }
}

defineExpose({ fetchDistribution })
</script>

<style scoped>
.stats-section h2 {
  margin-bottom: 1.25rem;
  font-size: 1.1rem;
}

.chart-empty {
  color: var(--text-muted);
  font-size: 0.85rem;
  font-style: italic;
  padding: 1rem 0;
}

.heatmap-container {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.heatmap-grid {
  display: grid;
  gap: 2px;
  min-width: 300px;
}

.heatmap-corner {
  background: transparent;
}

.heatmap-col-header {
  font-size: 0.65rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-align: center;
  padding: 0.25rem 0.15rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.heatmap-row-header {
  font-size: 0.65rem;
  font-weight: 600;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  padding-right: 0.35rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 80px;
}

.heatmap-cell {
  aspect-ratio: 1;
  min-height: 28px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s;
  border: 1px solid rgba(191, 155, 48, 0.1);
}

.heatmap-cell:hover {
  transform: scale(1.1);
  box-shadow: 0 0 8px rgba(191, 155, 48, 0.4);
  z-index: 1;
}

.heatmap-count {
  font-size: 0.65rem;
  font-weight: 700;
  color: var(--text-primary);
}

.heatmap-legend {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.75rem;
  justify-content: center;
}

.heatmap-legend-label {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.heatmap-legend-bar {
  width: 120px;
  height: 10px;
  border-radius: 5px;
  background: linear-gradient(to right, rgba(191, 155, 48, 0.1), rgba(191, 155, 48, 0.85));
}

@media (max-width: 480px) {
  .heatmap-grid {
    gap: 1px;
    min-width: 0;
  }
  .heatmap-col-header {
    font-size: 0.55rem;
    padding: 0.2rem 0.1rem;
  }
  .heatmap-row-header {
    font-size: 0.55rem;
    max-width: 60px;
    padding-right: 0.2rem;
  }
  .heatmap-cell {
    min-height: 24px;
    border-radius: 3px;
  }
  .heatmap-count {
    font-size: 0.55rem;
  }
}
</style>
