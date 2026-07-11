<template>
  <div class="flex flex-col gap-6">
    <div class="flex flex-col gap-2">
      <h2 class="text-xl font-medium text-heading">Collection Health</h2>
      <p class="text-base leading-relaxed text-text-secondary">
        Aggregate quality metrics across all active user collections.
      </p>
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <div class="spinner"></div>
    </div>

    <div
      v-else-if="error"
      class="flex items-center gap-2 rounded-sm border border-[rgba(231,76,60,0.3)] bg-[rgba(231,76,60,0.15)] p-4 text-base text-[var(--color-negative)]"
    >
      <AlertCircle :size="20" />
      <span>{{ error }}</span>
    </div>

    <div v-else-if="summary" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <div class="flex gap-4 rounded-md border border-border-subtle bg-card p-6 shadow-[var(--shadow-card)] transition-all hover:border-border-accent hover:shadow-[var(--shadow-glow)]">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-sm bg-[rgba(52,152,219,0.15)] text-[#3498db]">
          <Activity :size="24" />
        </div>
        <div class="flex flex-1 flex-col gap-[0.35rem]">
          <div class="section-label">Median Score</div>
          <div class="font-['Cinzel'] text-[1.75rem] font-bold leading-none text-gold md:text-2xl">{{ summary.medianScore }}</div>
          <div class="text-base text-text-secondary">
            Across {{ summary.eligibleCoinCount }} active coins
          </div>
        </div>
      </div>

      <div class="flex gap-4 rounded-md border border-border-subtle bg-card p-6 shadow-[var(--shadow-card)] transition-all hover:border-border-accent hover:shadow-[var(--shadow-glow)]">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-sm bg-[rgba(243,156,18,0.15)] text-[#f39c12]">
          <AlertTriangle :size="24" />
        </div>
        <div class="flex flex-1 flex-col gap-[0.35rem]">
          <div class="section-label">Low-Score Coins</div>
          <div class="font-['Cinzel'] text-[1.75rem] font-bold leading-none text-gold md:text-2xl">{{ summary.lowScorePercentage.toFixed(1) }}%</div>
          <div class="text-base text-text-secondary">
            Below {{ summary.lowScoreThreshold }}
          </div>
        </div>
      </div>

      <div class="flex gap-4 rounded-md border border-border-subtle bg-card p-6 shadow-[var(--shadow-card)] transition-all hover:border-border-accent hover:shadow-[var(--shadow-glow)] md:col-span-2 xl:col-span-3">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-sm bg-[rgba(231,76,60,0.15)] text-[#e74c3c]">
          <FileWarning :size="24" />
        </div>
        <div class="flex flex-1 flex-col gap-[0.35rem]">
          <div class="section-label">Top Missing Fields</div>
          <div v-if="summary.topMissingFields.length === 0" class="text-base italic text-text-muted">
            No missing fields detected
          </div>
          <div v-else class="mt-1 flex flex-col gap-2">
            <div
              v-for="(field, idx) in summary.topMissingFields"
              :key="idx"
              class="flex items-center justify-between gap-3 rounded-sm border border-border-subtle bg-input px-3 py-2"
            >
              <span class="text-base font-medium text-text-primary">{{ field.key }}</span>
              <span class="chip-sm">{{ field.count }} coins</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="flex flex-col items-center gap-4 rounded-md border border-border-subtle bg-card px-8 py-12 text-center text-text-muted">
      <FileQuestion :size="48" />
      <p class="m-0 text-base text-text-secondary">No collection health data available</p>
    </div>
  </div>
</template>



<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { AlertCircle, Activity, AlertTriangle, FileWarning, FileQuestion } from 'lucide-vue-next'
import * as api from '@/api/client'
import type { AdminHealthSummaryResponse } from '@/types'

const summary = ref<AdminHealthSummaryResponse | null>(null)
const loading = ref(false)
const error = ref('')

onMounted(() => {
  fetchAdminHealth()
})

async function fetchAdminHealth() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.getAdminHealthSummary()
    summary.value = res.data
  } catch (err) {
    error.value = 'Failed to load health metrics'
    console.error('Admin health fetch error:', err)
  } finally {
    loading.value = false
  }
}
</script>
