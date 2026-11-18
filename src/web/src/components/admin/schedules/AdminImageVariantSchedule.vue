<template>
  <hr class="my-6 border-0 border-t border-border-subtle" />

  <!-- Image Variant Check -->
  <h3 class="mb-4 text-lg font-medium text-heading">Image Variant Check</h3>
  <p class="mb-4 text-base text-text-secondary">Makes the small and medium copies of coin photos that do not have them, such as photos uploaded before variants existed. Without them, large photos do not show in clients that cannot fetch media links, such as the MCP coin viewer. Runs two minutes after the API starts, then daily. Photos that already have both copies are skipped.</p>
  <p v-if="statusLoading && !status" class="mb-4 text-body text-text-muted">Checking server status…</p>
  <p v-else-if="statusError" class="mb-4 text-body text-loss">{{ statusError }}</p>
  <p v-else-if="status" class="mb-4 text-body" :class="status.enabled ? 'text-gain' : 'text-text-muted'">
    Server status: <strong>{{ status.isRunning ? 'Running a pass now' : status.enabled ? 'Enabled' : 'Disabled' }}</strong>{{ !status.isRunning && status.enabled ? ` — next run in ${formatNextRunIn(status.nextRunIn)}` : '' }}
  </p>
  <div class="mb-4">
    <div class="form-group flex items-center justify-between gap-3">
      <label class="form-label" for="image-variant-check-enabled">Enable Image Variant Check</label>
      <BaseToggle
        size="sm"
        id="image-variant-check-enabled"
        label="Enable Image Variant Check"
        :model-value="settings.ImageVariantCheckEnabled === 'true'"
        @update:model-value="settings.ImageVariantCheckEnabled = $event ? 'true' : 'false'"
      />
    </div>
    <div class="form-group">
      <label class="form-label" for="image-variant-check-start-time">Start Time (daily)</label>
      <input
        id="image-variant-check-start-time"
        v-model="settings.ImageVariantCheckStartTime"
        class="form-input w-full max-w-[120px]"
        type="time"
        aria-describedby="image-variant-check-start-time-hint"
      />
      <span id="image-variant-check-start-time-hint" class="form-hint">Time of day when coin photos are checked for missing variants. Run Now works even when the daily check is off.</span>
      <AdminScheduleSummary :start-time="settings.ImageVariantCheckStartTime" default-start-time="03:17" :zone="settings.ScheduleTimezone" />
    </div>
    <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
      <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="emit('save')">
        {{ settingsSaving ? 'Saving...' : 'Save Image Variant Settings' }}
      </button>
      <span v-if="runMsg" class="text-body md:mr-auto" :class="runError ? 'text-loss' : 'text-gold'" role="status">{{ runMsg }}</span>
      <button class="btn btn-secondary btn-sm md:ml-auto" :disabled="triggerLoading || status?.isRunning" @click="runNow()">
        {{ triggerLoading ? 'Starting...' : status?.isRunning ? 'Running...' : 'Run Now' }}
      </button>
    </div>
  </div>

  <hr class="my-6 border-0 border-t border-border-subtle" />
  <h3 class="mb-4 text-lg font-medium text-heading">Image Variant Run History</h3>
  <div v-if="runsLoading && runs.length === 0" class="flex justify-center py-8"><div class="spinner"></div></div>
  <p v-else-if="runsError" class="px-8 py-8 text-center text-body text-loss">{{ runsError }}</p>
  <div v-else-if="runs.length === 0" class="px-8 py-8 text-center font-sans text-text-muted">No image variant passes recorded yet.</div>
  <template v-else>
    <div class="overflow-x-auto">
      <table class="data-table text-chip md:table-fixed [&_th]:px-[0.35rem] [&_th]:py-2 md:[&_th]:px-2 md:[&_th]:py-3 [&_td]:px-[0.35rem] [&_td]:py-2 md:[&_td]:px-2 md:[&_td]:py-3">
        <thead>
          <tr>
            <th>Date</th>
            <th class="hidden md:table-cell">Trigger</th>
            <th>Status</th>
            <th>Checked</th>
            <th>Made</th>
            <th class="hidden md:table-cell">Failed</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="run in runs" :key="run.id">
            <tr>
              <td class="text-body text-text-secondary">{{ formatDate(run.startedAt) }}</td>
              <td class="hidden md:table-cell">{{ run.triggerType }}</td>
              <td>
                <BaseStatusBadge :tone="statusTone(run.status)">{{ run.status }}</BaseStatusBadge>
              </td>
              <td>{{ run.checked }}</td>
              <td class="font-semibold text-gain">{{ run.made }}</td>
              <td class="hidden font-semibold md:table-cell" :class="run.failed ? 'text-loss' : ''">{{ run.failed }}</td>
              <td>{{ run.status === 'running' ? '—' : formatDuration(run.durationMs) }}</td>
            </tr>
            <tr v-if="run.errorMessage || run.failures" class="image-variant-run-detail">
              <td colspan="7" class="text-chip text-text-secondary">
                <p v-if="run.errorMessage" class="m-0">{{ run.errorMessage }}</p>
                <template v-if="run.failures">
                  <p class="m-0">Could not resize{{ run.failed > failureLines(run).length ? ` (first ${failureLines(run).length} of ${run.failed})` : '' }}:</p>
                  <ul class="m-0 pl-5 break-all">
                    <li v-for="line in failureLines(run)" :key="line">{{ line }}</li>
                  </ul>
                </template>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <div class="mt-4 flex items-center justify-center gap-3">
      <button class="btn btn-secondary btn-sm" :disabled="page <= 1 || runsLoading" @click="goToPage(page - 1)">Prev</button>
      <span class="text-chip text-text-secondary">Page {{ page }}</span>
      <button class="btn btn-secondary btn-sm" :disabled="page * pageSize >= total || runsLoading" @click="goToPage(page + 1)">Next</button>
    </div>
  </template>
</template>

<script setup lang="ts">
import { BaseStatusBadge, BaseToggle, type StatusTone } from '@/components/ui'
import AdminScheduleSummary from '@/components/admin/schedules/AdminScheduleSummary.vue'
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { getApiErrorMessage, getImageVariantRuns, getImageVariantStatus, triggerImageVariantRun } from '@/api/client'
import type { AppSettings, ImageVariantRun, SchedulerStatus } from '@/types'

const props = defineProps<{
  settings: AppSettings
  settingsSaving: boolean
}>()

const emit = defineEmits<{
  save: []
}>()

const pageSize = 5
// While a pass runs, the status and history refresh this often.
const pollMs = 5000

const status = ref<SchedulerStatus | null>(null)
const statusLoading = ref(false)
const statusError = ref('')
const runs = ref<ImageVariantRun[]>([])
const total = ref(0)
const page = ref(1)
const runsLoading = ref(false)
const runsError = ref('')
const triggerLoading = ref(false)
const runMsg = ref('')
const runError = ref(false)

let pollTimer: ReturnType<typeof setTimeout> | undefined
let msgTimer: ReturnType<typeof setTimeout> | undefined
let runsRequest = 0
let unmounted = false

watch(() => props.settingsSaving, (saving, wasSaving) => {
  if (wasSaving && !saving) void loadStatus()
})

async function loadStatus() {
  statusLoading.value = true
  try {
    const res = await getImageVariantStatus()
    status.value = res.data
    statusError.value = ''
  } catch (err) {
    status.value = null
    statusError.value = `Could not load the server status: ${getApiErrorMessage(err) || 'no response'}. Reload the page to try again.`
  } finally {
    statusLoading.value = false
  }
}

async function loadRuns() {
  // Only the latest request may update the table, so a slow earlier page cannot overwrite a newer one.
  const request = ++runsRequest
  runsLoading.value = true
  try {
    const res = await getImageVariantRuns(page.value, pageSize)
    if (request !== runsRequest) return
    runs.value = res.data.runs ?? []
    total.value = res.data.total ?? 0
    runsError.value = ''
  } catch (err) {
    if (request !== runsRequest) return
    runsError.value = `Could not load the run history: ${getApiErrorMessage(err) || 'no response'}. Reload the page to try again.`
  } finally {
    if (request === runsRequest) runsLoading.value = false
  }
}

async function goToPage(next: number) {
  page.value = Math.max(1, next)
  await loadRuns()
}

async function refresh() {
  await Promise.all([loadStatus(), loadRuns()])
  schedulePoll()
}

// Keep refreshing while a pass runs, so its result appears without a reload.
function schedulePoll() {
  clearTimeout(pollTimer)
  if (unmounted) return
  const running = status.value?.isRunning || runs.value.some(run => run.status === 'running')
  if (running) pollTimer = setTimeout(() => { void refresh() }, pollMs)
}

function showMessage(text: string, isError: boolean) {
  runMsg.value = text
  runError.value = isError
  clearTimeout(msgTimer)
  msgTimer = setTimeout(() => { runMsg.value = '' }, 10000)
}

async function runNow() {
  if (triggerLoading.value) return
  triggerLoading.value = true
  try {
    const res = await triggerImageVariantRun()
    showMessage(res.data?.message || 'Image variant pass started.', false)
  } catch (err) {
    showMessage(getApiErrorMessage(err) || 'Could not start the image variant pass. Try again in a moment.', true)
  } finally {
    triggerLoading.value = false
  }
  page.value = 1
  await refresh()
}

function statusTone(runStatus: ImageVariantRun['status']): StatusTone {
  if (runStatus === 'success') return 'success'
  if (runStatus === 'error') return 'error'
  if (runStatus === 'running') return 'info'
  return 'warning'
}

function failureLines(run: ImageVariantRun) {
  return (run.failures ?? '').split('\n').filter(Boolean)
}

function formatNextRunIn(nanoseconds: number) {
  const totalMinutes = Math.max(0, Math.round(nanoseconds / 1e9 / 60))
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  if (hours === 0) return `${minutes}m`
  return `${hours}h ${minutes}m`
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString()
}

function formatDuration(ms: number) {
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${Math.floor(ms / 60000)}m ${Math.round((ms % 60000) / 1000)}s`
}

onMounted(() => {
  void refresh()
})

onUnmounted(() => {
  unmounted = true
  clearTimeout(pollTimer)
  clearTimeout(msgTimer)
})
</script>
