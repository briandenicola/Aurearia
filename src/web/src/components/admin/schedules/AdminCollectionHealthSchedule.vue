<template>
  <hr class="my-6 border-0 border-t border-border-subtle" />

  <!-- Collection Health Snapshots -->
  <h3 class="mb-4 text-lg font-medium text-heading">Collection Health Snapshots</h3>
  <p class="mb-4 text-base text-text-secondary">Captures daily health baselines used by the 30-day collection health trend.</p>
  <p v-if="statusLoading" class="mb-4 text-body text-text-muted">Checking server status…</p>
  <p v-else-if="status" class="mb-4 text-body" :class="status.enabled ? 'text-gain' : 'text-text-muted'">
    Server status: <strong>{{ status.enabled ? 'Enabled' : 'Disabled' }}</strong>{{ status.enabled ? ` — next run in ${formatNextRunIn(status.nextRunIn)}` : '' }}
  </p>
  <div class="mb-4">
    <div class="form-group flex items-center justify-between gap-3">
      <label class="form-label">Enable Daily Snapshots</label>
      <BaseToggle
        size="sm"
        label="Enable Daily Snapshots"
        :model-value="settings.CollectionHealthSnapshotsEnabled === 'true'"
        @update:model-value="settings.CollectionHealthSnapshotsEnabled = $event ? 'true' : 'false'"
      />
    </div>
    <div class="form-group">
      <label class="form-label">Start Time (daily)</label>
      <input
        v-model="settings.CollectionHealthSnapshotsStartTime"
        class="form-input w-full max-w-[120px]"
        type="time"
      />
      <span class="form-hint">Time of day when collection health baselines are captured for trend calculations.</span>
      <AdminScheduleSummary :start-time="settings.CollectionHealthSnapshotsStartTime" default-start-time="04:30" :zone="settings.ScheduleTimezone" />
    </div>
    <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
      <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="emit('save')">
        {{ settingsSaving ? 'Saving...' : 'Save Snapshot Settings' }}
      </button>
      <span v-if="settingsMsg" class="text-body text-gold md:mr-auto" :class="settingsError ? 'text-loss' : ''">{{ settingsMsg }}</span>
      <button class="btn btn-secondary btn-sm md:ml-auto" :disabled="triggerLoading" @click="triggerManualSnapshots()">
        {{ triggerLoading ? 'Running...' : 'Run Now' }}
      </button>
    </div>
  </div>

  <hr class="my-6 border-0 border-t border-border-subtle" />
  <h3 class="mb-4 text-lg font-medium text-heading">Collection Health Snapshot Run History</h3>
  <div v-if="loading" class="flex justify-center py-8"><div class="spinner"></div></div>
  <div v-else-if="runs.length === 0" class="px-8 py-8 text-center font-sans text-text-muted">No collection health snapshot runs recorded yet.</div>
  <template v-else>
    <div class="overflow-x-auto">
      <table class="data-table text-chip md:table-fixed [&_th]:px-[0.35rem] [&_th]:py-2 md:[&_th]:px-2 md:[&_th]:py-3 [&_td]:px-[0.35rem] [&_td]:py-2 md:[&_td]:px-2 md:[&_td]:py-3">
        <thead>
          <tr>
            <th>Date</th>
            <th class="hidden md:table-cell">Trigger</th>
            <th>Status</th>
            <th>Eligible</th>
            <th>Snapshotted</th>
            <th class="hidden md:table-cell">Failed</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="run in runs" :key="run.id">
            <td class="text-body text-text-secondary">{{ formatDate(run.startedAt) }}</td>
            <td class="hidden md:table-cell">{{ run.triggerType }}</td>
            <td>
              <BaseStatusBadge :tone="run.status === 'error' ? 'error' : run.status === 'success' ? 'success' : 'warning'">{{ run.status }}</BaseStatusBadge>
            </td>
            <td>{{ run.usersEligible }}</td>
            <td class="font-semibold text-gain">{{ run.usersSnapshotted }}</td>
            <td class="hidden font-semibold text-loss md:table-cell">{{ run.usersFailed }}</td>
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
import { onMounted, onUnmounted, ref, watch } from 'vue'
import {
  getCollectionHealthSnapshotRuns,
  getCollectionHealthSnapshotStatus,
  triggerCollectionHealthSnapshots,
} from '@/api/client'
import { useRunHistoryPagination } from '@/composables/useRunHistoryPagination'
import type { AppSettings, CollectionHealthSnapshotRun, SchedulerStatus } from '@/types'

const props = defineProps<{
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
const status = ref<SchedulerStatus | null>(null)
const statusLoading = ref(false)
const timers: ReturnType<typeof setTimeout>[] = []

const {
  runs,
  page,
  loading,
  loadRuns,
  prevPage,
  nextPage,
} = useRunHistoryPagination<CollectionHealthSnapshotRun>(async (currentPage, limit) => {
  const res = await getCollectionHealthSnapshotRuns(currentPage, limit)
  return res.data ?? {}
})

watch(() => props.settingsSaving, (saving, wasSaving) => {
  if (wasSaving && !saving) {
    loadStatus()
  }
})

async function loadStatus() {
  statusLoading.value = true
  try {
    const res = await getCollectionHealthSnapshotStatus()
    status.value = res.data
  } catch {
    status.value = null
  } finally {
    statusLoading.value = false
  }
}

function formatNextRunIn(nanoseconds: number) {
  const totalMinutes = Math.max(0, Math.round(nanoseconds / 1e9 / 60))
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  if (hours === 0) return `${minutes}m`
  return `${hours}h ${minutes}m`
}

async function triggerManualSnapshots() {
  triggerLoading.value = true
  emit('update:settingsMsg', '')
  emit('update:settingsError', false)
  try {
    const res = await triggerCollectionHealthSnapshots()
    const { message, users, snapshotsCreated, skipped, errors, durationMs } = res.data
    const parts = [
      snapshotsCreated != null ? `${snapshotsCreated} snapshots` : null,
      users != null ? `${users} users` : null,
      skipped != null ? `${skipped} skipped` : null,
      errors ? `${errors} errors` : null,
      durationMs != null ? `${(durationMs / 1000).toFixed(1)}s` : null,
    ].filter((part): part is string => part != null)
    emit('update:settingsMsg', message ?? (parts.length ? `Snapshot run complete — ${parts.join(', ')}` : 'Snapshot run complete'))
    if (errors) {
      emit('update:settingsError', true)
    }
    timers.push(setTimeout(() => { emit('update:settingsMsg', '') }, 10000))
    timers.push(setTimeout(() => { loadRuns() }, 1000))
    timers.push(setTimeout(() => { loadStatus() }, 1000))
  } catch {
    emit('update:settingsMsg', 'Failed to run collection health snapshots')
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
  loadStatus()
})

onUnmounted(() => {
  timers.forEach(clearTimeout)
})
</script>
