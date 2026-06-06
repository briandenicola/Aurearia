<template>
  <div class="set-detail-page">
    <div v-if="loading" class="loading-state">
      Loading set details...
    </div>

    <div v-else-if="set" class="set-detail-container">
      <div class="set-header">
        <button class="btn" @click="$router.back()">Back</button>
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
          <button v-if="canManageMembership" class="btn btn-secondary" @click="openAddCoinModal">Add Coin</button>
          <button class="btn" @click="showEditModal = true">Edit</button>
          <button class="btn btn-danger" @click="deleteSet">Delete</button>
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
        @compare="compareSelectedSets"
      />

      <div class="coins-section">
        <h2>Coins in Set</h2>
        <div v-if="coins.length === 0" class="empty-coins">
          <p>No coins in this set yet</p>
          <button v-if="canManageMembership" class="btn btn-primary" @click="openAddCoinModal">Add Coins</button>
        </div>
        <div v-else class="coins-grid">
          <div v-for="coin in coins" :key="coin.id" class="coin-card" @click="goToCoin(coin.id)">
            <div class="coin-image">
              <img
                v-if="coin.images?.[0]"
                :src="`/uploads/${coin.images[0].filePath}`"
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
            <button type="button" class="btn" @click="showEditModal = false">Cancel</button>
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
import { FolderOpen, Coins } from 'lucide-vue-next'
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
const trendRange = ref('1y')
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
  const res = await compareSets([setId, ...setIds], trendRange.value)
  compareResults.value = res.data.sets
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

function goToCoin(coinId: number) {
  router.push({ name: 'coin-detail', params: { id: coinId } })
}

function formatNumber(value: number): string {
  return value.toFixed(2)
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
  margin-bottom: 2rem;
}

.set-header-content {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin: 1rem 0;
}

.set-icon-large {
  width: 64px;
  height: 64px;
  border-radius:var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--bg-primary);
}

.set-title-area h1 {
  margin: 0;
}

.set-description {
  color: var(--text-secondary);
  margin: 0.5rem 0 0;
}

.set-actions {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
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
  border-radius:var(--radius-sm);
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

.coins-section h2 {
  margin-bottom: 1rem;
}

.analytics-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius:var(--radius-md);
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

.coin-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius:var(--radius-sm);
  padding: 1rem;
  cursor: pointer;
  position: relative;
  transition: all var(--transition-fast);
}

.coin-card:hover {
  border-color: var(--accent-gold);
  box-shadow: var(--shadow-card);
}

.coin-image {
  width: 100%;
  aspect-ratio: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-input);
  border-radius:var(--radius-sm);
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
  border-radius:var(--radius-md);
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
  border-radius:var(--radius-sm);
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
</style>
