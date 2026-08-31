<template>
  <hr class="my-6 border-0 border-t border-border-subtle" />

  <!-- Collection Health Snapshots -->
  <h3 class="mb-4 text-base font-semibold text-text-primary">Collection Health Snapshots</h3>
  <p class="mb-4 text-base text-text-secondary">Captures daily health baselines used by the 30-day collection health trend.</p>
  <p v-if="statusLoading" class="mb-4 text-body text-text-muted">Checking server status…</p>
  <p v-else-if="status" class="mb-4 text-body" :class="status.enabled ? 'text-[var(--color-positive)]' : 'text-text-muted'">
    Server status: <strong>{{ status.enabled ? 'Enabled' : 'Disabled' }}</strong>{{ status.enabled ? ` — next run in ${formatNextRunIn(status.nextRunIn)}` : '' }}
  </p>
  <div class="mb-4">
    <div class="form-group flex items-center justify-between gap-3">
      <label class="form-label">Enable Daily Snapshots</label>
      <label class="relative inline-block h-[22px] w-[42px]">
        <input
          class="peer sr-only" type="checkbox"
          :checked="settings.CollectionHealthSnapshotsEnabled === 'true'"
          @change="settings.CollectionHealthSnapshotsEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
        />
        <span class="absolute inset-0 rounded-full border border-border-subtle bg-surface transition-colors after:absolute after:bottom-[2px] after:left-[2px] after:h-4 after:w-4 after:rounded-full after:bg-[var(--text-secondary)] after:transition-transform peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] peer-checked:after:translate-x-5 peer-checked:after:bg-gold peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2"></span>
      </label>
    </div>
    <div class="form-group">
      <label class="form-label">Start Time (daily)</label>
      <input
        v-model="settings.CollectionHealthSnapshotsStartTime"
        class="form-input w-full max-w-[120px]"
        type="time"
      />
      <span class="form-hint">Time of day when collection health baselines are captured for trend calculations.</span>
    </div>
    <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
      <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="emit('save')">
        {{ settingsSaving ? 'Saving...' : 'Save Snapshot Settings' }}
      </button>
      <span v-if="settingsMsg" class="text-body text-gold md:mr-auto" :class="settingsError ? 'text-[var(--color-negative)]' : ''">{{ settingsMsg }}</span>
      <button class="btn btn-secondary btn-sm md:ml-auto" :disabled="triggerLoading" @click="triggerManualSnapshots()">
        {{ triggerLoading ? 'Running...' : 'Run Now' }}
      </button>
    </div>
  </div>

  <hr class="my-6 border-0 border-t border-border-subtle" />
  <h3 class="mb-4 text-base font-semibold text-text-primary">Collection Health Snapshot Run History</h3>
  <div v-if="loading" class="flex justify-center py-8"><div class="spinner"></div></div>
  <div v-else-if="runs.length === 0" class="px-8 py-8 text-center font-sans text-text-muted">No collection health snapshot runs recorded yet.</div>
  <template v-else>
    <div class="overflow-x-auto">
      <table class="w-full border-collapse text-[0.8rem] md:table-fixed md:text-[0.82rem] [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-[0.35rem] [&_th]:py-2 [&_th]:text-left [&_th]:text-sm [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted md:[&_th]:px-2 md:[&_th]:py-3 [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-[0.35rem] [&_td]:py-2 [&_td]:text-left md:[&_td]:px-2 md:[&_td]:py-3">
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
              <span class="inline-block rounded-full px-2 py-[0.15rem] text-label font-semibold" :class="run.status === 'error' ? 'bg-[rgba(231,76,60,0.15)] text-[var(--color-negative)]' : run.status === 'success' ? 'bg-[rgba(46,204,113,0.15)] text-[var(--color-positive)]' : 'bg-[rgba(241,196,15,0.15)] text-warning'">{{ run.status }}</span>
            </td>
            <td>{{ run.usersEligible }}</td>
            <td class="font-semibold text-[var(--color-positive)]">{{ run.usersSnapshotted }}</td>
            <td class="hidden font-semibold text-[var(--color-negative)] md:table-cell">{{ run.usersFailed }}</td>
            <td>{{ formatDuration(run.durationMs) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="mt-4 flex items-center justify-center gap-3">
      <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="prevPage()">Prev</button>
      <span class="text-[0.82rem] text-text-secondary">Page {{ page }}</span>
      <button class="btn btn-secondary btn-sm" :disabled="runs.length < 5" @click="nextPage()">Next</button>
    </div>
  </template>
</template>

<script setup lang="ts">
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
