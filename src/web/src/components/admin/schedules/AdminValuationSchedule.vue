<template>
  <hr class="my-6 border-0 border-t border-border-subtle" />

  <!-- Collection Valuation -->
  <h3 class="mb-4 text-base font-semibold text-text-primary">Collection Valuation</h3>
  <div class="mb-4">
    <div class="form-group flex items-center justify-between gap-3">
      <label class="form-label">Enable Scheduled Valuation</label>
      <label class="relative inline-block h-[22px] w-[42px]">
        <input
          class="peer sr-only" type="checkbox"
          :checked="settings.ValuationCheckEnabled === 'true'"
          @change="settings.ValuationCheckEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
        />
        <span class="absolute inset-0 rounded-full border border-border-subtle bg-surface transition-colors after:absolute after:bottom-[2px] after:left-[2px] after:h-4 after:w-4 after:rounded-full after:bg-[var(--text-secondary)] after:transition-transform peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] peer-checked:after:translate-x-5 peer-checked:after:bg-gold peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2"></span>
      </label>
    </div>
    <div class="form-group">
      <label class="form-label">Start Time (daily anchor)</label>
      <input
        v-model="settings.ValuationCheckStartTime"
        class="form-input w-full max-w-[120px]"
        type="time"
      />
      <span class="form-hint">The valuation cycle starts at this time on scheduled days.</span>
    </div>
    <div class="form-group">
      <label class="form-label">Repeat Interval (days)</label>
      <input
        v-model="settings.ValuationCheckIntervalDays"
        class="form-input w-full max-w-[120px]"
        type="number"
        min="1"
        step="1"
      />
      <span class="form-hint">How often to run (e.g. 7 = weekly). AI valuations are costly so daily runs are not recommended.</span>
    </div>
    <div class="form-group">
      <label class="form-label">Max Coins Per Run</label>
      <input
        v-model="settings.ValuationMaxCoins"
        class="form-input w-full max-w-[120px]"
        type="number"
        min="1"
        step="10"
      />
      <span class="form-hint">Limit how many coins are valuated per run to control AI costs.</span>
    </div>
    <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
      <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="emit('save')">
        {{ settingsSaving ? 'Saving...' : 'Save Valuation Settings' }}
      </button>
      <span v-if="settingsMsg" class="text-body text-gold md:mr-auto" :class="settingsError ? 'text-[var(--color-negative)]' : ''">{{ settingsMsg }}</span>
      <button class="btn btn-secondary btn-sm md:ml-auto" :disabled="triggerLoading" @click="triggerManualValuation()">
        {{ triggerLoading ? 'Starting...' : 'Run Now' }}
      </button>
    </div>
  </div>

  <hr class="my-6 border-0 border-t border-border-subtle" />
  <h3 class="mb-4 text-base font-semibold text-text-primary">Valuation Run History</h3>

  <div v-if="loading" class="flex justify-center py-8"><div class="spinner"></div></div>
  <div v-else-if="runs.length === 0" class="px-8 py-8 text-center font-sans text-text-muted">No valuation runs recorded yet.</div>
  <template v-else>
    <table class="w-full border-collapse text-[0.8rem] md:table-fixed md:text-[0.82rem] [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-[0.35rem] [&_th]:py-2 [&_th]:text-left [&_th]:text-sm [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted md:[&_th]:px-2 md:[&_th]:py-3 [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-[0.35rem] [&_td]:py-2 [&_td]:text-left md:[&_td]:px-2 md:[&_td]:py-3">
      <thead>
        <tr>
          <th>Date</th>
          <th class="hidden md:table-cell">Trigger</th>
          <th>Status</th>
          <th>Checked</th>
          <th class="hidden md:table-cell">Updated</th>
          <th class="hidden md:table-cell">Skipped</th>
          <th class="hidden md:table-cell">Errors</th>
          <th>Duration</th>
        </tr>
      </thead>
      <tbody>
        <template v-for="run in runs" :key="run.id">
          <tr class="cursor-pointer transition-colors hover:bg-surface" :class="{ 'bg-surface': expandedRunId === run.id }" @click="toggleRunDetail(run.id)">
            <td class="text-body text-text-secondary">{{ formatDate(run.startedAt) }}</td>
            <td class="hidden md:table-cell">{{ run.triggerType }}</td>
            <td>
              <span class="inline-block rounded-full px-2 py-[0.15rem] text-label font-semibold" :class="run.status === 'running' ? 'bg-[rgba(52,152,219,0.15)] text-[#3498db]' : run.status === 'completed' ? 'bg-[rgba(46,204,113,0.15)] text-[var(--color-positive)]' : run.status === 'failed' ? 'bg-[rgba(231,76,60,0.15)] text-[var(--color-negative)]' : 'bg-[rgba(243,156,18,0.15)] text-[#f39c12]'">{{ run.status }}</span>
              <span v-if="run.status === 'running' && run.totalCoins > 0" class="ml-[0.35rem] text-label font-medium text-text-secondary">
                {{ run.coinsChecked + run.coinsSkipped + run.errors }} / {{ run.totalCoins }}
              </span>
              <button v-if="run.status === 'running'" class="ml-[0.4rem] rounded-full border border-[rgba(231,76,60,0.4)] bg-transparent px-[0.4rem] py-[0.1rem] text-[0.65rem] text-[var(--color-negative)] transition-colors hover:bg-[rgba(231,76,60,0.15)]" @click.stop="cancelRun(run.id)">Cancel</button>
            </td>
            <td>{{ run.coinsChecked }}</td>
            <td class="hidden font-semibold text-[var(--color-positive)] md:table-cell">{{ run.coinsUpdated }}</td>
            <td class="hidden font-semibold text-warning md:table-cell">{{ run.coinsSkipped }}</td>
            <td class="hidden font-semibold text-[var(--color-negative)] md:table-cell">{{ run.errors }}</td>
            <td>{{ formatDuration(run.durationMs) }}</td>
          </tr>
          <tr v-if="expandedRunId === run.id && expandedResults" class="bg-surface-secondary">
            <td :colspan="colspan">
              <div v-if="expandedLoading" class="flex justify-center py-8"><div class="spinner"></div></div>
              <table v-else-if="expandedResults.length" class="w-full border-collapse text-[0.78rem] md:table-fixed [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-2 [&_th]:py-[0.4rem] [&_th]:text-left [&_th]:text-label [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-2 [&_td]:py-[0.4rem] [&_td]:overflow-hidden [&_td]:text-ellipsis [&_td]:whitespace-nowrap">
                <thead>
                  <tr>
                    <th>Coin</th>
                    <th>Previous</th>
                    <th>Estimated</th>
                    <th>Confidence</th>
                    <th>Status</th>
                    <th>Explanation</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="result in expandedResults" :key="result.id">
                    <td>{{ result.coinName }}</td>
                    <td>{{ result.previousValue != null ? `$${result.previousValue.toFixed(2)}` : '--' }}</td>
                    <td class="font-semibold text-gold">{{ result.estimatedValue > 0 ? `$${result.estimatedValue.toFixed(2)}` : '--' }}</td>
                    <td>
                      <span v-if="result.confidence" class="inline-block rounded-sm px-[0.3rem] py-[0.1rem] text-label font-semibold" :class="result.confidence === 'high' ? 'bg-[rgba(46,204,113,0.15)] text-[var(--confidence-high)]' : result.confidence === 'medium' ? 'bg-[rgba(241,196,15,0.15)] text-[var(--confidence-medium)]' : 'bg-[rgba(231,76,60,0.15)] text-[var(--confidence-low)]'">{{ result.confidence }}</span>
                      <span v-else>--</span>
                    </td>
                    <td>
                      <span class="inline-block rounded-full px-2 py-[0.15rem] text-label font-semibold" :class="result.status === 'success' ? 'bg-[rgba(46,204,113,0.15)] text-[var(--color-positive)]' : result.status === 'skipped' ? 'bg-[rgba(149,165,166,0.15)] text-[#95a5a6]' : 'bg-[rgba(231,76,60,0.15)] text-[var(--color-negative)]'">{{ result.status }}</span>
                    </td>
                    <td class="max-w-[200px] overflow-hidden text-ellipsis whitespace-nowrap">
                      <div v-if="result.changeExplanation" class="mb-[0.35rem] font-medium text-gold">{{ result.changeExplanation }}</div>
                      <div>{{ result.reasoning || result.errorMessage || '--' }}</div>
                    </td>
                  </tr>
                </tbody>
              </table>
              <p v-else class="px-8 py-8 text-center font-sans text-text-muted">No results for this run.</p>
            </td>
          </tr>
        </template>
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
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { cancelValuationRun, getValuationRunDetail, getValuationRuns, triggerValuation } from '@/api/client'
import { useRunHistoryPagination } from '@/composables/useRunHistoryPagination'
import type { AppSettings, ValuationRun } from '@/types'

defineProps<{
  settings: AppSettings
  settingsSaving: boolean
  settingsMsg: string
  settingsError: boolean
}>()

const emit = defineEmits<{
  save: []
  'update:settingsMsg': [val: string]
  'update:settingsError': [val: boolean]
}>()

const isMobile = ref(window.innerWidth <= 600)
const colspan = computed(() => isMobile.value ? 4 : 8)
const triggerLoading = ref(false)
const expandedRunId = ref<number | null>(null)
const expandedResults = ref<ValuationRun['results']>(undefined)
const expandedLoading = ref(false)
const timers: ReturnType<typeof setTimeout>[] = []
let pollTimer: ReturnType<typeof setInterval> | null = null

const {
  runs,
  page,
  loading,
  loadRuns: loadRunsBase,
  prevPage,
  nextPage,
} = useRunHistoryPagination<ValuationRun>(async (currentPage, limit) => {
  const res = await getValuationRuns(currentPage, limit)
  return res.data ?? {}
})

function onResize() {
  isMobile.value = window.innerWidth <= 600
}

async function loadRuns() {
  try {
    await loadRunsBase()
    const hasRunning = runs.value.some(run => run.status === 'running')
    if (hasRunning && !pollTimer) {
      pollTimer = setInterval(() => { loadRuns() }, 5000)
    } else if (!hasRunning && pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  } catch { /* ignore */ }
}

async function toggleRunDetail(runId: number) {
  if (expandedRunId.value === runId) {
    expandedRunId.value = null
    expandedResults.value = undefined
    return
  }
  expandedRunId.value = runId
  expandedResults.value = []
  expandedLoading.value = true
  try {
    const res = await getValuationRunDetail(runId)
    expandedResults.value = res.data.results ?? []
  } catch {
    expandedResults.value = []
  } finally {
    expandedLoading.value = false
  }
}

async function triggerManualValuation() {
  triggerLoading.value = true
  emit('update:settingsMsg', '')
  emit('update:settingsError', false)
  try {
    await triggerValuation()
    emit('update:settingsMsg', 'Valuation started — progress updates below')
    timers.push(setTimeout(() => { emit('update:settingsMsg', '') }, 10000))
    timers.push(setTimeout(() => { loadRuns() }, 2000))
  } catch {
    emit('update:settingsMsg', 'Failed to trigger valuation')
    emit('update:settingsError', true)
  } finally {
    triggerLoading.value = false
  }
}

async function cancelRun(runId: number) {
  try {
    await cancelValuationRun(runId)
    emit('update:settingsMsg', 'Cancellation requested')
    timers.push(setTimeout(() => { emit('update:settingsMsg', '') }, 5000))
    timers.push(setTimeout(() => { loadRuns() }, 1000))
  } catch {
    emit('update:settingsMsg', 'Failed to cancel run')
    emit('update:settingsError', true)
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString()
}

function formatDuration(ms: number) {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

onMounted(() => {
  window.addEventListener('resize', onResize)
  loadRuns()
})

onUnmounted(() => {
  window.removeEventListener('resize', onResize)
  if (pollTimer) clearInterval(pollTimer)
  timers.forEach(clearTimeout)
})
</script>
