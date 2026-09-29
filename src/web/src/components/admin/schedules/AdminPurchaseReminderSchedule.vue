<template>
  <hr class="my-6 border-0 border-t border-border-subtle" />

  <!-- Purchase Reminder Delivery -->
  <h3 class="mb-4 text-lg font-medium text-heading">Purchase Reminder Delivery</h3>
  <p class="mb-4 text-base text-text-secondary">Sends in-app notifications for wishlist purchase reminders whose due date has arrived. Users set their own reminders per coin; this scheduler handles daily delivery.</p>
  <div class="mb-4">
    <div class="form-group flex items-center justify-between gap-3">
      <label class="form-label" for="reminder-check-enabled">Enable Reminder Delivery</label>
      <BaseToggle
        size="sm"
        id="reminder-check-enabled"
        label="Enable Reminder Delivery"
        :model-value="settings.ReminderCheckEnabled === 'true'"
        @update:model-value="settings.ReminderCheckEnabled = $event ? 'true' : 'false'"
      />
    </div>
    <div class="form-group">
      <label class="form-label" for="reminder-check-start-time">Start Time (daily)</label>
      <input
        id="reminder-check-start-time"
        v-model="settings.ReminderCheckStartTime"
        class="form-input w-full max-w-[120px]"
        type="time"
        aria-describedby="reminder-check-start-time-hint"
      />
      <span id="reminder-check-start-time-hint" class="form-hint">Time of day when due purchase reminders are checked and delivered.</span>
      <AdminScheduleSummary :start-time="settings.ReminderCheckStartTime" default-start-time="08:00" :zone="settings.ScheduleTimezone" />
    </div>
    <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
      <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="emit('save')">
        {{ settingsSaving ? 'Saving...' : 'Save Reminder Settings' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { BaseToggle } from '@/components/ui'
import AdminScheduleSummary from '@/components/admin/schedules/AdminScheduleSummary.vue'
import type { AppSettings } from '@/types'

defineProps<{
  settings: AppSettings
  settingsSaving: boolean
}>()

const emit = defineEmits<{
  save: []
}>()
</script>