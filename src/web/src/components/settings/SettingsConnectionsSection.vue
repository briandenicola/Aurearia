<template>
  <section class="card text-text-primary">
    <h2 class="mb-5 border-b border-border-subtle pb-3 text-lg text-heading">Connections</h2>

    <h3 class="mb-3 text-base text-text-secondary">NumisBids Integration</h3>
    <p class="mb-3 text-sm text-text-muted">
      Connect your NumisBids account for watchlist/import tracking. Won/lost outcomes, winning bids, and max bids should be checked on NumisBids and updated manually.
    </p>
    <div class="form-group">
      <label class="form-label">NumisBids Username</label>
      <input v-model="nbUsername" class="form-input" placeholder="Your NumisBids username" autocomplete="off" />
    </div>
    <div class="form-group">
      <label class="form-label">NumisBids Password</label>
      <input v-model="nbPassword" type="password" class="form-input" placeholder="Your NumisBids password" autocomplete="new-password" />
      <span class="mt-1 block text-chip text-text-muted">Encrypted at rest on the server. Used only for NumisBids watchlist/import tracking; legacy stored passwords migrate on next save or sync.</span>
    </div>
    <div
      v-if="nbValidating"
      class="mt-1 rounded-sm border border-[color-mix(in_srgb,var(--text-warning)_20%,transparent)] bg-[color-mix(in_srgb,var(--text-warning)_10%,transparent)] px-3 py-[0.4rem] text-chip text-warning"
    >
      Validating NumisBids credentials...
    </div>
    <div
      v-else-if="nbValidationError"
      class="mt-1 rounded-sm border border-[color-mix(in_srgb,var(--color-negative)_20%,transparent)] bg-[color-mix(in_srgb,var(--color-negative)_10%,transparent)] px-3 py-[0.4rem] text-chip text-[var(--color-negative)]"
    >
      {{ nbValidationError }}
    </div>
    <div
      v-else-if="auth.user?.numisBidsConfigured"
      class="mt-1 rounded-sm border border-[color-mix(in_srgb,var(--color-positive)_20%,transparent)] bg-[color-mix(in_srgb,var(--color-positive)_10%,transparent)] px-3 py-[0.4rem] text-chip text-[var(--color-positive)]"
    >
      NumisBids account connected
    </div>

    <h3 class="mb-3 mt-5 text-base text-text-secondary">CNG Auctions Integration</h3>
    <p class="mb-3 text-sm text-text-muted">
      Connect your CNG Auctions account to sync watched lots and auto-detect hosted-auction outcomes where CNG provides the data.
    </p>
    <div class="form-group">
      <label class="form-label">CNG Username</label>
      <input v-model="cngUsername" class="form-input" placeholder="Your CNG username or email" autocomplete="off" />
    </div>
    <div class="form-group">
      <label class="form-label">CNG Password</label>
      <input v-model="cngPassword" type="password" class="form-input" placeholder="Your CNG password" autocomplete="new-password" />
      <span class="mt-1 block text-chip text-text-muted">Encrypted at rest on the server. Used for CNG watched-lot sync and available hosted bid/outcome data; legacy stored passwords migrate on next save or sync.</span>
    </div>
    <div
      v-if="cngValidating"
      class="mt-1 rounded-sm border border-[color-mix(in_srgb,var(--text-warning)_20%,transparent)] bg-[color-mix(in_srgb,var(--text-warning)_10%,transparent)] px-3 py-[0.4rem] text-chip text-warning"
    >
      Validating CNG credentials...
    </div>
    <div
      v-else-if="cngValidationError"
      class="mt-1 rounded-sm border border-[color-mix(in_srgb,var(--color-negative)_20%,transparent)] bg-[color-mix(in_srgb,var(--color-negative)_10%,transparent)] px-3 py-[0.4rem] text-chip text-[var(--color-negative)]"
    >
      {{ cngValidationError }}
    </div>
    <div
      v-else-if="auth.user?.cngConfigured"
      class="mt-1 rounded-sm border border-[color-mix(in_srgb,var(--color-positive)_20%,transparent)] bg-[color-mix(in_srgb,var(--color-positive)_10%,transparent)] px-3 py-[0.4rem] text-chip text-[var(--color-positive)]"
    >
      CNG account connected
    </div>

    <h3 class="mb-3 mt-5 text-base text-text-secondary">ParcelApp Integration</h3>
    <p class="mb-3 text-sm text-text-muted">
      Connect ParcelApp so shipment tracking can be created from a tracking number and synced automatically when the admin setting is enabled.
    </p>
    <div class="form-group">
      <label class="form-label">ParcelApp API Key</label>
      <input v-model="parcelAppAPIKey" type="password" class="form-input" placeholder="Your ParcelApp API key" autocomplete="new-password" />
      <span class="mt-1 block text-chip text-text-muted">Generate this at web.parcelapp.net. The key is encrypted at rest and only used for your shipments.</span>
    </div>
    <div
      v-if="auth.user?.parcelAppConfigured"
      class="mt-1 rounded-sm border border-[color-mix(in_srgb,var(--color-positive)_20%,transparent)] bg-[color-mix(in_srgb,var(--color-positive)_10%,transparent)] px-3 py-[0.4rem] text-chip text-[var(--color-positive)]"
    >
      ParcelApp key connected
    </div>

    <h3 class="mb-3 mt-5 text-base text-text-secondary">Pushover Notifications</h3>
    <p class="mb-3 text-sm text-text-muted">
      Receive push notifications on your phone when wishlist items become unavailable or friends add new coins.
    </p>
    <div class="form-group">
      <label class="form-label">Pushover User Key</label>
      <input v-model="pushoverKey" type="password" class="form-input" placeholder="Your Pushover User Key" autocomplete="off" />
      <span class="mt-1 block text-chip text-text-muted">Find your User Key in the Pushover app or dashboard.</span>
    </div>
    <div
      v-if="auth.user?.pushoverEnabled"
      class="mb-2 mt-1 rounded-sm border border-[color-mix(in_srgb,var(--color-positive)_20%,transparent)] bg-[color-mix(in_srgb,var(--color-positive)_10%,transparent)] px-3 py-[0.4rem] text-chip text-[var(--color-positive)]"
    >
      Pushover notifications active
    </div>
    <button
      class="btn btn-secondary btn-sm mb-1 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
      :disabled="pushoverTesting || !auth.user?.pushoverEnabled"
      @click="handleTestPushover"
    >
      {{ pushoverTesting ? 'Sending...' : 'Test Notification' }}
    </button>
    <p
      v-if="pushoverTestMsg"
      class="mt-1 text-body text-gold"
      :class="{ 'text-[var(--color-negative)]': pushoverTestError }"
    >
      {{ pushoverTestMsg }}
    </p>

    <button
      class="btn btn-primary btn-sm mt-2 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
      @click="handleSaveProfile"
      :disabled="profileSaving || nbValidating || cngValidating"
    >
      {{ nbValidating || cngValidating ? 'Validating...' : profileSaving ? 'Saving...' : 'Save Connections' }}
    </button>
    <p v-if="profileMsg" class="mt-2 text-body text-gold" :class="{ 'text-[var(--color-negative)]': profileError }">{{ profileMsg }}</p>
  </section>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { useSettingsProfile } from '@/composables/useSettingsProfile'

const auth = useAuthStore()
const {
  nbUsername, nbPassword, cngUsername, cngPassword, parcelAppAPIKey, pushoverKey,
  pushoverTesting, pushoverTestMsg, pushoverTestError, handleTestPushover,
  profileMsg, profileError, profileSaving,
  nbValidating, nbValidationError, cngValidating, cngValidationError, handleSaveProfile,
} = useSettingsProfile()
</script>
