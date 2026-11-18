<template>
  <hr class="my-6 border-0 border-t border-border-subtle" />

  <!-- Image Variant Check -->
  <h3 class="mb-4 text-lg font-medium text-heading">Image Variant Check</h3>
  <p class="mb-4 text-base text-text-secondary">Makes the small and medium copies of coin photos that do not have them, such as photos uploaded before variants existed. Without them, large photos do not show in clients that cannot fetch media links, such as the MCP coin viewer. Runs two minutes after the API starts, then daily. Photos that already have both copies are skipped.</p>
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
      <span id="image-variant-check-start-time-hint" class="form-hint">Time of day when coin photos are checked for missing variants. Photos that cannot be resized are listed in the logs.</span>
      <AdminScheduleSummary :start-time="settings.ImageVariantCheckStartTime" default-start-time="03:17" :zone="settings.ScheduleTimezone" />
    </div>
    <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
      <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="emit('save')">
        {{ settingsSaving ? 'Saving...' : 'Save Image Variant Settings' }}
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
