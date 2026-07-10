<template>
  <div class="container pb-4 md:pb-6">
    <div v-if="loading" class="py-12 text-center text-text-secondary">
      Loading set details...
    </div>

    <div v-else-if="set" class="space-y-6">
      <div class="page-header items-start">
        <div class="flex min-w-0 flex-1 items-start gap-3 md:items-center">
          <span class="h-11 w-1 shrink-0 rounded-full shadow-[0_0_16px_var(--accent-gold-glow)]" :style="{ backgroundColor: set.color }" aria-hidden="true"></span>
          <div class="min-w-0">
            <h1>{{ set.name }}</h1>
            <p v-if="set.description" class="mt-0.5 truncate text-base text-text-secondary">{{ set.description }}</p>
          </div>
        </div>
        <div v-if="isPwa" class="pwa-actions">
          <button class="pwa-icon-btn" @click="router.push({ name: 'sets' })" title="Back to Sets">
            <ArrowLeft :size="22" />
          </button>
          <button v-if="canManageMembership" class="pwa-icon-btn" @click="openAddCoinModal" title="Add Coin">
            <CirclePlus :size="22" />
          </button>
          <button class="pwa-icon-btn" @click="showEditModal = true" title="Edit Set">
            <Pencil :size="22" />
          </button>
          <button class="pwa-icon-btn text-[var(--error-bg)]" @click="deleteSet" title="Delete Set">
            <Trash2 :size="22" />
          </button>
        </div>
        <div v-else class="header-actions">
          <button class="btn btn-ghost" @click="router.push({ name: 'sets' })">
            <ArrowLeft :size="16" /> Back
          </button>
          <button v-if="canManageMembership" class="btn btn-primary" @click="openAddCoinModal">
            <Plus :size="16" /> Add Coin
          </button>
          <button class="btn btn-secondary" @click="showEditModal = true">
            <Pencil :size="16" /> Edit
          </button>
          <button class="btn btn-danger" @click="deleteSet">
            <Trash2 :size="16" /> Delete
          </button>
        </div>
      </div>

      <SetCompletionChecklist
        v-if="completion"
        :completion="completion"
      />

      <section v-if="analytics" class="card p-6">
        <h2 class="mt-0">Analytics</h2>
        <div class="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(150px,1fr))]">
          <div class="flex flex-col gap-1.5">
            <span class="text-body text-text-secondary">ROI</span>
            <strong class="text-lg text-gold">{{ analytics.roiPercent == null ? 'N/A' : `${analytics.roiPercent.toFixed(1)}%` }}</strong>
          </div>
          <div class="flex flex-col gap-1.5">
            <span class="text-body text-text-secondary">Acquisition Rate</span>
            <strong class="text-lg text-gold">{{ analytics.acquisitionRatePerMonth == null ? 'N/A' : `${analytics.acquisitionRatePerMonth.toFixed(1)}/mo` }}</strong>
          </div>
        </div>
      </section>

      <SetTrendChart
        :snapshots="snapshots"
        :range="trendRange"
        @update:range="changeTrendRange"
      />

      <div class="-mt-4 mb-6">
        <button class="btn btn-secondary" @click="captureSnapshot">Capture Snapshot</button>
      </div>

      <SetComparePanel
        :sets="allSets"
        :results="compareResults"
        :loading="compareLoading"
        :error="compareError"
        @compare="compareSelectedSets"
      />

      <div class="space-y-4">
        <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
          <div>
            <p class="section-label">{{ canReorderCoins ? 'Manual sequence' : 'Set members' }}</p>
            <h2 class="m-0">Coins in Set</h2>
          </div>
          <p v-if="canReorderCoins && coins.length > 1" class="m-0 max-w-none text-left text-body text-text-secondary md:max-w-[24rem] md:text-right" :class="{ 'text-[var(--confidence-low)]': orderError }" aria-live="polite">
            <span v-if="savingOrder">Saving order...</span>
            <span v-else-if="orderError">{{ orderError }}</span>
            <span v-else>Drag rows or use the arrows to arrange this set.</span>
          </p>
        </div>
        <div v-if="coins.length === 0" class="card space-y-4 py-8 text-center">
          <p class="m-0 text-base text-text-secondary">No coins in this set yet</p>
          <button v-if="canManageMembership" class="btn btn-primary" @click="openAddCoinModal">Add Coins</button>
        </div>
        <div
          v-else
          class="space-y-4"
          :class="{ 'opacity-80': savingOrder }"
          aria-label="Coins in this set"
        >
          <div class="flex flex-col gap-4">
            <MuseumTray
              :coins="currentDrawerCoins"
              :felt-theme="feltColor"
              @coin-clicked="goToCoin"
            />
            <TrayControls
              v-if="totalDrawers > 1"
              :drawer-index="drawerIndex"
              :total-drawers="totalDrawers"
              :fixed="false"
              @prev="handlePrevDrawer"
              @next="handleNextDrawer"
            />
          </div>
          <div
            v-if="canManageMembership"
            class="mt-4 flex flex-col gap-2"
            aria-label="Manage set coin order and membership"
          >
            <div
              v-for="(coin, index) in coins"
              :key="coin.id"
              class="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 rounded-sm border border-border-subtle bg-card px-3 py-2.5 transition-all hover:border-border-accent md:grid-cols-[auto_minmax(0,1fr)_auto_auto]"
              :class="{
                'cursor-grab': canReorderCoins && !savingOrder,
                'opacity-[0.55] border-border-accent': draggingCoinId === coin.id,
                'border-border-accent shadow-[var(--shadow-glow)]': dragOverCoinId === coin.id,
              }"
              :draggable="canReorderCoins && !savingOrder"
              @dragstart="startDragging(coin.id, $event)"
              @dragover.prevent="trackDragOver(coin.id)"
              @dragleave="clearDragOver(coin.id)"
              @drop.prevent="dropCoin(coin.id)"
              @dragend="resetDragState"
            >
              <span class="inline-flex h-7 min-w-7 items-center justify-center rounded-full border border-border-accent text-sm font-semibold text-gold" :aria-label="`Position ${index + 1}`">{{ index + 1 }}</span>
              <button type="button" class="min-w-0 truncate bg-transparent p-0 text-left text-base font-semibold text-text-primary transition-colors hover:text-gold" @click="goToCoin(coin.id)">
                {{ coin.name }}
              </button>
              <span class="hidden whitespace-nowrap text-chip font-semibold text-gold md:inline">${{ coin.currentValue ?? 0 }}</span>
              <div class="flex flex-nowrap justify-end gap-1.5" aria-label="Set coin actions">
                <button
                  v-if="canReorderCoins"
                  type="button"
                  class="inline-flex h-8 w-8 items-center justify-center rounded-full border border-border-subtle bg-input text-text-secondary transition-all hover:border-border-accent hover:bg-[var(--accent-gold-glow)] hover:text-gold disabled:cursor-not-allowed disabled:opacity-30"
                  :disabled="index === 0 || savingOrder"
                  @click="moveCoinByButton(index, -1)"
                  title="Move earlier"
                  :aria-label="`Move ${coin.name} earlier`"
                >
                  <ChevronUp :size="16" />
                </button>
                <button
                  v-if="canReorderCoins"
                  type="button"
                  class="inline-flex h-8 w-8 items-center justify-center rounded-full border border-border-subtle bg-input text-text-secondary transition-all hover:border-border-accent hover:bg-[var(--accent-gold-glow)] hover:text-gold disabled:cursor-not-allowed disabled:opacity-30"
                  :disabled="index === coins.length - 1 || savingOrder"
                  @click="moveCoinByButton(index, 1)"
                  title="Move later"
                  :aria-label="`Move ${coin.name} later`"
                >
                  <ChevronDown :size="16" />
                </button>
                <button
                  v-if="canManageMembership"
                  type="button"
                  class="inline-flex h-8 w-8 items-center justify-center rounded-full border border-border-subtle bg-input text-text-secondary transition-all hover:border-border-accent hover:bg-[var(--accent-gold-glow)] hover:text-gold"
                  @click.stop="removeCoin(coin.id)"
                  title="Remove from set"
                  :aria-label="`Remove ${coin.name} from set`"
                >
                  <X :size="16" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="showAddCoinModal" class="fixed inset-0 z-[1000] flex items-center justify-center bg-black/60 p-4" @click.self="showAddCoinModal = false">
      <div class="card w-[90%] max-w-[500px] p-8">
        <h2 class="mt-0">Add Coin to Set</h2>
        <form @submit.prevent="addCoin">
          <div class="form-group">
            <label for="coinSearch" class="form-label">Search coins</label>
            <input
              id="coinSearch"
              v-model="coinSearch"
              type="search"
              class="form-input"
              placeholder="Search by name, ruler, denomination, or mint"
            />
          </div>
          <div class="form-group">
            <label for="coinToAdd" class="form-label">Coin</label>
            <select id="coinToAdd" v-model.number="coinIdToAdd" class="form-select" required>
              <option :value="null" disabled>Select a coin...</option>
              <option
                v-for="coin in filteredAvailableCoins"
                :key="coin.id"
                :value="coin.id"
              >
                {{ coin.name }}<template v-if="coin.ruler"> - {{ coin.ruler }}</template>
              </option>
            </select>
            <p v-if="availableCoins.length === 0" class="mt-1.5 text-chip text-text-secondary">All loaded coins are already in this set.</p>
            <p v-else-if="filteredAvailableCoins.length === 0" class="mt-1.5 text-chip text-text-secondary">No matching coins found.</p>
          </div>
          <div class="mt-6 flex justify-end gap-2">
            <button type="button" class="btn btn-secondary" @click="showAddCoinModal = false">Cancel</button>
            <button type="submit" class="btn btn-primary" :disabled="!coinIdToAdd">Add Coin</button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="showEditModal" class="fixed inset-0 z-[1000] flex items-center justify-center bg-black/60 p-4" @click.self="showEditModal = false">
      <div class="card w-[90%] max-w-[500px] p-8">
        <h2 class="mt-0">Edit Set</h2>
        <form @submit.prevent="updateSet">
          <div class="form-group">
            <label for="editName" class="form-label">Name</label>
            <input id="editName" v-model="editForm.name" type="text" class="form-input" required maxlength="80" />
          </div>
          <div class="form-group">
            <label for="editDescription" class="form-label">Description</label>
            <textarea id="editDescription" v-model="editForm.description" rows="3" class="form-input" maxlength="2000" />
          </div>
          <div class="form-group">
            <label for="editColor" class="form-label">Color</label>
            <input id="editColor" v-model="editForm.color" type="color" class="form-input h-11 cursor-pointer p-1" />
          </div>
          <div class="mt-6 flex justify-end gap-2">
            <button type="button" class="btn btn-secondary" @click="showEditModal = false">Cancel</button>
            <button type="submit" class="btn btn-primary">Update</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ArrowLeft, ChevronDown, ChevronUp, CirclePlus, Pencil, Plus, Trash2, X } from 'lucide-vue-next'
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
import MuseumTray from '@/components/tray/MuseumTray.vue'
import TrayControls from '@/components/tray/TrayControls.vue'
import { usePwa } from '@/composables/usePwa'
import { useTrayPreference } from '@/composables/useTrayPreference'
import { getDrawerCoins, getTotalDrawers, type TrayCoin } from '@/utils/trayLayout'

const router = useRouter()
const route = useRoute()
const { isPwa } = usePwa()
const { feltColor } = useTrayPreference()
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
const drawerIndex = ref(0)
const coinsPerDrawer = 12
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
const trayCoins = computed((): TrayCoin[] =>
  coins.value.map((coin) => ({
    id: coin.id,
    name: coin.name,
    diameterMm: coin.diameterMm,
    images: coin.images ?? [],
  })),
)
const currentDrawerCoins = computed(() => getDrawerCoins(trayCoins.value, drawerIndex.value, coinsPerDrawer))
const totalDrawers = computed(() => getTotalDrawers(trayCoins.value.length, coinsPerDrawer))

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

watch(totalDrawers, (drawers) => {
  if (drawers === 0) {
    drawerIndex.value = 0
    return
  }
  drawerIndex.value = Math.min(drawerIndex.value, drawers - 1)
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

function handlePrevDrawer() {
  drawerIndex.value = Math.max(0, drawerIndex.value - 1)
}

function handleNextDrawer() {
  drawerIndex.value = Math.min(totalDrawers.value - 1, drawerIndex.value + 1)
}

function goToCoin(coinId: number) {
  router.push({ name: 'coin-detail', params: { id: coinId } })
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
