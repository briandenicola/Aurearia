<template>
  <section class="admin-section card flex flex-col">
    <h2 class="mb-5 border-b border-border-subtle pb-3 text-xl font-medium">Schedules</h2>

    <AdminAvailabilitySchedule
      :settings="settings"
      :settings-saving="settingsSaving"
      :settings-msg="availSettingsMsg"
      :settings-error="availSettingsError"
      @save="emit('save')"
      @update:settings-msg="emit('update:availSettingsMsg', $event)"
      @update:settings-error="emit('update:availSettingsError', $event)"
    >
      <template #additional-settings>
        <hr class="my-6 border-0 border-t border-border-subtle" />

        <!-- ParcelApp Shipment Tracking -->
        <h3 class="mb-4 text-base font-semibold text-text-primary">ParcelApp Shipment Tracking</h3>
        <p class="mb-4 text-base text-text-secondary">Enables ParcelApp shipment checks for users who have saved a ParcelApp API key. Automated checks run no more often than every 20 minutes.</p>
        <div class="mb-4">
          <div class="form-group flex items-center justify-between gap-3">
            <label class="form-label">Enable ParcelApp Tracking</label>
            <label class="relative inline-block h-[22px] w-[42px]">
              <input
                class="peer sr-only" type="checkbox"
                :checked="settings.ParcelAppEnabled === 'true'"
                @change="settings.ParcelAppEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
              />
              <span class="absolute inset-0 rounded-full border border-border-subtle bg-surface transition-colors after:absolute after:bottom-[2px] after:left-[2px] after:h-4 after:w-4 after:rounded-full after:bg-[var(--text-secondary)] after:transition-transform peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] peer-checked:after:translate-x-5 peer-checked:after:bg-gold peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2"></span>
            </label>
          </div>
          <div class="form-group">
            <label class="form-label">Repeat Interval (minutes)</label>
            <input
              v-model="settings.ShipmentSyncInterval"
              class="form-input w-full max-w-[120px]"
              type="number"
              min="20"
              step="5"
            />
            <span class="form-hint">Minimum 20 minutes to stay within ParcelApp's 20 requests/hour limit.</span>
          </div>
          <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
            <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="emit('save')">
              {{ settingsSaving ? 'Saving...' : 'Save ParcelApp Settings' }}
            </button>
          </div>
        </div>

        <hr class="my-6 border-0 border-t border-border-subtle" />

        <!-- Wishlist Search Alerts -->
        <h3 class="mb-4 text-base font-semibold text-text-primary">Wishlist Search Alerts</h3>
        <p class="mb-4 text-base text-text-secondary">Runs the daily sweep that queues automatic discovery runs for wishlist search alerts whose cadence (daily/weekly/monthly) has elapsed. Individual alerts also support Run Now from the Wishlist Alerts page.</p>
        <div class="mb-4">
          <div class="form-group flex items-center justify-between gap-3">
            <label class="form-label">Enable Automatic Checks</label>
            <label class="relative inline-block h-[22px] w-[42px]">
              <input
                class="peer sr-only" type="checkbox"
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
      </template>
    </AdminAvailabilitySchedule>

    <AdminAuctionEndingSchedule
      :settings="settings"
      :settings-saving="settingsSaving"
      :settings-msg="auctionSettingsMsg"
      :settings-error="auctionSettingsError"
      @save="emit('save')"
      @update:settings-msg="emit('update:auctionSettingsMsg', $event)"
      @update:settings-error="emit('update:auctionSettingsError', $event)"
    />

    <AdminAuctionAlertReminderSchedule
      :settings="settings"
      :settings-saving="settingsSaving"
      :settings-msg="alertReminderSettingsMsg"
      :settings-error="alertReminderSettingsError"
      @save="emit('save')"
      @update:settings-msg="emit('update:alertReminderSettingsMsg', $event)"
      @update:settings-error="emit('update:alertReminderSettingsError', $event)"
    />

    <AdminAuctionWatchBidDigestSchedule
      :settings="settings"
      :settings-saving="settingsSaving"
      :settings-msg="watchBidDigestSettingsMsg"
      :settings-error="watchBidDigestSettingsError"
      @save="emit('save')"
      @update:settings-msg="emit('update:watchBidDigestSettingsMsg', $event)"
      @update:settings-error="emit('update:watchBidDigestSettingsError', $event)"
    />

    <AdminValuationSchedule
      :settings="settings"
      :settings-saving="settingsSaving"
      :settings-msg="valSettingsMsg"
      :settings-error="valSettingsError"
      @save="emit('save')"
      @update:settings-msg="emit('update:valSettingsMsg', $event)"
      @update:settings-error="emit('update:valSettingsError', $event)"
    />

    <AdminCollectionHealthSchedule
      :settings="settings"
      :settings-saving="settingsSaving"
      :settings-msg="healthSettingsMsg"
      :settings-error="healthSettingsError"
      @save="emit('save')"
      @update:settings-msg="emit('update:healthSettingsMsg', $event)"
      @update:settings-error="emit('update:healthSettingsError', $event)"
    />

    <AdminCoinOfDaySchedule
      :settings="settings"
      :settings-saving="settingsSaving"
      @save="emit('save')"
    />

    <AdminPurchaseReminderSchedule
      :settings="settings"
      :settings-saving="settingsSaving"
      @save="emit('save')"
    />
  </section>
</template>

<script setup lang="ts">
import AdminAuctionAlertReminderSchedule from '@/components/admin/schedules/AdminAuctionAlertReminderSchedule.vue'
import AdminAuctionEndingSchedule from '@/components/admin/schedules/AdminAuctionEndingSchedule.vue'
import AdminAuctionWatchBidDigestSchedule from '@/components/admin/schedules/AdminAuctionWatchBidDigestSchedule.vue'
import AdminAvailabilitySchedule from '@/components/admin/schedules/AdminAvailabilitySchedule.vue'
import AdminCoinOfDaySchedule from '@/components/admin/schedules/AdminCoinOfDaySchedule.vue'
import AdminPurchaseReminderSchedule from '@/components/admin/schedules/AdminPurchaseReminderSchedule.vue'
import AdminCollectionHealthSchedule from '@/components/admin/schedules/AdminCollectionHealthSchedule.vue'
import AdminValuationSchedule from '@/components/admin/schedules/AdminValuationSchedule.vue'
import type { AppSettings } from '@/types'

defineProps<{
  settings: AppSettings
  settingsSaving: boolean
  availSettingsMsg: string
  availSettingsError: boolean
  auctionSettingsMsg: string
  auctionSettingsError: boolean
  alertReminderSettingsMsg: string
  alertReminderSettingsError: boolean
  watchBidDigestSettingsMsg: string
  watchBidDigestSettingsError: boolean
  healthSettingsMsg: string
  healthSettingsError: boolean
  valSettingsMsg: string
  valSettingsError: boolean
}>()

const emit = defineEmits<{
  save: []
  'update:valSettingsMsg': [val: string]
  'update:valSettingsError': [val: boolean]
  'update:auctionSettingsMsg': [val: string]
  'update:auctionSettingsError': [val: boolean]
  'update:alertReminderSettingsMsg': [val: string]
  'update:alertReminderSettingsError': [val: boolean]
  'update:watchBidDigestSettingsMsg': [val: string]
  'update:watchBidDigestSettingsError': [val: boolean]
  'update:availSettingsMsg': [val: string]
  'update:availSettingsError': [val: boolean]
  'update:healthSettingsMsg': [val: string]
  'update:healthSettingsError': [val: boolean]
}>()
</script>
