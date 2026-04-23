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
      <AdminAiConfigSection
        v-if="activeTab === 'ai'"
        :settings="settings"
        :setting-defaults="settingDefaults"
        :settings-msg="settingsMsg"
        :settings-error="settingsError"
        :settings-saving="settingsSaving"
        :ollama-testing="ollamaTesting"
        :ollama-test-result="ollamaTestResult"
        :ollama-test-ok="ollamaTestOk"
        :anthropic-testing="anthropicTesting"
        :anthropic-test-result="anthropicTestResult"
        :anthropic-test-ok="anthropicTestOk"
        :anthropic-models="anthropicModels"
        :searxng-testing="searxngTesting"
        :searxng-test-result="searxngTestResult"
        :searxng-test-ok="searxngTestOk"
        :coin-search-prompt-default="coinSearchPromptDefault"
        :coin-shows-prompt-default="coinShowsPromptDefault"
        :valuation-prompt-default="valuationPromptDefault"
        @save="saveSettings"
        @test-ollama="testOllamaConnection"
        @test-anthropic="testAnthropicConn"
        @test-searxng="testSearxngConn"
      />

      <!-- System Tab -->
      <AdminSystemSection
        v-if="activeTab === 'system'"
        :numista-api-key="settings.NumistaAPIKey ?? ''"
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
      <section v-if="activeTab === 'schedules'" class="admin-section card">
        <h2>Schedules</h2>
        <AvailabilityScheduleSection
          :settings="settings"
          :settings-saving="settingsSaving"
          :avail-settings-msg="availSettingsMsg"
          :avail-settings-error="availSettingsError"
          @save-settings="saveSettings"
        />
        <hr class="section-divider" />
        <ValuationScheduleSection
          :settings="settings"
          :settings-saving="settingsSaving"
          v-model:val-settings-msg="valSettingsMsg"
          v-model:val-settings-error="valSettingsError"
          @save-settings="saveSettings"
        />
      </section>

      <!-- Reset Password Modal -->
      <ResetPasswordModal :user="resetTarget" @close="resetTarget = null" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, type Component } from 'vue'
import { useAuthStore } from '@/stores/auth'
import {
  getUsers, deleteUser,
  getAdminLogs,
} from '@/api/client'
import { LOG_LEVELS } from '@/types'
import type { UserInfo, LogEntry } from '@/types'
import { useDialog } from '@/composables/useDialog'
import { useAdminConfig } from '@/composables/useAdminConfig'
import ResetPasswordModal from '@/components/admin/ResetPasswordModal.vue'
import AdminUsersSection from '@/components/admin/AdminUsersSection.vue'
import AdminSystemSection from '@/components/admin/AdminSystemSection.vue'
import AdminAiConfigSection from '@/components/admin/AdminAiConfigSection.vue'
import AdminLogsSection from '@/components/admin/AdminLogsSection.vue'
import AvailabilityScheduleSection from '@/components/admin/AvailabilityScheduleSection.vue'
import ValuationScheduleSection from '@/components/admin/ValuationScheduleSection.vue'
import { Users, Cpu, Wrench, ScrollText, Download, ShieldCheck, CalendarClock } from 'lucide-vue-next'

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
  availSettingsMsg, availSettingsError, valSettingsMsg, valSettingsError,
  loadSettings, saveSettings,
  testOllamaConnection, testAnthropicConn, testSearxngConn,
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

function onSystemSave(payload: { numistaApiKey: string; logLevel: string }) {
  settings.value.NumistaAPIKey = payload.numistaApiKey
  settings.value.LogLevel = payload.logLevel
  saveSettings()
}

onMounted(() => {
  loadUsers()
  loadSettings()
})

onUnmounted(() => {
  if (logsInterval) clearInterval(logsInterval)
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

.admin-section h2 {
  font-size: 1.1rem;
  margin-bottom: 1.25rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
}

.users-table {
  width: 100%;
  border-collapse: collapse;
}

.users-table th,
.users-table td {
  text-align: left;
  padding: 0.75rem 0.5rem;
  border-bottom: 1px solid var(--border-subtle);
}

.users-table th {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  font-weight: 600;
}

.date-cell {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.text-muted {
  color: var(--text-muted);
}

.form-hint {
  display: block;
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-top: 0.25rem;
}

.section-divider {
  border: none;
  border-top: 1px solid var(--border-subtle, #333);
  margin: 1.5rem 0;
}

.subsection-title {
  font-size: 1rem;
  font-weight: 600;
  margin-bottom: 1rem;
  color: var(--text-primary, #e0e0e0);
}

@media (max-width: 640px) {
  .tab-nav {
    flex-wrap: wrap;
  }
  .users-table {
    font-size: 0.85rem;
  }
}

/* Logs */
.logs-empty {
  text-align: center;
  padding: 2rem;
  color: var(--text-muted);
  font-family: 'Inter', sans-serif;
}

/* Availability */

</style>
