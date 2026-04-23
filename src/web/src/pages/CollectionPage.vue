<template>
  <div ref="pullContainer" class="container" :class="{ 'pwa-mode': isPwa }" :style="pullDistance > 0 ? `transform: translateY(${pullDistance}px); transition: none` : ''">
    <div class="pull-indicator" :class="{ visible: pullDistance > 0 || refreshing, refreshing }" :style="`top: ${-50 + pullDistance * 0.6}px; opacity: ${Math.min(pullDistance / 60, 1)}`">
      <div class="pull-spinner" :style="refreshing ? '' : `transform: rotate(${pullDistance * 3}deg)`"></div>
      <span class="pull-text">{{ refreshing ? 'Refreshing...' : pullDistance >= 60 ? 'Release to refresh' : 'Pull to refresh' }}</span>
    </div>

    <CollectionPwaHeader
      v-if="isPwa"
      v-model:search="search"
      v-model:selected-category="selectedCategory"
      v-model:selected-tag="selectedTag"
      v-model:sort-key="sortKey"
      v-model:view-mode="viewMode"
      v-model:grid-side="gridSide"
      v-model:menu-open="menuOpen"
      :select-mode="selectMode"
      :user-tags="userTags"
      @toggle-select-mode="toggleSelectMode"
    />

    <CollectionDesktopHeader
      v-if="!isPwa"
      v-model:search="search"
      v-model:selected-category="selectedCategory"
      v-model:selected-tag="selectedTag"
      v-model:sort-key="sortKey"
      v-model:grid-side="gridSide"
      :select-mode="selectMode"
      :user-tags="userTags"
      @toggle-select-mode="toggleSelectMode"
    />

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
import type { ImageType } from '@/types'
import { bulkAction } from '@/api/client'
import { usePullToRefresh } from '@/composables/usePullToRefresh'
import { useBulkSelect } from '@/composables/useBulkSelect'
import { usePwa } from '@/composables/usePwa'
import { useCollectionFilters } from '@/composables/useCollectionFilters'
import CoinCard from '@/components/CoinCard.vue'
import SwipeGallery from '@/components/SwipeGallery.vue'
import CollectionPwaHeader from '@/components/collection/CollectionPwaHeader.vue'
import CollectionDesktopHeader from '@/components/collection/CollectionDesktopHeader.vue'
import CollectionPagination from '@/components/CollectionPagination.vue'
import BulkActionBar from '@/components/BulkActionBar.vue'
import BulkTagPickerModal from '@/components/BulkTagPickerModal.vue'

const store = useCoinsStore()
const auth = useAuthStore()
const router = useRouter()

const {
  selectedCategory, search, page, sortKey, selectedTag, userTags,
  fetchUserTags, loadCoins,
} = useCollectionFilters()

const menuOpen = ref(false)

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
