<template>
  <PullToRefresh :on-refresh="handleRefresh">
  <div class="container">
    <div class="page-header">
      <h1>Settings</h1>
      <div v-if="isPwa" class="settings-menu-wrapper">
        <button class="btn btn-secondary btn-sm settings-menu-btn" @click="settingsMenuOpen = !settingsMenuOpen">
          <component :is="tabIcons[activeTab]" :size="16" />
          {{ tabs.find(t => t.id === activeTab)?.label }}
          <Menu :size="16" />
        </button>
        <Transition name="fade">
          <div v-if="settingsMenuOpen" class="settings-dropdown">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              class="settings-dropdown-item"
              :class="{ active: activeTab === tab.id }"
              @click="selectTab(tab.id); settingsMenuOpen = false"
            >
              <component :is="tabIcons[tab.id]" :size="16" />
              {{ tab.label }}
            </button>
          </div>
        </Transition>
      </div>
    </div>

    <div class="settings-layout">
      <!-- Tab Nav (desktop only) -->
      <div v-if="!isPwa" class="tab-nav">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          class="tab-btn"
          :class="{ active: activeTab === tab.id }"
          @click="selectTab(tab.id)"
        ><component :is="tabIcons[tab.id]" :size="16" /> {{ tab.label }}</button>
      </div>

      <!-- Account Tab -->
      <section v-if="activeTab === 'account'" class="settings-section card">
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

      <!-- Appearance Tab -->
      <SettingsAppearanceSection
        v-if="activeTab === 'appearance'"
        :theme="theme"
        :timezone="timezone"
        :timezones="timezones"
        :default-view="defaultView"
        :default-sort="defaultSort"
        @set-theme="setTheme"
        @save-timezone="(tz: string) => { timezone = tz; saveTimezone() }"
        @set-default-view="setDefaultView"
        @save-default-sort="(sort: string) => { defaultSort = sort; saveDefaultSort() }"
      />

      <!-- Data Tab -->
      <section v-if="activeTab === 'data'" class="settings-section card">
        <h2>Data Management</h2>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Export Collection</span>
            <span class="setting-desc">Download your collection data and photos as a zip archive</span>
          </div>
          <button class="btn btn-secondary btn-sm" :disabled="exporting" @click="handleExport">
            {{ exporting ? 'Exporting...' : 'Export ZIP' }}
          </button>
        </div>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">PDF Catalog</span>
            <span class="setting-desc">Generate a styled PDF catalog with photos, grades, and valuations</span>
          </div>
          <button class="btn btn-secondary btn-sm" :disabled="exportingPdf" @click="handleExportPDF">
            {{ exportingPdf ? 'Generating...' : 'Export PDF' }}
          </button>
        </div>
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Import Collection</span>
            <span class="setting-desc">Import coins from a JSON file</span>
          </div>
          <label class="btn btn-secondary btn-sm import-btn">
            📤 Import
            <input type="file" accept=".json" hidden @change="handleImport" />
          </label>
        </div>
        <p v-if="dataMsg" class="msg" :class="{ error: dataError }">{{ dataMsg }}</p>

        <h3>API Keys</h3>
        <p class="setting-desc" style="margin-bottom: 1rem">
          Generate API keys to access your collection from external tools and scripts. Use the <code>X-API-Key</code> header to authenticate.
        </p>

        <div class="apikey-generate">
          <input
            v-model="apiKeyName"
            type="text"
            class="form-input"
            placeholder="Key name (e.g. My Script)"
            :disabled="generatingKey"
          />
          <button
            class="btn btn-primary btn-sm"
            :disabled="!apiKeyName.trim() || generatingKey"
            @click="handleGenerateKey"
          >
            {{ generatingKey ? 'Generating...' : '🔑 Generate Key' }}
          </button>
        </div>

        <div v-if="newlyGeneratedKey" class="apikey-reveal">
          <p class="apikey-reveal-warning">
            ⚠️ Copy this key now — it will not be shown again.
          </p>
          <div class="apikey-reveal-box">
            <code class="apikey-reveal-value">{{ newlyGeneratedKey }}</code>
            <button class="btn btn-secondary btn-sm" @click="copyKey">
              {{ keyCopied ? '✓ Copied' : '📋 Copy' }}
            </button>
          </div>
        </div>

        <p v-if="apiKeyMsg" class="msg" :class="{ error: apiKeyError }">{{ apiKeyMsg }}</p>

        <div v-if="apiKeys.length" class="apikey-list">
          <div
            v-for="key in apiKeys"
            :key="key.id"
            class="apikey-item"
            :class="{ revoked: key.revokedAt }"
          >
            <div class="apikey-item-info">
              <span class="apikey-item-name">{{ key.name }}</span>
              <span class="apikey-item-meta">
                ...{{ key.keyPrefix }}
                · Created {{ formatDate(key.createdAt) }}
                <template v-if="key.lastUsedAt"> · Last used {{ formatDate(key.lastUsedAt) }}</template>
              </span>
            </div>
            <span v-if="key.revokedAt" class="apikey-item-badge revoked-badge">Revoked</span>
            <button
              v-else
              class="btn btn-danger btn-sm"
              @click="handleRevokeKey(key.id)"
            >
              Revoke
            </button>
          </div>
        </div>
        <p v-else-if="!generatingKey" class="setting-desc" style="margin-top: 0.5rem">No API keys yet.</p>

        <h3 style="margin-top: 2rem">Tags</h3>
        <p class="setting-desc">Create custom tags to organize and filter your coins.</p>

        <div class="tag-create-form">
          <input
            v-model="newTagName"
            type="text"
            class="form-input"
            placeholder="New tag name..."
            maxlength="50"
            @keydown.enter="handleCreateTag"
          />
          <div class="tag-color-picker">
            <button
              v-for="c in TAG_COLORS"
              :key="c"
              class="color-swatch"
              :class="{ active: newTagColor === c }"
              :style="{ backgroundColor: c }"
              @click="newTagColor = c"
            ></button>
          </div>
          <button class="btn btn-primary btn-sm" @click="handleCreateTag" :disabled="!newTagName.trim()">Create Tag</button>
        </div>
        <p v-if="tagError" class="tag-error">{{ tagError }}</p>

        <div v-if="tagList.length" class="tag-list">
          <div v-for="tag in tagList" :key="tag.id" class="tag-list-item">
            <template v-if="editingTag?.id === tag.id">
              <input v-model="editTagName" class="form-input tag-edit-input" maxlength="50" @keydown.enter="handleSaveTag" />
              <div class="tag-color-picker">
                <button
                  v-for="c in TAG_COLORS"
                  :key="c"
                  class="color-swatch sm"
                  :class="{ active: editTagColor === c }"
                  :style="{ backgroundColor: c }"
                  @click="editTagColor = c"
                ></button>
              </div>
              <button class="btn btn-primary btn-sm" @click="handleSaveTag">Save</button>
              <button class="btn btn-secondary btn-sm" @click="editingTag = null">Cancel</button>
            </template>
            <template v-else>
              <span class="tag-preview" :style="{ backgroundColor: tag.color + '22', color: tag.color, borderColor: tag.color + '44' }">{{ tag.name }}</span>
              <div class="tag-actions">
                <button class="btn btn-secondary btn-sm" @click="startEditTag(tag)">Edit</button>
                <button class="btn btn-danger btn-sm" @click="handleDeleteTag(tag)">Delete</button>
              </div>
            </template>
          </div>
        </div>
        <p v-else class="empty-tags">No tags created yet. Create your first tag above.</p>
      </section>

      <!-- Tools Tab -->
      <SettingsToolsSection
        v-if="activeTab === 'tools'"
        :blocked-users="blockedUsers"
        :blocked-loading="blockedLoading"
        @saved="handleProcessSaved"
        @unblock="handleUnblock"
      />

      <!-- Conversations Tab -->
      <SavedConversationsSection
        v-if="activeTab === 'conversations'"
        :conversations="conversations"
        :loading="conversationsLoading"
        @open="openConversation"
        @delete="handleDeleteConversation"
      />

      <!-- Help Tab -->
      <HelpSection v-if="activeTab === 'help'" />

      <CoinSearchChat
        v-if="showChat"
        :loadConversation="chatConversation"
        @close="showChat = false; chatConversation = null"
        @added="() => {}"
      />
    </div>
  </div>
  </PullToRefresh>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import PullToRefresh from '@/components/PullToRefresh.vue'
import {
  changePassword, exportCollection, exportCatalogPDF, importCollection,
  generateApiKey, listApiKeys, revokeApiKey,
  webauthnRegisterBegin, webauthnRegisterFinish,
  webauthnListCredentials, webauthnDeleteCredential,
  listConversations, getConversation, deleteConversation,
  uploadAvatar, deleteAvatar, updateProfile, getMe,
  getBlockedUsers, unblockFollower, validateNumisBidsCredentials,
  getTags, createTag, updateTag as updateTagApi, deleteTag,
} from '@/api/client'
import type { ConversationSummary } from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import type { Coin, Theme, ApiKey, WebAuthnCredentialInfo, Tag } from '@/types'
import CoinSearchChat from '@/components/CoinSearchChat.vue'
import ImageProcessor from '@/components/ImageProcessor.vue'
import HelpSection from '@/components/HelpSection.vue'
import SettingsAppearanceSection from '@/components/settings/SettingsAppearanceSection.vue'
import SavedConversationsSection from '@/components/settings/SavedConversationsSection.vue'
import SettingsToolsSection from '@/components/settings/SettingsToolsSection.vue'
import { User, Palette, Database, MessageSquare, HelpCircle, Wrench, Menu, ShieldCheck, Tags } from 'lucide-vue-next'

const tabIcons: Record<string, Component> = {
  account: User,
  appearance: Palette,
  data: Database,
  tools: Wrench,
  conversations: MessageSquare,
  help: HelpCircle,
  admin: ShieldCheck,
}

const { showConfirm, showAlert } = useDialog()
const activeTab = ref('account')
const settingsMenuOpen = ref(false)
const isPwa = window.matchMedia('(display-mode: standalone)').matches
  || (window.navigator as any).standalone === true

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const baseTabs = [
  { id: 'account', label: 'Account' },
  { id: 'appearance', label: 'Appearance' },
  { id: 'data', label: 'Data' },
  { id: 'tools', label: 'Tools' },
  { id: 'conversations', label: 'Conversations' },
  { id: 'help', label: 'Help' },
]
const tabs = computed(() => {
  if (isPwa && auth.isAdmin) {
    return [
      { id: 'account', label: 'Account' },
      { id: 'admin', label: 'Admin' },
      { id: 'appearance', label: 'Appearance' },
      { id: 'data', label: 'Data' },
      { id: 'tools', label: 'Tools' },
      { id: 'conversations', label: 'Conversations' },
      { id: 'help', label: 'Help' },
    ]
  }
  return baseTabs
})

// Set active tab from query param (e.g. /settings?tab=process)
if (route.query.tab && baseTabs.map(t => t.id).concat('admin').includes(route.query.tab as string)) {
  activeTab.value = route.query.tab as string
}

function selectTab(tabId: string) {
  if (tabId === 'admin') {
    router.push('/admin')
    return
  }
  activeTab.value = tabId
}

function handleProcessSaved(savedCoinId: number) {
  router.push(`/edit/${savedCoinId}`)
}

// Blocked users
const blockedUsers = ref<{ id: number; username: string; avatarPath: string }[]>([])
const blockedLoading = ref(false)

async function loadBlockedUsers() {
  try {
    const res = await getBlockedUsers()
    blockedUsers.value = res.data.blocked
  } catch {
    blockedUsers.value = []
  }
}

async function handleUnblock(user: { id: number; username: string; avatarPath: string }) {
  blockedLoading.value = true
  try {
    await unblockFollower(user.id)
    blockedUsers.value = blockedUsers.value.filter(u => u.id !== user.id)
  } catch {
    // ignore
  } finally {
    blockedLoading.value = false
  }
}

// Avatar
const avatarUrl = ref('/coin-logo.jpg')

// Tag management
const tagList = ref<Tag[]>([])
const newTagName = ref('')
const newTagColor = ref('#6b7280')
const editingTag = ref<Tag | null>(null)
const editTagName = ref('')
const editTagColor = ref('')
const tagError = ref('')

const TAG_COLORS = ['#6b7280', '#ef4444', '#f59e0b', '#10b981', '#3b82f6', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316', '#6366f1']

async function loadTags() {
  try {
    const res = await getTags()
    tagList.value = res.data?.tags ?? []
  } catch { tagList.value = [] }
}

async function handleCreateTag() {
  tagError.value = ''
  const name = newTagName.value.trim()
  if (!name) return
  try {
    await createTag({ name, color: newTagColor.value })
    newTagName.value = ''
    newTagColor.value = '#6b7280'
    await loadTags()
  } catch (e: any) {
    tagError.value = e?.response?.data?.error ?? 'Failed to create tag'
  }
}

function startEditTag(tag: Tag) {
  editingTag.value = tag
  editTagName.value = tag.name
  editTagColor.value = tag.color
}

async function handleSaveTag() {
  tagError.value = ''
  if (!editingTag.value) return
  try {
    await updateTagApi(editingTag.value.id, { name: editTagName.value.trim(), color: editTagColor.value })
    editingTag.value = null
    await loadTags()
  } catch (e: any) {
    tagError.value = e?.response?.data?.error ?? 'Failed to update tag'
  }
}

async function handleDeleteTag(tag: Tag) {
  const confirmed = await showConfirm(`Delete tag "${tag.name}"? It will be removed from all coins.`, { title: 'Delete Tag', variant: 'danger' })
  if (!confirmed) return
  try {
    await deleteTag(tag.id)
    await loadTags()
  } catch { /* ignore */ }
}

function updateAvatarUrl() {
  avatarUrl.value = auth.user?.avatarPath ? `/uploads/${auth.user.avatarPath}` : '/coin-logo.jpg'
}
updateAvatarUrl()

async function handleAvatarUpload(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  try {
    const res = await uploadAvatar(file)
    if (auth.user) {
      auth.user.avatarPath = res.data.avatarPath
      localStorage.setItem('user', JSON.stringify(auth.user))
    }
    updateAvatarUrl()
  } catch { /* ignore */ }
}

async function handleAvatarDelete() {
  try {
    await deleteAvatar()
    if (auth.user) {
      auth.user.avatarPath = ''
      localStorage.setItem('user', JSON.stringify(auth.user))
    }
    updateAvatarUrl()
  } catch { /* ignore */ }
}

// Profile
const profileEmail = ref(auth.user?.email || '')
const profileBio = ref(auth.user?.bio || '')
const profileZipCode = ref(auth.user?.zipCode || '')
const nbUsername = ref(auth.user?.numisBidsUsername || '')
const nbPassword = ref('')
const profilePublic = ref(auth.user?.isPublic || false)
const profileMsg = ref('')
const profileError = ref(false)
const profileSaving = ref(false)
const showPrivacyWarning = ref(false)

function onPublicToggle(e: Event) {
  const checked = (e.target as HTMLInputElement).checked
  if (!checked && profilePublic.value) {
    // Going from public → private: show warning
    ;(e.target as HTMLInputElement).checked = true // revert checkbox visually
    showPrivacyWarning.value = true
  } else {
    profilePublic.value = checked
  }
}

function confirmGoPrivate() {
  profilePublic.value = false
  showPrivacyWarning.value = false
}

function cancelGoPrivate() {
  showPrivacyWarning.value = false
}

const nbValidating = ref(false)
const nbValidationError = ref('')

async function handleSaveProfile() {
  profileMsg.value = ''
  profileError.value = false
  profileSaving.value = true
  nbValidationError.value = ''
  try {
    // If user entered new NB credentials, validate them first
    if (nbPassword.value && nbUsername.value) {
      nbValidating.value = true
      try {
        const valRes = await validateNumisBidsCredentials(nbUsername.value, nbPassword.value)
        if (!valRes.data.valid) {
          nbValidationError.value = valRes.data.error || 'Invalid NumisBids credentials'
          profileSaving.value = false
          nbValidating.value = false
          return
        }
      } catch {
        nbValidationError.value = 'Could not validate NumisBids credentials. Check your username and password.'
        profileSaving.value = false
        nbValidating.value = false
        return
      } finally {
        nbValidating.value = false
      }
    }

    const data: Record<string, unknown> = {
      email: profileEmail.value,
      bio: profileBio.value,
      zipCode: profileZipCode.value,
      isPublic: profilePublic.value,
      numisBidsUsername: nbUsername.value,
    }
    if (nbPassword.value) {
      data.numisBidsPassword = nbPassword.value
    }
    const res = await updateProfile(data as Parameters<typeof updateProfile>[0])
    if (auth.user) {
      auth.user.email = res.data.email
      auth.user.bio = res.data.bio
      auth.user.zipCode = res.data.zipCode
      auth.user.isPublic = res.data.isPublic
      auth.user.numisBidsUsername = res.data.numisBidsUsername
      auth.user.numisBidsConfigured = res.data.numisBidsConfigured
      localStorage.setItem('user', JSON.stringify(auth.user))
    }
    nbPassword.value = ''
    profileMsg.value = 'Profile saved'
  } catch {
    profileMsg.value = 'Failed to save profile'
    profileError.value = true
  } finally {
    profileSaving.value = false
  }
}

// Password
const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const passwordMsg = ref('')
const passwordError = ref(false)
const passwordLoading = ref(false)

async function handleChangePassword() {
  passwordMsg.value = ''
  passwordError.value = false

  if (newPassword.value !== confirmPassword.value) {
    passwordMsg.value = 'New passwords do not match'
    passwordError.value = true
    return
  }

  passwordLoading.value = true
  try {
    await changePassword(currentPassword.value, newPassword.value)
    passwordMsg.value = 'Password changed successfully'
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
  } catch {
    passwordMsg.value = 'Failed — check your current password'
    passwordError.value = true
  } finally {
    passwordLoading.value = false
  }
}

// Theme
const theme = ref<Theme>((localStorage.getItem('theme') as Theme) || 'dark')

function setTheme(t: Theme) {
  theme.value = t
  localStorage.setItem('theme', t)
  document.documentElement.setAttribute('data-theme', t)
}

// Timezone
const timezones = (Intl as any).supportedValuesOf('timeZone') as string[]
const timezone = ref(localStorage.getItem('timezone') || Intl.DateTimeFormat().resolvedOptions().timeZone)

function saveTimezone() {
  localStorage.setItem('timezone', timezone.value)
}

// Default view
const defaultView = ref<'swipe' | 'grid'>((localStorage.getItem('defaultView') as 'swipe' | 'grid') || 'swipe')

function setDefaultView(v: 'swipe' | 'grid') {
  defaultView.value = v
  localStorage.setItem('defaultView', v)
}

// Default sort
const defaultSort = ref(localStorage.getItem('defaultSort') || 'updated_at_desc')

function saveDefaultSort() {
  localStorage.setItem('defaultSort', defaultSort.value)
}

// Data
const exporting = ref(false)
const exportingPdf = ref(false)
const dataMsg = ref('')
const dataError = ref(false)

async function handleExport() {
  exporting.value = true
  dataMsg.value = ''
  try {
    const res = await exportCollection()
    const blob = new Blob([res.data], { type: 'application/zip' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `ancient-coins-export-${new Date().toISOString().slice(0, 10)}.zip`
    a.click()
    URL.revokeObjectURL(url)
    dataMsg.value = 'Export downloaded'
  } catch {
    dataMsg.value = 'Export failed'
    dataError.value = true
  } finally {
    exporting.value = false
  }
}

async function handleExportPDF() {
  exportingPdf.value = true
  dataMsg.value = ''
  try {
    const res = await exportCatalogPDF()
    const blob = new Blob([res.data], { type: 'application/pdf' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `coin-catalog-${new Date().toISOString().slice(0, 10)}.pdf`
    a.click()
    URL.revokeObjectURL(url)
    dataMsg.value = 'PDF catalog downloaded'
  } catch {
    dataMsg.value = 'PDF generation failed'
    dataError.value = true
  } finally {
    exportingPdf.value = false
  }
}

async function handleImport(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return

  dataMsg.value = ''
  dataError.value = false

  try {
    const text = await file.text()
    const coins: Coin[] = JSON.parse(text)
    const res = await importCollection(coins)
    dataMsg.value = `Imported ${res.data.imported} coins`
  } catch {
    dataMsg.value = 'Import failed — ensure valid JSON format'
    dataError.value = true
  }
}

// API Keys
const apiKeys = ref<ApiKey[]>([])
const apiKeyName = ref('')
const newlyGeneratedKey = ref('')
const keyCopied = ref(false)
const generatingKey = ref(false)
const apiKeyMsg = ref('')
const apiKeyError = ref(false)

async function loadApiKeys() {
  try {
    const res = await listApiKeys()
    apiKeys.value = res.data
  } catch {
    // silently fail on load
  }
}

async function handleGenerateKey() {
  if (!apiKeyName.value.trim()) return

  generatingKey.value = true
  apiKeyMsg.value = ''
  apiKeyError.value = false
  newlyGeneratedKey.value = ''
  keyCopied.value = false

  try {
    const res = await generateApiKey(apiKeyName.value.trim())
    newlyGeneratedKey.value = res.data.key
    apiKeyName.value = ''
    await loadApiKeys()
  } catch {
    apiKeyMsg.value = 'Failed to generate API key'
    apiKeyError.value = true
  } finally {
    generatingKey.value = false
  }
}

async function copyKey() {
  try {
    await navigator.clipboard.writeText(newlyGeneratedKey.value)
    keyCopied.value = true
    setTimeout(() => { keyCopied.value = false }, 3000)
  } catch {
    // Fallback for non-HTTPS contexts
    const textarea = document.createElement('textarea')
    textarea.value = newlyGeneratedKey.value
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    keyCopied.value = true
    setTimeout(() => { keyCopied.value = false }, 3000)
  }
}

async function handleRevokeKey(id: number) {
  apiKeyMsg.value = ''
  apiKeyError.value = false
  try {
    await revokeApiKey(id)
    await loadApiKeys()
    newlyGeneratedKey.value = ''
  } catch {
    apiKeyMsg.value = 'Failed to revoke key'
    apiKeyError.value = true
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric',
  })
}

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
    // Begin registration — get options from server
    const beginRes = await webauthnRegisterBegin()
    const options = beginRes.data

    // Convert base64url fields to ArrayBuffers for the browser API
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
      excludeCredentials: (options.publicKey.excludeCredentials || []).map((c: any) => ({
        id: base64urlToBuffer(c.id),
        type: c.type,
        transports: c.transports,
      })),
    }

    // Call browser WebAuthn API (triggers Face ID / fingerprint prompt)
    const credential = await navigator.credentials.create({
      publicKey: publicKeyOptions,
    }) as PublicKeyCredential

    // Finish registration — send attestation to server
    await webauthnRegisterFinish(credential)

    credentialMsg.value = 'Biometric credential registered!'
    await loadCredentials()
  } catch (e: any) {
    credentialMsg.value = e?.message || 'Registration failed'
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

// Saved Conversations
const conversations = ref<ConversationSummary[]>([])
const conversationsLoading = ref(false)
const showChat = ref(false)
const chatConversation = ref<{ id: number; title: string; messages: string } | null>(null)

async function loadConversations() {
  conversationsLoading.value = true
  try {
    const res = await listConversations()
    conversations.value = res.data
  } catch {
    // silently fail
  } finally {
    conversationsLoading.value = false
  }
}

async function openConversation(id: number) {
  try {
    const res = await getConversation(id)
    chatConversation.value = {
      id: res.data.id,
      title: res.data.title,
      messages: res.data.messages,
    }
    showChat.value = true
  } catch {
    await showAlert('Failed to load conversation', { title: 'Error' })
  }
}

async function handleDeleteConversation(id: number) {
  if (!await showConfirm('Delete this saved conversation?', { title: 'Delete Conversation', variant: 'danger' })) return
  try {
    await deleteConversation(id)
    conversations.value = conversations.value.filter(c => c.id !== id)
  } catch {
    await showAlert('Failed to delete conversation', { title: 'Error' })
  }
}

async function handleRefresh() {
  await Promise.all([
    loadApiKeys(),
    loadConversations(),
    loadBlockedUsers(),
    supportsWebAuthn ? loadCredentials() : Promise.resolve(),
  ])
}

onMounted(() => {
  loadApiKeys()
  loadConversations()
  loadBlockedUsers()
  loadTags()
  if (supportsWebAuthn) loadCredentials()
})
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

.settings-layout {
  max-width: 800px;
  margin-left: auto;
  margin-right: auto;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.tab-nav {
  display: flex;
  gap: 0.25rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 0.3rem;
}

.tab-btn {
  flex: 1;
  padding: 0.6rem 1rem;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
}

.tab-btn.active {
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.tab-btn:hover:not(.active) {
  color: var(--text-primary);
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

.import-btn {
  cursor: pointer;
}

.msg {
  font-size: 0.85rem;
  color: var(--accent-gold);
  margin: 0.5rem 0;
}

.msg.error {
  color: #e74c3c;
}

.apikey-generate {
  display: flex;
  gap: 0.75rem;
  align-items: center;
  margin-bottom: 0.75rem;
}

.apikey-generate .form-input {
  flex: 1;
  max-width: 280px;
}

.apikey-reveal {
  background: var(--bg-primary);
  border: 1px solid var(--accent-gold-dim);
  border-radius: var(--radius-sm);
  padding: 0.75rem 1rem;
  margin-bottom: 0.75rem;
}

.apikey-reveal-warning {
  font-size: 0.8rem;
  color: var(--accent-gold);
  margin-bottom: 0.5rem;
  font-weight: 500;
}

.apikey-reveal-box {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.apikey-reveal-value {
  flex: 1;
  font-size: 0.78rem;
  background: var(--bg-card);
  padding: 0.4rem 0.6rem;
  border-radius: var(--radius-sm);
  word-break: break-all;
  user-select: all;
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

.apikey-item.revoked {
  opacity: 0.5;
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

.revoked-badge {
  font-size: 0.7rem;
  padding: 0.15rem 0.5rem;
  background: var(--bg-primary);
  border-radius: var(--radius-full);
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

/* Settings hamburger menu (PWA) */
.page-header {
  display: flex;
  flex-direction: row !important;
  justify-content: space-between;
  align-items: center !important;
  margin-bottom: 1.5rem;
  flex-wrap: nowrap;
}

.settings-menu-wrapper {
  position: relative;
}

.settings-menu-btn {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.85rem;
}

.settings-dropdown {
  position: absolute;
  top: calc(100% + 0.5rem);
  right: 0;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
  z-index: 50;
  min-width: 180px;
  padding: 0.3rem;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.settings-dropdown-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.6rem 0.75rem;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
  text-align: left;
  width: 100%;
}

.settings-dropdown-item.active {
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.settings-dropdown-item:hover:not(.active) {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@media (max-width: 640px) {
  .setting-item {
    flex-direction: column;
    align-items: stretch;
  }

  .tab-nav {
    flex-wrap: wrap;
  }

  .tab-btn {
    font-size: 0.78rem;
    padding: 0.5rem 0.6rem;
  }
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

/* Tag Manager */
.tag-create-form {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
  margin: 1rem 0;
}

.tag-create-form .form-input {
  flex: 1;
  min-width: 150px;
}

.tag-color-picker {
  display: flex;
  gap: 0.3rem;
  align-items: center;
}

.color-swatch {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  padding: 0;
}

.color-swatch.active {
  border-color: var(--text-primary);
  box-shadow: 0 0 0 2px var(--bg-card);
}

.color-swatch.sm {
  width: 18px;
  height: 18px;
}

.tag-error {
  color: #ef4444;
  font-size: 0.85rem;
  margin-top: 0.25rem;
}

.tag-list {
  margin-top: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.tag-list-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  flex-wrap: wrap;
}

.tag-preview {
  font-size: 0.8rem;
  padding: 0.2rem 0.6rem;
  border-radius: 9999px;
  border: 1px solid;
  flex-shrink: 0;
}

.tag-edit-input {
  flex: 1;
  min-width: 120px;
}

.tag-actions {
  margin-left: auto;
  display: flex;
  gap: 0.25rem;
}

.empty-tags {
  color: var(--text-secondary);
  font-size: 0.85rem;
  margin-top: 1rem;
}
</style>
