<template>
  <div class="container">
    <div class="page-header">
      <h1>Admin</h1>
    </div>

    <div v-if="!auth.isAdmin" class="empty-state">
      <h3>Access Denied</h3>
      <p>Admin privileges required</p>
    </div>

    <div v-else class="mx-auto grid max-w-[1200px] gap-6 md:grid-cols-[minmax(250px,300px)_minmax(0,1fr)]">
      <aside class="flex flex-col gap-4">
        <section
          v-for="group in tabGroups"
          :key="group.id"
          class="flex flex-col gap-2"
        >
          <p class="section-label mb-0 ml-1">{{ group.label }}</p>
          <div class="overflow-hidden rounded-md border border-border-subtle bg-card">
            <button
              v-for="tab in group.items"
              :key="tab.id"
              class="flex min-h-[52px] w-full items-center justify-between border-b border-border-subtle bg-transparent px-[0.9rem] py-3 text-left text-base font-medium text-text-secondary transition-all last:border-b-0 hover:bg-card-hover hover:text-text-primary"
              :class="activeTab === tab.id ? 'bg-gold-dim text-gold' : ''"
              @click="activeTab = tab.id"
            >
              <span class="inline-flex items-center gap-[0.65rem]">
                <component :is="tabIcons[tab.id]" :size="18" />
                <span>{{ tab.label }}</span>
              </span>
              <ChevronRight :size="18" class="opacity-65" />
            </button>
          </div>
        </section>
      </aside>

      <div class="min-w-0 md:mt-[1.62rem]">
        <!-- Users Tab -->
        <AdminUsersSection
          v-if="activeTab === 'users'"
          :users="users"
          :loading="usersLoading"
          :current-user-id="auth.user?.id ?? 0"
          @edit="openEditModal"
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
          :public-app-url="settings.PublicAppURL ?? ''"
          :log-level="settings.LogLevel ?? ''"
          :log-levels="LOG_LEVELS"
          :saving="settingsSaving"
          :msg="settingsMsg"
          :error="settingsError"
          :app-version="appVersion"
          :build-date="buildDate"
          @save="onSystemSave"
        />

        <!-- Coin Properties Tab -->
        <AdminCoinPropertiesSection
          v-if="activeTab === 'properties'"
          :category-options="settings.CoinCategories ?? ''"
          :era-options="settings.CoinEras ?? ''"
          :saving="settingsSaving"
          :msg="settingsMsg"
          :error="settingsError"
          @update:category-options="settings.CoinCategories = $event"
          @update:era-options="settings.CoinEras = $event"
          @save="saveSettings"
        />

        <!-- Catalogs Tab -->
        <AdminCatalogsSection v-if="activeTab === 'catalogs'" />

        <!-- OIDC Tab -->
        <AdminOIDCSection v-if="activeTab === 'oidc'" />

        <!-- Security Tab -->
        <AdminSecuritySection
          v-if="activeTab === 'security'"
          :users="users"
          :registration-mode="settings.RegistrationMode ?? ''"
          @unlocked="handleUserUnlocked"
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
          :alert-reminder-settings-msg="alertReminderSettingsMsg"
          :alert-reminder-settings-error="alertReminderSettingsError"
          :watch-bid-digest-settings-msg="watchBidDigestSettingsMsg"
          :watch-bid-digest-settings-error="watchBidDigestSettingsError"
          :health-settings-msg="healthSettingsMsg"
          :health-settings-error="healthSettingsError"
          :val-settings-msg="valSettingsMsg"
          :val-settings-error="valSettingsError"
          @save="saveSettings"
          @update:avail-settings-msg="availSettingsMsg = $event"
          @update:avail-settings-error="availSettingsError = $event"
          @update:val-settings-msg="valSettingsMsg = $event"
          @update:val-settings-error="valSettingsError = $event"
          @update:auction-settings-msg="auctionSettingsMsg = $event"
          @update:auction-settings-error="auctionSettingsError = $event"
          @update:alert-reminder-settings-msg="alertReminderSettingsMsg = $event"
          @update:alert-reminder-settings-error="alertReminderSettingsError = $event"
          @update:watch-bid-digest-settings-msg="watchBidDigestSettingsMsg = $event"
          @update:watch-bid-digest-settings-error="watchBidDigestSettingsError = $event"
          @update:health-settings-msg="healthSettingsMsg = $event"
          @update:health-settings-error="healthSettingsError = $event"
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

      <AdminUserEditModal
        :user="editTarget"
        :current-user-id="auth.user?.id ?? 0"
        @close="editTarget = null"
        @role-updated="handleRoleUpdated"
        @deleted="handleUserDeleted"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, type Component } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useRoute, useRouter } from 'vue-router'
import { getUsers, getAdminLogs } from '@/api/client'
import { LOG_LEVELS } from '@/types'
import type { UserInfo, LogEntry } from '@/types'
import { useAdminConfig } from '@/composables/useAdminConfig'
import AdminUserEditModal from '@/components/admin/AdminUserEditModal.vue'
import AdminUsersSection from '@/components/admin/AdminUsersSection.vue'
import AdminSystemSection from '@/components/admin/AdminSystemSection.vue'
import AdminLogsSection from '@/components/admin/AdminLogsSection.vue'
import AdminAISection from '@/components/admin/AdminAISection.vue'
import AdminSchedulesSection from '@/components/admin/AdminSchedulesSection.vue'
import AdminHealthSection from '@/components/admin/AdminHealthSection.vue'
import AdminCatalogsSection from '@/components/admin/AdminCatalogsSection.vue'
import AdminCoinPropertiesSection from '@/components/admin/AdminCoinPropertiesSection.vue'
import AdminSecuritySection from '@/components/admin/AdminSecuritySection.vue'
import AdminOIDCSection from '@/components/admin/AdminOIDCSection.vue'
import { Users, Cpu, Wrench, ScrollText, CalendarClock, Activity, ChevronRight, BookMarked, Settings2, ShieldAlert, KeyRound } from 'lucide-vue-next'

type AdminTabId = 'users' | 'ai' | 'system' | 'properties' | 'catalogs' | 'oidc' | 'security' | 'schedules' | 'health' | 'logs'
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
  properties: Settings2,
  catalogs: BookMarked,
  oidc: KeyRound,
  security: ShieldAlert,
  schedules: CalendarClock,
  health: Activity,
  logs: ScrollText,
}

const tabs: AdminTab[] = [
  { id: 'users', label: 'Users', group: 'configuration' },
  { id: 'ai', label: 'AI', group: 'configuration' },
  { id: 'system', label: 'System', group: 'configuration' },
  { id: 'properties', label: 'Coin Properties', group: 'configuration' },
  { id: 'catalogs', label: 'Catalogs', group: 'configuration' },
  { id: 'oidc', label: 'OIDC Login', group: 'configuration' },
  { id: 'security', label: 'Security', group: 'operations' },
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

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const rawVersion = import.meta.env.VITE_APP_VERSION || 'dev'
const appVersion = computed(() => {
  if (rawVersion === 'dev') return 'dev'
  return rawVersion
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

const editTarget = ref<UserInfo | null>(null)

function openEditModal(user: UserInfo) {
  editTarget.value = user
}

function handleRoleUpdated(payload: { userId: number; role: UserInfo['role'] }) {
  users.value = users.value.map((user) =>
    user.id === payload.userId
      ? { ...user, role: payload.role }
      : user
  )
}

function handleUserDeleted(userId: number) {
  users.value = users.value.filter((user) => user.id !== userId)
  editTarget.value = null
}

function handleUserUnlocked(userId: number) {
  users.value = users.value.map((user) =>
    user.id === userId
      ? { ...user, lockedUntil: null, failedLoginAttempts: 0 }
      : user
  )
}

// Settings (from composable)
const {
  settings, settingDefaults, settingsMsg, settingsError, settingsSaving,
  ollamaTesting, ollamaTestResult, ollamaTestOk,
  anthropicTesting, anthropicTestResult, anthropicTestOk, anthropicModels,
  searxngTesting, searxngTestResult, searxngTestOk,
  coinSearchPromptDefault, coinShowsPromptDefault, valuationPromptDefault,
  availSettingsMsg, availSettingsError, auctionSettingsMsg, auctionSettingsError, alertReminderSettingsMsg, alertReminderSettingsError, watchBidDigestSettingsMsg, watchBidDigestSettingsError, healthSettingsMsg, healthSettingsError, valSettingsMsg, valSettingsError,
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

function onSystemSave(payload: { numistaApiKey: string; logLevel: string; pushoverAppToken: string; publicAppUrl: string }) {
  settings.value.NumistaAPIKey = payload.numistaApiKey
  settings.value.LogLevel = payload.logLevel
  settings.value.PushoverAppToken = payload.pushoverAppToken
  settings.value.PublicAppURL = payload.publicAppUrl
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
