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
      <SettingsAccountSection v-if="activeTab === 'account'" ref="accountSection" />

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
      <SettingsDataSection v-if="activeTab === 'data'" ref="dataSection" />

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
  listConversations, getConversation, deleteConversation,
  getBlockedUsers, unblockFollower,
} from '@/api/client'
import type { ConversationSummary } from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import { usePwa } from '@/composables/usePwa'
import type { Theme } from '@/types'
import CoinSearchChat from '@/components/CoinSearchChat.vue'
import HelpSection from '@/components/HelpSection.vue'
import SettingsAccountSection from '@/components/settings/SettingsAccountSection.vue'
import SettingsAppearanceSection from '@/components/settings/SettingsAppearanceSection.vue'
import SettingsDataSection from '@/components/settings/SettingsDataSection.vue'
import SavedConversationsSection from '@/components/settings/SavedConversationsSection.vue'
import SettingsToolsSection from '@/components/settings/SettingsToolsSection.vue'
import { User, Palette, Database, MessageSquare, HelpCircle, Wrench, Menu, ShieldCheck } from 'lucide-vue-next'

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
const { isPwa } = usePwa()

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const accountSection = ref<InstanceType<typeof SettingsAccountSection> | null>(null)
const dataSection = ref<InstanceType<typeof SettingsDataSection> | null>(null)

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

// Theme
const theme = ref<Theme>((localStorage.getItem('theme') as Theme) || 'dark')

function setTheme(t: Theme) {
  theme.value = t
  localStorage.setItem('theme', t)
  document.documentElement.setAttribute('data-theme', t)
}

// Timezone
const timezones = 'supportedValuesOf' in Intl
  ? (Intl as unknown as { supportedValuesOf: (key: string) => string[] }).supportedValuesOf('timeZone')
  : [] as string[]
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
    dataSection.value?.loadApiKeys() ?? Promise.resolve(),
    loadConversations(),
    loadBlockedUsers(),
    accountSection.value?.loadCredentials() ?? Promise.resolve(),
  ])
}

onMounted(() => {
  loadConversations()
  loadBlockedUsers()
})
</script>

<style scoped>
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
  .tab-nav {
    flex-wrap: wrap;
  }

  .tab-btn {
    font-size: 0.78rem;
    padding: 0.5rem 0.6rem;
  }
}
</style>
