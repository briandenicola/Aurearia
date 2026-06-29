<template>
  <div class="app">
    <!-- Nav bar — brand + hamburger for both desktop and PWA -->
    <nav v-if="auth.isAuthenticated" class="nav-bar" :class="{ 'pwa-mode': isPwa }">
      <div class="nav-content">
        <button class="nav-brand" @click="sidebarOpen = !sidebarOpen">
          <img src="/coin-logo.jpg" alt="Aurearia - Coin Collection" class="nav-logo" />
          <span class="nav-title">Aurearia<span class="nav-title-suffix"> - Coin Collection</span></span>
        </button>
        <div class="nav-actions">
          <template v-if="showCollectionActions">
            <button 
              class="nav-bell"
              :class="{ active: bulkSelectActive }" 
              :aria-label="bulkSelectActive ? 'Cancel selection mode' : 'Select coins'"
              :title="bulkSelectActive ? 'Cancel selection mode' : 'Select coins'"
              @click="toggleCollectionSelectMode"
            >
              <CheckSquare :size="20" />
            </button>
            <router-link to="/add" class="nav-bell" aria-label="Add Coin" title="Add Coin">
              <CirclePlus :size="20" />
            </router-link>
          </template>
          <router-link v-if="isPwa" to="/add" class="nav-bell nav-add" aria-label="Add Coin">
            <Plus :size="20" />
          </router-link>
          <router-link to="/notifications" class="nav-bell" aria-label="Notifications">
            <Bell :size="20" />
            <span v-if="unreadCount > 0" class="nav-bell-badge">{{ unreadCount > 99 ? '99+' : unreadCount }}</span>
          </router-link>
        </div>
      </div>
    </nav>

    <!-- Sidebar overlay -->
    <Transition name="sidebar-fade">
      <div v-if="sidebarOpen" class="sidebar-overlay" @click="sidebarOpen = false"></div>
    </Transition>

    <!-- Slide-in sidebar -->
    <Transition name="sidebar-slide">
      <aside v-if="sidebarOpen" class="sidebar" :class="{ 'pwa-mode': isPwa }">
        <div class="sidebar-header">
          <img src="/coin-logo.jpg" alt="Aurearia - Coin Collection" class="sidebar-logo" />
          <span class="sidebar-title">Aurearia<span class="sidebar-title-suffix"> - Coin Collection</span></span>
          <button class="sidebar-header-btn" :class="{ active: editMode }" @click="toggleEditMode" :title="editMode ? 'Done' : 'Reorder menu'">
            <GripVertical :size="18" />
          </button>
          <button class="sidebar-close" @click="sidebarOpen = false">
            <X :size="20" />
          </button>
        </div>
        <nav ref="navRef" class="sidebar-nav" :class="{ 'edit-mode': editMode }">
          <div
            v-for="item in orderedNavItems"
            :key="item.id"
            class="sidebar-item"
            :data-id="item.id"
          >
            <component
              :is="!editMode && item.to ? 'router-link' : 'button'"
              v-bind="!editMode && item.to ? { to: item.to, 'active-class': 'active' } : {}"
              class="sidebar-link"
              @click="handleNavClick(item)"
            >
              <span v-if="editMode" class="drag-handle"><GripVertical :size="16" /></span>
              <component :is="item.icon" :size="20" />
              <span>{{ item.label }}</span>
              <span v-if="item.badge && item.badge() > 0" class="sidebar-badge">{{ item.badge() }}</span>
              <ChevronDown
                v-if="item.children?.length && !editMode"
                :size="16"
                class="sidebar-chevron"
                :class="{
                  expanded: (item.id === 'stats' && statsExpanded) || (item.id === 'collection' && collectionExpanded)
                }"
              />
            </component>
            <div
              v-if="item.children?.length && !editMode && ((item.id === 'stats' && statsExpanded) || (item.id === 'collection' && collectionExpanded))"
              class="sidebar-submenu"
              :aria-label="`${item.label} views`"
            >
              <router-link
                v-for="child in item.children"
                :key="child.id"
                :to="child.to"
                class="sidebar-sublink"
                active-class="active"
                @click="sidebarOpen = false"
              >
                <ChevronRight :size="14" class="sidebar-subchevron" />
                <span>{{ child.label }}</span>
              </router-link>
            </div>
          </div>
        </nav>
        <div class="sidebar-footer">
          <router-link to="/settings" class="sidebar-link" active-class="active" @click="sidebarOpen = false">
            <Settings :size="20" />
            <span>Settings</span>
          </router-link>
          <button class="sidebar-link" @click="openOnboardingGuide">
            <BookOpen :size="20" />
            <span>Getting Started</span>
          </button>
          <router-link v-if="auth.isAdmin" to="/admin" class="sidebar-link" active-class="active" @click="sidebarOpen = false">
            <ShieldCheck :size="20" />
            <span>Admin</span>
          </router-link>
          <button class="sidebar-link sidebar-logout" @click="handleLogout">
            <LogOut :size="20" />
            <span>Logout</span>
          </button>
        </div>
      </aside>
    </Transition>

    <main class="main-content" :class="{ 'with-nav': auth.isAuthenticated }">
      <router-view />
    </main>

    <Teleport to="body">
      <!-- PWA floating agent button -->
      <button
        v-if="isPwa && auth.isAuthenticated && !showChat && !bulkSelectActive"
        class="agent-fab"
        @click="showChat = true"
        aria-label="Open AI Agent"
      >
        <Bot :size="22" />
      </button>
    </Teleport>

    <!-- AI Agent Chat -->
    <CoinSearchChat v-if="showChat" @close="showChat = false" />

    <!-- Email prompt modal for legacy users -->
    <div v-if="showEmailPrompt" class="modal-overlay" @click.self="dismissEmailPrompt">
      <div class="modal card">
        <h3>Add Your Email</h3>
        <p>An email address is now required. Please add yours to continue using all features.</p>
        <div class="form-group" style="margin: 1rem 0">
          <input v-model="promptEmail" type="email" class="form-input" placeholder="you@example.com" />
        </div>
        <div class="modal-actions">
          <button class="btn btn-secondary" @click="dismissEmailPrompt">Later</button>
          <button class="btn btn-primary" @click="savePromptEmail" :disabled="!promptEmail">Save</button>
        </div>
      </div>
    </div>

    <div v-if="showOnboardingPrompt" class="modal-overlay" @click.self="dismissOnboardingPrompt">
      <div class="modal card">
        <h3>Aurearia - Coin Collection</h3>
        <p>Start with the Getting Started guide to download the CSV template, build your file, and import your first collection.</p>
        <div class="modal-actions">
          <button class="btn btn-secondary" @click="dismissOnboardingPrompt">Not now</button>
          <button class="btn btn-primary" @click="openOnboardingGuide">Open Guide</button>
        </div>
      </div>
    </div>

    <AppDialog />
    <AppToasts />
    <PwaInstallPrompt />
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
  if (auth.isAuthenticated) {
    startPolling()
    try {
      const res = await getMe()
      const data = res.data
      onboardingPromptKey.value = `onboardingPromptSeen:${data.id}`

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
  destroySortable()
  stopPolling()
})
</script>

<style scoped>
.app {
  min-height: 100vh;
}

/* ── Top nav bar (shared structure) ── */
.nav-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  background: rgba(15, 15, 26, 0.95);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--border-subtle);
}

.nav-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 1rem;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.nav-brand {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  text-decoration: none;
  background: none;
  border: none;
  cursor: pointer;
  padding: 0.4rem 0.6rem;
  border-radius: var(--radius-sm);
  transition: background var(--transition-fast);
  flex-shrink: 0;
}

.nav-brand:hover {
  background: var(--accent-gold-glow);
}

.nav-logo {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--accent-gold-dim);
}

.nav-title {
  font-family: 'Cinzel', serif;
  font-size: 1.2rem;
  color: var(--accent-gold);
  font-weight: 600;
  white-space: nowrap;
}

@media (max-width: 480px) {
  .nav-title-suffix,
  .sidebar-title-suffix {
    display: none;
  }
}

.nav-bar.pwa-mode .nav-title-suffix,
.sidebar.pwa-mode .sidebar-title-suffix {
  display: none;
}

.nav-bell {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  padding: 0.4rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: color var(--transition-fast), background var(--transition-fast);
  text-decoration: none;
}

.nav-actions {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.nav-bell:hover {
  color: var(--accent-gold);
  background: var(--accent-gold-glow);
}

.nav-bell.active {
  color: var(--accent-gold);
  background: var(--accent-gold-glow);
}

.nav-bell-badge {
  position: absolute;
  top: 0;
  right: -2px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 8px;
  background: var(--accent-gold);
  color: var(--bg-primary);
  font-size: 0.7rem;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
}

/* ── Sidebar ── */
/* z-index ladder: Leaflet controls ≤1000 → MintCoinDrawer 1100 → sidebar-overlay 1200 → sidebar 1300 */
.sidebar-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 1200;
}

.sidebar {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: 280px;
  background: var(--bg-card);
  border-right: 1px solid var(--border-subtle);
  z-index: 1300;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}

.sidebar-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1.25rem 1.25rem 1rem;
  border-bottom: 1px solid var(--border-subtle);
}

.sidebar-logo {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--accent-gold-dim);
}

.sidebar-title {
  font-family: 'Cinzel', serif;
  font-size: 1rem;
  color: var(--accent-gold);
  font-weight: 600;
  flex: 1;
}

.sidebar-close {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.3rem;
  border-radius: var(--radius-sm);
  transition: color var(--transition-fast), background var(--transition-fast);
}

.sidebar-close:hover {
  color: var(--text-primary);
  background: var(--accent-gold-glow);
}

.sidebar-header-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.3rem;
  border-radius: var(--radius-sm);
  transition: color var(--transition-fast), background var(--transition-fast);
}

.sidebar-header-btn:hover {
  color: var(--text-primary);
  background: var(--accent-gold-glow);
}

.sidebar-header-btn.active {
  color: var(--accent-gold);
  background: var(--accent-gold-glow);
}

.sidebar-nav {
  flex: 1;
  padding: 0.75rem 0;
}

.sidebar-item {
  width: 100%;
}

.sidebar-link {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.7rem 1.25rem;
  color: var(--text-secondary);
  font-size: 0.9rem;
  text-decoration: none;
  transition: all var(--transition-fast);
  border: none;
  background: none;
  width: 100%;
  cursor: pointer;
  font-family: inherit;
}

.sidebar-link:hover {
  color: var(--accent-gold);
  background: var(--accent-gold-glow);
}

.sidebar-link.active {
  color: var(--accent-gold);
  background: var(--accent-gold-glow);
  border-right: 3px solid var(--accent-gold);
}

.sidebar-submenu {
  display: flex;
  flex-direction: column;
  padding: 0.15rem 0 0.35rem;
}

.sidebar-sublink {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.45rem 1.25rem 0.45rem 3.25rem;
  color: var(--text-muted);
  font-size: 0.85rem;
  text-decoration: none;
  transition: color var(--transition-fast), background var(--transition-fast);
}

.sidebar-sublink:hover,
.sidebar-sublink.active {
  color: var(--accent-gold);
  background: var(--accent-gold-glow);
}

.sidebar-subchevron {
  flex-shrink: 0;
}

.sidebar-chevron {
  margin-left: auto;
  flex-shrink: 0;
  color: var(--text-muted);
  transition: transform var(--transition-fast);
}

.sidebar-chevron.expanded {
  transform: rotate(180deg);
}

.sidebar-badge {
  margin-left: auto;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 10px;
  background: var(--accent-gold);
  color: var(--bg-primary);
  font-size: 0.7rem;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
}

.sidebar-footer {
  border-top: 1px solid var(--border-subtle);
  padding: 0.5rem 0;
}

/* Drag handle & edit mode */
.drag-handle {
  color: var(--text-secondary);
  cursor: grab;
  display: flex;
  align-items: center;
  flex-shrink: 0;
  opacity: 0.5;
  transition: opacity var(--transition-fast);
}

.drag-handle:active {
  cursor: grabbing;
}

.sidebar-link:hover .drag-handle {
  opacity: 1;
}

.edit-mode .sidebar-link {
  cursor: default;
  user-select: none;
}

.sortable-ghost {
  background: var(--accent-gold-glow);
  border-right: 3px solid var(--accent-gold);
  opacity: 0.6;
}

.sortable-chosen {
  background: var(--accent-gold-glow);
}

.sidebar-logout {
  color: var(--text-secondary);
}

.sidebar-logout:hover {
  color: #f87171;
  background: rgba(248, 113, 113, 0.1);
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

.main-content {
  min-height: 100vh;
}

.main-content.with-nav {
  padding-top: 76px;
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  max-width: 360px;
  width: 90%;
  padding: 2rem;
}

.modal h3 {
  margin-bottom: 0.5rem;
}

.modal p {
  color: var(--text-secondary);
  font-size: 0.9rem;
  margin-bottom: 0;
}

.modal-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
  margin-top: 1.5rem;
}

</style>
