<template>
  <div ref="pullContainer" class="container" :class="{ 'pwa-mode': isPwa }" :style="pullDistance > 0 ? `transform: translateY(${pullDistance}px); transition: none` : ''">
    <div
      class="pointer-events-none fixed left-1/2 z-[100] flex -translate-x-1/2 items-center gap-2 rounded-full border border-border-subtle bg-card px-4 py-[0.4rem] opacity-0 shadow-[0_2px_12px_rgba(0,0,0,0.3)] transition-opacity"
      :class="{ 'pointer-events-auto': pullDistance > 0 || refreshing }"
      :style="`top: ${-50 + pullDistance * 0.6}px; opacity: ${Math.min(pullDistance / 60, 1)}`"
    >
      <div
        class="h-[18px] w-[18px] rounded-full border-2 border-border-subtle border-t-gold"
        :class="{ 'animate-spin': refreshing }"
        :style="refreshing ? '' : `transform: rotate(${pullDistance * 3}deg)`"
      ></div>
      <span class="whitespace-nowrap text-sm text-text-secondary">{{ refreshing ? 'Refreshing...' : pullDistance >= 60 ? 'Release to refresh' : 'Pull to refresh' }}</span>
    </div>

    <PwaCollectionHeader
      v-if="isPwa"
      v-model:search="search"
      v-model:menu-open="menuOpen"
      v-model:selected-category="selectedCategory"
      v-model:selected-era="selectedEra"
      v-model:selected-tag="selectedTag"
      v-model:sort-key="sortKey"
      v-model:view-mode="viewMode"
      v-model:grid-side="gridSide"
      :select-mode="selectMode"
      :user-tags="userTags"
      :era-options="eraOptions"
      @toggle-select-mode="toggleSelectMode"
    />

    <DesktopCollectionHeader
      v-if="!isPwa"
      v-model:search="search"
      v-model:selected-category="selectedCategory"
      v-model:selected-era="selectedEra"
      v-model:selected-tag="selectedTag"
      v-model:sort-key="sortKey"
      v-model:grid-side="gridSide"
      :user-tags="userTags"
      :era-options="eraOptions"
    />

    <!-- Needs Attention Queue (when filter is active) -->
    <div v-if="showNeedsAttention && !selectMode" class="mb-6">
      <NeedsAttentionQueue
        :coins="store.coinHealthList"
        :loading="store.healthLoading"
        :total="healthTotal"
        :page="healthPage"
        :limit="healthLimit"
        @quick-action="handleHealthQuickAction"
        @page-change="handleHealthPageChange"
      />
    </div>

    <CollectionContent
      :loading="store.loading"
      :coins="store.coins"
      :select-mode="selectMode"
      :selected-coin-ids="selectedCoinIds"
      :selected-count="selectedCoinIds.size"
      :is-pwa="isPwa"
      :view-mode="viewMode"
      :grid-side="gridSide"
      :has-filters="!!(search || selectedCategory || selectedEra || selectedTag)"
      :total="store.total"
      :page="page"
      :per-page="COINS_PER_PAGE"
      @select-all="selectAll"
      @deselect-all="deselectAll"
      @toggle-coin-select="toggleCoinSelect"
      @page-change="handlePageChange"
    />

    <CollectionPagination
      :page="page"
      :total="store.total"
      :per-page="COINS_PER_PAGE"
      :view-mode="viewMode"
      @prev="page--"
      @next="page++"
    />

    <BulkActionBar
      :visible="selectMode && selectedCoinIds.size > 0"
      :selected-count="selectedCoinIds.size"
      @tag="showTagPicker = true"
      @location="showLocationPicker = true"
      @sell="bulkSell"
      @delete="bulkDelete"
    />

    <BulkTagPickerModal
      :open="showTagPicker"
      :tags="userTags"
      @select="bulkTag"
      @close="showTagPicker = false"
    />

    <BulkLocationPickerModal
      :open="showLocationPicker"
      :locations="storageLocations"
      @select="bulkAssignLocation"
      @close="showLocationPicker = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useCoinsStore } from '@/stores/coins'
import type { ImageType, HealthQuickAction, StorageLocation } from '@/types'
import { bulkAction, getStorageLocations } from '@/api/client'
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
import BulkLocationPickerModal from '@/components/BulkLocationPickerModal.vue'
import NeedsAttentionQueue from '@/components/collection/NeedsAttentionQueue.vue'

const store = useCoinsStore()
const router = useRouter()

const {
  selectedCategory, search, page, sortKey, selectedTag, userTags,
  selectedEra, eraOptions,
  fetchUserTags, loadCoins,
} = useCollectionFilters()

const menuOpen = ref(false)
const COINS_PER_PAGE = 50

onMounted(() => {
  fetchUserTags()
  fetchStorageLocations()
  // Reset bulkSelectActive on mount to prevent stale state from previous navigation
  bulkSelectActive.value = false
})

// Health queue state
const showNeedsAttention = computed(() => sortKey.value === 'needs_attention')
const healthPage = ref(1)
const healthLimit = ref(25)
const healthTotal = ref(0)

watch(showNeedsAttention, (show) => {
  if (show) {
    fetchHealthQueue()
  }
})

async function fetchHealthQueue() {
  try {
    const res = await store.fetchCoinHealthList('needs_attention', healthPage.value, healthLimit.value)
    healthTotal.value = res.pagination.total
  } catch (err) {
    console.error('Failed to fetch health queue:', err)
  }
}

function handleHealthPageChange(newPage: number) {
  healthPage.value = newPage
  fetchHealthQueue()
}

function handleHealthQuickAction(coinId: number, action: HealthQuickAction) {
  switch (action) {
    case 'edit_metadata':
      router.push(`/coins/${coinId}/edit`)
      break
    case 'upload_images':
      router.push(`/coins/${coinId}/edit?tab=images`)
      break
    case 'run_valuation':
      router.push(`/coins/${coinId}?action=valuation`)
      break
    case 'run_ai_analysis':
      router.push(`/coins/${coinId}?action=analysis`)
      break
  }
}

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
const showLocationPicker = ref(false)
const storageLocations = ref<StorageLocation[]>([])
const { bulkSelectActive } = useBulkSelect()

async function fetchStorageLocations() {
  try {
    const res = await getStorageLocations()
    storageLocations.value = res.data.storageLocations ?? []
  } catch {
    // Silent failure - locations will be empty if request fails
  }
}

function toggleSelectMode() {
  selectMode.value = !selectMode.value
  bulkSelectActive.value = selectMode.value
  if (!selectMode.value) {
    selectedCoinIds.value = new Set()
    showTagPicker.value = false
    showLocationPicker.value = false
  }
}

// Sync with global bulkSelectActive changes (e.g., from title bar)
watch(bulkSelectActive, (active) => {
  if (selectMode.value !== active) {
    selectMode.value = active
    if (!active) {
      selectedCoinIds.value = new Set()
      showTagPicker.value = false
      showLocationPicker.value = false
    }
  }
})

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

async function bulkTag(target: string) {
  const applyingSet = target.startsWith('set:')
  try {
    if (applyingSet) {
      const setId = Number(target.slice(4))
      await bulkAction([...selectedCoinIds.value], 'set', { setId })
    } else {
      const tagId = Number(target.startsWith('tag:') ? target.slice(4) : target)
      await bulkAction([...selectedCoinIds.value], 'tag', { tagId })
    }
    showTagPicker.value = false
    selectedCoinIds.value = new Set()
    selectMode.value = false
    bulkSelectActive.value = false
    loadCoins()
  } catch {
    alert(applyingSet ? 'Failed to apply set' : 'Failed to apply tag')
  }
}

async function bulkAssignLocation(locationId: number | null) {
  try {
    await bulkAction([...selectedCoinIds.value], 'assign-location', { storageLocationId: locationId })
    showLocationPicker.value = false
    selectedCoinIds.value = new Set()
    selectMode.value = false
    bulkSelectActive.value = false
    loadCoins()
  } catch {
    alert('Failed to assign location')
  }
}

function handlePageChange(newPage: number) {
  page.value = newPage
}

onUnmounted(() => {
  // Clean up module-level state when navigating away
  if (selectMode.value) {
    bulkSelectActive.value = false
  }
})
</script>
