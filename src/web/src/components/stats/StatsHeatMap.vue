<template>
  <div class="stats-section card">
    <h2 class="mb-5 text-[1.1rem]">Collection Distribution</h2>
    <div v-if="heatMapEras.length && heatMapCategories.length" class="flex flex-col gap-3">
      <ZoomableSurface aria-label="Zoomable collection distribution heat map. Use controls, wheel, pinch, drag, or keyboard shortcuts to inspect dense cells.">
        <div class="grid min-w-[300px] gap-[2px] p-3 max-[480px]:min-w-0 max-[480px]:gap-px" :style="{ gridTemplateColumns: `minmax(60px, 80px) repeat(${heatMapCategories.length}, 1fr)` }">
          <div></div>
          <div v-for="cat in heatMapCategories" :key="cat" class="overflow-hidden text-ellipsis whitespace-nowrap px-[0.15rem] py-[0.25rem] text-center text-[0.65rem] font-semibold text-text-secondary max-[480px]:px-[0.1rem] max-[480px]:py-[0.2rem] max-[480px]:text-[0.55rem]">{{ cat }}</div>
          <template v-for="era in heatMapEras" :key="era">
            <div class="flex max-w-[80px] items-center overflow-hidden text-ellipsis whitespace-nowrap pr-[0.35rem] text-[0.65rem] font-semibold text-text-secondary max-[480px]:max-w-[60px] max-[480px]:pr-[0.2rem] max-[480px]:text-[0.55rem]">{{ era }}</div>
            <div
              v-for="cat in heatMapCategories"
              :key="`${era}-${cat}`"
              class="flex aspect-square min-h-[28px] cursor-pointer items-center justify-center rounded-sm border border-border-subtle transition hover:z-10 hover:scale-110 hover:shadow-[var(--shadow-gold-hover)] max-[480px]:min-h-[24px]"
              :style="{ backgroundColor: cellColor(heatMapData[`${era}|${cat}`] ?? 0) }"
              :title="`${era} / ${cat}: ${heatMapData[`${era}|${cat}`] ?? 0} coins`"
              @click="navigateToFiltered(era, cat)"
            >
              <span v-if="(heatMapData[`${era}|${cat}`] ?? 0) > 0" class="text-[0.65rem] font-bold text-text-primary max-[480px]:text-[0.55rem]">{{ heatMapData[`${era}|${cat}`] }}</span>
            </div>
          </template>
        </div>
      </ZoomableSurface>
      <div class="mt-3 flex items-center justify-center gap-2">
        <span class="text-label text-text-muted">0</span>
        <div class="h-[10px] w-[120px] rounded-full bg-[linear-gradient(to_right,color-mix(in_srgb,var(--color-gold)_10%,transparent),color-mix(in_srgb,var(--color-gold)_85%,transparent))]"></div>
        <span class="text-label text-text-muted">{{ heatMapMax }}</span>
      </div>
    </div>
    <p v-else class="py-4 text-body italic text-text-muted">Add coins with era and category to see distribution.</p>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { getDistribution } from '@/api/client'
import ZoomableSurface from '@/components/ZoomableSurface.vue'

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
  if (count === 0) return 'color-mix(in srgb, var(--accent-gold) 5%, transparent)'
  const intensity = Math.min(count / heatMapMax.value, 1)
  const percentage = Math.round((0.15 + intensity * 0.7) * 100)
  return `color-mix(in srgb, var(--accent-gold) ${percentage}%, transparent)`
}

function navigateToFiltered(era: string, category: string) {
  if ((heatMapData.value[`${era}|${category}`] ?? 0) > 0) {
    router.push({ path: '/', query: { category } })
  }
}

defineExpose({ fetchDistribution })
</script>
