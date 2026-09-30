<template>
  <hr class="my-6 border-0 border-t border-border-subtle" />

  <!-- Auction Ending Alerts -->
  <h3 class="mb-4 text-lg font-medium text-heading">Auction Ending Alerts</h3>
  <p class="mb-4 text-base text-text-secondary">Checks watched auction lots that are ending soon and sends Pushover reminders before bidding closes.</p>
  <div class="mb-4">
    <div class="form-group flex items-center justify-between gap-3">
      <label class="form-label">Enable Automatic Alerts</label>
      <BaseToggle
        size="sm"
        label="Enable Automatic Alerts"
        :model-value="settings.AuctionEndingCheckEnabled === 'true'"
        @update:model-value="settings.AuctionEndingCheckEnabled = $event ? 'true' : 'false'"
      />
    </div>
    <div class="form-group">
      <label class="form-label">Start Time (daily anchor)</label>
      <input
        v-model="settings.AuctionEndingCheckStartTime"
        class="form-input w-full max-w-[120px]"
        type="time"
      />
      <span class="form-hint">The first ending-alert check runs at this time each day.</span>
    </div>
    <div class="form-group">
      <label class="form-label">Repeat Interval (minutes)</label>
      <input
        v-model="settings.AuctionEndingCheckInterval"
        class="form-input w-full max-w-[120px]"
        type="number"
        min="60"
        step="60"
      />
      <span class="form-hint">How often to check for lots ending soon after the start time. Default 1440 (daily).</span>
      <AdminScheduleSummary :start-time="settings.AuctionEndingCheckStartTime" default-start-time="08:00" :interval-minutes="settings.AuctionEndingCheckInterval" :zone="settings.ScheduleTimezone" />
    </div>
    <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
      <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="emit('save')">
        {{ settingsSaving ? 'Saving...' : 'Save Alert Settings' }}
      </button>
      <span v-if="settingsMsg" class="text-body text-gold md:mr-auto" :class="settingsError ? 'text-loss' : ''">{{ settingsMsg }}</span>
      <button class="btn btn-secondary btn-sm md:ml-auto" :disabled="triggerLoading" @click="triggerManualAuctionCheck()">
        {{ triggerLoading ? 'Starting...' : 'Run Now' }}
      </button>
    </div>
  </div>

  <hr class="my-6 border-0 border-t border-border-subtle" />
  <h3 class="mb-4 text-lg font-medium text-heading">Auction Ending Alert Run History</h3>

  <div v-if="loading" class="flex justify-center py-8"><div class="spinner"></div></div>
  <div v-else-if="runs.length === 0" class="px-8 py-8 text-center font-sans text-text-muted">No auction ending alert runs recorded yet.</div>
  <template v-else>
    <div class="overflow-x-auto">
      <table class="data-table text-chip md:table-fixed [&_th]:px-[0.35rem] [&_th]:py-2 md:[&_th]:px-2 md:[&_th]:py-3 [&_td]:px-[0.35rem] [&_td]:py-2 md:[&_td]:px-2 md:[&_td]:py-3">
        <thead>
          <tr>
            <th>Date</th>
            <th class="hidden md:table-cell">Trigger</th>
            <th>Lots</th>
            <th>Alerts</th>
            <th class="hidden md:table-cell">Status</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="run in runs" :key="run.id">
            <td class="text-body text-text-secondary">{{ formatDate(run.startedAt) }}</td>
            <td class="hidden md:table-cell">
              <BaseStatusBadge :tone="run.triggerType === 'manual' ? 'error' : 'warning'">
                {{ run.triggerType }}
              </BaseStatusBadge>
            </td>
            <td>{{ run.lotsChecked }}</td>
            <td class="font-semibold text-gain">{{ run.alertsSent }}</td>
            <td class="hidden md:table-cell">
              <BaseStatusBadge :tone="run.status === 'error' ? 'error' : (run.status === 'success' ? 'success' : 'warning')">
                {{ run.status }}
              </BaseStatusBadge>
            </td>
            <td>{{ formatDuration(run.durationMs) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="mt-4 flex items-center justify-center gap-3">
      <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="prevPage()">Prev</button>
      <span class="text-chip text-text-secondary">Page {{ page }}</span>
      <button class="btn btn-secondary btn-sm" :disabled="runs.length < 5" @click="nextPage()">Next</button>
    </div>
  </template>
</template>

<script setup lang="ts">
import { BaseStatusBadge, BaseToggle } from '@/components/ui'
import AdminScheduleSummary from '@/components/admin/schedules/AdminScheduleSummary.vue'
import { onMounted, onUnmounted, ref } from 'vue'
import { getAuctionEndingRun, getAuctionEndingRuns, triggerAuctionEndingCheck } from '@/api/client'
import { useRunHistoryPagination } from '@/composables/useRunHistoryPagination'
import type { AppSettings, AuctionEndingRun } from '@/types'

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

const triggerLoading = ref(false)
const timers: ReturnType<typeof setTimeout>[] = []
let pollTimer: ReturnType<typeof setInterval> | null = null

const {
  runs,
  page,
  loading,
  loadRuns: loadRunsBase,
  prevPage,
  nextPage,
} = useRunHistoryPagination<AuctionEndingRun>(async (currentPage, limit) => {
  const res = await getAuctionEndingRuns(currentPage, limit)
  return res.data ?? {}
})

async function loadRuns() {
  try {
    await loadRunsBase()
    const hasActive = runs.value.some(run => run.status === 'queued' || run.status === 'running')
    if (hasActive && !pollTimer) {
      pollTimer = setInterval(() => { loadRuns() }, 3000)
    } else if (!hasActive && pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  } catch { /* ignore */ }
}

async function triggerManualAuctionCheck() {
  triggerLoading.value = true
  emit('update:settingsMsg', '')
  emit('update:settingsError', false)
  try {
    const res = await triggerAuctionEndingCheck()
    const { runId, status } = res.data
    if (status === 'queued' || status === 'running') {
      emit('update:settingsMsg', `Run #${runId} queued — checking for results…`)
      const pollCompleted = async () => {
        try {
          const runRes = await getAuctionEndingRun(runId)
          const run = runRes.data
          if (run.status === 'success') {
            const durationSec = run.durationMs != null ? ` in ${(run.durationMs / 1000).toFixed(1)}s` : ''
            emit('update:settingsMsg', `Run #${runId} completed — ${run.lotsChecked} lots checked, ${run.alertsSent} alerts sent${durationSec}`)
            timers.push(setTimeout(() => { emit('update:settingsMsg', '') }, 10000))
            loadRuns()
          } else if (run.status === 'error') {
            emit('update:settingsMsg', `Run #${runId} failed`)
            emit('update:settingsError', true)
            loadRuns()
          } else {
            timers.push(setTimeout(pollCompleted, 2000))
          }
        } catch {
          loadRuns()
        }
      }
      timers.push(setTimeout(pollCompleted, 1500))
    } else if (status === 'error') {
      emit('update:settingsMsg', `Run #${runId} failed`)
      emit('update:settingsError', true)
      timers.push(setTimeout(() => { loadRuns() }, 1000))
    } else {
      timers.push(setTimeout(() => { loadRuns() }, 2000))
    }
  } catch {
    emit('update:settingsMsg', 'Failed to trigger auction ending alerts')
    emit('update:settingsError', true)
  } finally {
    triggerLoading.value = false
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
  loadRuns()
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
  timers.forEach(clearTimeout)
})
</script>
