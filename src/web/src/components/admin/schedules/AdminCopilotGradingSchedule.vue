<template>
  <hr class="my-6 border-0 border-t border-border-subtle" />

  <h3 class="mb-4 text-lg font-medium text-heading">Copilot Answer Grading</h3>
  <p class="mb-4 text-base text-text-secondary">
    Grades completed Coin Copilot conversations once a day with a rubric judge, for review under Copilot Quality.
    Each graded conversation costs about one judge call on the configured Anthropic key.
  </p>
  <p v-if="status" class="mb-4 text-body" :class="status.enabled ? 'text-gain' : 'text-text-muted'">
    Server status: <strong>{{ status.enabled ? 'Enabled' : 'Disabled' }}</strong>{{ status.enabled ? ` — next run in ${formatNextRunIn(status.nextRunIn)}` : '' }}
    <span v-if="status.isRunning"> — grading now</span>
  </p>
  <p v-if="lastCycleText" class="mb-4 text-body text-text-secondary">{{ lastCycleText }}</p>
  <div class="mb-4">
    <div class="form-group flex items-center justify-between gap-3">
      <label class="form-label">Enable Nightly Grading</label>
      <BaseToggle
        size="sm"
        label="Enable Nightly Grading"
        :model-value="settings.CopilotGradingEnabled === 'true'"
        @update:model-value="settings.CopilotGradingEnabled = $event ? 'true' : 'false'"
      />
    </div>
    <div class="form-group">
      <label class="form-label">Start Time (daily)</label>
      <input v-model="settings.CopilotGradingStartTime" class="form-input w-full max-w-[120px]" type="time" />
      <AdminScheduleSummary :start-time="settings.CopilotGradingStartTime" default-start-time="05:00" :zone="settings.ScheduleTimezone" />
    </div>
    <div class="form-group">
      <label class="form-label">Daily Limit</label>
      <input v-model="settings.CopilotGradingDailyCap" class="form-input w-full max-w-[120px]" type="number" min="1" max="500" />
      <span class="form-hint">Most conversations graded per run, newest first; 1–500. Default 50.</span>
    </div>
    <div class="form-group">
      <label class="form-label">Judge Model</label>
      <input v-model="settings.CopilotGradingJudgeModel" class="form-input w-full max-w-[260px]" type="text" />
      <span class="form-hint">Anthropic model that grades answers. Default claude-sonnet-5-5.</span>
    </div>
    <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
      <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="emit('save')">
        {{ settingsSaving ? 'Saving...' : 'Save Grading Settings' }}
      </button>
      <span v-if="message" class="text-body md:mr-auto" :class="messageError ? 'text-loss' : 'text-gold'">{{ message }}</span>
      <button class="btn btn-secondary btn-sm md:ml-auto" :disabled="triggerLoading || status?.isRunning" @click="runNow()">
        {{ triggerLoading ? 'Starting...' : 'Grade Now' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { BaseToggle } from '@/components/ui'
import AdminScheduleSummary from '@/components/admin/schedules/AdminScheduleSummary.vue'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { getCopilotGradingStatus, runCopilotGrading } from '@/api/client'
import type { AppSettings, CopilotGradingStatus } from '@/types'

const props = defineProps<{
  settings: AppSettings
  settingsSaving: boolean
}>()

const emit = defineEmits<{
  save: []
}>()

const status = ref<CopilotGradingStatus | null>(null)
const triggerLoading = ref(false)
const message = ref('')
const messageError = ref(false)
const timers: ReturnType<typeof setTimeout>[] = []

const lastCycleText = computed(() => {
  const cycle = status.value?.lastCycle
  if (!cycle) return ''
  const when = new Date(cycle.startedAt).toLocaleString()
  if (cycle.skipped) return `Last run (${cycle.triggerType}, ${when}): skipped. ${cycle.skipped}`
  return `Last run (${cycle.triggerType}, ${when}): ${cycle.graded} graded, ${cycle.failed} failed.`
})

watch(() => props.settingsSaving, (saving, wasSaving) => {
  if (wasSaving && !saving) loadStatus()
})

async function loadStatus() {
  try {
    const res = await getCopilotGradingStatus()
    status.value = res.data
  } catch {
    status.value = null
  }
}

async function runNow() {
  triggerLoading.value = true
  message.value = ''
  messageError.value = false
  try {
    await runCopilotGrading()
    message.value = 'Grading started. Results appear under Copilot Quality as each conversation is graded.'
    timers.push(setTimeout(() => { loadStatus() }, 1000))
  } catch (err: unknown) {
    const conflict = (err as { response?: { status?: number } })?.response?.status === 409
    message.value = conflict ? 'Grading is already running.' : 'Failed to start Copilot grading.'
    messageError.value = true
  } finally {
    triggerLoading.value = false
    timers.push(setTimeout(() => { message.value = '' }, 10000))
  }
}

function formatNextRunIn(nanoseconds: number) {
  const totalMinutes = Math.max(0, Math.round(nanoseconds / 1e9 / 60))
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  if (hours === 0) return `${minutes}m`
  return `${hours}h ${minutes}m`
}

onMounted(loadStatus)
onUnmounted(() => timers.forEach(clearTimeout))
</script>
