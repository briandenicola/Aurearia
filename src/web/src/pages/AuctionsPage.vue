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

    <div class="filter-bar">
      <div class="status-filters">
        <button
          v-for="s in statuses"
          :key="s.value"
          class="filter-btn"
          :class="{ active: activeStatus === s.value }"
          @click="activeStatus = s.value"
        >
          {{ s.label }}
          <span v-if="statusCounts[s.value]" class="count-badge">{{ statusCounts[s.value] }}</span>
        </button>
      </div>
    </div>

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

    <!-- Lot detail drawer -->
    <div v-if="selectedLot" class="modal-overlay" @click.self="selectedLot = null">
      <div class="lot-detail card">
        <div class="detail-header">
          <h2>{{ selectedLot.title }}</h2>
          <button class="btn-close" @click="selectedLot = null"><X :size="18" /></button>
        </div>

        <div v-if="selectedLot.imageUrl" class="detail-image-container">
          <img :src="proxiedDetailImageUrl" :alt="selectedLot.title" class="detail-image" />
        </div>

        <div class="detail-body">
          <div class="detail-row" v-if="selectedLot.auctionHouse">
            <span class="detail-label">Auction House</span>
            <span>{{ selectedLot.auctionHouse }}</span>
          </div>
          <div class="detail-row" v-if="selectedLot.saleName">
            <span class="detail-label">Sale</span>
            <span>{{ selectedLot.saleName }}</span>
          </div>
          <div class="detail-row" v-if="selectedLot.lotNumber">
            <span class="detail-label">Lot #</span>
            <span>{{ selectedLot.lotNumber }}</span>
          </div>
          <div class="detail-row" v-if="selectedLot.saleDate">
            <span class="detail-label">Sale Date</span>
            <span>{{ formatDate(selectedLot.saleDate) }}</span>
          </div>
          <div class="detail-row" v-if="selectedLot.estimate">
            <span class="detail-label">Estimate</span>
            <span>{{ formatCurrency(selectedLot.estimate, selectedLot.currency) }}</span>
          </div>
          <div class="detail-row" v-if="selectedLot.currentBid">
            <span class="detail-label">Current Bid</span>
            <span class="bid-value">{{ formatCurrency(selectedLot.currentBid, selectedLot.currency) }}</span>
          </div>
          <div class="detail-row" v-if="selectedLot.maxBid">
            <span class="detail-label">Max Bid</span>
            <span class="max-bid-value">{{ formatCurrency(selectedLot.maxBid, selectedLot.currency) }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Status</span>
            <span class="status-tag" :class="`status-${selectedLot.status}`">{{ selectedLot.status }}</span>
          </div>
          <div v-if="selectedLot.description" class="detail-description">
            <span class="detail-label">Description</span>
            <p>{{ selectedLot.description }}</p>
          </div>
        </div>

        <div class="detail-actions">
          <div class="action-row">
            <select v-model="newStatus" class="form-input status-select">
              <option value="watching">Watching</option>
              <option value="bidding">Bidding</option>
              <option value="won">Won</option>
              <option value="lost">Lost</option>
              <option value="passed">Passed</option>
            </select>
            <button class="btn btn-secondary" @click="changeStatus" :disabled="newStatus === selectedLot.status">
              Update Status
            </button>
          </div>
          <div v-if="newStatus === 'bidding'" class="action-row bid-input-row">
            <label class="detail-label">Max Bid</label>
            <input
              v-model.number="maxBidInput"
              type="number"
              class="form-input bid-input"
              :placeholder="selectedLot.currency || 'USD'"
              min="0"
              step="1"
            />
          </div>
          <div class="action-row event-link-row">
            <label class="detail-label"><CalendarDays :size="14" /> Calendar Event</label>
            <div class="event-link-controls">
              <select v-model="selectedEventId" class="form-input event-select">
                <option value="">None</option>
                <option v-for="evt in calendarEvents" :key="evt.id" :value="evt.id">
                  {{ evt.title }}
                </option>
              </select>
              <button
                class="btn btn-secondary btn-sm"
                @click="linkEvent"
                :disabled="(selectedEventId === '' ? null : Number(selectedEventId)) === (selectedLot?.eventId ?? null)"
              >
                Link
              </button>
            </div>
          </div>
          <div class="action-row">
            <a :href="selectedLot.numisBidsUrl" class="btn btn-primary" target="_blank" rel="noopener noreferrer">
              <ExternalLink :size="14" /> View on NumisBids
            </a>
            <button v-if="selectedLot.status === 'won'" class="btn btn-primary" @click="convertToCoin">
              <ArrowRightCircle :size="14" /> Add to Collection
            </button>
            <button class="btn btn-danger" @click="removeLot">
              <Trash2 :size="14" /> Remove
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Floating bulk action bar -->
    <Transition name="bar-slide">
      <div v-if="selectMode && selectedLotIds.size > 0" class="bulk-action-bar">
        <span class="bulk-count">{{ selectedLotIds.size }} lot{{ selectedLotIds.size === 1 ? '' : 's' }} selected</span>
        <div class="bulk-actions">
          <select v-model="bulkEventId" class="form-input bulk-event-select">
            <option value="">Unlink Event</option>
            <option v-for="evt in calendarEvents" :key="evt.id" :value="evt.id">
              {{ evt.title }}
            </option>
          </select>
          <button class="bulk-btn bulk-btn-link" @click="bulkLinkEvent">
            <CalendarDays :size="16" /> {{ bulkEventId === '' ? 'Unlink' : 'Link' }}
          </button>
        </div>
      </div>
    </Transition>
  </div>
  </PullToRefresh>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getAuctionLots, getAuctionLotCounts, updateAuctionLotStatus, convertAuctionLotToCoin, deleteAuctionLot, syncNumisBidsWatchlist, listCalendarEvents, linkAuctionLotEvent, bulkLinkAuctionLotEvent } from '@/api/client'
import type { AuctionLot, AuctionLotStatus } from '@/types'
import AuctionLotCard from '@/components/AuctionLotCard.vue'
import ImportLotModal from '@/components/ImportLotModal.vue'
import PullToRefresh from '@/components/PullToRefresh.vue'
import { Plus, CirclePlus, X, ExternalLink, ArrowRightCircle, Trash2, RefreshCw, CalendarDays, CheckSquare } from 'lucide-vue-next'
import { usePwa } from '@/composables/usePwa'
import { formatCurrency } from '@/utils/format'

const router = useRouter()
const API_BASE = import.meta.env.VITE_API_BASE_URL || ''
const { isPwa } = usePwa()

const lots = ref<AuctionLot[]>([])
const statusCounts = ref<Record<string, number>>({})
const loading = ref(true)
const showImport = ref(false)
const selectedLot = ref<AuctionLot | null>(null)
const newStatus = ref<AuctionLotStatus>('watching')
const maxBidInput = ref<number | null>(null)
const activeStatus = ref('bidding')
const syncing = ref(false)
const syncMessage = ref('')
const calendarEvents = ref<Array<{ id: number; title: string; auctionHouse: string; startDate: string | null }>>([])
const selectedEventId = ref<number | string>('')

const selectMode = ref(false)
const selectedLotIds = ref(new Set<number>())
const bulkEventId = ref<number | string>('')

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
  if (next.has(lotId)) {
    next.delete(lotId)
  } else {
    next.add(lotId)
  }
  selectedLotIds.value = next
}

function selectAllLots() {
  selectedLotIds.value = new Set(lots.value.map(l => l.id))
}

function deselectAllLots() {
  selectedLotIds.value = new Set()
}

async function bulkLinkEvent() {
  const eventId = bulkEventId.value === '' ? null : Number(bulkEventId.value)
  try {
    await bulkLinkAuctionLotEvent([...selectedLotIds.value], eventId)
    selectedLotIds.value = new Set()
    selectMode.value = false
    bulkEventId.value = ''
    fetchLots()
  } catch { /* ignore */ }
}

async function fetchCalendarEvents() {
  try {
    const res = await listCalendarEvents()
    calendarEvents.value = res.data?.events ?? []
  } catch { /* ignore */ }
}

async function linkEvent() {
  if (!selectedLot.value) return
  const eventId = selectedEventId.value === '' ? null : Number(selectedEventId.value)
  try {
    const res = await linkAuctionLotEvent(selectedLot.value.id, eventId)
    selectedLot.value = res.data
    fetchLots()
  } catch { /* ignore */ }
}

const proxiedDetailImageUrl = computed(() => {
  if (!selectedLot.value?.imageUrl) return ''
  const token = localStorage.getItem('token') ?? ''
  return `${API_BASE}/api/proxy-image?url=${encodeURIComponent(selectedLot.value.imageUrl)}&token=${encodeURIComponent(token)}`
})

const statuses = [
  { value: '', label: 'All' },
  { value: 'watching', label: 'Watching' },
  { value: 'bidding', label: 'Bidding' },
  { value: 'won', label: 'Won' },
  { value: 'lost', label: 'Lost' },
  { value: 'passed', label: 'Passed' },
]

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
  newStatus.value = lot.status
  maxBidInput.value = lot.maxBid ?? null
  selectedEventId.value = lot.eventId ?? ''
  fetchCalendarEvents()
}

function handleImported() {
  showImport.value = false
  fetchLots()
  fetchAllCounts()
}

async function changeStatus() {
  if (!selectedLot.value) return
  try {
    const bid = newStatus.value === 'bidding' ? maxBidInput.value : undefined
    const res = await updateAuctionLotStatus(selectedLot.value.id, newStatus.value, bid)
    selectedLot.value = res.data

    // When marked as Won, automatically convert to a coin and open edit page
    if (newStatus.value === 'won') {
      try {
        const coinRes = await convertAuctionLotToCoin(selectedLot.value.id)
        selectedLot.value = null
        router.push(`/edit/${coinRes.data.id}`)
        return
      } catch { /* fall through — show drawer with manual convert button */ }
    }

    fetchLots()
    fetchAllCounts()
  } catch { /* ignore */ }
}

async function convertToCoin() {
  if (!selectedLot.value) return
  try {
    const coinRes = await convertAuctionLotToCoin(selectedLot.value.id)
    selectedLot.value = null
    router.push(`/edit/${coinRes.data.id}`)
  } catch { /* ignore */ }
}

async function removeLot() {
  if (!selectedLot.value) return
  try {
    await deleteAuctionLot(selectedLot.value.id)
    selectedLot.value = null
    fetchLots()
    fetchAllCounts()
  } catch { /* ignore */ }
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

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' })
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

.filter-bar {
  margin-bottom: 1.25rem;
}

.status-filters {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.filter-btn {
  padding: 0.4rem 0.9rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.82rem;
  cursor: pointer;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.filter-btn:hover {
  border-color: var(--accent-gold-dim);
  color: var(--text-primary);
}

.filter-btn.active {
  background: var(--accent-gold-glow);
  border-color: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.count-badge {
  background: var(--bg-primary);
  padding: 0.05rem 0.4rem;
  border-radius: var(--radius-full);
  font-size: 0.7rem;
  font-weight: 600;
}

.lots-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 1.25rem;
}

/* Lot detail modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.lot-detail {
  max-width: 560px;
  width: 100%;
  max-height: 90vh;
  overflow-y: auto;
  padding: 0;
}

.detail-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--border-subtle);
  gap: 1rem;
}

.detail-header h2 {
  font-size: 1.1rem;
  line-height: 1.35;
  margin: 0;
}

.btn-close {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
}

.btn-close:hover {
  color: var(--text-primary);
}

.detail-image-container {
  width: 100%;
  max-height: 300px;
  overflow: hidden;
  background: var(--bg-primary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.detail-image {
  width: 100%;
  height: 300px;
  object-fit: contain;
}

.detail-body {
  padding: 1.25rem 1.5rem;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 0;
  border-bottom: 1px solid var(--border-subtle);
  font-size: 0.88rem;
}

.detail-label {
  color: var(--text-secondary);
  font-size: 0.82rem;
}

.bid-value {
  font-weight: 600;
  color: var(--accent-gold);
}

.max-bid-value {
  font-weight: 600;
  color: var(--accent-gold);
  opacity: 0.8;
}

.bid-input-row {
  align-items: center;
}

.bid-input {
  flex: 1;
  max-width: 140px;
}

.status-tag {
  padding: 0.15rem 0.55rem;
  border-radius: var(--radius-full);
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
}

.status-watching { background: rgba(100, 150, 255, 0.2); color: #6496ff; }
.status-bidding { background: var(--accent-gold-glow); color: var(--accent-gold); }
.status-won { background: rgba(74, 222, 128, 0.15); color: #4ade80; }
.status-lost { background: rgba(248, 113, 113, 0.15); color: #f87171; }
.status-passed { background: rgba(120, 120, 120, 0.15); color: #999; }

.detail-description {
  margin-top: 0.75rem;
}

.detail-description p {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin-top: 0.4rem;
  line-height: 1.5;
  max-height: 120px;
  overflow-y: auto;
}

.detail-actions {
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--border-subtle);
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.action-row {
  display: flex;
  gap: 0.6rem;
  flex-wrap: wrap;
}

.status-select {
  flex: 1;
  min-width: 120px;
  max-width: 160px;
}

.btn-danger {
  background: transparent;
  border: 1px solid rgba(248, 113, 113, 0.4);
  color: #f87171;
  padding: 0.5rem 0.9rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 0.82rem;
  display: flex;
  align-items: center;
  gap: 0.35rem;
  transition: all var(--transition-fast);
}

.btn-danger:hover {
  background: rgba(248, 113, 113, 0.1);
  border-color: rgba(248, 113, 113, 0.6);
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

.event-link-row {
  flex-direction: column;
  gap: 0.4rem;
}

.event-link-row .detail-label {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.event-link-controls {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.event-select {
  flex: 1;
  min-width: 160px;
}

.btn-sm {
  padding: 0.35rem 0.7rem;
  font-size: 0.8rem;
}

/* Select controls */
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

/* Floating bulk action bar */
.bulk-action-bar {
  position: fixed;
  bottom: 1.5rem;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 1rem;
  background: var(--bg-card);
  border: 1px solid var(--accent-gold-dim);
  border-radius: var(--radius-md);
  padding: 0.75rem 1.25rem;
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.5);
  z-index: 200;
  white-space: nowrap;
}

.bulk-count {
  font-size: 0.85rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.bulk-actions {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.bulk-event-select {
  min-width: 160px;
  font-size: 0.82rem;
}

.bulk-btn {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.4rem 0.75rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-primary);
  color: var(--text-primary);
  font-size: 0.8rem;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.bulk-btn:hover {
  border-color: var(--accent-gold);
  color: var(--accent-gold);
}

/* Bar slide transition */
.bar-slide-enter-active,
.bar-slide-leave-active {
  transition: transform 0.25s ease, opacity 0.25s ease;
}

.bar-slide-enter-from,
.bar-slide-leave-to {
  transform: translateX(-50%) translateY(20px);
  opacity: 0;
}
</style>
