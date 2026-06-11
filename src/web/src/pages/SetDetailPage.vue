<template>
  <div class="set-detail-page">
    <div v-if="loading" class="loading-state">
      Loading set details...
    </div>

    <div v-else-if="set" class="set-detail-container">
      <div class="set-header">
        <button class="btn btn-ghost btn-sm" @click="router.push({ name: 'sets' })">
          <ArrowLeft :size="16" />
          Back to Sets
        </button>
        <div class="set-header-content">
          <div class="set-icon-large" :style="{ backgroundColor: set.color }">
            <FolderOpen :size="32" />
          </div>
          <div class="set-title-area">
            <h1>{{ set.name }}</h1>
            <p v-if="set.description" class="set-description">{{ set.description }}</p>
          </div>
        </div>
        <div class="set-actions">
          <button v-if="canManageMembership" class="btn btn-primary btn-sm" @click="openAddCoinModal">
            <Plus :size="16" />
            Add Coin
          </button>
          <button class="btn btn-secondary btn-sm" @click="showEditModal = true">
            <Pencil :size="16" />
            Edit
          </button>
          <button class="btn btn-danger btn-sm" @click="deleteSet">
            <Trash2 :size="16" />
            Delete
          </button>
        </div>
      </div>

      <div class="set-summary">
        <div class="summary-card">
          <span class="summary-label">Coins</span>
          <span class="summary-value">{{ set.coinCount }}</span>
        </div>
        <div class="summary-card">
          <span class="summary-label">Total Value</span>
          <span class="summary-value">${{ formatNumber(set.totalValue) }}</span>
        </div>
        <div class="summary-card">
          <span class="summary-label">Total Invested</span>
          <span class="summary-value">${{ formatNumber(set.totalInvested) }}</span>
        </div>
        <div v-if="set.avgValuePerCoin != null" class="summary-card">
          <span class="summary-label">Avg Value/Coin</span>
          <span class="summary-value">${{ formatNumber(set.avgValuePerCoin) }}</span>
        </div>
      </div>

      <SetCompletionChecklist
        v-if="completion"
        :completion="completion"
      />

      <section v-if="analytics" class="analytics-card">
        <h2>Analytics</h2>
        <div class="analytics-grid">
          <div>
            <span class="summary-label">ROI</span>
            <strong>{{ analytics.roiPercent == null ? 'N/A' : `${analytics.roiPercent.toFixed(1)}%` }}</strong>
          </div>
          <div>
            <span class="summary-label">Acquisition Rate</span>
            <strong>{{ analytics.acquisitionRatePerMonth == null ? 'N/A' : `${analytics.acquisitionRatePerMonth.toFixed(1)}/mo` }}</strong>
          </div>
        </div>
      </section>

      <SetTrendChart
        :snapshots="snapshots"
        :range="trendRange"
        @update:range="changeTrendRange"
      />

      <div class="trend-actions">
        <button class="btn btn-secondary" @click="captureSnapshot">Capture Snapshot</button>
      </div>

      <SetComparePanel
        :sets="allSets"
        :results="compareResults"
        :loading="compareLoading"
        :error="compareError"
        @compare="compareSelectedSets"
      />

      <div class="coins-section">
        <div class="coins-heading">
          <div>
            <p class="section-label">{{ canReorderCoins ? 'Manual sequence' : 'Set members' }}</p>
            <h2>Coins in Set</h2>
          </div>
          <p v-if="canReorderCoins && coins.length > 1" class="order-status" :class="{ error: orderError }" aria-live="polite">
            <span v-if="savingOrder">Saving order...</span>
            <span v-else-if="orderError">{{ orderError }}</span>
            <span v-else>Drag cards or use the arrows to arrange this set.</span>
          </p>
        </div>
        <div v-if="coins.length === 0" class="empty-coins">
          <p>No coins in this set yet</p>
          <button v-if="canManageMembership" class="btn btn-primary" @click="openAddCoinModal">Add Coins</button>
        </div>
        <div
          v-else
          class="coins-grid"
          :class="{ 'is-reorderable': canReorderCoins, 'is-saving-order': savingOrder }"
          aria-label="Coins in this set"
        >
          <div
            v-for="(coin, index) in coins"
            :key="coin.id"
            class="coin-card"
            :class="{ dragging: draggingCoinId === coin.id, 'drag-over': dragOverCoinId === coin.id }"
            :draggable="canReorderCoins && !savingOrder"
            @click="goToCoin(coin.id)"
            @dragstart="startDragging(coin.id, $event)"
            @dragover.prevent="trackDragOver(coin.id)"
            @dragleave="clearDragOver(coin.id)"
            @drop.prevent="dropCoin(coin.id)"
            @dragend="resetDragState"
          >
            <div v-if="canReorderCoins" class="order-controls" @click.stop>
              <span class="order-rank" :aria-label="`Position ${index + 1}`">{{ index + 1 }}</span>
              <div class="order-buttons" aria-label="Reorder coin">
                <button
                  type="button"
                  class="btn btn-ghost btn-xs"
                  :disabled="index === 0 || savingOrder"
                  @click="moveCoinByButton(index, -1)"
                  title="Move earlier"
                  :aria-label="`Move ${coin.name} earlier`"
                >
                  Up
                </button>
                <button
                  type="button"
                  class="btn btn-ghost btn-xs"
                  :disabled="index === coins.length - 1 || savingOrder"
                  @click="moveCoinByButton(index, 1)"
                  title="Move later"
                  :aria-label="`Move ${coin.name} later`"
                >
                  Down
                </button>
              </div>
            </div>
            <div class="coin-image">
              <img
                v-if="coin.images?.[0]?.filePath"
                :src="`/uploads/${coin.images?.[0]?.filePath ?? ''}`"
                :alt="coin.name"
              />
              <Coins v-else :size="48" />
            </div>
            <div class="coin-info">
              <h3>{{ coin.name }}</h3>
              <p>${{ coin.currentValue || 0 }}</p>
            </div>
            <button
              v-if="canManageMembership"
              class="remove-coin-btn btn btn-ghost btn-xs"
              @click.stop="removeCoin(coin.id)"
              title="Remove from set"
            >
              Remove
            </button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="showAddCoinModal" class="modal-overlay" @click.self="showAddCoinModal = false">
      <div class="modal-content">
        <h2>Add Coin to Set</h2>
        <form @submit.prevent="addCoin">
          <div class="form-group">
            <label for="coinSearch">Search coins</label>
            <input
              id="coinSearch"
              v-model="coinSearch"
              type="search"
              placeholder="Search by name, ruler, denomination, or mint"
            />
          </div>
          <div class="form-group">
            <label for="coinToAdd">Coin</label>
            <select id="coinToAdd" v-model.number="coinIdToAdd" required>
              <option :value="null" disabled>Select a coin...</option>
              <option
                v-for="coin in filteredAvailableCoins"
                :key="coin.id"
                :value="coin.id"
              >
                {{ coin.name }}<template v-if="coin.ruler"> - {{ coin.ruler }}</template>
              </option>
            </select>
            <p v-if="availableCoins.length === 0" class="form-hint">All loaded coins are already in this set.</p>
            <p v-else-if="filteredAvailableCoins.length === 0" class="form-hint">No matching coins found.</p>
          </div>
          <div class="form-actions">
            <button type="button" class="btn btn-secondary" @click="showAddCoinModal = false">Cancel</button>
            <button type="submit" class="btn btn-primary" :disabled="!coinIdToAdd">Add Coin</button>
          </div>
        </form>
      </div>
    </div>

    <!-- Edit Modal -->
    <div v-if="showEditModal" class="modal-overlay" @click.self="showEditModal = false">
      <div class="modal-content">
        <h2>Edit Set</h2>
        <form @submit.prevent="updateSet">
          <div class="form-group">
            <label for="editName">Name</label>
            <input id="editName" v-model="editForm.name" type="text" required maxlength="80" />
          </div>
          <div class="form-group">
            <label for="editDescription">Description</label>
            <textarea id="editDescription" v-model="editForm.description" rows="3" maxlength="2000" />
          </div>
          <div class="form-group">
            <label for="editColor">Color</label>
            <input id="editColor" v-model="editForm.color" type="color" />
          </div>
          <div class="form-actions">
            <button type="button" class="btn btn-secondary" @click="showEditModal = false">Cancel</button>
            <button type="submit" class="btn btn-primary">Update</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ArrowLeft, FolderOpen, Coins, Pencil, Plus, Trash2 } from 'lucide-vue-next'
import {
  addCoinToSet,
  compareSets,
  createSetSnapshot,
  deleteSet as deleteSetApi,
  getCoins,
  getCoinsInSet,
  getSet,
  getSetAnalytics,
  getSetCompletion,
  getSets,
  getSetTrends,
  reorderSetCoins,
  removeCoinFromSet,
  updateSet as updateSetApi,
} from '@/api/client'
import type { CoinSetAnalytics, CoinSetComparison, CoinSetCompletion, CoinSetDetail, CoinSetSnapshot, CoinSetSummary, Coin } from '@/types'
import SetCompletionChecklist from '@/components/sets/SetCompletionChecklist.vue'
import SetTrendChart from '@/components/sets/SetTrendChart.vue'
import SetComparePanel from '@/components/sets/SetComparePanel.vue'

const router = useRouter()
const route = useRoute()
const loading = ref(true)
const set = ref<CoinSetDetail | null>(null)
const coins = ref<Coin[]>([])
const allCoins = ref<Coin[]>([])
const completion = ref<CoinSetCompletion | null>(null)
const snapshots = ref<CoinSetSnapshot[]>([])
const analytics = ref<CoinSetAnalytics | null>(null)
const allSets = ref<CoinSetSummary[]>([])
const compareResults = ref<CoinSetComparison[]>([])
const compareLoading = ref(false)
const compareError = ref<string | null>(null)
const trendRange = ref('1y')
const savingOrder = ref(false)
const orderError = ref<string | null>(null)
const draggingCoinId = ref<number | null>(null)
const dragOverCoinId = ref<number | null>(null)
const showAddCoinModal = ref(false)
const showEditModal = ref(false)
const coinIdToAdd = ref<number | null>(null)
const coinSearch = ref('')
const editForm = ref({
  name: '',
  description: '',
  color: '#6b7280',
})

const setId = Number(route.params.id)

const canManageMembership = computed(() => set.value?.setType !== 'smart')
const canReorderCoins = computed(() => canManageMembership.value && coins.value.length > 1)

const availableCoins = computed(() => {
  const existingIds = new Set(coins.value.map((coin) => coin.id))
  return allCoins.value.filter((coin) => !existingIds.has(coin.id))
})

const filteredAvailableCoins = computed(() => {
  const term = coinSearch.value.trim().toLowerCase()
  if (!term) return availableCoins.value
  return availableCoins.value.filter((coin) => [
    coin.name,
    coin.ruler,
    coin.denomination,
    coin.mint,
  ].some((field) => field?.toLowerCase().includes(term)))
})

onMounted(async () => {
  await loadSetDetails()
})

async function loadSetDetails() {
  loading.value = true
  try {
    const [setRes, coinsRes, trendsRes, analyticsRes, setsRes, allCoinsRes] = await Promise.all([
      getSet(setId),
      getCoinsInSet(setId),
      getSetTrends(setId, trendRange.value),
      getSetAnalytics(setId),
      getSets(),
      getCoins({ wishlist: 'false', sold: 'false', limit: 100, sort: 'name', order: 'asc' }),
    ])
    set.value = setRes.data
    coins.value = coinsRes.data.coins
    orderError.value = null
    allCoins.value = allCoinsRes.data.coins
    snapshots.value = trendsRes.data.snapshots
    analytics.value = analyticsRes.data
    allSets.value = setsRes.data.sets.filter((candidate) => candidate.id !== setId)
    if (set.value.setType === 'defined' || set.value.setType === 'goal') {
      const completionRes = await getSetCompletion(setId)
      completion.value = completionRes.data
    } else {
      completion.value = null
    }
    editForm.value = {
      name: set.value.name,
      description: set.value.description || '',
      color: set.value.color,
    }

  } catch (error) {
    console.error('Failed to load set:', error)
  } finally {
    loading.value = false
  }
}

async function changeTrendRange(range: string) {
  trendRange.value = range
  compareResults.value = []
  compareError.value = null
  const res = await getSetTrends(setId, trendRange.value)
  snapshots.value = res.data.snapshots
}

async function captureSnapshot() {
  try {
    await createSetSnapshot(setId)
    await changeTrendRange(trendRange.value)
    const analyticsRes = await getSetAnalytics(setId)
    analytics.value = analyticsRes.data
  } catch (error) {
    console.error('Failed to capture snapshot:', error)
    alert('Failed to capture snapshot')
  }
}

async function compareSelectedSets(setIds: number[]) {
  compareLoading.value = true
  compareError.value = null
  try {
    const uniqueSetIds = Array.from(new Set([setId, ...setIds]))
    if (uniqueSetIds.length < 2) {
      compareResults.value = []
      compareError.value = 'Choose at least one other set to compare.'
      return
    }
    const res = await compareSets(uniqueSetIds, trendRange.value)
    compareResults.value = res.data.sets
    if (compareResults.value.length === 0) {
      compareError.value = 'No comparison data is available for the selected sets.'
    }
  } catch (error) {
    console.error('Failed to compare sets:', error)
    compareResults.value = []
    compareError.value = getErrorMessage(error, 'Unable to compare these sets. Please try again.')
  } finally {
    compareLoading.value = false
  }
}

async function updateSet() {
  try {
    await updateSetApi(setId, editForm.value)
    showEditModal.value = false
    await loadSetDetails()
  } catch (error) {
    console.error('Failed to update set:', error)
    alert('Failed to update set')
  }
}

function openAddCoinModal() {
  coinIdToAdd.value = null
  coinSearch.value = ''
  showAddCoinModal.value = true
}

async function addCoin() {
  if (!coinIdToAdd.value) return
  try {
    await addCoinToSet(setId, { coinId: coinIdToAdd.value })
    coinIdToAdd.value = null
    showAddCoinModal.value = false
    await loadSetDetails()
  } catch (error) {
    console.error('Failed to add coin:', error)
    alert('Failed to add coin')
  }
}

async function deleteSet() {
  if (!confirm('Are you sure you want to delete this set?')) return
  try {
    await deleteSetApi(setId)
    router.push({ name: 'sets' })
  } catch (error) {
    console.error('Failed to delete set:', error)
    alert('Failed to delete set')
  }
}

async function removeCoin(coinId: number) {
  if (!confirm('Remove this coin from the set?')) return
  try {
    await removeCoinFromSet(setId, coinId)
    await loadSetDetails()
  } catch (error) {
    console.error('Failed to remove coin:', error)
    alert('Failed to remove coin')
  }
}

function startDragging(coinId: number, event: DragEvent) {
  if (!canReorderCoins.value || savingOrder.value) return
  draggingCoinId.value = coinId
  orderError.value = null
  event.dataTransfer?.setData('text/plain', String(coinId))
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
}

function trackDragOver(coinId: number) {
  if (!canReorderCoins.value || draggingCoinId.value === null || draggingCoinId.value === coinId) return
  dragOverCoinId.value = coinId
}

function clearDragOver(coinId: number) {
  if (dragOverCoinId.value === coinId) {
    dragOverCoinId.value = null
  }
}

async function dropCoin(targetCoinId: number) {
  if (!canReorderCoins.value || draggingCoinId.value === null || draggingCoinId.value === targetCoinId) {
    resetDragState()
    return
  }
  await moveCoin(draggingCoinId.value, targetCoinId, 'before')
}

async function moveCoinByButton(index: number, direction: -1 | 1) {
  const targetIndex = index + direction
  const coinToMove = coins.value[index]
  const targetCoin = coins.value[targetIndex]
  if (!coinToMove || !targetCoin || savingOrder.value) return
  await moveCoin(coinToMove.id, targetCoin.id, direction === 1 ? 'after' : 'before')
}

async function moveCoin(sourceCoinId: number, targetCoinId: number, placement: 'before' | 'after') {
  const fromIndex = coins.value.findIndex((coin) => coin.id === sourceCoinId)
  const toIndex = coins.value.findIndex((coin) => coin.id === targetCoinId)
  if (fromIndex === -1 || toIndex === -1 || fromIndex === toIndex) {
    resetDragState()
    return
  }

  const previousCoins = [...coins.value]
  const nextCoins = [...coins.value]
  const [movedCoin] = nextCoins.splice(fromIndex, 1)
  if (!movedCoin) {
    resetDragState()
    return
  }
  const targetIndexAfterRemoval = nextCoins.findIndex((coin) => coin.id === targetCoinId)
  if (targetIndexAfterRemoval === -1) {
    resetDragState()
    return
  }
  nextCoins.splice(placement === 'after' ? targetIndexAfterRemoval + 1 : targetIndexAfterRemoval, 0, movedCoin)
  coins.value = nextCoins
  resetDragState()
  await persistCoinOrder(previousCoins)
}

async function persistCoinOrder(previousCoins: Coin[]) {
  savingOrder.value = true
  orderError.value = null
  try {
    await reorderSetCoins(setId, { coinIds: coins.value.map((coin) => coin.id) })
  } catch (error) {
    console.error('Failed to save coin order:', error)
    coins.value = previousCoins
    orderError.value = getErrorMessage(error, 'Unable to save this order. Please try again.')
  } finally {
    savingOrder.value = false
  }
}

function resetDragState() {
  draggingCoinId.value = null
  dragOverCoinId.value = null
}

function goToCoin(coinId: number) {
  router.push({ name: 'coin-detail', params: { id: coinId } })
}

function formatNumber(value: number): string {
  return value.toFixed(2)
}

function getErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error !== null && 'response' in error) {
    const response = (error as { response?: { data?: { error?: unknown } } }).response
    if (typeof response?.data?.error === 'string') {
      return response.data.error
    }
  }
  if (error instanceof Error && error.message) {
    return error.message
  }
  return fallback
}
</script>

<style scoped>
.set-detail-page {
  padding: 1.5rem;
  max-width: 1200px;
  margin: 0 auto;
}

.loading-state {
  text-align: center;
  padding: 3rem 1rem;
}

.set-header {
  position: sticky;
  top: 0;
  z-index: 20;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 0;
  margin-bottom: 1.5rem;
  background: var(--bg-primary);
  border-bottom: 1px solid var(--border-subtle);
}

.set-header-content {
  display: flex;
  align-items: center;
  gap: 1rem;
  min-width: 0;
}

.set-icon-large {
  width: 3.5rem;
  height: 3.5rem;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--bg-primary);
  flex-shrink: 0;
}

.set-title-area h1 {
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.set-description {
  color: var(--text-secondary);
  margin: 0.25rem 0 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.set-actions {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.set-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.summary-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.summary-label {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.summary-value {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--accent-gold);
}

.coins-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
}

.coins-heading h2 {
  margin: 0;
}

.order-status {
  max-width: 24rem;
  margin: 0;
  color: var(--text-secondary);
  font-size: 0.85rem;
  text-align: right;
}

.order-status.error {
  color: var(--confidence-low);
}

.analytics-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1.5rem;
  margin-bottom: 1.5rem;
}

.analytics-card h2 {
  margin-top: 0;
}

.analytics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 1rem;
}

.analytics-grid div {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.analytics-grid strong {
  color: var(--accent-gold);
  font-size: 1.2rem;
}

.trend-actions {
  margin-top: -1rem;
  margin-bottom: 1.5rem;
}

.empty-coins {
  text-align: center;
  padding: 2rem;
}

.coins-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 1rem;
}

.coins-grid.is-saving-order {
  opacity: 0.8;
}

.coin-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 1rem;
  cursor: pointer;
  position: relative;
  transition: all var(--transition-fast);
}

.coin-card:hover {
  border-color: var(--accent-gold);
  box-shadow: var(--shadow-card);
}

.coin-card[draggable="true"] {
  cursor: grab;
}

.coin-card.dragging {
  opacity: 0.55;
  border-color: var(--accent-gold);
}

.coin-card.drag-over {
  border-color: var(--accent-gold);
  box-shadow: var(--shadow-glow);
}

.order-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.order-rank {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.75rem;
  height: 1.75rem;
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-full);
  color: var(--accent-gold);
  font-size: 0.75rem;
  font-weight: 600;
}

.order-buttons {
  display: flex;
  gap: 0.35rem;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.coin-image {
  width: 100%;
  aspect-ratio: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-input);
  border-radius: var(--radius-sm);
  margin-bottom: 0.5rem;
  overflow: hidden;
}

.coin-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.coin-info h3 {
  font-size: 0.85rem;
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.coin-info p {
  margin: 0.25rem 0 0;
  font-weight: 600;
}

.remove-coin-btn {
  position: absolute;
  top: 0.5rem;
  right: 0.5rem;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--modal-backdrop, rgba(0, 0, 0, 0.6));
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  padding: 2rem;
  border-radius: var(--radius-md);
  max-width: 500px;
  width: 90%;
  box-shadow: var(--shadow-card);
}

.modal-content h2 {
  margin-top: 0;
}

.form-group {
  margin-bottom: 1rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
}

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-input);
  color: var(--text-primary);
  font-family: inherit;
}

.form-hint {
  margin: 0.35rem 0 0;
  color: var(--text-secondary);
  font-size: 0.8rem;
}

.form-actions {
  display: flex;
  gap: 0.5rem;
  justify-content: flex-end;
  margin-top: 1.5rem;
}

@media (max-width: 768px) {
  .set-detail-page {
    padding: 1rem;
  }

  .set-header {
    grid-template-columns: 1fr;
    align-items: stretch;
  }

  .set-header-content {
    order: -1;
  }

  .set-actions {
    justify-content: flex-start;
  }

  .coins-heading {
    flex-direction: column;
  }

  .order-status {
    max-width: none;
    text-align: left;
  }

  .order-controls {
    align-items: flex-start;
  }
}
</style>
