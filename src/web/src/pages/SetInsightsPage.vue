<template>
  <div class="container pb-4 md:pb-6">
    <div v-if="loading" class="py-12 text-center text-text-secondary">
      Loading set insights...
    </div>

    <div v-else-if="set" class="space-y-6">
      <div class="page-header items-start">
        <div class="flex min-w-0 flex-1 items-start gap-3 md:items-center">
          <span class="h-11 w-1 shrink-0 rounded-full shadow-[0_0_16px_var(--accent-gold-glow)]" :style="{ backgroundColor: set.color }" aria-hidden="true"></span>
          <div class="min-w-0">
            <h1>{{ set.name }}</h1>
            <p class="section-label mb-0 mt-1 inline-flex items-center gap-1.5">
              <Info :size="13" />
              Set information
            </p>
          </div>
        </div>
        <div class="header-actions">
          <button class="btn btn-ghost" @click="router.push({ name: 'set-detail', params: { id: setId } })">
            <ArrowLeft :size="16" /> Back to Set
          </button>
        </div>
      </div>

      <section v-if="analytics" class="card p-6">
        <h2 class="mt-0">Analytics</h2>
        <div class="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(150px,1fr))]">
          <div class="flex flex-col gap-1.5">
            <span class="text-body text-text-secondary">ROI</span>
            <strong class="text-lg text-gold">{{ analytics.roiPercent == null ? 'N/A' : `${analytics.roiPercent.toFixed(1)}%` }}</strong>
          </div>
          <div class="flex flex-col gap-1.5">
            <span class="text-body text-text-secondary">Acquisition Rate</span>
            <strong class="text-lg text-gold">{{ analytics.acquisitionRatePerMonth == null ? 'N/A' : `${analytics.acquisitionRatePerMonth.toFixed(1)}/mo` }}</strong>
          </div>
        </div>
      </section>

      <SetTrendChart
        :snapshots="snapshots"
        :range="trendRange"
        @update:range="changeTrendRange"
      >
        <template #actions>
          <button class="btn btn-secondary" @click="captureSnapshot">Capture Snapshot</button>
        </template>
      </SetTrendChart>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Info } from 'lucide-vue-next'
import { createSetSnapshot, getSet, getSetAnalytics, getSetTrends } from '@/api/client'
import type { CoinSetAnalytics, CoinSetDetail, CoinSetSnapshot } from '@/types'
import SetTrendChart from '@/components/sets/SetTrendChart.vue'

const router = useRouter()
const route = useRoute()
const setId = Number(route.params.id)

const loading = ref(true)
const set = ref<CoinSetDetail | null>(null)
const analytics = ref<CoinSetAnalytics | null>(null)
const snapshots = ref<CoinSetSnapshot[]>([])
const trendRange = ref('1y')

onMounted(async () => {
  await loadInsights()
})

async function loadInsights() {
  loading.value = true
  try {
    const [setRes, analyticsRes, trendsRes] = await Promise.all([
      getSet(setId),
      getSetAnalytics(setId),
      getSetTrends(setId, trendRange.value),
    ])
    set.value = setRes.data
    analytics.value = analyticsRes.data
    snapshots.value = trendsRes.data.snapshots
  } catch (error) {
    console.error('Failed to load set insights:', error)
  } finally {
    loading.value = false
  }
}

async function changeTrendRange(range: string) {
  trendRange.value = range
  const res = await getSetTrends(setId, trendRange.value)
  snapshots.value = res.data.snapshots
}

async function captureSnapshot() {
  try {
    await createSetSnapshot(setId)
    await changeTrendRange(trendRange.value)
    const analyticsRes = await getSetAnalytics(setId)
    analytics.value = analyticsRes.data
  } catch (error) {
    console.error('Failed to capture snapshot:', error)
    alert('Failed to capture snapshot')
  }
}
</script>
