<template>
  <div class="min-h-screen">
    <!-- Nav bar — brand + hamburger for both desktop and PWA -->
    <nav v-if="auth.isAuthenticated" class="fixed inset-x-0 top-0 z-[100] border-b border-border-subtle bg-surface/95 backdrop-blur-md">
      <div class="mx-auto flex h-[60px] max-w-[1200px] items-center justify-between gap-4 px-4">
        <button class="flex shrink-0 cursor-pointer items-center gap-2 rounded-sm border-0 bg-transparent px-2.5 py-1.5 transition-colors hover:bg-gold-glow" @click="sidebarOpen = !sidebarOpen">
          <img src="/coin-logo.jpg" alt="Aurearia - Coin Collection" class="h-9 w-9 rounded-full border-2 border-gold-dim object-cover" />
          <span class="font-display text-lg font-semibold whitespace-nowrap text-gold">Aurearia<span v-if="!isPwa" class="hidden sm:inline"> - Coin Collection</span></span>
        </button>
        <div class="flex items-center gap-1">
          <template v-if="showCollectionActions">
            <button 
              class="relative flex items-center justify-center rounded-sm p-1.5 text-text-secondary transition-colors hover:bg-gold-glow hover:text-gold"
              :class="{ 'bg-gold-glow text-gold': bulkSelectActive }"
              :aria-label="bulkSelectActive ? 'Cancel selection mode' : 'Select coins'"
              :title="bulkSelectActive ? 'Cancel selection mode' : 'Select coins'"
              @click="toggleCollectionSelectMode"
            >
              <CheckSquare :size="20" />
            </button>
            <router-link to="/add" class="relative flex items-center justify-center rounded-sm p-1.5 text-text-secondary no-underline transition-colors hover:bg-gold-glow hover:text-gold" aria-label="Add Coin" title="Add Coin">
              <CirclePlus :size="20" />
            </router-link>
          </template>
          <router-link v-if="isPwa" to="/add" class="relative flex items-center justify-center rounded-sm p-1.5 text-text-secondary no-underline transition-colors hover:bg-gold-glow hover:text-gold" aria-label="Add Coin">
            <Plus :size="20" />
          </router-link>
          <router-link to="/notifications" class="relative flex items-center justify-center rounded-sm p-1.5 text-text-secondary no-underline transition-colors hover:bg-gold-glow hover:text-gold" aria-label="Notifications">
            <Bell :size="20" />
            <span v-if="unreadCount > 0" class="absolute top-0 -right-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-gold px-1 text-label font-bold leading-none text-surface">{{ unreadCount > 99 ? '99+' : unreadCount }}</span>
          </router-link>
        </div>
      </div>
    </nav>

    <!-- Sidebar overlay -->
    <Transition name="sidebar-fade">
      <div v-if="sidebarOpen" class="fixed inset-0 z-[1200] bg-black/50" @click="sidebarOpen = false"></div>
    </Transition>

    <!-- Slide-in sidebar -->
    <Transition name="sidebar-slide">
      <aside v-if="sidebarOpen" class="fixed inset-y-0 left-0 z-[1300] flex w-[280px] flex-col overflow-y-auto border-r border-border-subtle bg-card">
        <div class="flex items-center gap-3 border-b border-border-subtle px-5 pt-5 pb-4">
          <img src="/coin-logo.jpg" alt="Aurearia - Coin Collection" class="h-10 w-10 rounded-full border-2 border-gold-dim object-cover" />
          <span class="flex-1 font-display text-base font-semibold text-gold">Aurearia<span v-if="!isPwa" class="hidden sm:inline"> - Coin Collection</span></span>
          <button class="rounded-sm border-0 bg-transparent p-[0.3rem] text-text-secondary transition-colors hover:bg-gold-glow hover:text-text-primary" :class="{ 'bg-gold-glow text-gold': editMode }" @click="toggleEditMode" :title="editMode ? 'Done' : 'Reorder menu'">
            <GripVertical :size="18" />
          </button>
          <button class="rounded-sm border-0 bg-transparent p-[0.3rem] text-text-secondary transition-colors hover:bg-gold-glow hover:text-text-primary" @click="sidebarOpen = false">
            <X :size="20" />
          </button>
        </div>
        <nav ref="navRef" class="flex-1 py-3">
          <div
            v-for="item in orderedNavItems"
            :key="item.id"
            class="w-full"
            :data-id="item.id"
          >
            <component
              :is="!editMode && item.to ? 'router-link' : 'button'"
              v-bind="!editMode && item.to ? { to: item.to, 'active-class': 'border-r-[3px] border-gold bg-gold-glow text-gold' } : {}"
              class="sidebar-link group flex w-full items-center gap-3 border-0 bg-transparent px-5 py-[0.7rem] text-left font-sans text-base text-text-secondary no-underline transition-colors hover:bg-gold-glow hover:text-gold"
              :class="editMode ? 'cursor-default select-none' : 'cursor-pointer'"
              @click="handleNavClick(item)"
            >
              <span v-if="editMode" class="drag-handle flex shrink-0 cursor-grab items-center text-text-secondary opacity-50 transition-opacity group-hover:opacity-100 active:cursor-grabbing"><GripVertical :size="16" /></span>
              <component :is="item.icon" :size="20" />
              <span>{{ item.label }}</span>
              <span v-if="item.badge && item.badge() > 0" class="ml-auto flex h-5 min-w-5 items-center justify-center rounded-full bg-gold px-1.5 text-label font-bold text-surface">{{ item.badge() }}</span>
              <ChevronDown
                v-if="item.children?.length && !editMode"
                :size="16"
                class="ml-auto shrink-0 text-text-muted transition-transform"
                :class="{
                  'rotate-180': (item.id === 'stats' && statsExpanded) || (item.id === 'collection' && collectionExpanded)
                }"
              />
            </component>
            <div
              v-if="item.children?.length && !editMode && ((item.id === 'stats' && statsExpanded) || (item.id === 'collection' && collectionExpanded))"
              class="flex flex-col pt-[0.15rem] pb-[0.35rem]"
              :aria-label="`${item.label} views`"
            >
              <router-link
                v-for="child in item.children"
                :key="child.id"
                :to="child.to"
                class="flex items-center gap-[0.35rem] py-[0.45rem] pr-5 pl-[3.25rem] text-body text-text-muted no-underline transition-colors hover:bg-gold-glow hover:text-gold"
                active-class="bg-gold-glow text-gold"
                @click="sidebarOpen = false"
              >
                <ChevronRight :size="14" class="shrink-0" />
                <span>{{ child.label }}</span>
              </router-link>
            </div>
          </div>
        </nav>
        <div class="border-t border-border-subtle py-2">
          <router-link to="/settings" class="sidebar-link flex w-full items-center gap-3 border-0 bg-transparent px-5 py-[0.7rem] text-left font-sans text-base text-text-secondary no-underline transition-colors hover:bg-gold-glow hover:text-gold" active-class="border-r-[3px] border-gold bg-gold-glow text-gold" @click="sidebarOpen = false">
            <Settings :size="20" />
            <span>Settings</span>
          </router-link>
          <button class="sidebar-link flex w-full cursor-pointer items-center gap-3 border-0 bg-transparent px-5 py-[0.7rem] text-left font-sans text-base text-text-secondary transition-colors hover:bg-gold-glow hover:text-gold" @click="openOnboardingGuide">
            <BookOpen :size="20" />
            <span>Getting Started</span>
          </button>
          <router-link v-if="auth.isAdmin" to="/admin" class="sidebar-link flex w-full items-center gap-3 border-0 bg-transparent px-5 py-[0.7rem] text-left font-sans text-base text-text-secondary no-underline transition-colors hover:bg-gold-glow hover:text-gold" active-class="border-r-[3px] border-gold bg-gold-glow text-gold" @click="sidebarOpen = false">
            <ShieldCheck :size="20" />
            <span>Admin</span>
          </router-link>
          <button class="sidebar-link flex w-full cursor-pointer items-center gap-3 border-0 bg-transparent px-5 py-[0.7rem] text-left font-sans text-base text-text-secondary transition-colors hover:bg-red-400/10 hover:text-red-400" @click="handleLogout">
            <LogOut :size="20" />
            <span>Logout</span>
          </button>
        </div>
      </aside>
    </Transition>

    <main class="min-h-screen" :class="{ 'pt-[76px]': auth.isAuthenticated }">
      <router-view />
    </main>

    <Teleport to="body">
      <!-- PWA floating agent button -->
      <button
        v-if="isPwa && auth.isAuthenticated && !showChat && !bulkSelectActive"
        class="fixed z-[1100] flex h-[52px] w-[52px] touch-none items-center justify-center rounded-full border border-border-accent bg-card text-gold shadow-card bottom-[calc(24px+env(safe-area-inset-bottom))] right-[calc(24px+env(safe-area-inset-right))]"
        :style="fabPositionStyle"
        @click="handleAgentFabClick"
        @pointerdown="startAgentFabDrag"
        @pointermove="moveAgentFabDrag"
        @pointerup="stopAgentFabDrag"
        @pointercancel="stopAgentFabDrag"
        aria-label="Open AI Agent"
      >
        <Bot :size="22" />
      </button>
    </Teleport>

    <!-- AI Agent Chat -->
    <CoinSearchChat v-if="showChat" @close="showChat = false" />

    <!-- Email prompt modal for legacy users -->
    <div v-if="showEmailPrompt" class="fixed inset-0 z-[1000] flex items-center justify-center bg-black/70" @click.self="dismissEmailPrompt">
      <div class="card w-[90%] max-w-[360px] p-8">
        <h3 class="mb-2">Add Your Email</h3>
        <p class="mb-0 text-base text-text-secondary">An email address is now required. Please add yours to continue using all features.</p>
        <div class="form-group my-4">
          <input v-model="promptEmail" type="email" class="form-input" placeholder="you@example.com" />
        </div>
        <div class="mt-6 flex justify-end gap-3">
          <button class="btn btn-secondary" @click="dismissEmailPrompt">Later</button>
          <button class="btn btn-primary" @click="savePromptEmail" :disabled="!promptEmail">Save</button>
        </div>
      </div>
    </div>

    <div v-if="showOnboardingPrompt" class="fixed inset-0 z-[1000] flex items-center justify-center bg-black/70" @click.self="dismissOnboardingPrompt">
      <div class="card w-[90%] max-w-[360px] p-8">
        <h3 class="mb-2">Aurearia - Coin Collection</h3>
        <p class="mb-0 text-base text-text-secondary">Start with the Getting Started guide to download the CSV template, build your file, and import your first collection.</p>
        <div class="mt-6 flex justify-end gap-3">
          <button class="btn btn-secondary" @click="dismissOnboardingPrompt">Not now</button>
          <button class="btn btn-primary" @click="openOnboardingGuide">Open Guide</button>
        </div>
      </div>
    </div>

    <AppDialog />
    <AppToasts />
    <PwaInstallPrompt />
    <PwaUpdateBanner />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted, markRaw, type Component } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useRouter, useRoute } from 'vue-router'
import { Landmark, Bookmark, BadgeDollarSign, BarChart3, CirclePlus, Settings, ShieldCheck, LogOut, Users as UsersIcon, Bot, Gavel, X, Bell, Plus, CalendarDays, Share2, GripVertical, BookOpen, Layers3, Search, NotebookPen, ChevronRight, ChevronDown, CheckSquare } from 'lucide-vue-next'
import { updateProfile, getMe } from '@/api/client'
import { useNotifications } from '@/composables/useNotifications'
import { useBulkSelect } from '@/composables/useBulkSelect'
import { usePwa } from '@/composables/usePwa'
import CoinSearchChat from '@/components/CoinSearchChat.vue'
import AppDialog from '@/components/AppDialog.vue'
import AppToasts from '@/components/AppToasts.vue'
import PwaInstallPrompt from '@/components/PwaInstallPrompt.vue'
import PwaUpdateBanner from '@/components/PwaUpdateBanner.vue'
import Sortable from 'sortablejs'

interface NavItem {
  id: string
  label: string
  icon: Component
  to?: string
  action?: () => void
  visible: boolean
  badge?: () => number
  children?: NavSubItem[]
}

interface NavSubItem {
  id: string
  label: string
  to: string
}

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const { isPwa } = usePwa()

const showChat = ref(false)
const sidebarOpen = ref(false)
const showEmailPrompt = ref(false)
const showOnboardingPrompt = ref(false)
const promptEmail = ref('')
const onboardingPromptKey = ref('')
const editMode = ref(false)
const navRef = ref<HTMLElement | null>(null)
let sortableInstance: Sortable | null = null
const { unreadCount, startPolling, stopPolling } = useNotifications()
const { bulkSelectActive } = useBulkSelect()
const statsExpanded = ref(false)
const collectionExpanded = ref(false)
const agentFabPosition = ref<{ x: number; y: number } | null>(null)
const isDraggingAgentFab = ref(false)
const agentFabSuppressClick = ref(false)
const agentFabPointerId = ref<number | null>(null)
const agentFabPointerOffset = ref({ x: 0, y: 0 })
const agentFabDragStart = ref({ x: 0, y: 0 })

const AGENT_FAB_SIZE = 52
const AGENT_FAB_VIEWPORT_MARGIN = 8

const isCollectionPage = computed(() => route.name === 'collection')
const showCollectionActions = computed(() => isCollectionPage.value && !isPwa)

const defaultNavItems: NavItem[] = [
  {
    id: 'collection',
    label: 'Collection',
    icon: markRaw(Landmark),
    to: '/',
    visible: true,
    children: [
      { id: 'collection-gallery', label: 'Gallery', to: '/' },
      { id: 'collection-tray', label: 'Tray', to: '/tray' },
    ],
  },
  { id: 'add-coin', label: 'Add Coin', icon: markRaw(CirclePlus), to: '/add', visible: isPwa },
  { id: 'lookup', label: 'Identify Coin', icon: markRaw(Search), to: '/lookup', visible: true },
  { id: 'wishlist', label: 'Wishlist', icon: markRaw(Bookmark), to: '/wishlist', visible: true },
  { id: 'sold', label: 'Sold', icon: markRaw(BadgeDollarSign), to: '/sold', visible: true },
  { id: 'auctions', label: 'Auctions', icon: markRaw(Gavel), to: '/auctions', visible: true },
  { id: 'followers', label: 'Followers', icon: markRaw(UsersIcon), to: '/followers', visible: true },
  { id: 'agent', label: 'Agent', icon: markRaw(Bot), action: () => { showChat.value = true; sidebarOpen.value = false }, visible: true },
  {
    id: 'stats',
    label: 'Stats',
    icon: markRaw(BarChart3),
    to: '/stats',
    visible: true,
    children: [
      { id: 'stats-timeline', label: 'Timeline', to: '/stats/timeline' },
      { id: 'stats-map', label: 'Map', to: '/stats/mint-map' },
      { id: 'stats-health', label: 'Health', to: '/stats/health' },
      { id: 'stats-value-trends', label: 'Value Details', to: '/stats/value-trends' },
      { id: 'stats-investment-breakdown', label: 'Investment Breakdown', to: '/stats/investment-breakdown' },
    ],
  },
  { id: 'sets', label: 'Sets', icon: markRaw(Layers3), to: '/sets', visible: true },
  { id: 'notes', label: 'Notes', icon: markRaw(NotebookPen), to: '/notes', visible: true },
  { id: 'calendar', label: 'Calendar', icon: markRaw(CalendarDays), to: '/calendar', visible: true },
  { id: 'showcases', label: 'Showcases', icon: markRaw(Share2), to: '/showcases', visible: true },
  { id: 'notifications', label: 'Notifications', icon: markRaw(Bell), to: '/notifications', visible: true, badge: () => unreadCount.value },
]

function getStorageKey() {
  const userId = auth.user?.id || 'default'
  return `sidebarNavOrder:${userId}`
}

function loadSavedOrder(): string[] {
  try {
    const saved = localStorage.getItem(getStorageKey())
    return saved ? JSON.parse(saved) : []
  } catch { return [] }
}

function applyOrder(order: string[]): NavItem[] {
  const itemMap = new Map(defaultNavItems.map(item => [item.id, item]))
  const ordered: NavItem[] = []
  for (const id of order) {
    const item = itemMap.get(id)
    if (item) {
      ordered.push(item)
      itemMap.delete(id)
    }
  }
  // Append any new items not in saved order
  for (const item of itemMap.values()) {
    ordered.push(item)
  }
  return ordered
}

const navOrder = ref<string[]>(loadSavedOrder())
const orderedNavItems = computed(() => {
  const items = navOrder.value.length ? applyOrder(navOrder.value) : defaultNavItems
  return items.filter(item => item.visible)
})

const fabPositionStyle = computed<Record<string, string> | undefined>(() => {
  if (!agentFabPosition.value) return undefined
  return {
    left: `${agentFabPosition.value.x}px`,
    top: `${agentFabPosition.value.y}px`,
    right: 'auto',
    bottom: 'auto',
  }
})

// Full order including hidden items for persistence
const fullOrder = computed(() => {
  return navOrder.value.length ? applyOrder(navOrder.value).map(i => i.id) : defaultNavItems.map(i => i.id)
})

function handleNavClick(item: NavItem) {
  if (editMode.value) return
  if (item.id === 'collection' && item.children?.length) {
    collectionExpanded.value = !collectionExpanded.value
    if (!collectionExpanded.value && item.to) {
      router.push(item.to)
      sidebarOpen.value = false
    }
  } else if (item.id === 'stats' && item.children?.length) {
    statsExpanded.value = !statsExpanded.value
    if (!statsExpanded.value && item.to) {
      router.push(item.to)
      sidebarOpen.value = false
    }
  } else if (item.action) {
    item.action()
  } else if (item.to) {
    router.push(item.to)
    sidebarOpen.value = false
  }
}

function toggleEditMode() {
  editMode.value = !editMode.value
}

function toggleCollectionSelectMode() {
  bulkSelectActive.value = !bulkSelectActive.value
}

function clampAgentFabPosition(x: number, y: number): { x: number; y: number } {
  const maxX = Math.max(AGENT_FAB_VIEWPORT_MARGIN, window.innerWidth - AGENT_FAB_SIZE - AGENT_FAB_VIEWPORT_MARGIN)
  const maxY = Math.max(AGENT_FAB_VIEWPORT_MARGIN, window.innerHeight - AGENT_FAB_SIZE - AGENT_FAB_VIEWPORT_MARGIN)
  return {
    x: Math.min(Math.max(x, AGENT_FAB_VIEWPORT_MARGIN), maxX),
    y: Math.min(Math.max(y, AGENT_FAB_VIEWPORT_MARGIN), maxY),
  }
}

function startAgentFabDrag(event: PointerEvent) {
  if (event.pointerType === 'mouse' && event.button !== 0) return
  const target = event.currentTarget
  if (!(target instanceof HTMLElement)) return
  const rect = target.getBoundingClientRect()
  agentFabPointerId.value = event.pointerId
  agentFabPointerOffset.value = { x: event.clientX - rect.left, y: event.clientY - rect.top }
  agentFabDragStart.value = { x: event.clientX, y: event.clientY }
  isDraggingAgentFab.value = false
  target.setPointerCapture(event.pointerId)
}

function moveAgentFabDrag(event: PointerEvent) {
  if (agentFabPointerId.value !== event.pointerId) return
  const moved = Math.hypot(event.clientX - agentFabDragStart.value.x, event.clientY - agentFabDragStart.value.y) > 4
  if (moved) {
    isDraggingAgentFab.value = true
  }
  if (!isDraggingAgentFab.value) return
  const next = clampAgentFabPosition(
    event.clientX - agentFabPointerOffset.value.x,
    event.clientY - agentFabPointerOffset.value.y,
  )
  agentFabPosition.value = next
}

function stopAgentFabDrag(event: PointerEvent) {
  if (agentFabPointerId.value !== event.pointerId) return
  const target = event.currentTarget
  if (target instanceof HTMLElement && target.hasPointerCapture(event.pointerId)) {
    target.releasePointerCapture(event.pointerId)
  }
  agentFabPointerId.value = null
  if (isDraggingAgentFab.value) {
    agentFabSuppressClick.value = true
    window.setTimeout(() => {
      agentFabSuppressClick.value = false
    }, 0)
  }
}

function handleAgentFabClick(event: MouseEvent) {
  if (agentFabSuppressClick.value) {
    event.preventDefault()
    event.stopPropagation()
    return
  }
  showChat.value = true
}

function handleAgentFabViewportResize() {
  if (!agentFabPosition.value) return
  agentFabPosition.value = clampAgentFabPosition(agentFabPosition.value.x, agentFabPosition.value.y)
}

function initSortable() {
  if (!navRef.value) return
  sortableInstance = Sortable.create(navRef.value, {
    animation: 150,
    handle: '.drag-handle',
    ghostClass: 'sortable-ghost',
    chosenClass: 'sortable-chosen',
    onEnd: (evt) => {
      if (evt.oldIndex == null || evt.newIndex == null) return
      const visibleIds = orderedNavItems.value.map(i => i.id)
      const moved = visibleIds.splice(evt.oldIndex, 1)[0]!
      visibleIds.splice(evt.newIndex, 0, moved)
      const full = [...fullOrder.value]
      const newFull: string[] = []
      let visIdx = 0
      for (const id of full) {
        const item = defaultNavItems.find(n => n.id === id)
        if (item && !item.visible) {
          newFull.push(id)
        } else if (visIdx < visibleIds.length) {
          newFull.push(visibleIds[visIdx]!)
          visIdx++
        }
      }
      while (visIdx < visibleIds.length) {
        newFull.push(visibleIds[visIdx]!)
        visIdx++
      }
      navOrder.value = newFull
      localStorage.setItem(getStorageKey(), JSON.stringify(newFull))
    },
  })
}

function destroySortable() {
  if (sortableInstance) {
    sortableInstance.destroy()
    sortableInstance = null
  }
}

watch([sidebarOpen, editMode], async ([open, edit]) => {
  destroySortable()
  if (open && edit) {
    await nextTick()
    initSortable()
  }
})

// Turn off edit mode when sidebar closes
watch(sidebarOpen, (open) => {
  if (!open) editMode.value = false
})

onMounted(async () => {
  window.addEventListener('resize', handleAgentFabViewportResize)
  if (auth.isAuthenticated) {
    startPolling()
    try {
      const res = await getMe()
      const data = res.data
      onboardingPromptKey.value = `onboardingPromptSeen:${data.id}`

      // Sync auth store with fresh server-side user data so fields like
      // numisBidsConfigured and cngConfigured are always up-to-date.
      if (auth.user) {
        auth.user.numisBidsUsername = data.numisBidsUsername
        auth.user.numisBidsConfigured = data.numisBidsConfigured
        auth.user.cngUsername = data.cngUsername
        auth.user.cngConfigured = data.cngConfigured
        auth.user.pushoverEnabled = data.pushoverEnabled
        auth.user.coinOfDayEnabled = data.coinOfDayEnabled
        localStorage.setItem('user', JSON.stringify(auth.user))
      }

      if (data.emailMissing) {
        const dismissed = localStorage.getItem('emailPromptDismissed')
        if (!dismissed || Date.now() - Number.parseInt(dismissed, 10) > 7 * 24 * 60 * 60 * 1000) {
          showEmailPrompt.value = true
        }
      } else if (shouldShowOnboardingPrompt(data.createdAt, onboardingPromptKey.value)) {
        showOnboardingPrompt.value = true
      }
    } catch { /* ignore */ }
  }
})

function shouldShowOnboardingPrompt(createdAt: string, storageKey: string): boolean {
  if (!storageKey || localStorage.getItem(storageKey) === 'true') {
    return false
  }
  const createdAtMs = Date.parse(createdAt)
  if (Number.isNaN(createdAtMs)) {
    return false
  }
  const ageMs = Date.now() - createdAtMs
  return ageMs >= 0 && ageMs <= 14 * 24 * 60 * 60 * 1000
}

function dismissEmailPrompt() {
  showEmailPrompt.value = false
  localStorage.setItem('emailPromptDismissed', Date.now().toString())
}

function dismissOnboardingPrompt() {
  showOnboardingPrompt.value = false
  if (onboardingPromptKey.value) {
    localStorage.setItem(onboardingPromptKey.value, 'true')
  }
}

async function savePromptEmail() {
  if (!promptEmail.value) return
  try {
    await updateProfile({ email: promptEmail.value })
    showEmailPrompt.value = false
    localStorage.removeItem('emailPromptDismissed')
  } catch { /* ignore */ }
}

function openOnboardingGuide() {
  dismissOnboardingPrompt()
  sidebarOpen.value = false
  router.push({ path: '/settings', query: { tab: 'help' } })
}

function handleLogout() {
  stopPolling()
  auth.logout()
  router.push('/login')
}

onUnmounted(() => {
  window.removeEventListener('resize', handleAgentFabViewportResize)
  destroySortable()
  stopPolling()
})
</script>

<style scoped>
.sortable-ghost {
  background: var(--accent-gold-glow);
  border-right: 3px solid var(--accent-gold);
  opacity: 0.6;
}

.sortable-chosen {
  background: var(--accent-gold-glow);
}

/* Sidebar transitions */
.sidebar-slide-enter-active,
.sidebar-slide-leave-active {
  transition: transform 0.25s ease;
}

.sidebar-slide-enter-from,
.sidebar-slide-leave-to {
  transform: translateX(-100%);
}

.sidebar-fade-enter-active,
.sidebar-fade-leave-active {
  transition: opacity 0.25s ease;
}

.sidebar-fade-enter-from,
.sidebar-fade-leave-to {
  opacity: 0;
}
</style>
