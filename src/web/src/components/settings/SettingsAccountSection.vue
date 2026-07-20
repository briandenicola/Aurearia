<template>
  <section class="card text-text-primary">
    <h2 class="mb-5 border-b border-border-subtle pb-3 text-lg text-heading">Account</h2>

    <!-- Avatar -->
    <div class="flex items-center justify-between gap-4 border-b border-border-subtle py-3 last:border-0">
      <div class="shrink-0">
        <AuthenticatedImage :media-path="avatarUrl" alt="Avatar" class="h-16 w-16 rounded-full border-2 border-gold-dim object-cover" />
      </div>
      <div class="flex flex-wrap gap-2">
        <label class="btn btn-secondary btn-sm cursor-pointer focus-within:outline-2 focus-within:outline-gold focus-within:outline-offset-2">
          Upload Avatar
          <input type="file" accept="image/*" hidden @change="handleAvatarUpload" />
        </label>
        <button
          v-if="auth.user?.avatarPath"
          class="btn btn-danger btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          @click="handleAvatarDelete"
        >
          Remove
        </button>
      </div>
    </div>

    <div class="flex items-center justify-between gap-4 border-b border-border-subtle py-3 last:border-0">
      <div class="flex flex-col gap-[0.15rem]">
        <span class="text-base font-medium">Username</span>
        <span class="text-base text-text-secondary">{{ auth.user?.username }}</span>
      </div>
    </div>
    <div class="flex items-center justify-between gap-4 border-b border-border-subtle py-3 last:border-0">
      <div class="flex flex-col gap-[0.15rem]">
        <span class="text-base font-medium">Role</span>
        <span class="badge text-base text-text-secondary" :class="`badge-${auth.user?.role === 'admin' ? 'roman' : 'modern'}`">
          {{ auth.user?.role }}
        </span>
      </div>
    </div>

    <!-- Profile / Social Settings -->
    <h3 class="mb-3 mt-5 text-base text-text-secondary">Profile</h3>
    <div class="form-group">
      <label class="form-label">Email</label>
      <input v-model="profileEmail" type="email" class="form-input" placeholder="you@example.com" />
    </div>
    <div class="form-group">
      <label class="form-label">Bio</label>
      <input v-model="profileBio" class="form-input" placeholder="Tell collectors about yourself..." maxlength="200" />
    </div>
    <div class="form-group">
      <label class="form-label">ZIP Code</label>
      <input v-model="profileZipCode" class="form-input" placeholder="e.g. 90210" maxlength="10" />
      <span class="mt-1 block text-chip text-text-muted">Used by the Agent to find nearby coin shows and dealers</span>
    </div>

    <h3 class="mb-3 mt-5 text-base text-text-secondary">NumisBids Integration</h3>
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
    <div class="flex items-center justify-between gap-4 border-b border-border-subtle py-3 last:border-0">
      <div class="flex flex-col gap-[0.15rem]">
        <span class="text-base font-medium">Public Collection</span>
        <span class="text-sm text-text-muted">Allow other users to follow you and view your coins</span>
      </div>
      <label class="relative inline-block h-7 w-[50px] shrink-0">
        <input type="checkbox" class="peer sr-only" :checked="profilePublic" @change="onPublicToggle" />
        <span
          class="absolute inset-0 cursor-pointer rounded-full border border-border-subtle bg-[var(--bg-primary)] transition-colors peer-checked:border-gold peer-checked:bg-gold-dim peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2 after:absolute after:bottom-[3px] after:left-[3px] after:h-5 after:w-5 after:rounded-full after:bg-text-secondary after:transition-transform after:content-[''] peer-checked:after:translate-x-[22px] peer-checked:after:bg-gold"
        ></span>
      </label>
    </div>
    <div class="flex items-center justify-between gap-4 border-b border-border-subtle py-3 last:border-0">
      <div class="flex flex-col gap-[0.15rem]">
        <span class="text-base font-medium">Coin of the Day</span>
        <span class="text-sm text-text-muted">Receive a daily featured coin notification from your collection</span>
      </div>
      <label class="relative inline-block h-7 w-[50px] shrink-0">
        <input v-model="coinOfDayEnabled" type="checkbox" class="peer sr-only" />
        <span
          class="absolute inset-0 cursor-pointer rounded-full border border-border-subtle bg-[var(--bg-primary)] transition-colors peer-checked:border-gold peer-checked:bg-gold-dim peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2 after:absolute after:bottom-[3px] after:left-[3px] after:h-5 after:w-5 after:rounded-full after:bg-text-secondary after:transition-transform after:content-[''] peer-checked:after:translate-x-[22px] peer-checked:after:bg-gold"
        ></span>
      </label>
    </div>
    <div class="flex items-center justify-between gap-4 border-b border-border-subtle py-3 last:border-0">
      <div class="flex flex-col gap-[0.15rem]">
        <span class="text-base font-medium">Emperor Tracker</span>
        <span class="text-sm text-text-muted">Track your collection's progress toward every Roman Emperor, under Stats</span>
      </div>
      <label class="relative inline-block h-7 w-[50px] shrink-0">
        <input v-model="emperorTrackerEnabled" type="checkbox" class="peer sr-only" />
        <span
          class="absolute inset-0 cursor-pointer rounded-full border border-border-subtle bg-[var(--bg-primary)] transition-colors peer-checked:border-gold peer-checked:bg-gold-dim peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2 after:absolute after:bottom-[3px] after:left-[3px] after:h-5 after:w-5 after:rounded-full after:bg-text-secondary after:transition-transform after:content-[''] peer-checked:after:translate-x-[22px] peer-checked:after:bg-gold"
        ></span>
      </label>
    </div>
    <template v-if="emperorTrackerEnabled">
      <div class="flex items-center justify-between gap-4 border-b border-border-subtle py-3 pl-4 last:border-0">
        <div class="flex flex-col gap-[0.15rem]">
          <span class="text-sm font-medium">Also track usurpers</span>
          <span class="text-xs text-text-muted">Track completion of coins from usurpers/pretenders, as a separate goal</span>
        </div>
        <label class="relative inline-block h-6 w-[44px] shrink-0">
          <input v-model="emperorTrackerShowUsurpers" type="checkbox" class="peer sr-only" />
          <span
            class="absolute inset-0 cursor-pointer rounded-full border border-border-subtle bg-[var(--bg-primary)] transition-colors peer-checked:border-gold peer-checked:bg-gold-dim peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2 after:absolute after:bottom-[3px] after:left-[3px] after:h-4 after:w-4 after:rounded-full after:bg-text-secondary after:transition-transform after:content-[''] peer-checked:after:translate-x-[19px] peer-checked:after:bg-gold"
          ></span>
        </label>
      </div>
      <div class="flex items-center justify-between gap-4 border-b border-border-subtle py-3 pl-4 last:border-0">
        <div class="flex flex-col gap-[0.15rem]">
          <span class="text-sm font-medium">Also track empresses</span>
          <span class="text-xs text-text-muted">Track completion of coins from empresses, as a separate goal</span>
        </div>
        <label class="relative inline-block h-6 w-[44px] shrink-0">
          <input v-model="emperorTrackerShowEmpresses" type="checkbox" class="peer sr-only" />
          <span
            class="absolute inset-0 cursor-pointer rounded-full border border-border-subtle bg-[var(--bg-primary)] transition-colors peer-checked:border-gold peer-checked:bg-gold-dim peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2 after:absolute after:bottom-[3px] after:left-[3px] after:h-4 after:w-4 after:rounded-full after:bg-text-secondary after:transition-transform after:content-[''] peer-checked:after:translate-x-[19px] peer-checked:after:bg-gold"
          ></span>
        </label>
      </div>
      <div class="flex items-center justify-between gap-4 border-b border-border-subtle py-3 pl-4 last:border-0">
        <div class="flex flex-col gap-[0.15rem]">
          <span class="text-sm font-medium">Also track other figures</span>
          <span class="text-xs text-text-muted">Track completion of Caesars who never acceded and other precursor figures, as a separate goal</span>
        </div>
        <label class="relative inline-block h-6 w-[44px] shrink-0">
          <input v-model="emperorTrackerShowOtherFigures" type="checkbox" class="peer sr-only" />
          <span
            class="absolute inset-0 cursor-pointer rounded-full border border-border-subtle bg-[var(--bg-primary)] transition-colors peer-checked:border-gold peer-checked:bg-gold-dim peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2 after:absolute after:bottom-[3px] after:left-[3px] after:h-4 after:w-4 after:rounded-full after:bg-text-secondary after:transition-transform after:content-[''] peer-checked:after:translate-x-[19px] peer-checked:after:bg-gold"
          ></span>
        </label>
      </div>
    </template>
    <button
      class="btn btn-primary btn-sm mt-2 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
      @click="handleSaveProfile"
      :disabled="profileSaving || nbValidating || cngValidating"
    >
      {{ nbValidating || cngValidating ? 'Validating...' : profileSaving ? 'Saving...' : 'Save Profile' }}
    </button>
    <p v-if="profileMsg" class="mt-2 text-body text-gold" :class="{ 'text-[var(--color-negative)]': profileError }">{{ profileMsg }}</p>

    <!-- Privacy Warning Modal -->
    <Teleport to="body">
      <div v-if="showPrivacyWarning" class="fixed inset-0 z-[1000] flex items-start justify-center bg-[var(--overlay-dark)] p-4 pt-[15vh]" @click.self="cancelGoPrivate">
        <div class="card w-full max-w-[440px]">
          <div class="border-b border-border-subtle px-5 py-4">
            <h2 class="m-0 flex items-center gap-2 text-base text-heading">
              Make Collection Private?
            </h2>
          </div>
          <div class="p-5">
            <p class="mb-3 mt-0 leading-6 text-text-secondary">
              Setting your profile to private will <strong class="text-text-primary">permanently remove all your followers</strong>.
              They will need to send new follow requests if you make your profile public again.
            </p>
            <p class="mb-4 mt-0 leading-6 text-text-secondary">
              You will also be hidden from user search results.
            </p>
            <div class="flex justify-end gap-3">
              <button class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" @click="cancelGoPrivate">Cancel</button>
              <button class="btn btn-danger btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" @click="confirmGoPrivate">Make Private</button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <h3 class="mb-3 mt-5 text-base text-text-secondary">Change Password</h3>
    <form class="max-w-[350px]" @submit.prevent="handleChangePassword">
      <div class="form-group">
        <label class="form-label">Current Password</label>
        <input v-model="currentPassword" type="password" class="form-input" required />
      </div>
      <div class="form-group">
        <label class="form-label">New Password</label>
        <input v-model="newPassword" type="password" class="form-input" required minlength="6" />
      </div>
      <div class="form-group">
        <label class="form-label">Confirm New Password</label>
        <input v-model="confirmPassword" type="password" class="form-input" required />
      </div>
      <p v-if="passwordMsg" class="my-2 text-body text-gold" :class="{ 'text-[var(--color-negative)]': passwordError }">{{ passwordMsg }}</p>
      <button
        type="submit"
        class="btn btn-primary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
        :disabled="passwordLoading"
      >
        {{ passwordLoading ? 'Changing...' : 'Change Password' }}
      </button>
    </form>

    <h3 class="mb-3 mt-5 text-base text-text-secondary">Connected Sign-in Providers</h3>
    <p class="mt-2 text-sm text-text-muted">
      Link an external provider after signing in locally. This avoids unsafe automatic account merges.
    </p>

    <div v-if="oidcMsg" class="my-2 text-body text-gold" :class="{ 'text-[var(--color-negative)]': oidcError }" role="status">
      {{ oidcMsg }}
    </div>

    <div v-if="oidcLoading" class="my-2 text-body text-gold">
      Loading linked providers...
    </div>
    <div v-else>
      <div v-if="oidcIdentities.length" class="mt-4 flex flex-col gap-3">
        <div
          v-for="identity in oidcIdentities"
          :key="identity.id"
          class="flex flex-col items-stretch gap-3 rounded-sm border border-border-subtle bg-input p-3 md:flex-row md:items-start md:justify-between"
        >
          <div class="flex min-w-0 flex-col gap-[0.2rem]">
            <div class="flex flex-wrap items-center gap-[0.35rem]">
              <span class="text-base font-medium text-text-primary">{{ identity.providerDisplayName }}</span>
              <span
                class="chip-sm"
                :class="identity.emailVerified ? 'border-[var(--color-positive)] text-[var(--color-positive)]' : 'border-[var(--color-negative)] text-[var(--color-negative)]'"
              >
                {{ identity.emailVerified ? 'Email verified' : 'Email unverified' }}
              </span>
            </div>
            <span class="text-sm text-text-muted [overflow-wrap:anywhere]">Issuer: {{ identity.issuer }}</span>
            <span class="text-sm text-text-muted [overflow-wrap:anywhere]">Subject: {{ identity.subjectPreview }}</span>
            <span class="text-sm text-text-muted [overflow-wrap:anywhere]">Email: {{ identity.email }}</span>
            <span class="text-sm text-text-muted [overflow-wrap:anywhere]">
              Linked {{ formatDateTime(identity.createdAt) }}
              <template v-if="identity.lastLoginAt"> · Last login {{ formatDateTime(identity.lastLoginAt) }}</template>
            </span>
          </div>
          <button
            class="btn btn-danger btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
            :disabled="unlinkingIdentityId === identity.id"
            @click="handleUnlinkIdentity(identity.id, identity.providerDisplayName)"
          >
            {{ unlinkingIdentityId === identity.id ? 'Unlinking...' : 'Unlink' }}
          </button>
        </div>
      </div>
      <p v-else class="mt-2 text-sm text-text-muted">No external sign-in providers linked.</p>

      <div v-if="linkableProviders.length" class="mt-3 flex flex-wrap gap-2">
        <button
          v-for="provider in linkableProviders"
          :key="provider.id"
          type="button"
          class="btn btn-secondary btn-sm inline-flex items-center gap-[0.35rem] focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          :disabled="linkingProviderId === provider.id"
          @click="handleLinkProvider(provider.id, provider.displayName)"
        >
          <LinkIcon :size="16" aria-hidden="true" />
          {{ linkingProviderId === provider.id ? 'Starting...' : `Link ${provider.displayName}` }}
        </button>
      </div>
      <p v-else-if="!oidcProviders.length" class="mt-2 text-sm text-text-muted">
        No enabled OIDC providers are available for linking.
      </p>
    </div>

    <template v-if="supportsWebAuthn">
      <h3 class="mb-3 mt-5 text-base text-text-secondary">Biometric Login</h3>
      <p class="mb-3 text-sm text-text-muted">
        Register Face ID, Touch ID, or fingerprint for quick sign-in on this device.
      </p>

      <button
        class="btn btn-primary btn-sm inline-flex items-center gap-[0.35rem] focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
        :disabled="registeringCredential"
        @click="handleRegisterCredential"
      >
        <LockKeyhole v-if="!registeringCredential" :size="16" aria-hidden="true" />
        {{ registeringCredential ? 'Registering...' : 'Register Biometric' }}
      </button>
      <p v-if="credentialMsg" class="mt-2 text-body text-gold" :class="{ 'text-[var(--color-negative)]': credentialError }">{{ credentialMsg }}</p>

      <div v-if="webauthnCredentials.length" class="mt-4 flex flex-col gap-2">
        <div
          v-for="cred in webauthnCredentials"
          :key="cred.id"
          class="flex items-center justify-between gap-3 border-b border-border-subtle py-[0.6rem] last:border-0"
        >
          <div class="flex min-w-0 flex-col gap-[0.1rem]">
            <span class="text-base font-medium">{{ cred.name }}</span>
            <span class="text-sm text-text-muted">Registered {{ formatDate(cred.createdAt) }}</span>
          </div>
          <button class="btn btn-danger btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" @click="handleDeleteCredential(cred.id)">
            Remove
          </button>
        </div>
      </div>
      <p v-else-if="!registeringCredential" class="mt-2 text-sm text-text-muted">No biometric credentials registered.</p>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import {
  webauthnRegisterBegin, webauthnRegisterFinish,
  webauthnListCredentials, webauthnDeleteCredential,
  deleteOIDCIdentity, getApiErrorMessage, getOIDCIdentities, getOIDCPublicProviders, startOIDCLink,
} from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import { useSettingsProfile } from '@/composables/useSettingsProfile'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'
import type { OIDCLinkedIdentity, OIDCPublicProvider, WebAuthnCredentialInfo } from '@/types'
import { Link as LinkIcon, LockKeyhole } from 'lucide-vue-next'

const auth = useAuthStore()
const { showConfirm } = useDialog()

const {
  avatarUrl, handleAvatarUpload, handleAvatarDelete,
  profileEmail, profileBio, profileZipCode,
  nbUsername, nbPassword, cngUsername, cngPassword, pushoverKey, pushoverTesting, pushoverTestMsg, pushoverTestError,
  handleTestPushover, profilePublic, profileMsg, profileError, profileSaving,
  showPrivacyWarning, onPublicToggle, confirmGoPrivate, cancelGoPrivate,
  nbValidating, nbValidationError, cngValidating, cngValidationError, handleSaveProfile, coinOfDayEnabled,
  emperorTrackerEnabled, emperorTrackerShowUsurpers, emperorTrackerShowEmpresses, emperorTrackerShowOtherFigures,
  currentPassword, newPassword, confirmPassword,
  passwordMsg, passwordError, passwordLoading, handleChangePassword,
} = useSettingsProfile()

// WebAuthn Biometric
const supportsWebAuthn = !!window.PublicKeyCredential
const webauthnCredentials = ref<WebAuthnCredentialInfo[]>([])
const registeringCredential = ref(false)
const credentialMsg = ref('')
const credentialError = ref(false)
const oidcIdentities = ref<OIDCLinkedIdentity[]>([])
const oidcProviders = ref<OIDCPublicProvider[]>([])
const oidcLoading = ref(true)
const oidcMsg = ref('')
const oidcError = ref(false)
const linkingProviderId = ref<number | null>(null)
const unlinkingIdentityId = ref<number | null>(null)

const linkableProviders = computed(() => {
  const linkedProviderIds = new Set(oidcIdentities.value.map(identity => identity.providerId))
  return oidcProviders.value.filter(provider => !linkedProviderIds.has(provider.id))
})

async function loadOIDCAccounts() {
  oidcLoading.value = true
  try {
    const [identitiesResponse, providersResponse] = await Promise.all([
      getOIDCIdentities(),
      getOIDCPublicProviders(),
    ])
    oidcIdentities.value = identitiesResponse.data.identities ?? []
    oidcProviders.value = providersResponse.data.providers ?? []
  } catch (error: unknown) {
    oidcMsg.value = getApiErrorMessage(error) || 'Failed to load linked sign-in providers.'
    oidcError.value = true
  } finally {
    oidcLoading.value = false
  }
}

async function handleLinkProvider(providerId: number, displayName: string) {
  oidcMsg.value = ''
  oidcError.value = false
  linkingProviderId.value = providerId
  try {
    const response = await startOIDCLink(providerId, {
      redirectPath: '/settings?tab=account',
      callbackPath: `/settings/oidc/link/callback/${providerId}`,
    })
    const authorizationUrl = response.data.authorizationUrl
    if (!authorizationUrl) {
      oidcMsg.value = `${displayName} did not return an authorization URL. Ask an administrator to test the provider.`
      oidcError.value = true
      return
    }
    window.location.assign(authorizationUrl)
  } catch (error: unknown) {
    oidcMsg.value = mapOIDCAccountError(error, 'link')
    oidcError.value = true
  } finally {
    linkingProviderId.value = null
  }
}

async function handleUnlinkIdentity(identityId: number, displayName: string) {
  const confirmed = await showConfirm(
    `Unlink ${displayName} from your account?`,
    { title: 'Unlink Sign-in Provider', variant: 'danger' },
  )
  if (!confirmed) return

  oidcMsg.value = ''
  oidcError.value = false
  unlinkingIdentityId.value = identityId
  try {
    await deleteOIDCIdentity(identityId)
    await loadOIDCAccounts()
    oidcMsg.value = `${displayName} unlinked.`
  } catch (error: unknown) {
    oidcMsg.value = mapOIDCAccountError(error, 'unlink')
    oidcError.value = true
  } finally {
    unlinkingIdentityId.value = null
  }
}

function mapOIDCAccountError(error: unknown, action: 'link' | 'unlink') {
  const response = getErrorResponse(error)
  const message = getApiErrorMessage(error)
  const normalized = message.toLowerCase()

  if (response?.status === 409 && action === 'link') {
    if (normalized.includes('another user') || normalized.includes('already linked')) {
      return 'This provider account is already linked to another user. Sign in with a different provider account or ask an administrator for help.'
    }
    return 'This provider account cannot be linked automatically. Sign in locally with the intended account, then try linking again.'
  }

  if (response?.status === 409 && action === 'unlink') {
    return 'This identity cannot be unlinked because your account would have no usable sign-in method. Add a password or another sign-in method first.'
  }

  if (response?.status === 404) {
    return 'That linked identity was not found for your account. Refresh settings and try again.'
  }

  if (normalized.includes('state') || normalized.includes('claims') || response?.status === 400) {
    return 'The provider response could not be validated. Start the linking flow again from Account Settings.'
  }

  if (normalized.includes('configuration') || normalized.includes('discovery') || response?.status === 500) {
    return 'The sign-in provider is not configured correctly. Ask an administrator to test the provider settings.'
  }

  return message || `Failed to ${action} OIDC identity.`
}

function getErrorResponse(error: unknown): { status?: number } | null {
  if (typeof error !== 'object' || error === null || !('response' in error)) return null
  const response = (error as { response?: unknown }).response
  if (typeof response !== 'object' || response === null) return null
  return response as { status?: number }
}

async function loadCredentials() {
  try {
    const res = await webauthnListCredentials()
    webauthnCredentials.value = res.data
  } catch {
    // silently fail
  }
}

function base64urlToBuffer(base64url: string): ArrayBuffer {
  const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/')
  const pad = base64.length % 4 === 0 ? '' : '='.repeat(4 - (base64.length % 4))
  const binary = atob(base64 + pad)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  return bytes.buffer
}

async function handleRegisterCredential() {
  registeringCredential.value = true
  credentialMsg.value = ''
  credentialError.value = false

  try {
    const beginRes = await webauthnRegisterBegin()
    const options = beginRes.data
    const publicKey = options.publicKey
    if (!publicKey?.challenge || !publicKey.user?.id) {
      throw new Error('Biometric registration is temporarily unavailable. Missing challenge data.')
    }

    const publicKeyOptions: PublicKeyCredentialCreationOptions = {
      challenge: base64urlToBuffer(publicKey.challenge),
      rp: publicKey.rp,
      user: {
        id: base64urlToBuffer(publicKey.user.id),
        name: publicKey.user.name,
        displayName: publicKey.user.displayName,
      },
      pubKeyCredParams: publicKey.pubKeyCredParams,
      timeout: publicKey.timeout || 60000,
      authenticatorSelection: publicKey.authenticatorSelection,
      attestation: publicKey.attestation || 'none',
      excludeCredentials: (publicKey.excludeCredentials || []).map((c: { id: string; type: string; transports?: string[] }) => ({
        id: base64urlToBuffer(c.id),
        type: c.type,
        transports: c.transports,
      })),
    }

    const credential = await navigator.credentials.create({
      publicKey: publicKeyOptions,
    }) as PublicKeyCredential

    await webauthnRegisterFinish(credential)

    credentialMsg.value = 'Biometric credential registered!'
    await loadCredentials()
  } catch (e: unknown) {
    credentialMsg.value = e instanceof Error ? e.message : 'Registration failed'
    credentialError.value = true
  } finally {
    registeringCredential.value = false
  }
}

async function handleDeleteCredential(id: number) {
  if (!await showConfirm('Remove this biometric credential?', { title: 'Remove Credential' })) return
  try {
    await webauthnDeleteCredential(id)
    await loadCredentials()
  } catch {
    credentialMsg.value = 'Failed to remove credential'
    credentialError.value = true
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric',
  })
}

function formatDateTime(dateStr: string) {
  return new Date(dateStr).toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  })
}

onMounted(() => {
  if (supportsWebAuthn) loadCredentials()
  void loadOIDCAccounts()
})

defineExpose({ loadCredentials, loadOIDCAccounts })
</script>
