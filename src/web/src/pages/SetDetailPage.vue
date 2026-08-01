<template>
  <div class="container pb-4 md:pb-6">
    <div v-if="loading" class="py-12 text-center text-text-secondary">
      Loading set details...
    </div>

    <div v-else-if="set" class="space-y-6">
      <div class="page-header relative items-start">
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
          <button
            class="pwa-icon-btn"
            :class="menuOpen ? 'border-border-accent bg-gold-glow text-gold' : ''"
            @click="menuOpen = !menuOpen"
            title="Set actions"
            aria-label="Set actions"
          >
            <Menu :size="22" />
          </button>
        </div>
        <div v-else class="header-actions">
          <button class="btn btn-ghost" @click="router.push({ name: 'sets' })">
            <ArrowLeft :size="16" /> Back
          </button>
          <button
            class="btn btn-secondary"
            :class="menuOpen ? 'border-border-accent bg-gold-glow text-gold' : ''"
            @click="menuOpen = !menuOpen"
            aria-haspopup="menu"
            :aria-expanded="menuOpen"
          >
            <Menu :size="16" /> Actions
          </button>
        </div>

        <div v-if="menuOpen" class="absolute right-0 top-[calc(100%+0.5rem)] z-20 w-full max-w-[260px] rounded-md border border-border-subtle bg-card p-2 shadow-[0_10px_26px_rgba(0,0,0,0.45)]" role="menu">
          <button
            class="w-full rounded-sm px-3 py-2 text-left text-body text-text-secondary transition-all hover:bg-card-hover hover:text-text-primary"
            role="menuitem"
            @click="openSetInfoPage"
          >
            <span class="inline-flex items-center gap-2">
              <Info :size="15" />
              Analytics & Value Trend
            </span>
          </button>
          <button
            v-if="canManageMembership"
            class="w-full rounded-sm px-3 py-2 text-left text-body text-text-secondary transition-all hover:bg-card-hover hover:text-text-primary"
            role="menuitem"
            @click="openAddCoinModal"
          >
            <span class="inline-flex items-center gap-2">
              <CirclePlus :size="15" />
              Add Coin
            </span>
          </button>
          <button
            class="w-full rounded-sm px-3 py-2 text-left text-body text-text-secondary transition-all hover:bg-card-hover hover:text-text-primary"
            role="menuitem"
            @click="openEditModal"
          >
            <span class="inline-flex items-center gap-2">
              <Pencil :size="15" />
              Edit Set
            </span>
          </button>
          <button
            class="w-full rounded-sm px-3 py-2 text-left text-body text-[var(--error-bg)] transition-all hover:bg-card-hover"
            role="menuitem"
            @click="deleteSet"
          >
            <span class="inline-flex items-center gap-2">
              <Trash2 :size="15" />
              Delete Set
            </span>
          </button>
        </div>
        <button v-if="menuOpen" class="fixed inset-0 z-10 bg-transparent" aria-label="Close menu" @click="menuOpen = false"></button>
      </div>

      <SetCompletionChecklist
        v-if="completion"
        :completion="completion"
      />

      <div class="space-y-4">
        <p v-if="canReorderCoins && coins.length > 1" class="m-0 max-w-none text-left text-body text-text-secondary md:text-right" :class="{ 'text-[var(--confidence-low)]': orderError }" aria-live="polite">
          <span v-if="savingOrder">Saving order...</span>
          <span v-else-if="orderError">{{ orderError }}</span>
          <span v-else>Drag rows or use the arrows to arrange this set.</span>
        </p>
        <p v-if="normalizedSetType === 'agentic'" class="m-0 text-body text-text-secondary">
          Click any tray slot to assign or replace the coin for that target.
        </p>
        <div v-if="displayTrayCoins.length === 0" class="card space-y-4 py-8 text-center">
          <p class="m-0 text-base text-text-secondary">{{ emptySetMessage }}</p>
          <button v-if="canManageMembership" class="btn btn-primary" @click="openAddCoinModal">Add Coins</button>
        </div>
        <div
          v-else
          class="space-y-4"
          :class="{ 'opacity-80': savingOrder }"
          aria-label="Coins in this set"
        >
          <div class="flex flex-col gap-4">
            <div class="flex justify-center md:justify-center">
              <label class="inline-flex w-full items-center justify-between gap-3 rounded-full border border-border-subtle bg-[rgba(255,255,255,0.04)] px-3 py-2 text-[0.8rem] font-semibold uppercase tracking-[0.04em] text-text-secondary md:w-auto md:justify-start">
                <span>Coin size</span>
                <input
                  id="set-tray-size-slider"
                  v-model.number="traySizeScale"
                  class="w-[140px] accent-[var(--accent-gold)]"
                  type="range"
                  min="0.75"
                  max="1.4"
                  step="0.05"
                />
                <span class="min-w-[3.2rem] text-right text-text-primary">{{ traySizeScale.toFixed(2) }}x</span>
              </label>
            </div>
            <MuseumTray
              :coins="currentDrawerCoins"
              :felt-theme="feltColor"
              :size-scale="traySizeScale"
              @coin-clicked="handleTrayCoinClick"
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

    <div v-if="showAssignSlotModal" class="fixed inset-0 z-[1000] flex items-center justify-center bg-black/60 p-4" @click.self="showAssignSlotModal = false">
      <div class="card w-[90%] max-w-[500px] p-8">
        <h2 class="mt-0">Assign Coin to Slot</h2>
        <p class="section-label mb-3">{{ slotTargetLabel }}</p>
        <form @submit.prevent="assignCoinToSlot">
          <div class="form-group">
            <label for="slotCoinSearch" class="form-label">Search coins</label>
            <input
              id="slotCoinSearch"
              v-model="slotCoinSearch"
              type="search"
              class="form-input"
              placeholder="Search by name, ruler, denomination, or mint"
            />
          </div>
          <div class="form-group">
            <label for="slotCoinToAssign" class="form-label">Coin</label>
            <select id="slotCoinToAssign" v-model.number="slotCoinIdToAssign" class="form-select" required>
              <option :value="null" disabled>Select a coin...</option>
              <option
                v-for="coin in filteredAssignableCoins"
                :key="coin.id"
                :value="coin.id"
              >
                {{ coin.name }}<template v-if="coin.ruler"> - {{ coin.ruler }}</template>
              </option>
            </select>
            <p v-if="filteredAssignableCoins.length === 0" class="mt-1.5 text-chip text-text-secondary">No matching coins found.</p>
            <p v-if="slotAssignmentError" class="mt-1.5 text-chip text-[var(--error-bg)]">{{ slotAssignmentError }}</p>
          </div>
          <div class="mt-6 flex justify-end gap-2">
            <button type="button" class="btn btn-ghost" :disabled="slotAssignmentSaving" @click="showAssignSlotModal = false">Cancel</button>
            <button v-if="currentSlotCoinId" type="button" class="btn btn-danger" :disabled="slotAssignmentSaving" @click="clearSlotAssignment">Clear Slot</button>
            <button type="submit" class="btn btn-primary" :disabled="!slotCoinIdToAssign || slotAssignmentSaving">
              {{ currentSlotCoinId ? 'Replace Coin' : 'Assign Coin' }}
            </button>
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
import { ArrowLeft, ChevronDown, ChevronUp, CirclePlus, Info, Menu, Pencil, Trash2, X } from 'lucide-vue-next'
import {
  addCoinToSet,
  deleteSet as deleteSetApi,
  getCoins,
  getCoinsInSet,
  getSet,
  getSetCompletion,
  reorderSetCoins,
  removeCoinFromSet,
  updateSet as updateSetApi,
} from '@/api/client'
import { normalizeCoinSetType } from '@/types'
import type { CoinSetCompletion, CoinSetDetail, Coin } from '@/types'
import SetCompletionChecklist from '@/components/sets/SetCompletionChecklist.vue'
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
const drawerIndex = ref(0)
const coinsPerDrawer = 12
const traySizeScale = ref(1)
const savingOrder = ref(false)
const orderError = ref<string | null>(null)
const draggingCoinId = ref<number | null>(null)
const dragOverCoinId = ref<number | null>(null)
const showAddCoinModal = ref(false)
const showAssignSlotModal = ref(false)
const showEditModal = ref(false)
const coinIdToAdd = ref<number | null>(null)
const coinSearch = ref('')
const slotCoinSearch = ref('')
const slotCoinIdToAssign = ref<number | null>(null)
const slotTargetId = ref<number | null>(null)
const slotTargetLabel = ref('')
const currentSlotCoinId = ref<number | null>(null)
const slotAssignmentSaving = ref(false)
const slotAssignmentError = ref<string | null>(null)
const menuOpen = ref(false)
const editForm = ref({
  name: '',
  description: '',
  color: '#6b7280',
})

const setId = Number(route.params.id)

const canManageMembership = computed(() => {
  if (!set.value) return false
  const normalizedType = normalizeCoinSetType(set.value.setType)
  return normalizedType !== 'smart' && normalizedType !== 'agentic'
})
const canReorderCoins = computed(() => canManageMembership.value && coins.value.length > 1)
const normalizedSetType = computed(() => set.value ? normalizeCoinSetType(set.value.setType) : null)
type AgenticTrayCoin = TrayCoin & { targetId: number; assignedCoinId?: number | null }
const trayCoins = computed((): TrayCoin[] =>
  coins.value.map((coin) => ({
    id: coin.id,
    name: coin.name,
    diameterMm: coin.diameterMm,
    images: coin.images ?? [],
    purchaseDate: coin.purchaseDate,
    wishlistPlaceholder: normalizedSetType.value === 'goal' && coin.isWishlist,
  })),
)
const agenticTrayCoins = computed((): AgenticTrayCoin[] => {
  if (normalizedSetType.value !== 'agentic') return []
  const targetMatches = completion.value?.targetMatches
  if (targetMatches && targetMatches.length > 0) {
    return targetMatches.map(({ target, coin }) => {
      if (coin) {
        return {
          id: coin.id,
          targetId: target.id,
          assignedCoinId: coin.id,
          name: coin.name,
          diameterMm: coin.diameterMm,
          images: coin.images ?? [],
          purchaseDate: coin.purchaseDate,
        }
      }
      return {
        id: target.id,
        targetId: target.id,
        assignedCoinId: null,
        name: target.label,
        diameterMm: null,
        images: [],
        placeholder: true,
        placeholderLabel: target.year != null ? String(target.year) : target.label,
      }
    })
  }
  const targets = completion.value?.targets ?? completion.value?.missingTargets ?? []
  return targets.map((target) => ({
    id: target.id,
    targetId: target.id,
    assignedCoinId: null,
    name: target.label,
    diameterMm: null,
    images: [],
    placeholder: true,
    placeholderLabel: target.year != null ? String(target.year) : target.label,
  }))
})
const displayTrayCoins = computed(() => normalizedSetType.value === 'agentic' ? agenticTrayCoins.value : trayCoins.value)
const currentDrawerCoins = computed(() => getDrawerCoins(displayTrayCoins.value, drawerIndex.value, coinsPerDrawer))
const totalDrawers = computed(() => getTotalDrawers(displayTrayCoins.value.length, coinsPerDrawer))
const emptySetMessage = computed(() => {
  if (normalizedSetType.value === 'agentic' && set.value?.agenticStatus === 'generating') {
    return 'Aurearia is generating this agentic roster. You will receive a notification when the tray slots are ready.'
  }
  if (normalizedSetType.value === 'agentic') return 'No agentic tray slots have been generated yet.'
  return 'No coins in this set yet'
})

const availableCoins = computed(() => {
  const existingIds = new Set(coins.value.map((coin) => coin.id))
  return allCoins.value.filter((coin) => !existingIds.has(coin.id))
})

const assignableCoins = computed(() => {
  if (normalizedSetType.value !== 'agentic') return availableCoins.value

  const assignedToOtherSlots = new Set<number>()
  for (const match of completion.value?.targetMatches ?? []) {
    if (match.target.id !== slotTargetId.value && match.coin?.id != null) {
      assignedToOtherSlots.add(match.coin.id)
    }
  }
  return allCoins.value.filter((coin) => !assignedToOtherSlots.has(coin.id))
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

const filteredAssignableCoins = computed(() => {
  const term = slotCoinSearch.value.trim().toLowerCase()
  if (!term) return assignableCoins.value
  return assignableCoins.value.filter((coin) => [
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

watch(traySizeScale, (value) => {
  const normalizedValue = Math.min(1.4, Math.max(0.75, Number(value) || 1))
  traySizeScale.value = normalizedValue
  localStorage.setItem('tray:sizeScale', normalizedValue.toString())
})

onMounted(async () => {
  const storedScale = localStorage.getItem('tray:sizeScale')
  if (storedScale !== null) {
    const parsedScale = Number(storedScale)
    if (Number.isFinite(parsedScale)) {
      traySizeScale.value = Math.min(1.4, Math.max(0.75, parsedScale))
    }
  }
  await loadSetDetails()
})

async function loadSetDetails() {
  loading.value = true
  try {
    const [setRes, coinsRes, allCoinsRes] = await Promise.all([
      getSet(setId),
      getCoinsInSet(setId),
      getCoins({ wishlist: 'false', sold: 'false', limit: 100, sort: 'name', order: 'asc' }),
    ])
    set.value = setRes.data
    coins.value = coinsRes.data.coins
    orderError.value = null
    allCoins.value = allCoinsRes.data.coins
    const normalizedSetType = normalizeCoinSetType(set.value.setType)
    if (normalizedSetType === 'goal' || normalizedSetType === 'agentic') {
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

function openSetInfoPage() {
  menuOpen.value = false
  router.push({ name: 'set-insights', params: { id: setId } })
}

function openEditModal() {
  menuOpen.value = false
  showEditModal.value = true
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
  menuOpen.value = false
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

function openAssignSlotModal(targetId: number, targetLabel: string, assignedCoinId: number | null) {
  slotTargetId.value = targetId
  slotTargetLabel.value = targetLabel
  currentSlotCoinId.value = assignedCoinId
  slotCoinIdToAssign.value = assignedCoinId
  slotCoinSearch.value = ''
  slotAssignmentError.value = null
  showAssignSlotModal.value = true
}

async function assignCoinToSlot() {
  if (!slotCoinIdToAssign.value || !slotTargetId.value) return
  slotAssignmentSaving.value = true
  slotAssignmentError.value = null
  try {
    await addCoinToSet(setId, { coinId: slotCoinIdToAssign.value, targetId: slotTargetId.value })
    showAssignSlotModal.value = false
    await loadSetDetails()
  } catch (error) {
    console.error('Failed to assign coin to slot:', error)
    slotAssignmentError.value = getErrorMessage(error, 'Unable to assign this coin to the slot.')
  } finally {
    slotAssignmentSaving.value = false
  }
}

async function clearSlotAssignment() {
  if (!currentSlotCoinId.value) return
  slotAssignmentSaving.value = true
  slotAssignmentError.value = null
  try {
    await removeCoinFromSet(setId, currentSlotCoinId.value)
    showAssignSlotModal.value = false
    await loadSetDetails()
  } catch (error) {
    console.error('Failed to clear slot assignment:', error)
    slotAssignmentError.value = getErrorMessage(error, 'Unable to clear this slot.')
  } finally {
    slotAssignmentSaving.value = false
  }
}

async function deleteSet() {
  menuOpen.value = false
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

function handleTrayCoinClick(coinOrTargetId: number) {
  if (normalizedSetType.value !== 'agentic') {
    goToCoin(coinOrTargetId)
    return
  }
  const slot = currentDrawerCoins.value.find((coin) => coin.id === coinOrTargetId) as AgenticTrayCoin | undefined
  if (!slot || slot.targetId == null) return
  openAssignSlotModal(slot.targetId, slot.name, slot.assignedCoinId ?? null)
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
