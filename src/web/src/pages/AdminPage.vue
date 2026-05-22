<template>
  <div class="container">
    <div class="page-header">
      <h1>Admin</h1>
    </div>

    <div v-if="!auth.isAdmin" class="empty-state">
      <h3>Access Denied</h3>
      <p>Admin privileges required</p>
    </div>

    <div v-else class="admin-layout">
      <!-- Tab Nav -->
      <div class="tab-nav">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          class="tab-btn"
          :class="{ active: activeTab === tab.id }"
          @click="activeTab = tab.id"
        ><component :is="tabIcons[tab.id]" :size="16" /> {{ tab.label }}</button>
      </div>

      <!-- Users Tab -->
      <AdminUsersSection
        v-if="activeTab === 'users'"
        :users="users"
        :loading="usersLoading"
        :current-user-id="auth.user?.id ?? 0"
        @reset="openResetModal"
        @delete="handleDeleteUser"
      />

      <!-- AI Tab -->
      <AdminAISection
        v-if="activeTab === 'ai'"
        :settings="settings"
        :setting-defaults="settingDefaults"
        :settings-msg="settingsMsg"
        :settings-error="settingsError"
        :settings-saving="settingsSaving"
        :anthropic-models="anthropicModels"
        :anthropic-testing="anthropicTesting"
        :anthropic-test-result="anthropicTestResult"
        :anthropic-test-ok="anthropicTestOk"
        :ollama-testing="ollamaTesting"
        :ollama-test-result="ollamaTestResult"
        :ollama-test-ok="ollamaTestOk"
        :searxng-testing="searxngTesting"
        :searxng-test-result="searxngTestResult"
        :searxng-test-ok="searxngTestOk"
        :coin-search-prompt-default="coinSearchPromptDefault"
        :coin-shows-prompt-default="coinShowsPromptDefault"
        :valuation-prompt-default="valuationPromptDefault"
        @save="saveSettings"
        @test-anthropic-conn="testAnthropicConn"
        @test-ollama-connection="testOllamaConnection"
        @test-searxng-conn="testSearxngConn"
      />

      <!-- System Tab -->
      <AdminSystemSection
        v-if="activeTab === 'system'"
        :numista-api-key="settings.NumistaAPIKey ?? ''"
        :pushover-app-token="settings.PushoverAppToken ?? ''"
        :log-level="settings.LogLevel ?? ''"
        :log-levels="LOG_LEVELS"
        :saving="settingsSaving"
        :msg="settingsMsg"
        :error="settingsError"
        :app-version="appVersion"
        :build-date="buildDate"
        @save="onSystemSave"
      />

      <!-- Logs Tab -->
      <AdminLogsSection
        v-if="activeTab === 'logs'"
        :logs="logs"
        :loading="logsLoading"
        :filter="logsFilter"
        :auto-refresh="logsAutoRefresh"
        @load="loadLogs"
        @toggle-auto-refresh="toggleAutoRefresh"
        @export="exportLogs"
        @update:filter="logsFilter = $event"
      />

      <!-- Schedules Tab -->
      <AdminSchedulesSection
        v-if="activeTab === 'schedules'"
        :settings="settings"
        :settings-saving="settingsSaving"
        :avail-settings-msg="availSettingsMsg"
        :avail-settings-error="availSettingsError"
        :auction-settings-msg="auctionSettingsMsg"
        :auction-settings-error="auctionSettingsError"
        :val-settings-msg="valSettingsMsg"
        :val-settings-error="valSettingsError"
        @save="saveSettings"
        @update:val-settings-msg="valSettingsMsg = $event"
        @update:val-settings-error="valSettingsError = $event"
        @update:auction-settings-msg="auctionSettingsMsg = $event"
        @update:auction-settings-error="auctionSettingsError = $event"
      />

      <!-- Reset Password Modal -->
      <ResetPasswordModal :user="resetTarget" @close="resetTarget = null" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, type Component } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { getUsers, deleteUser, getAdminLogs } from '@/api/client'
import { LOG_LEVELS } from '@/types'
import type { UserInfo, LogEntry } from '@/types'
import { useDialog } from '@/composables/useDialog'
import { useAdminConfig } from '@/composables/useAdminConfig'
import ResetPasswordModal from '@/components/admin/ResetPasswordModal.vue'
import AdminUsersSection from '@/components/admin/AdminUsersSection.vue'
import AdminSystemSection from '@/components/admin/AdminSystemSection.vue'
import AdminLogsSection from '@/components/admin/AdminLogsSection.vue'
import AdminAISection from '@/components/admin/AdminAISection.vue'
import AdminSchedulesSection from '@/components/admin/AdminSchedulesSection.vue'
import { Users, Cpu, Wrench, ScrollText, CalendarClock } from 'lucide-vue-next'

const tabIcons: Record<string, Component> = { users: Users, ai: Cpu, system: Wrench, logs: ScrollText, schedules: CalendarClock }

const { showConfirm, showAlert } = useDialog()
const auth = useAuthStore()

const rawVersion = import.meta.env.VITE_APP_VERSION || 'dev'
const appVersion = computed(() => {
  if (rawVersion === 'dev') return 'dev'
  return rawVersion.length > 8 ? rawVersion.substring(0, 7) : rawVersion
})
const buildDate = computed(() => {
  const raw = import.meta.env.VITE_BUILD_DATE
  if (!raw) return ''
  try {
    return new Date(raw).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
  } catch {
    return raw
  }
})

const tabs = [
  { id: 'users', icon: 'users', label: 'Users' },
  { id: 'ai', icon: 'cpu', label: 'AI' },
  { id: 'system', icon: 'wrench', label: 'System' },
  { id: 'logs', icon: 'scroll-text', label: 'Logs' },
  { id: 'schedules', icon: 'calendar-clock', label: 'Schedules' },
]
const activeTab = ref('users')

// Users
const users = ref<UserInfo[]>([])
const usersLoading = ref(true)

async function loadUsers() {
  usersLoading.value = true
  try {
    const res = await getUsers()
    users.value = res.data
  } finally {
    usersLoading.value = false
  }
}

async function handleDeleteUser(user: UserInfo) {
  if (!await showConfirm(`Delete user "${user.username}" and all their data? This cannot be undone.`, { title: 'Delete User', variant: 'danger' })) return
  try {
    await deleteUser(user.id)
    users.value = users.value.filter((u) => u.id !== user.id)
  } catch {
    await showAlert('Failed to delete user', { title: 'Error' })
  }
}

// Reset password modal
const resetTarget = ref<UserInfo | null>(null)

function openResetModal(user: UserInfo) {
  resetTarget.value = user
}

// Settings (from composable)
const {
  settings, settingDefaults, settingsMsg, settingsError, settingsSaving,
  ollamaTesting, ollamaTestResult, ollamaTestOk,
  anthropicTesting, anthropicTestResult, anthropicTestOk, anthropicModels,
  searxngTesting, searxngTestResult, searxngTestOk,
  coinSearchPromptDefault, coinShowsPromptDefault, valuationPromptDefault,
  availSettingsMsg, availSettingsError, auctionSettingsMsg, auctionSettingsError, valSettingsMsg, valSettingsError,
  loadSettings, saveSettings,
  testOllamaConnection, testAnthropicConn, testSearxngConn,
  cleanup: cleanupAdminConfig,
} = useAdminConfig()

// Logs
const logs = ref<LogEntry[]>([])
const logsLoading = ref(false)
const logsFilter = ref('')
const logsAutoRefresh = ref(false)
let logsInterval: ReturnType<typeof setInterval> | null = null

async function loadLogs() {
  logsLoading.value = true
  try {
    const res = await getAdminLogs(500, logsFilter.value || undefined)
    logs.value = res.data.logs || []
  } catch { /* ignore */ } finally {
    logsLoading.value = false
  }
}

function toggleAutoRefresh() {
  logsAutoRefresh.value = !logsAutoRefresh.value
  if (logsAutoRefresh.value) {
    logsInterval = setInterval(loadLogs, 3000)
  } else if (logsInterval) {
    clearInterval(logsInterval)
    logsInterval = null
  }
}

function exportLogs() {
  if (logs.value.length === 0) return
  const lines = logs.value.map(
    (e) => `${e.timestamp} [${e.level.padEnd(5)}] ${e.message}`
  )
  const blob = new Blob([lines.join('\n')], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  const date = new Date().toISOString().slice(0, 10)
  a.download = `ancientcoins-logs-${date}.log`
  a.click()
  URL.revokeObjectURL(url)
}

function onSystemSave(payload: { numistaApiKey: string; logLevel: string; pushoverAppToken: string }) {
  settings.value.NumistaAPIKey = payload.numistaApiKey
  settings.value.LogLevel = payload.logLevel
  settings.value.PushoverAppToken = payload.pushoverAppToken
  saveSettings()
}

onMounted(() => {
  loadUsers()
  loadSettings()
})

onUnmounted(() => {
  if (logsInterval) clearInterval(logsInterval)
  cleanupAdminConfig()
})
</script>

<style scoped>
.admin-layout {
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
}

.tab-btn.active {
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.tab-btn:hover:not(.active) {
  color: var(--text-primary);
}

@media (max-width: 640px) {
  .tab-nav {
    flex-wrap: wrap;
  }
}
</style>
