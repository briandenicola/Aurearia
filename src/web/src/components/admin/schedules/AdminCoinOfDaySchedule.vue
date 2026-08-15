<template>
  <hr class="my-6 border-0 border-t border-border-subtle" />

  <!-- Coin of the Day -->
  <h3 class="mb-4 text-base font-semibold text-text-primary">Coin of the Day</h3>
  <p class="mb-4 text-base text-text-secondary">Picks one coin per day from each user's collection and sends an in-app and Pushover notification. Each coin in a user's collection appears once before any coin repeats.</p>
  <div class="mb-4">
    <div class="form-group flex items-center justify-between gap-3">
      <label class="form-label">Enable Daily Feature</label>
      <label class="relative inline-block h-[22px] w-[42px]">
        <input
          class="peer sr-only" type="checkbox"
          :checked="settings.CoinOfDayEnabled === 'true'"
          @change="settings.CoinOfDayEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
        />
        <span class="absolute inset-0 rounded-full border border-border-subtle bg-surface transition-colors after:absolute after:bottom-[2px] after:left-[2px] after:h-4 after:w-4 after:rounded-full after:bg-[var(--text-secondary)] after:transition-transform peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] peer-checked:after:translate-x-5 peer-checked:after:bg-gold peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2"></span>
      </label>
    </div>
    <div class="form-group">
      <label class="form-label">Start Time (daily)</label>
      <input
        v-model="settings.CoinOfDayStartTime"
        class="form-input w-full max-w-[120px]"
        type="time"
      />
      <span class="form-hint">Time of day when the daily featured coin is picked for each enrolled user.</span>
    </div>
    <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
      <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="emit('save')">
        {{ settingsSaving ? 'Saving...' : 'Save Coin of the Day Settings' }}
      </button>
      <span v-if="settingsMsg" class="text-body text-gold md:mr-auto" :class="settingsError ? 'text-[var(--color-negative)]' : ''">{{ settingsMsg }}</span>
      <button class="btn btn-secondary btn-sm md:ml-auto" :disabled="triggerLoading" @click="triggerManualCoinOfDay()">
        {{ triggerLoading ? 'Running...' : 'Run Now' }}
      </button>
    </div>
  </div>

  <hr class="my-6 border-0 border-t border-border-subtle" />
  <h3 class="mb-4 text-base font-semibold text-text-primary">Coin of the Day Run History</h3>
  <div v-if="loading" class="flex justify-center py-8"><div class="spinner"></div></div>
  <div v-else-if="runs.length === 0" class="px-8 py-8 text-center font-sans text-text-muted">No Coin of the Day runs recorded yet.</div>
  <template v-else>
    <table class="w-full border-collapse text-[0.8rem] md:table-fixed md:text-[0.82rem] [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-[0.35rem] [&_th]:py-2 [&_th]:text-left [&_th]:text-sm [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted md:[&_th]:px-2 md:[&_th]:py-3 [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-[0.35rem] [&_td]:py-2 [&_td]:text-left md:[&_td]:px-2 md:[&_td]:py-3">
      <thead>
        <tr>
          <th>Date</th>
          <th>Status</th>
          <th>Picked</th>
          <th>Skipped</th>
          <th>Errors</th>
          <th class="hidden md:table-cell">Trigger</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="run in runs" :key="run.id">
          <td class="text-body text-text-secondary">{{ formatDate(run.startedAt) }}</td>
          <td>{{ run.status }}</td>
          <td>{{ run.picked }}</td>
          <td>{{ run.skipped }}</td>
          <td>{{ run.errors }}</td>
          <td class="hidden md:table-cell">{{ run.triggerType }}</td>
        </tr>
      </tbody>
    </table>

    <div class="mt-4 flex items-center justify-center gap-3">
      <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="prevPage()">Prev</button>
      <span class="text-[0.82rem] text-text-secondary">Page {{ page }}</span>
      <button class="btn btn-secondary btn-sm" :disabled="runs.length < 5" @click="nextPage()">Next</button>
    </div>
  </template>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { getCoinOfDayRunDetail, getCoinOfDayRuns, triggerCoinOfDayRun } from '@/api/client'
import { useRunHistoryPagination } from '@/composables/useRunHistoryPagination'
import type { AppSettings, CoinOfDayRun } from '@/types'

defineProps<{
  settings: AppSettings
  settingsSaving: boolean
}>()

const emit = defineEmits<{
  save: []
}>()

const triggerLoading = ref(false)
const settingsMsg = ref('')
const settingsError = ref(false)
const timers: ReturnType<typeof setTimeout>[] = []
let pollTimer: ReturnType<typeof setInterval> | null = null

const {
  runs,
  page,
  loading,
  loadRuns,
  prevPage,
  nextPage,
} = useRunHistoryPagination<CoinOfDayRun>(async (currentPage, limit) => {
  const res = await getCoinOfDayRuns(currentPage, limit)
  return res.data ?? {}
})

function runIsTerminal(runStatus: string) {
  return runStatus === 'completed' || runStatus === 'failed'
}

function refreshPolling() {
  const hasActive = runs.value.some(run => !runIsTerminal(run.status))
  if (hasActive && !pollTimer) {
    pollTimer = setInterval(() => { loadRuns() }, 5000)
  } else if (!hasActive && pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }

  watch(runs, () => {
    refreshPolling()
  })
}

async function triggerManualCoinOfDay() {
  triggerLoading.value = true
  settingsMsg.value = ''
  settingsError.value = false
  try {
    const res = await triggerCoinOfDayRun()
    const runId = Number(res.data.runId ?? 0)
    settingsMsg.value = runId ? `Coin of the Day run #${runId} queued` : 'Coin of the Day run queued'
    if (runId) {
      const detail = await getCoinOfDayRunDetail(runId)
      const run = detail.data
      if (runIsTerminal(run.status)) {
        settingsMsg.value = `Picked ${run.picked}, skipped ${run.skipped}${run.errors ? `, errors ${run.errors}` : ''}`
        settingsError.value = run.status === 'failed'
      }
    }
    await loadRuns()
    refreshPolling()
    timers.push(setTimeout(() => { settingsMsg.value = '' }, 10000))
  } catch {
    settingsMsg.value = 'Failed to run Coin of the Day'
    settingsError.value = true
  } finally {
    triggerLoading.value = false
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString()
}

onMounted(() => {
  loadRuns()
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
  timers.forEach(clearTimeout)
})
</script>
