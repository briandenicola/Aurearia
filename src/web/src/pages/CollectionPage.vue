<template>
  <div ref="pullContainer" class="container" :class="{ 'pwa-mode': isPwa }" :style="pullDistance > 0 ? `transform: translateY(${pullDistance}px); transition: none` : ''">
    <div class="pull-indicator" :class="{ visible: pullDistance > 0 || refreshing, refreshing }" :style="`top: ${-50 + pullDistance * 0.6}px; opacity: ${Math.min(pullDistance / 60, 1)}`">
      <div class="pull-spinner" :style="refreshing ? '' : `transform: rotate(${pullDistance * 3}deg)`"></div>
      <span class="pull-text">{{ refreshing ? 'Refreshing...' : pullDistance >= 60 ? 'Release to refresh' : 'Pull to refresh' }}</span>
    </div>

    <!-- PWA compact header: search + filter/sort -->
    <div v-if="isPwa" class="pwa-header">
      <SearchBar v-model="search" />
      <button class="pwa-icon-btn" :class="{ active: selectMode }" @click="toggleSelectMode" title="Select">
        <CheckSquare :size="22" />
      </button>
      <router-link to="/add" class="pwa-add-btn">
        <CirclePlus :size="22" />
      </router-link>
      <div class="hamburger-wrapper">
        <button class="hamburger-btn" @click="menuOpen = !menuOpen" :class="{ active: menuOpen }">
          <SlidersHorizontal :size="22" />
        </button>
        <Transition name="menu-slide">
          <div v-if="menuOpen" class="pwa-menu">
            <div class="pwa-menu-section">
              <span class="pwa-menu-label">Category</span>
              <CategoryFilter v-model="selectedCategory" />
            </div>
            <div v-if="userTags.length" class="pwa-menu-section">
              <span class="pwa-menu-label">Tag</span>
              <select v-model="selectedTag" class="tag-filter-select pwa-tag-select">
                <option value="">All Tags</option>
                <option v-for="tag in userTags" :key="tag.id" :value="String(tag.id)">{{ tag.name }}</option>
              </select>
            </div>
            <div class="pwa-menu-section">
              <span class="pwa-menu-label">Sort</span>
              <SortSelect v-model="sortKey" />
            </div>
            <div class="pwa-menu-section">
              <span class="pwa-menu-label">View</span>
              <div class="pwa-menu-row">
                <div class="view-toggle">
                  <button class="view-btn" :class="{ active: viewMode === 'swipe' }" @click="viewMode = 'swipe'" title="Swipe view">
                    <Layers :size="18" />
                  </button>
                  <button class="view-btn" :class="{ active: viewMode === 'grid' }" @click="viewMode = 'grid'" title="Grid view">
                    <LayoutGrid :size="18" />
                  </button>
                </div>
                <div v-if="viewMode === 'grid'" class="side-toggle">
                  <button class="toggle-btn" :class="{ active: gridSide === null }" @click="gridSide = null">Primary</button>
                  <button class="toggle-btn" :class="{ active: gridSide === 'obverse' }" @click="gridSide = 'obverse'">Obverse</button>
                  <button class="toggle-btn" :class="{ active: gridSide === 'reverse' }" @click="gridSide = 'reverse'">Reverse</button>
                </div>
              </div>
            </div>
          </div>
        </Transition>
      </div>
    </div>
    <div v-if="isPwa && menuOpen" class="pwa-menu-backdrop" @click="menuOpen = false"></div>

    <!-- Desktop header (hidden in PWA) -->
    <div v-if="!isPwa" class="desktop-sticky-header">
      <div class="page-header collection-header">
        <div class="header-spacer"></div>
        <SearchBar v-model="search" />
        <div class="header-sort">
          <SortSelect v-model="sortKey" />
        </div>
      </div>

      <div class="collection-toolbar">
        <div class="toolbar-filters">
          <CategoryFilter v-model="selectedCategory" />
          <select v-if="userTags.length" v-model="selectedTag" class="tag-filter-select">
            <option value="">All Tags</option>
            <option v-for="tag in userTags" :key="tag.id" :value="String(tag.id)">{{ tag.name }}</option>
          </select>
        </div>
        <div class="toolbar-right">
          <button class="btn" :class="selectMode ? 'btn-primary' : 'btn-secondary'" @click="toggleSelectMode">
            <CheckSquare :size="16" /> {{ selectMode ? 'Cancel' : 'Select' }}
          </button>
          <div class="side-toggle">
            <button class="btn btn-primary toggle-btn" :class="{ active: gridSide === null }" @click="gridSide = null">
              Primary
            </button>
            <button class="btn btn-primary toggle-btn" :class="{ active: gridSide === 'obverse' }" @click="gridSide = 'obverse'">
              Obverse
            </button>
            <button class="btn btn-primary toggle-btn" :class="{ active: gridSide === 'reverse' }" @click="gridSide = 'reverse'">
              Reverse
            </button>
          </div>
          <router-link to="/add" class="btn btn-primary"><CirclePlus :size="16" /> Add Coin</router-link>
        </div>
      </div>
    </div>

    <div v-if="store.loading" class="loading-overlay">
      <div class="spinner"></div>
      <p>Loading collection...</p>
    </div>

    <template v-else-if="store.coins.length">
      <div v-if="selectMode" class="select-controls">
        <button class="btn btn-sm btn-secondary" @click="selectAll">Select All</button>
        <button class="btn btn-sm btn-secondary" @click="deselectAll">Deselect All</button>
        <span class="select-count">{{ selectedCoinIds.size }} selected</span>
      </div>
      <SwipeGallery v-if="isPwa && viewMode === 'swipe' && !selectMode" :coins="store.coins" />
      <div v-else class="coins-grid">
        <CoinCard
          v-for="coin in store.coins"
          :key="coin.id"
          :coin="coin"
          :image-side="gridSide"
          :selectable="selectMode"
          :selected="selectedCoinIds.has(coin.id)"
          @toggle-select="toggleCoinSelect"
        />
      </div>
    </template>

    <div v-else class="empty-state">
      <h3>{{ search || selectedCategory ? 'No coins match your search' : 'Your collection is empty' }}</h3>
      <p>{{ search || selectedCategory ? 'Try different filters' : 'Add your first coin to get started' }}</p>
      <router-link v-if="!search && !selectedCategory" to="/add" class="btn btn-primary" style="margin-top: 1rem">
        Add Your First Coin
      </router-link>
    </div>

    <CollectionPagination
      :page="page"
      :total="store.total"
      :per-page="50"
      :view-mode="viewMode"
      @prev="page--"
      @next="page++"
    />

    <BulkActionBar
      :visible="selectMode && selectedCoinIds.size > 0"
      :selected-count="selectedCoinIds.size"
      @tag="showTagPicker = true"
      @sell="bulkSell"
      @delete="bulkDelete"
    />

    <BulkTagPickerModal
      :open="showTagPicker"
      :tags="userTags"
      @select="bulkTag"
      @close="showTagPicker = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useCoinsStore } from '@/stores/coins'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'
import type { ImageType, Tag } from '@/types'
import { getTags, bulkAction } from '@/api/client'
import { usePullToRefresh } from '@/composables/usePullToRefresh'
import { useBulkSelect } from '@/composables/useBulkSelect'
import { usePwa } from '@/composables/usePwa'
import CoinCard from '@/components/CoinCard.vue'
import SwipeGallery from '@/components/SwipeGallery.vue'
import CategoryFilter from '@/components/CategoryFilter.vue'
import SearchBar from '@/components/SearchBar.vue'
import SortSelect from '@/components/SortSelect.vue'
import CollectionPagination from '@/components/CollectionPagination.vue'
import BulkActionBar from '@/components/BulkActionBar.vue'
import BulkTagPickerModal from '@/components/BulkTagPickerModal.vue'

import { Layers, LayoutGrid, CirclePlus, SlidersHorizontal, CheckSquare } from 'lucide-vue-next'

const store = useCoinsStore()
const auth = useAuthStore()
const router = useRouter()
const selectedCategory = store.selectedCategory !== undefined ? ref(store.selectedCategory) : ref('')
const search = ref(store.searchQuery)
const page = ref(1)
const sortKey = ref(localStorage.getItem('defaultSort') || 'updated_at_desc')
const menuOpen = ref(false)
const selectedTag = ref('')
const userTags = ref<Tag[]>([])

async function fetchUserTags() {
  try {
    const res = await getTags()
    userTags.value = res.data?.tags ?? []
  } catch { /* ignore */ }
}

onMounted(fetchUserTags)

// Use saved preference if set, otherwise default to swipe in PWA mode
const savedView = localStorage.getItem('defaultView') as 'grid' | 'swipe' | null
const { isPwa } = usePwa()
const viewMode = ref<'grid' | 'swipe'>(isPwa ? (savedView || 'swipe') : 'grid')
const gridSide = ref<ImageType | null>(null)

const pullContainer = ref<HTMLElement | null>(null)
const { pullDistance, refreshing } = usePullToRefresh(pullContainer, async () => {
  await new Promise<void>((resolve) => {
    loadCoins()
    const unwatch = watch(() => store.loading, (loading) => {
      if (!loading) { unwatch(); resolve() }
    })
    if (!store.loading) { unwatch(); resolve() }
  })
})

let debounceTimer: ReturnType<typeof setTimeout>

function loadCoins() {
  const [sort, order] = sortKey.value.split('_').length === 3
    ? [sortKey.value.split('_').slice(0, 2).join('_'), sortKey.value.split('_')[2]]
    : [sortKey.value.split('_')[0], sortKey.value.split('_')[1]]
  store.selectedCategory = selectedCategory.value
  store.searchQuery = search.value
  store.fetchCoins({
    category: selectedCategory.value || undefined,
    search: search.value || undefined,
    tag: selectedTag.value || undefined,
    wishlist: 'false',
    sold: 'false',
    page: page.value,
    sort,
    order,
  })
}

watch(selectedCategory, () => {
  page.value = 1
  loadCoins()
})

watch(selectedTag, () => {
  page.value = 1
  loadCoins()
})

watch(search, () => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    page.value = 1
    loadCoins()
  }, 300)
})

watch(page, loadCoins)
watch(sortKey, () => {
  page.value = 1
  loadCoins()
})

loadCoins()

// Select mode state
const selectMode = ref(false)
const selectedCoinIds = ref(new Set<number>())
const showTagPicker = ref(false)
const { bulkSelectActive } = useBulkSelect()

function toggleSelectMode() {
  selectMode.value = !selectMode.value
  bulkSelectActive.value = selectMode.value
  if (!selectMode.value) {
    selectedCoinIds.value = new Set()
    showTagPicker.value = false
  }
}

function toggleCoinSelect(coinId: number) {
  const next = new Set(selectedCoinIds.value)
  if (next.has(coinId)) {
    next.delete(coinId)
  } else {
    next.add(coinId)
  }
  selectedCoinIds.value = next
}

function selectAll() {
  selectedCoinIds.value = new Set(store.coins.map(c => c.id))
}

function deselectAll() {
  selectedCoinIds.value = new Set()
}

async function bulkDelete() {
  const count = selectedCoinIds.value.size
  if (!confirm(`Delete ${count} coin${count === 1 ? '' : 's'}? This cannot be undone.`)) return
  try {
    await bulkAction([...selectedCoinIds.value], 'delete')
    selectedCoinIds.value = new Set()
    selectMode.value = false
    bulkSelectActive.value = false
    loadCoins()
  } catch {
    alert('Failed to delete coins')
  }
}

async function bulkSell() {
  const count = selectedCoinIds.value.size
  if (!confirm(`Mark ${count} coin${count === 1 ? '' : 's'} as sold?`)) return
  try {
    await bulkAction([...selectedCoinIds.value], 'sell')
    selectedCoinIds.value = new Set()
    selectMode.value = false
    bulkSelectActive.value = false
    loadCoins()
  } catch {
    alert('Failed to mark coins as sold')
  }
}

async function bulkTag(tagId: number) {
  try {
    await bulkAction([...selectedCoinIds.value], 'tag', tagId)
    showTagPicker.value = false
    selectedCoinIds.value = new Set()
    selectMode.value = false
    bulkSelectActive.value = false
    loadCoins()
  } catch {
    alert('Failed to apply tag')
  }
}

</script>

<style scoped>
/* --- PWA compact header --- */
.pwa-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
  position: sticky;
  top: 60px;
  z-index: 150;
  background: var(--bg-primary);
  padding: 0.5rem 0;
}

.pwa-header :deep(.search-bar) {
  flex: 1;
  max-width: none;
}

.hamburger-wrapper {
  position: relative;
  flex-shrink: 0;
}

.hamburger-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.hamburger-btn.active,
.hamburger-btn:hover {
  border-color: var(--accent-gold);
  color: var(--accent-gold);
  background: var(--accent-gold-dim);
}

/* --- PWA popover menu --- */
.pwa-menu {
  position: absolute;
  top: calc(100% + 0.5rem);
  right: 0;
  z-index: 100;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1rem;
  min-width: 260px;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.4);
}

.pwa-menu-backdrop {
  position: fixed;
  inset: 0;
  z-index: 90;
}

.pwa-menu-section {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.pwa-menu-label {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  font-weight: 600;
}

.pwa-menu-row {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  flex-wrap: wrap;
}

.pwa-add-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  text-decoration: none;
}

/* Menu slide transition */
.menu-slide-enter-active,
.menu-slide-leave-active {
  transition: all 0.2s ease;
}
.menu-slide-enter-from,
.menu-slide-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* --- Desktop sticky header --- */
.desktop-sticky-header {
  position: sticky;
  top: 60px;
  z-index: 50;
  background: var(--bg-primary);
  padding-bottom: 0.5rem;
  margin: 0 -2rem;
  padding-left: 2rem;
  padding-right: 2rem;
}

/* --- Desktop header --- */
.collection-header {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.header-spacer {
  flex: 1;
}

.collection-header :deep(.search-bar) {
  flex: 0 1 600px;
}

.collection-header :deep(.search-input) {
  padding: 0.75rem 2.5rem;
  font-size: 0.95rem;
}

.header-sort {
  flex: 1;
  display: flex;
  justify-content: flex-end;
}

.collection-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
  margin-bottom: 1rem;
}

.toolbar-filters {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.tag-filter-select {
  padding: 0.35rem 0.5rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text-primary);
  font-size: 0.85rem;
  cursor: pointer;
}

.pwa-tag-select {
  width: 100%;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.view-toggle {
  display: flex;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.view-btn {
  padding: 0.4rem 0.6rem;
  border: none;
  background: var(--bg-card);
  color: var(--text-secondary);
  font-size: 1rem;
  cursor: pointer;
  transition: all var(--transition-fast);
  line-height: 1;
}

.view-btn.active {
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.view-btn:hover:not(.active) {
  background: var(--bg-card-hover);
}

.side-toggle {
  display: flex;
  gap: 0;
}

.side-toggle .toggle-btn {
  border-radius: 0;
  border-right: 1px solid rgba(255, 255, 255, 0.15);
}

.side-toggle .toggle-btn:first-child {
  border-radius: var(--radius-sm) 0 0 var(--radius-sm);
}

.side-toggle .toggle-btn:last-child {
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  border-right: none;
}

.toggle-btn {
  opacity: 0.6;
}

.toggle-btn.active {
  opacity: 1;
  background: var(--accent-gold);
  color: #1a1a2e;
}

.toggle-btn:hover:not(.active) {
  opacity: 0.8;
}

/* --- Pull to refresh --- */
.pull-indicator {
  position: fixed;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.4rem 1rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.3);
  z-index: 100;
  pointer-events: none;
  opacity: 0;
  transition: opacity 0.2s;
}

.pull-indicator.visible {
  pointer-events: auto;
}

.pull-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--border-subtle);
  border-top-color: var(--accent-gold);
  border-radius: 50%;
}

.pull-indicator.refreshing .pull-spinner {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.pull-text {
  font-size: 0.75rem;
  color: var(--text-secondary);
  white-space: nowrap;
}

@media (max-width: 768px) {
  .collection-header {
    flex-direction: column;
    align-items: stretch;
  }

  .header-spacer {
    display: none;
  }

  .header-sort {
    justify-content: flex-start;
  }

  .collection-header :deep(.search-bar) {
    max-width: 100%;
  }

  .collection-header :deep(.search-input) {
    padding: 0.6rem 2.5rem;
    font-size: 0.85rem;
  }

  .header-filters {
    justify-content: flex-start;
  }
}

/* --- Select mode controls --- */
.select-controls {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.select-count {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin-left: 0.5rem;
}

</style>
