<template>
  <div class="container flex flex-col gap-6">
    <header class="page-header flex flex-nowrap items-center justify-between gap-4">
      <div>
        <p class="section-label">Emperors</p>
        <h1>Stats</h1>
        <p class="mt-[0.35rem] text-base text-text-secondary">
          Completion and coverage metrics for your Emperor tracker.
        </p>
      </div>
      <router-link
        class="inline-flex shrink-0 items-center justify-center rounded-sm border border-border-subtle bg-transparent p-[0.4rem] text-text-secondary transition hover:border-border-accent hover:bg-gold-glow hover:text-gold"
        to="/sets/emperors"
        aria-label="Back to Emperors"
      >
        <ArrowLeft :size="20" />
      </router-link>
    </header>

    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
    </div>

    <div v-else-if="errorMessage" class="card px-8 py-12 text-center">
      <p class="text-text-secondary">{{ errorMessage }}</p>
    </div>

    <template v-else-if="result">
      <section class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <article class="card p-4">
          <p class="section-label mb-1">Overall Completion</p>
          <p class="m-0 text-lg font-semibold text-heading">
            {{ totalOwned }} of {{ totalFigures }}
            <span class="text-base font-normal text-text-secondary">({{ formatPct(totalPct) }}%)</span>
          </p>
        </article>
        <article class="card p-4">
          <p class="section-label mb-1">Remaining Augustuses</p>
          <p class="m-0 text-lg font-semibold text-heading">{{ remainingAugustuses }}</p>
        </article>
        <article class="card p-4">
          <p class="section-label mb-1">Completed Dynasties</p>
          <p class="m-0 text-lg font-semibold text-heading">{{ completedDynasties }} of {{ totalDynasties }}</p>
        </article>
        <article class="card p-4">
          <p class="section-label mb-1">Top Dynasty Progress</p>
          <p class="m-0 text-lg font-semibold text-heading">
            {{ bestDynasty?.dynasty ?? 'N/A' }}
            <span v-if="bestDynasty" class="text-base font-normal text-text-secondary">
              ({{ bestDynasty.owned }} of {{ bestDynasty.total }})
            </span>
          </p>
        </article>
        <article class="card p-4">
          <p class="section-label mb-1">Pursuit Suggestions</p>
          <p class="m-0 text-lg font-semibold text-heading">{{ result.suggestions.length }}</p>
        </article>
      </section>

      <section v-if="categoryRows.length" class="card p-5">
        <h2 class="m-0 mb-3 text-lg text-heading">Category Coverage</h2>
        <div class="flex flex-col gap-2">
          <div
            v-for="category in categoryRows"
            :key="category.label"
            class="flex items-center justify-between gap-3 border-b border-border-subtle pb-2 text-body last:border-0 last:pb-0"
          >
            <span class="text-text-secondary">{{ category.label }}</span>
            <span class="font-semibold text-heading">
              {{ category.owned }} of {{ category.total }}
              <span class="font-normal text-text-secondary">({{ formatPct(category.percentage) }}%)</span>
            </span>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ArrowLeft } from 'lucide-vue-next'
import { getApiErrorMessage, getEmperorTrackerProgress } from '@/api/client'
import type { CategoryProgress, DynastyProgress, EmperorTrackerResult } from '@/types'

const loading = ref(true)
const errorMessage = ref('')
const result = ref<EmperorTrackerResult | null>(null)

const categoryRows = computed(() => {
  if (!result.value) return []
  const rows: Array<{ label: string } & CategoryProgress> = []
  if (result.value.usurpers) rows.push({ label: 'Usurpers', ...result.value.usurpers })
  if (result.value.empresses) rows.push({ label: 'Empresses', ...result.value.empresses })
  if (result.value.other) rows.push({ label: 'Other Figures', ...result.value.other })
  return rows
})

const totalOwned = computed(() => result.value?.emperor.owned ?? 0)
const totalFigures = computed(() => result.value?.emperor.total ?? 0)
const totalPct = computed(() => totalFigures.value > 0 ? (totalOwned.value / totalFigures.value) * 100 : 0)
const remainingAugustuses = computed(() => Math.max(0, (result.value?.emperor.total ?? 0) - (result.value?.emperor.owned ?? 0)))
const emperorDynasties = computed(() => result.value?.emperor.dynasties ?? [])
const totalDynasties = computed(() => emperorDynasties.value.length)
const completedDynasties = computed(() => emperorDynasties.value.filter((dynasty) => dynasty.total > 0 && dynasty.owned >= dynasty.total).length)
const bestDynasty = computed<DynastyProgress | null>(() => {
  if (emperorDynasties.value.length === 0) return null
  return [...emperorDynasties.value].sort((a, b) => {
    const aPct = a.total > 0 ? a.owned / a.total : 0
    const bPct = b.total > 0 ? b.owned / b.total : 0
    if (bPct === aPct) return b.owned - a.owned
    return bPct - aPct
  })[0] ?? null
})

function formatPct(value: number): string {
  return Math.round(value).toString()
}

async function load() {
  loading.value = true
  errorMessage.value = ''
  try {
    const res = await getEmperorTrackerProgress()
    result.value = res.data
  } catch (error) {
    errorMessage.value = getApiErrorMessage(error) || 'Failed to load emperor stats.'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
