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
      <aside class="settings-nav">
        <section
          v-for="group in tabGroups"
          :key="group.id"
          class="settings-group"
        >
          <p class="section-label settings-group-title">{{ group.label }}</p>
          <div class="settings-group-card">
            <button
              v-for="tab in group.items"
              :key="tab.id"
              class="settings-item"
              :class="{ active: activeTab === tab.id }"
              @click="activeTab = tab.id"
            >
              <span class="settings-item-main">
                <component :is="tabIcons[tab.id]" :size="18" />
                <span>{{ tab.label }}</span>
              </span>
              <ChevronRight :size="18" class="settings-item-chevron" />
            </button>
          </div>
        </section>
      </aside>

      <div class="settings-content">
        <header class="settings-content-header card">
          <p class="section-label">{{ activeTabMeta.groupLabel }}</p>
          <h2>{{ activeTabMeta.label }}</h2>
        </header>

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

        <!-- Health Tab -->
        <AdminHealthSection v-if="activeTab === 'health'" />

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
      </div>

      <!-- Reset Password Modal -->
      <ResetPasswordModal :user="resetTarget" @close="resetTarget = null" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, type Component } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useRoute, useRouter } from 'vue-router'
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
import AdminHealthSection from '@/components/admin/AdminHealthSection.vue'
import { Users, Cpu, Wrench, ScrollText, CalendarClock, Activity, ChevronRight } from 'lucide-vue-next'

type AdminTabId = 'users' | 'ai' | 'system' | 'schedules' | 'health' | 'logs'
type AdminGroupId = 'configuration' | 'operations'
type AdminTab = {
  id: AdminTabId
  label: string
  group: AdminGroupId
  aliases?: string[]
}

const tabIcons: Record<AdminTabId, Component> = {
  users: Users,
  ai: Cpu,
  system: Wrench,
  schedules: CalendarClock,
  health: Activity,
  logs: ScrollText,
}

const tabs: AdminTab[] = [
  { id: 'users', label: 'Users', group: 'configuration' },
  { id: 'ai', label: 'AI', group: 'configuration' },
  { id: 'system', label: 'System', group: 'configuration' },
  { id: 'schedules', label: 'Schedules', group: 'operations', aliases: ['schedule'] },
  { id: 'health', label: 'Health', group: 'operations' },
  { id: 'logs', label: 'Logs', group: 'operations', aliases: ['log'] },
]
const DEFAULT_TAB: AdminTabId = 'users'
const groupLabels: Record<AdminGroupId, string> = {
  configuration: 'Configuration',
  operations: 'Operations',
}
const tabAliasMap = new Map<string, AdminTabId>()
for (const tab of tabs) {
  tabAliasMap.set(tab.id, tab.id)
  for (const alias of tab.aliases ?? []) {
    tabAliasMap.set(alias, tab.id)
  }
}

function normalizeTab(value: unknown): AdminTabId | null {
  const tab = Array.isArray(value) ? value[0] : value
  if (typeof tab !== 'string') return null
  return tabAliasMap.get(tab.toLowerCase()) ?? null
}

const { showConfirm, showAlert } = useDialog()
const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

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

const activeTab = ref<AdminTabId>(DEFAULT_TAB)
const tabGroups = computed(() => ([
  {
    id: 'configuration',
    label: groupLabels.configuration,
    items: tabs.filter(tab => tab.group === 'configuration'),
  },
  {
    id: 'operations',
    label: groupLabels.operations,
    items: tabs.filter(tab => tab.group === 'operations'),
  },
]))
const fallbackTab = tabs[0] as AdminTab
const activeTabMeta = computed(() => {
  const active = tabs.find(tab => tab.id === activeTab.value) ?? fallbackTab
  return {
    ...active,
    groupLabel: groupLabels[active.group],
  }
})

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

watch(() => route.query.tab, (rawTab) => {
  const normalized = normalizeTab(rawTab)
  if (normalized && normalized !== activeTab.value) {
    activeTab.value = normalized
  }
}, { immediate: true })

watch(activeTab, (nextTab) => {
  const normalized = normalizeTab(route.query.tab)
  if (normalized === nextTab) return
  void router.replace({
    query: { ...route.query, tab: nextTab },
  })
})

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
  max-width: 1200px;
  margin-left: auto;
  margin-right: auto;
  display: grid;
  grid-template-columns: minmax(250px, 300px) minmax(0, 1fr);
  gap: 1.5rem;
}

.settings-nav {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.settings-group-title {
  margin: 0 0 0 0.25rem;
}

.settings-group-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.settings-item {
  width: 100%;
  min-height: 52px;
  padding: 0.75rem 0.9rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-subtle);
  font-size: 0.9rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
  text-align: left;
}

.settings-item:last-child {
  border-bottom: none;
}

.settings-item-main {
  display: inline-flex;
  align-items: center;
  gap: 0.65rem;
}

.settings-item-chevron {
  opacity: 0.65;
}

.settings-item:hover:not(.active) {
  background: var(--bg-card-hover);
  color: var(--text-primary);
}

.settings-item.active {
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.settings-content {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.settings-content-header {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin: 0;
}

.settings-content-header h2 {
  margin: 0;
  font-size: 1.5rem;
  color: var(--text-heading);
}

@media (max-width: 980px) {
  .admin-layout {
    grid-template-columns: 1fr;
  }
}
</style>
