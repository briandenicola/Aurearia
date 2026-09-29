<template>
  <h3 class="mb-4 text-lg font-medium text-heading">Schedule Time Zone</h3>
  <p class="mb-4 text-base text-text-secondary">Start times below are read in this time zone. Coin of the Day can override it.</p>
  <div class="mb-4">
    <div class="form-group">
      <label class="form-label" for="schedule-timezone">Time Zone</label>
      <select
        id="schedule-timezone"
        class="form-input w-full max-w-[320px]"
        :value="settings.ScheduleTimezone ?? ''"
        @change="settings.ScheduleTimezone = ($event.target as HTMLSelectElement).value"
      >
        <option value="">Server time (usually UTC in Docker)</option>
        <option v-for="zone in timezoneOptions" :key="zone" :value="zone">{{ zone }}</option>
      </select>
      <span class="form-hint">Schedule changes apply within a minute of saving; no restart needed.</span>
      <button
        v-if="browserTimezone && settings.ScheduleTimezone !== browserTimezone"
        type="button"
        class="mt-1 self-start bg-transparent p-0 text-sm text-gold underline"
        @click="settings.ScheduleTimezone = browserTimezone"
      >
        Use my time zone ({{ browserTimezone }})
      </button>
    </div>
    <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
      <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="emit('save')">
        {{ settingsSaving ? 'Saving...' : 'Save Time Zone' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getBrowserTimezone } from '@/composables/usePurchaseReminder'
import { timezoneOptionsWith } from '@/composables/useScheduleTimezone'
import type { AppSettings } from '@/types'

const props = defineProps<{
  settings: AppSettings
  settingsSaving: boolean
}>()

const emit = defineEmits<{
  save: []
}>()

const browserTimezone = getBrowserTimezone()
const timezoneOptions = computed(() => timezoneOptionsWith(props.settings.ScheduleTimezone))
</script>
