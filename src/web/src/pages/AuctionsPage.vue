<template>
  <PullToRefresh :on-refresh="handleRefresh">
    <div class="container">
      <div class="page-header">
        <h1>Auctions</h1>
        <!-- PWA: icon-only buttons inline with title -->
        <div v-if="isPwa" class="pwa-actions">
          <button class="pwa-icon-btn" :disabled="syncing" @click="syncWatchlist" title="Sync Watchlist">
            <RefreshCw :size="22" :class="{ spinning: syncing }" />
          </button>
          <button class="pwa-icon-btn" :class="{ active: selectMode }" @click="toggleSelectMode" title="Select">
            <CheckSquare :size="22" />
          </button>
          <button class="pwa-icon-btn" @click="showImport = true" title="Add Lot">
            <CirclePlus :size="22" />
          </button>
        </div>
        <!-- Desktop: full text buttons -->
        <div v-else class="header-actions">
          <button class="btn btn-secondary" :disabled="syncing" @click="syncWatchlist">
            <RefreshCw :size="16" :class="{ spinning: syncing }" />
            {{ syncing ? 'Syncing...' : 'Sync Watchlist' }}
          </button>
          <button class="btn" :class="selectMode ? 'btn-primary' : 'btn-secondary'" @click="toggleSelectMode">
            <CheckSquare :size="16" /> {{ selectMode ? 'Cancel' : 'Select' }}
          </button>
          <button class="btn btn-primary" @click="showImport = true"><Plus :size="16" /> Add Lot</button>
        </div>
      </div>

      <div v-if="syncMessage" class="sync-toast">{{ syncMessage }}</div>

      <AuctionStatusFilter v-model="activeStatus" :counts="statusCounts" />

      <div v-if="selectMode" class="select-controls">
        <button class="btn btn-sm btn-secondary" @click="selectAllLots">Select All</button>
        <button class="btn btn-sm btn-secondary" @click="deselectAllLots">Deselect All</button>
        <span class="select-count">{{ selectedLotIds.size }} selected</span>
      </div>

      <div v-if="loading" class="loading-overlay">
        <div class="spinner"></div>
      </div>

      <div v-else-if="lots.length" class="lots-grid">
        <AuctionLotCard
          v-for="lot in lots"
          :key="lot.id"
          :lot="lot"
          :selectable="selectMode"
          :selected="selectedLotIds.has(lot.id)"
          @select="openLot"
          @toggle-select="toggleLotSelect"
        />
      </div>

      <div v-else class="empty-state">
        <h3>No auction lots{{ activeStatus ? ` with status "${activeStatus}"` : '' }}</h3>
        <p>Import lots from NumisBids to start tracking auctions</p>
        <button class="btn btn-primary" @click="showImport = true" style="margin-top: 0.75rem">
          <Plus :size="16" /> Import Your First Lot
        </button>
      </div>

      <ImportLotModal v-if="showImport" @close="showImport = false" @imported="handleImported" />

      <AuctionLotDetailModal
        v-if="selectedLot"
        :lot="selectedLot"
        @close="selectedLot = null"
        @updated="handleLotUpdated"
      />

      <AuctionBulkActionBar
        v-if="selectMode"
        :selected-count="selectedLotIds.size"
        :calendar-events="calendarEvents"
        @link-event="handleBulkLinkEvent"
      />
    </div>
  </PullToRefresh>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { getAuctionLots, getAuctionLotCounts, syncNumisBidsWatchlist, listCalendarEvents, bulkLinkAuctionLotEvent } from '@/api/client'
import type { AuctionLot } from '@/types'
import AuctionLotCard from '@/components/AuctionLotCard.vue'
import ImportLotModal from '@/components/ImportLotModal.vue'
import PullToRefresh from '@/components/PullToRefresh.vue'
import AuctionStatusFilter from '@/components/auction/AuctionStatusFilter.vue'
import AuctionLotDetailModal from '@/components/auction/AuctionLotDetailModal.vue'
import AuctionBulkActionBar from '@/components/auction/AuctionBulkActionBar.vue'
import { Plus, CirclePlus, RefreshCw, CheckSquare } from 'lucide-vue-next'
import { usePwa } from '@/composables/usePwa'

const { isPwa } = usePwa()

const lots = ref<AuctionLot[]>([])
const statusCounts = ref<Record<string, number>>({})
const loading = ref(true)
const showImport = ref(false)
const selectedLot = ref<AuctionLot | null>(null)
const activeStatus = ref('bidding')
const syncing = ref(false)
const syncMessage = ref('')
const calendarEvents = ref<Array<{ id: number; title: string; auctionHouse: string; startDate: string | null }>>([])

const selectMode = ref(false)
const selectedLotIds = ref(new Set<number>())

function toggleSelectMode() {
  selectMode.value = !selectMode.value
  if (!selectMode.value) {
    selectedLotIds.value = new Set()
  } else {
    fetchCalendarEvents()
  }
}

function toggleLotSelect(lotId: number) {
  const next = new Set(selectedLotIds.value)
  if (next.has(lotId)) next.delete(lotId)
  else next.add(lotId)
  selectedLotIds.value = next
}

function selectAllLots() {
  selectedLotIds.value = new Set(lots.value.map(l => l.id))
}

function deselectAllLots() {
  selectedLotIds.value = new Set()
}

async function fetchCalendarEvents() {
  try {
    const res = await listCalendarEvents()
    calendarEvents.value = res.data?.events ?? []
  } catch { /* ignore */ }
}

async function handleBulkLinkEvent(eventIdRaw: number | string) {
  const eventId = eventIdRaw === '' ? null : Number(eventIdRaw)
  try {
    await bulkLinkAuctionLotEvent([...selectedLotIds.value], eventId)
    selectedLotIds.value = new Set()
    selectMode.value = false
    fetchLots()
  } catch { /* ignore */ }
}

watch(activeStatus, () => fetchLots())

async function fetchLots() {
  loading.value = true
  try {
    const params: Record<string, string> = { sort: 'updated_at', order: 'desc' }
    if (activeStatus.value) params.status = activeStatus.value
    const res = await getAuctionLots(params)
    lots.value = res.data?.lots ?? []
  } catch {
    lots.value = []
  } finally {
    loading.value = false
  }
}

async function fetchAllCounts() {
  try {
    const res = await getAuctionLotCounts()
    statusCounts.value = res.data?.counts ?? {}
  } catch { /* ignore */ }
}

async function handleRefresh() {
  await fetchLots()
  await fetchAllCounts()
}

function openLot(lot: AuctionLot) {
  selectedLot.value = lot
}

function handleImported() {
  showImport.value = false
  fetchLots()
  fetchAllCounts()
}

function handleLotUpdated() {
  fetchLots()
  fetchAllCounts()
}

async function syncWatchlist() {
  syncing.value = true
  syncMessage.value = ''
  try {
    const res = await syncNumisBidsWatchlist()
    const count = res.data?.synced ?? 0
    syncMessage.value = `Synced ${count} lot${count !== 1 ? 's' : ''} from NumisBids`
    fetchLots()
    fetchAllCounts()
    setTimeout(() => { syncMessage.value = '' }, 4000)
  } catch (err: unknown) {
    const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Sync failed'
    syncMessage.value = msg
    setTimeout(() => { syncMessage.value = '' }, 5000)
  } finally {
    syncing.value = false
  }
}

fetchLots()
fetchAllCounts()
</script>

<style scoped>
.header-actions {
  display: flex;
  gap: 0.75rem;
  align-items: center;
}

.lots-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 1.25rem;
}

.sync-toast {
  padding: 0.6rem 1rem;
  margin-bottom: 1rem;
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  color: var(--text-primary);
  font-size: 0.85rem;
  text-align: center;
  animation: fadeIn 0.2s ease;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(-4px); }
  to { opacity: 1; transform: translateY(0); }
}

.select-controls {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  margin-bottom: 1rem;
}

.select-count {
  font-size: 0.85rem;
  color: var(--text-secondary);
  font-weight: 500;
}
</style>
