<template>
  <hr class="my-6 border-0 border-t border-border-subtle" />

  <!-- Purchase Reminder Delivery -->
  <h3 class="mb-4 text-base font-semibold text-text-primary">Purchase Reminder Delivery</h3>
  <p class="mb-4 text-base text-text-secondary">Sends in-app notifications for wishlist purchase reminders whose due date has arrived. Users set their own reminders per coin; this scheduler handles daily delivery.</p>
  <div class="mb-4">
    <div class="form-group flex items-center justify-between gap-3">
      <label class="form-label" for="reminder-check-enabled">Enable Reminder Delivery</label>
      <label class="relative inline-block h-[22px] w-[42px]">
        <input
          id="reminder-check-enabled"
          class="peer sr-only" type="checkbox"
          :checked="settings.ReminderCheckEnabled === 'true'"
          @change="settings.ReminderCheckEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
        />
        <span class="absolute inset-0 rounded-full border border-border-subtle bg-surface transition-colors after:absolute after:bottom-[2px] after:left-[2px] after:h-4 after:w-4 after:rounded-full after:bg-[var(--text-secondary)] after:transition-transform peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] peer-checked:after:translate-x-5 peer-checked:after:bg-gold peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2"></span>
      </label>
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
    </div>
    <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
      <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="emit('save')">
        {{ settingsSaving ? 'Saving...' : 'Save Reminder Settings' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AppSettings } from '@/types'

defineProps<{
  settings: AppSettings
  settingsSaving: boolean
}>()

const emit = defineEmits<{
  save: []
}>()
</script>