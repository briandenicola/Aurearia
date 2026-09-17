<template>
  <hr class="my-6 border-0 border-t border-border-subtle" />

  <h3 class="mb-4 text-base font-semibold text-text-primary">Wishlist Search Alerts</h3>
  <p class="mb-4 text-base text-text-secondary">Runs the daily sweep that queues automatic discovery runs for wishlist search alerts whose cadence (daily/weekly/monthly) has elapsed. Individual alerts also support Run Now from the Wishlist Alerts page.</p>
  <div class="mb-4">
    <div class="form-group flex items-center justify-between gap-3">
      <label class="form-label">Enable Automatic Checks</label>
      <label class="relative inline-block h-[22px] w-[42px]">
        <input
          class="peer sr-only"
          type="checkbox"
          :checked="settings.WishlistSearchAlertsCheckEnabled === 'true'"
          @change="settings.WishlistSearchAlertsCheckEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
        />
        <span class="absolute inset-0 rounded-full border border-border-subtle bg-surface transition-colors after:absolute after:bottom-[2px] after:left-[2px] after:h-4 after:w-4 after:rounded-full after:bg-[var(--text-secondary)] after:transition-transform peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] peer-checked:after:translate-x-5 peer-checked:after:bg-gold peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2"></span>
      </label>
    </div>
    <div class="form-group">
      <label class="form-label">Start Time (daily anchor)</label>
      <input
        v-model="settings.WishlistSearchAlertsCheckStartTime"
        class="form-input w-full max-w-[120px]"
        type="time"
      />
      <span class="form-hint">The daily sweep runs at this time and queues any alerts whose cadence has elapsed since their last run.</span>
    </div>
    <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
      <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="emit('save')">
        {{ settingsSaving ? 'Saving...' : 'Save Schedule Settings' }}
      </button>
    </div>
  </div>

  <hr class="my-6 border-0 border-t border-border-subtle" />
  <h3 class="mb-4 text-base font-semibold text-text-primary">Wishlist Search Alert Run History</h3>

  <div v-if="loading" class="flex justify-center py-8"><div class="spinner"></div></div>
  <div v-else-if="historyError" class="px-8 py-8 text-center font-sans text-[var(--color-negative)]" role="alert">{{ historyError }}</div>
  <div v-else-if="runs.length === 0" class="px-8 py-8 text-center font-sans text-text-muted">No wishlist search alert runs recorded yet.</div>
  <template v-else>
    <div class="overflow-x-auto">
      <table class="w-full border-collapse text-body [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-2 [&_th]:py-3 [&_th]:text-left [&_th]:text-sm [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.08em] [&_th]:text-text-muted [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-2 [&_td]:py-3 [&_td]:text-left">
        <thead>
          <tr>
            <th>Date</th>
            <th>Alert</th>
            <th class="hidden md:table-cell">Owner</th>
            <th class="hidden md:table-cell">Trigger</th>
            <th>Status</th>
            <th>Results</th>
            <th>New</th>
            <th class="hidden md:table-cell">Duplicates</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="run in runs" :key="run.id">
            <tr>
              <td class="text-text-secondary">{{ formatDate(run.startedAt) }}</td>
              <td class="max-w-[180px] overflow-hidden text-ellipsis whitespace-nowrap" :title="run.alertName || `Alert #${run.alertId}`">
                {{ run.alertName || `Alert #${run.alertId}` }}
              </td>
              <td class="hidden md:table-cell">{{ run.userName || `User #${run.userId}` }}</td>
              <td class="hidden md:table-cell">{{ run.triggerType }}</td>
              <td><span class="chip-sm" :class="statusClass(run.status)">{{ run.status }}</span></td>
              <td>{{ run.resultCount }}</td>
              <td class="font-semibold text-[var(--color-positive)]">{{ run.newCount }}</td>
              <td class="hidden md:table-cell">{{ run.duplicateCount }}</td>
              <td>{{ formatDuration(run.durationMs) }}</td>
            </tr>
            <tr v-if="run.errorMessage || run.partialWarnings.length > 0">
              <td colspan="9" class="italic text-text-secondary">
                Run #{{ run.id }}: {{ run.errorMessage || run.partialWarnings.join('; ') }}
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <div class="mt-4 flex items-center justify-center gap-3">
      <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="prevPage()">Prev</button>
      <span class="text-body text-text-secondary">Page {{ page }}</span>
      <button class="btn btn-secondary btn-sm" :disabled="page * pageLimit >= total" @click="nextPage()">Next</button>
    </div>
  </template>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getAdminWishlistSearchAlertRuns } from '@/api/client'
import type { AdminWishlistSearchAlertRun, AppSettings } from '@/types'

defineProps<{
  settings: AppSettings
  settingsSaving: boolean
}>()

const emit = defineEmits<{ save: [] }>()
const pageLimit = 5
const runs = ref<AdminWishlistSearchAlertRun[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const historyError = ref('')

async function loadRuns() {
  loading.value = true
  historyError.value = ''
  try {
    const response = await getAdminWishlistSearchAlertRuns(page.value, pageLimit)
    runs.value = response.data?.runs ?? []
    total.value = response.data?.total ?? 0
  } catch {
    runs.value = []
    total.value = 0
    historyError.value = 'Unable to load wishlist search alert run history.'
  } finally {
    loading.value = false
  }
}

async function prevPage() {
  page.value = Math.max(1, page.value - 1)
  await loadRuns()
}

async function nextPage() {
  page.value++
  await loadRuns()
}

function statusClass(status: AdminWishlistSearchAlertRun['status']): string {
  if (status === 'queued') return 'text-gold'
  if (status === 'running') return 'text-[var(--accent-bronze)]'
  if (status === 'failed' || status === 'cancelled' || status === 'rate_limited') return 'text-[var(--color-negative)]'
  if (status === 'partial') return 'text-warning'
  return 'text-[var(--color-positive)]'
}

function formatDate(date: string): string {
  return new Date(date).toLocaleString()
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

onMounted(loadRuns)
</script>
