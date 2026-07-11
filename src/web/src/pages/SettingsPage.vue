<template>
  <PullToRefresh :on-refresh="handleRefresh">
    <div class="container">
      <div class="page-header !flex-nowrap !items-center">
        <h1>Settings</h1>
        <div v-if="isPwa" class="relative">
          <button class="btn btn-secondary btn-sm gap-1.5 text-body" @click="settingsMenuOpen = !settingsMenuOpen">
            <component :is="tabIcons[activeTab]" :size="16" />
            {{ tabs.find(t => t.id === activeTab)?.label }}
            <Menu :size="16" />
          </button>
          <Transition
            enter-active-class="transition-opacity duration-150 ease-out"
            enter-from-class="opacity-0"
            leave-active-class="transition-opacity duration-150 ease-in"
            leave-to-class="opacity-0"
          >
            <div v-if="settingsMenuOpen" class="absolute right-0 top-full z-50 mt-2 flex min-w-[180px] flex-col gap-0.5 rounded-md border border-border-subtle bg-card p-[0.3rem] shadow-[0_4px_20px_rgba(0,0,0,0.4)]">
              <button
                v-for="tab in tabs"
                :key="tab.id"
                class="flex w-full items-center gap-2 rounded-sm px-3 py-2 text-left text-body font-medium text-text-secondary transition-colors hover:bg-surface hover:text-text-primary"
                :class="{ 'bg-[var(--accent-gold-dim)] text-gold hover:bg-[var(--accent-gold-dim)] hover:text-gold': activeTab === tab.id }"
                @click="selectTab(tab.id); settingsMenuOpen = false"
              >
                <component :is="tabIcons[tab.id]" :size="16" />
                {{ tab.label }}
              </button>
            </div>
          </Transition>
        </div>
      </div>

      <div class="mx-auto flex max-w-[800px] flex-col gap-6">
        <div v-if="!isPwa" class="flex flex-wrap gap-1 rounded-md border border-border-subtle bg-card p-[0.3rem]">
          <button
            v-for="tab in tabs"
            :key="tab.id"
            class="flex flex-1 items-center justify-center gap-1.5 rounded-sm px-2.5 py-2 text-[0.78rem] font-medium text-text-secondary transition-colors hover:text-text-primary md:px-4 md:py-2.5 md:text-body"
            :class="{ 'bg-[var(--accent-gold-dim)] text-gold hover:text-gold': activeTab === tab.id }"
            @click="selectTab(tab.id)"
          >
            <component :is="tabIcons[tab.id]" :size="16" /> {{ tab.label }}
          </button>
        </div>

        <SettingsAccountSection v-if="activeTab === 'account'" ref="accountSection" />

        <SettingsAppearanceSection
          v-if="activeTab === 'appearance'"
          :theme="theme"
          :timezone="timezone"
          :timezones="timezones"
          :default-view="defaultView"
          :default-sort="defaultSort"
          :tray-felt-color="feltColor"
          @set-theme="setTheme"
          @save-timezone="(tz: string) => { timezone = tz; saveTimezone() }"
          @set-default-view="setDefaultView"
          @save-default-sort="(sort: string) => { defaultSort = sort; saveDefaultSort() }"
          @set-tray-felt-color="(color) => { feltColor = color }"
        />

        <SettingsDataSection v-if="activeTab === 'data'" ref="dataSection" />

        <SettingsBackupsSection v-if="activeTab === 'backups'" ref="backupsSection" />

        <SettingsApiKeysSection v-if="activeTab === 'apikeys'" ref="apiKeysSection" />

        <SettingsToolsSection
          v-if="activeTab === 'tools'"
          :blocked-users="blockedUsers"
          :blocked-loading="blockedLoading"
          @saved="handleProcessSaved"
          @unblock="handleUnblock"
        />

        <SavedConversationsSection
          v-if="activeTab === 'conversations'"
          :conversations="conversations"
          :loading="conversationsLoading"
          @open="openConversation"
          @delete="handleDeleteConversation"
        />

        <HelpSection v-if="activeTab === 'help'" />

        <CoinSearchChat
          v-if="showChat"
          :load-conversation="chatConversation"
          @close="showChat = false; chatConversation = null"
          @added="() => {}"
        />
      </div>
    </div>
  </PullToRefresh>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, type Component } from 'vue'
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
import { useTrayPreference } from '@/composables/useTrayPreference'
import type { Theme } from '@/types'
import CoinSearchChat from '@/components/CoinSearchChat.vue'
import HelpSection from '@/components/HelpSection.vue'
import SettingsAccountSection from '@/components/settings/SettingsAccountSection.vue'
import SettingsAppearanceSection from '@/components/settings/SettingsAppearanceSection.vue'
import SettingsDataSection from '@/components/settings/SettingsDataSection.vue'
import SettingsBackupsSection from '@/components/settings/SettingsBackupsSection.vue'
import SettingsApiKeysSection from '@/components/settings/SettingsApiKeysSection.vue'
import SavedConversationsSection from '@/components/settings/SavedConversationsSection.vue'
import SettingsToolsSection from '@/components/settings/SettingsToolsSection.vue'
import { User, Palette, Database, MessageSquare, HelpCircle, Wrench, Menu, ShieldCheck, Archive, KeyRound } from 'lucide-vue-next'

const tabIcons: Record<string, Component> = {
  account: User,
  appearance: Palette,
  data: Database,
  backups: Archive,
  apikeys: KeyRound,
  tools: Wrench,
  conversations: MessageSquare,
  help: HelpCircle,
  admin: ShieldCheck,
}

const { showConfirm, showAlert } = useDialog()
const activeTab = ref('account')
const settingsMenuOpen = ref(false)
const { isPwa } = usePwa()
const { feltColor } = useTrayPreference()

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const accountSection = ref<InstanceType<typeof SettingsAccountSection> | null>(null)
const dataSection = ref<InstanceType<typeof SettingsDataSection> | null>(null)
const backupsSection = ref<InstanceType<typeof SettingsBackupsSection> | null>(null)
const apiKeysSection = ref<InstanceType<typeof SettingsApiKeysSection> | null>(null)

const baseTabs = [
  { id: 'account', label: 'Account' },
  { id: 'appearance', label: 'Appearance' },
  { id: 'data', label: 'Data' },
  { id: 'backups', label: 'Backups' },
  { id: 'apikeys', label: 'API Keys' },
  { id: 'tools', label: 'Tools' },
  { id: 'conversations', label: 'Conversations' },
  { id: 'help', label: 'Help' },
]
const validTabIds = baseTabs.map(t => t.id).concat('admin')
const tabs = computed(() => {
  if (isPwa && auth.isAdmin) {
    return [
      { id: 'account', label: 'Account' },
      { id: 'admin', label: 'Admin' },
      { id: 'appearance', label: 'Appearance' },
      { id: 'data', label: 'Data' },
      { id: 'backups', label: 'Backups' },
      { id: 'apikeys', label: 'API Keys' },
      { id: 'tools', label: 'Tools' },
      { id: 'conversations', label: 'Conversations' },
      { id: 'help', label: 'Help' },
    ]
  }
  return baseTabs
})

function applyTabFromRoute(tabValue: unknown) {
  if (typeof tabValue !== 'string' || !validTabIds.includes(tabValue)) {
    return
  }
  if (tabValue === 'admin') {
    if (auth.isAdmin) {
      router.push('/admin')
    }
    return
  }
  activeTab.value = tabValue
}

function selectTab(tabId: string) {
  if (tabId === 'admin') {
    router.push('/admin')
    return
  }
  activeTab.value = tabId
  router.replace({ query: { ...route.query, tab: tabId } })
}

applyTabFromRoute(route.query.tab)

watch(() => route.query.tab, (tab) => {
  applyTabFromRoute(tab)
})

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
    apiKeysSection.value?.loadApiKeys() ?? Promise.resolve(),
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
