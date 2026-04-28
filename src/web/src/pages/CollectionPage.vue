<template>
  <div ref="pullContainer" class="container" :class="{ 'pwa-mode': isPwa }" :style="pullDistance > 0 ? `transform: translateY(${pullDistance}px); transition: none` : ''">
    <div class="pull-indicator" :class="{ visible: pullDistance > 0 || refreshing, refreshing }" :style="`top: ${-50 + pullDistance * 0.6}px; opacity: ${Math.min(pullDistance / 60, 1)}`">
      <div class="pull-spinner" :style="refreshing ? '' : `transform: rotate(${pullDistance * 3}deg)`"></div>
      <span class="pull-text">{{ refreshing ? 'Refreshing...' : pullDistance >= 60 ? 'Release to refresh' : 'Pull to refresh' }}</span>
    </div>

    <PwaCollectionHeader
      v-if="isPwa"
      v-model:search="search"
      v-model:menuOpen="menuOpen"
      v-model:selectedCategory="selectedCategory"
      v-model:selectedTag="selectedTag"
      v-model:sortKey="sortKey"
      v-model:viewMode="viewMode"
      v-model:gridSide="gridSide"
      :select-mode="selectMode"
      :user-tags="userTags"
      @toggle-select-mode="toggleSelectMode"
    />

    <DesktopCollectionHeader
      v-if="!isPwa"
      v-model:search="search"
      v-model:selectedCategory="selectedCategory"
      v-model:selectedTag="selectedTag"
      v-model:sortKey="sortKey"
      v-model:gridSide="gridSide"
      :select-mode="selectMode"
      :user-tags="userTags"
      @toggle-select-mode="toggleSelectMode"
    />

    <CollectionContent
      :loading="store.loading"
      :coins="store.coins"
      :select-mode="selectMode"
      :selected-coin-ids="selectedCoinIds"
      :selected-count="selectedCoinIds.size"
      :is-pwa="isPwa"
      :view-mode="viewMode"
      :grid-side="gridSide"
      :has-filters="!!(search || selectedCategory)"
      @select-all="selectAll"
      @deselect-all="deselectAll"
      @toggle-coin-select="toggleCoinSelect"
    />

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
import type { ImageType } from '@/types'
import { bulkAction } from '@/api/client'
import { usePullToRefresh } from '@/composables/usePullToRefresh'
import { useBulkSelect } from '@/composables/useBulkSelect'
import { usePwa } from '@/composables/usePwa'
import { useCollectionFilters } from '@/composables/useCollectionFilters'
import PwaCollectionHeader from '@/components/collection/PwaCollectionHeader.vue'
import DesktopCollectionHeader from '@/components/collection/DesktopCollectionHeader.vue'
import CollectionContent from '@/components/collection/CollectionContent.vue'
import CollectionPagination from '@/components/CollectionPagination.vue'
import BulkActionBar from '@/components/BulkActionBar.vue'
import BulkTagPickerModal from '@/components/BulkTagPickerModal.vue'

const store = useCoinsStore()

const {
  selectedCategory, search, page, sortKey, selectedTag, userTags,
  fetchUserTags, loadCoins,
} = useCollectionFilters()

const menuOpen = ref(false)

onMounted(fetchUserTags)

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
</style>
