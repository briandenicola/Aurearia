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
            {{ syncing ? 'Syncing...' : 'Sync Watchlists' }}
          </button>
          <button class="btn" :class="selectMode ? 'btn-primary' : 'btn-secondary'" @click="toggleSelectMode">
            <CheckSquare :size="16" /> {{ selectMode ? 'Cancel' : 'Select' }}
          </button>
          <button class="btn btn-primary" @click="showImport = true"><Plus :size="16" /> Add Lot</button>
        </div>
      </div>

      <div v-if="syncMessage" class="sync-toast">{{ syncMessage }}</div>

      <div class="auction-filter-toolbar">
        <div class="source-filter" aria-label="Auction source filter">
          <button
            v-for="source in sourceOptions"
            :key="source.value"
            class="chip"
            :class="{ active: activeSource === source.value }"
            @click="activeSource = source.value"
          >
            {{ source.label }}
          </button>
        </div>
        <AuctionStatusFilter v-model="activeStatus" :counts="statusCounts" />
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
          :price-alerts="alertsByLot[lot.id] ?? []"
          :bid-reminders="remindersByLot[lot.id] ?? []"
          @select="openLot"
          @toggle-select="toggleLotSelect"
        />
      </div>

      <div v-else class="empty-state">
        <h3>No auction lots{{ emptyStateSuffix }}</h3>
        <p>Import lots from NumisBids or CNG Auctions to start tracking auctions</p>
        <button class="btn btn-primary import-first-btn" @click="showImport = true">
          <Plus :size="16" /> Import Your First Lot
        </button>
        <SafeExternalLink href="https://www.numisbids.com/" class="btn btn-secondary auction-house-link">
          <ExternalLink :size="16" /> Visit NumisBids
        </SafeExternalLink>
        <SafeExternalLink href="https://auctions.cngcoins.com/" class="btn btn-secondary auction-house-link">
          <ExternalLink :size="16" /> Visit CNG Auctions
        </SafeExternalLink>
      </div>

      <ImportLotModal v-if="showImport" @close="showImport = false" @imported="handleImported" />

      <Teleport to="body">
        <AuctionLotDetailModal
          v-if="selectedLot"
          :lot="selectedLot"
          :price-alerts="alertsByLot[selectedLot.id] ?? []"
          :bid-reminders="remindersByLot[selectedLot.id] ?? []"
          @close="selectedLot = null"
          @updated="handleLotUpdated"
          @alerts-updated="fetchAlertState"
        />
      </Teleport>

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
import { computed, ref, watch } from 'vue'
import { getAuctionLots, getAuctionLotCounts, syncNumisBidsWatchlist, listCalendarEvents, bulkLinkAuctionLotEvent, listAlerts, listReminders } from '@/api/client'
import type { AuctionLot, BidReminder, PriceAlert } from '@/types'
import AuctionLotCard from '@/components/AuctionLotCard.vue'
import ImportLotModal from '@/components/ImportLotModal.vue'
import PullToRefresh from '@/components/PullToRefresh.vue'
import AuctionStatusFilter from '@/components/auction/AuctionStatusFilter.vue'
import AuctionLotDetailModal from '@/components/auction/AuctionLotDetailModal.vue'
import AuctionBulkActionBar from '@/components/auction/AuctionBulkActionBar.vue'
import { Plus, CirclePlus, RefreshCw, CheckSquare, ExternalLink } from 'lucide-vue-next'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import { usePwa } from '@/composables/usePwa'
import { useAuthStore } from '@/stores/auth'

const { isPwa } = usePwa()
const auth = useAuthStore()

const lots = ref<AuctionLot[]>([])
const statusCounts = ref<Record<string, number>>({})
const loading = ref(true)
const showImport = ref(false)
const selectedLot = ref<AuctionLot | null>(null)
const activeStatus = ref('bidding')
const activeSource = ref('')
const syncing = ref(false)
const syncMessage = ref('')
const calendarEvents = ref<Array<{ id: number; title: string; auctionHouse: string; startDate: string | null }>>([])
const priceAlerts = ref<PriceAlert[]>([])
const bidReminders = ref<BidReminder[]>([])

const selectMode = ref(false)
const selectedLotIds = ref(new Set<number>())
const sourceOptions = [
  { value: '', label: 'All' },
  { value: 'numisbids', label: 'NumisBids' },
  { value: 'cng', label: 'CNG' },
]
const alertsByLot = computed(() => groupByLot(priceAlerts.value))
const remindersByLot = computed(() => groupByLot(bidReminders.value))

function groupByLot<T extends { auctionLotId: number }>(items: T[]): Record<number, T[]> {
  return items.reduce<Record<number, T[]>>((acc, item) => {
    if (!acc[item.auctionLotId]) acc[item.auctionLotId] = []
    acc[item.auctionLotId]?.push(item)
    return acc
  }, {})
}

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

watch([activeStatus, activeSource], () => {
  selectedLotIds.value = new Set()
  fetchLots()
  fetchAllCounts()
})

async function fetchLots() {
  loading.value = true
  try {
    const params: Record<string, string> = { sort: 'updated_at', order: 'desc' }
    if (activeStatus.value) params.status = activeStatus.value
    if (activeSource.value) params.source = activeSource.value
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
    const params = activeSource.value ? { source: activeSource.value } : undefined
    const res = await getAuctionLotCounts(params)
    statusCounts.value = res.data?.counts ?? {}
  } catch { /* ignore */ }
}

async function handleRefresh() {
  await Promise.all([fetchLots(), fetchAllCounts(), fetchAlertState()])
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

async function fetchAlertState() {
  try {
    const [alertsRes, remindersRes] = await Promise.all([listAlerts(), listReminders()])
    priceAlerts.value = alertsRes.data?.alerts ?? []
    bidReminders.value = remindersRes.data?.reminders ?? []
  } catch {
    priceAlerts.value = []
    bidReminders.value = []
  }
}

async function syncWatchlist() {
  syncing.value = true
  syncMessage.value = ''
  try {
    const providers = configuredAuctionProviders()
    if (!providers.length) {
      syncMessage.value = 'Configure auction provider credentials in Settings before syncing'
      setTimeout(() => { syncMessage.value = '' }, 5000)
      return
    }
    const results = await Promise.allSettled(providers.map((source) => syncNumisBidsWatchlist(source)))
    const synced = results.reduce((total, result) => total + (result.status === 'fulfilled' ? result.value.data?.synced ?? 0 : 0), 0)
    const failed = results.filter((result) => result.status === 'rejected')
    const providerLabel = providers.length > 1 ? 'watchlists' : providerName(providers[0] ?? 'numisbids')
    syncMessage.value = failed.length
      ? `Synced ${synced} lot${synced !== 1 ? 's' : ''}; ${failed.length} provider${failed.length !== 1 ? 's' : ''} failed`
      : `Synced ${synced} lot${synced !== 1 ? 's' : ''} from ${providerLabel}`
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

function configuredAuctionProviders(): string[] {
  const providers: string[] = []
  if (auth.user?.numisBidsConfigured) providers.push('numisbids')
  if (auth.user?.cngConfigured) providers.push('cng')
  return providers
}

function providerName(source: string): string {
  return source === 'cng' ? 'CNG Auctions' : 'NumisBids'
}

const emptyStateSuffix = computed(() => {
  const parts: string[] = []
  if (activeStatus.value) parts.push(`status "${activeStatus.value}"`)
  if (activeSource.value) parts.push(providerName(activeSource.value))
  return parts.length ? ` matching ${parts.join(' and ')}` : ''
})

fetchLots()
fetchAllCounts()
fetchAlertState()
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

.auction-filter-toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: nowrap;
  margin-bottom: 1rem;
}

.source-filter {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  flex: 1 1 auto;
  min-width: 0;
}

/*
 * :deep() audit — child-component class override
 * .status-filter-menu is rendered inside a child component (CoinStatusFilter).
 * The auction toolbar controls its position in the filter row (margin-left auto
 * to right-align); the child owns the class name but the layout context is the
 * parent's responsibility.
 */
.auction-filter-toolbar :deep(.status-filter-menu) {
  flex: 0 0 auto;
  margin-left: auto;
}

.import-first-btn {
  margin-top: 0.75rem;
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

.auction-house-link {
  margin-top: 0.75rem;
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  text-decoration: none;
}
</style>
