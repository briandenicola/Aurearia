<template>
  <section class="settings-section card">
    <h2>Account</h2>

    <!-- Avatar -->
    <div class="setting-item avatar-section">
      <div class="avatar-preview">
        <img :src="avatarUrl" alt="Avatar" class="avatar-img" />
      </div>
      <div class="avatar-actions">
        <label class="btn btn-secondary btn-sm">
          Upload Avatar
          <input type="file" accept="image/*" hidden @change="handleAvatarUpload" />
        </label>
        <button v-if="auth.user?.avatarPath" class="btn btn-danger btn-sm" @click="handleAvatarDelete">Remove</button>
      </div>
    </div>

    <div class="setting-item">
      <div class="setting-info">
        <span class="setting-label">Username</span>
        <span class="setting-value">{{ auth.user?.username }}</span>
      </div>
    </div>
    <div class="setting-item">
      <div class="setting-info">
        <span class="setting-label">Role</span>
        <span class="setting-value badge" :class="`badge-${auth.user?.role === 'admin' ? 'roman' : 'modern'}`">
          {{ auth.user?.role }}
        </span>
      </div>
    </div>

    <!-- Profile / Social Settings -->
    <h3>Profile</h3>
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
      <span class="setting-desc" style="font-size: 0.8rem; margin-top: 0.25rem; display: block">Used by the Agent to find nearby coin shows and dealers</span>
    </div>

    <h3>NumisBids Integration</h3>
    <p class="setting-desc" style="margin-bottom: 0.75rem">
      Connect your NumisBids account to sync your watchlist and track auction lots.
    </p>
    <div class="form-group">
      <label class="form-label">NumisBids Username</label>
      <input v-model="nbUsername" class="form-input" placeholder="Your NumisBids username" autocomplete="off" />
    </div>
    <div class="form-group">
      <label class="form-label">NumisBids Password</label>
      <input v-model="nbPassword" type="password" class="form-input" placeholder="Your NumisBids password" autocomplete="new-password" />
      <span class="setting-desc" style="font-size: 0.8rem; margin-top: 0.25rem; display: block">Stored securely on the server. Used only for watchlist sync.</span>
    </div>
    <div v-if="nbValidating" class="nb-status validating">
      Validating NumisBids credentials...
    </div>
    <div v-else-if="nbValidationError" class="nb-status error">
      {{ nbValidationError }}
    </div>
    <div v-else-if="auth.user?.numisBidsConfigured" class="nb-status connected">
      NumisBids account connected
    </div>
    <div class="setting-item">
      <div class="setting-info">
        <span class="setting-label">Public Collection</span>
        <span class="setting-desc">Allow other users to follow you and view your coins</span>
      </div>
      <label class="toggle">
        <input type="checkbox" :checked="profilePublic" @change="onPublicToggle" />
        <span class="toggle-slider"></span>
      </label>
    </div>
    <button class="btn btn-primary btn-sm" @click="handleSaveProfile" :disabled="profileSaving || nbValidating" style="margin-top: 0.5rem">
      {{ nbValidating ? 'Validating...' : profileSaving ? 'Saving...' : 'Save Profile' }}
    </button>
    <p v-if="profileMsg" class="msg" :class="{ error: profileError }" style="margin-top: 0.5rem">{{ profileMsg }}</p>

    <!-- Privacy Warning Modal -->
    <Teleport to="body">
      <div v-if="showPrivacyWarning" class="modal-overlay" @click.self="cancelGoPrivate">
        <div class="modal-content card" style="max-width: 440px;">
          <div class="modal-header">
            <h2 style="display: flex; align-items: center; gap: 0.5rem; margin: 0; font-size: 1rem;">
              ⚠️ Make Collection Private?
            </h2>
          </div>
          <div class="modal-body" style="padding: 1.25rem;">
            <p style="color: var(--text-secondary); line-height: 1.5; margin: 0 0 0.75rem;">
              Setting your profile to private will <strong style="color: var(--text-primary);">permanently remove all your followers</strong>.
              They will need to send new follow requests if you make your profile public again.
            </p>
            <p style="color: var(--text-secondary); line-height: 1.5; margin: 0 0 1rem;">
              You will also be hidden from user search results.
            </p>
            <div style="display: flex; gap: 0.75rem; justify-content: flex-end;">
              <button class="btn btn-secondary btn-sm" @click="cancelGoPrivate">Cancel</button>
              <button class="btn btn-danger btn-sm" @click="confirmGoPrivate">Make Private</button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <h3>Change Password</h3>
    <form class="password-form" @submit.prevent="handleChangePassword">
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
      <p v-if="passwordMsg" class="msg" :class="{ error: passwordError }">{{ passwordMsg }}</p>
      <button type="submit" class="btn btn-primary btn-sm" :disabled="passwordLoading">
        {{ passwordLoading ? 'Changing...' : 'Change Password' }}
      </button>
    </form>

    <template v-if="supportsWebAuthn">
      <h3>Biometric Login</h3>
      <p class="setting-desc" style="margin-bottom: 0.75rem">
        Register Face ID, Touch ID, or fingerprint for quick sign-in on this device.
      </p>

      <button
        class="btn btn-primary btn-sm"
        :disabled="registeringCredential"
        @click="handleRegisterCredential"
      >
        {{ registeringCredential ? 'Registering...' : '🔐 Register Biometric' }}
      </button>
      <p v-if="credentialMsg" class="msg" :class="{ error: credentialError }" style="margin-top: 0.5rem">{{ credentialMsg }}</p>

      <div v-if="webauthnCredentials.length" class="apikey-list">
        <div v-for="cred in webauthnCredentials" :key="cred.id" class="apikey-item">
          <div class="apikey-item-info">
            <span class="apikey-item-name">{{ cred.name }}</span>
            <span class="apikey-item-meta">Registered {{ formatDate(cred.createdAt) }}</span>
          </div>
          <button class="btn btn-danger btn-sm" @click="handleDeleteCredential(cred.id)">Remove</button>
        </div>
      </div>
      <p v-else-if="!registeringCredential" class="setting-desc" style="margin-top: 0.5rem">No biometric credentials registered.</p>
    </template>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import {
  webauthnRegisterBegin, webauthnRegisterFinish,
  webauthnListCredentials, webauthnDeleteCredential,
} from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import { useSettingsProfile } from '@/composables/useSettingsProfile'
import type { WebAuthnCredentialInfo } from '@/types'

const auth = useAuthStore()
const { showConfirm } = useDialog()

const {
  avatarUrl, handleAvatarUpload, handleAvatarDelete,
  profileEmail, profileBio, profileZipCode,
  nbUsername, nbPassword, profilePublic, profileMsg, profileError, profileSaving,
  showPrivacyWarning, onPublicToggle, confirmGoPrivate, cancelGoPrivate,
  nbValidating, nbValidationError, handleSaveProfile,
  currentPassword, newPassword, confirmPassword,
  passwordMsg, passwordError, passwordLoading, handleChangePassword,
} = useSettingsProfile()

// WebAuthn Biometric
const supportsWebAuthn = !!window.PublicKeyCredential
const webauthnCredentials = ref<WebAuthnCredentialInfo[]>([])
const registeringCredential = ref(false)
const credentialMsg = ref('')
const credentialError = ref(false)

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

    const publicKeyOptions: PublicKeyCredentialCreationOptions = {
      challenge: base64urlToBuffer(options.publicKey.challenge),
      rp: options.publicKey.rp,
      user: {
        id: base64urlToBuffer(options.publicKey.user.id),
        name: options.publicKey.user.name,
        displayName: options.publicKey.user.displayName,
      },
      pubKeyCredParams: options.publicKey.pubKeyCredParams,
      timeout: options.publicKey.timeout || 60000,
      authenticatorSelection: options.publicKey.authenticatorSelection,
      attestation: options.publicKey.attestation || 'none',
      excludeCredentials: (options.publicKey.excludeCredentials || []).map((c: { id: string; type: string; transports?: string[] }) => ({
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

onMounted(() => {
  if (supportsWebAuthn) loadCredentials()
})

defineExpose({ loadCredentials })
</script>

<style scoped>
.avatar-section {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.avatar-preview {
  flex-shrink: 0;
}

.avatar-img {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--accent-gold-dim, #c9a84c);
}

.avatar-actions {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.nb-status {
  font-size: 0.82rem;
  padding: 0.4rem 0.75rem;
  border-radius: var(--radius-sm);
  margin-top: 0.25rem;
}

.nb-status.connected {
  background: rgba(74, 222, 128, 0.1);
  color: #4ade80;
  border: 1px solid rgba(74, 222, 128, 0.2);
}

.nb-status.validating {
  background: rgba(250, 204, 21, 0.1);
  color: #facc15;
  border: 1px solid rgba(250, 204, 21, 0.2);
}

.nb-status.error {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.2);
}

.setting-value {
  font-size: 0.9rem;
  color: var(--text-secondary);
}

.password-form {
  max-width: 350px;
}

.msg {
  font-size: 0.85rem;
  color: var(--accent-gold);
  margin: 0.5rem 0;
}

.msg.error {
  color: #e74c3c;
}

.apikey-list {
  margin-top: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.apikey-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.6rem 0;
  border-bottom: 1px solid var(--border-subtle);
  gap: 0.75rem;
}

.apikey-item:last-child {
  border-bottom: none;
}

.apikey-item-info {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  min-width: 0;
}

.apikey-item-name {
  font-size: 0.9rem;
  font-weight: 500;
}

.apikey-item-meta {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 15vh 1rem;
  z-index: 1000;
}

.modal-content {
  width: 100%;
}

.modal-header {
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--border-subtle);
}

.modal-body {
  padding: 1.25rem;
}

.settings-section h2 {
  font-size: 1.1rem;
  margin-bottom: 1.25rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
}

.settings-section h3 {
  font-size: 0.95rem;
  margin-top: 1.25rem;
  margin-bottom: 0.75rem;
  color: var(--text-secondary);
}

.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 0;
  border-bottom: 1px solid var(--border-subtle);
  gap: 1rem;
}

.setting-item:last-child {
  border-bottom: none;
}

.setting-info {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.setting-label {
  font-size: 0.9rem;
  font-weight: 500;
}

.setting-desc {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.btn-danger {
  background: #e74c3c;
  color: #fff;
  border: none;
  cursor: pointer;
}

.btn-danger:hover {
  background: #c0392b;
}

@media (max-width: 640px) {
  .setting-item {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
