<template>
  <div class="container">
    <div class="page-header">
      <h1>Wishlist</h1>
      <!-- PWA: icon-only buttons inline with title -->
      <div v-if="isPwa" class="pwa-actions">
        <button
          class="pwa-icon-btn focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
          :disabled="checking"
          @click="handleCheckAvailability"
          title="Check Availability"
        >
          <span
            v-if="checking"
            class="inline-block h-[14px] w-[14px] animate-spin rounded-full border-2 border-border-subtle border-t-gold"
          ></span>
          <ShieldCheck v-else :size="22" />
        </button>
        <router-link
          v-if="store.coins.length"
          to="/wishlist/search-alerts"
          class="pwa-icon-btn focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
          title="Add Wish List Finder Agent"
          aria-label="Add Wish List Finder Agent"
        >
          <CalendarClock :size="22" />
        </router-link>
        <router-link
          to="/wishlist/availability-runs"
          class="pwa-icon-btn focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
          title="Run History"
          aria-label="Run History"
        >
          <History :size="22" />
        </router-link>
        <button
          type="button"
          class="pwa-icon-btn focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
          title="Add to wishlist"
          aria-label="Add to wishlist"
          @click="showAddDialog = true"
        >
          <CirclePlus :size="22" />
        </button>
      </div>
      <!-- Desktop: full text buttons -->
      <div v-else class="header-actions flex-wrap gap-3">
        <button
          class="btn btn-secondary focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
          :disabled="checking"
          @click="handleCheckAvailability"
        >
          <span
            v-if="checking"
            class="inline-block h-[14px] w-[14px] animate-spin rounded-full border-2 border-border-subtle border-t-gold"
          ></span>
          <ShieldCheck v-else :size="16" />
          {{ checking ? 'Checking...' : 'Check Availability' }}
        </button>
        <router-link
          v-if="store.coins.length"
          to="/wishlist/search-alerts"
          class="btn btn-secondary focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
          title="Search Alerts"
        >
          <CalendarClock :size="16" /> Search Alerts
        </router-link>
        <router-link
          to="/wishlist/availability-runs"
          class="btn btn-secondary focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
          title="Run History"
        >
          <History :size="16" /> Run History
        </router-link>
        <button
          type="button"
          class="btn btn-secondary focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
          @click="showAddDialog = true"
        >
          <CirclePlus :size="16" /> Add Coin
        </button>
      </div>
    </div>

    <div
      v-if="checkResult"
      class="mb-4 flex flex-wrap items-center gap-4 rounded-md border border-border-subtle bg-card px-4 py-3 text-body"
    >
      <span class="inline-flex rounded-full bg-[rgba(74,222,128,0.15)] px-2.5 py-1 text-chip font-semibold text-green-400">
        {{ checkResult.available }} available
      </span>
      <span class="inline-flex rounded-full bg-[rgba(248,113,113,0.15)] px-2.5 py-1 text-chip font-semibold text-red-400">
        {{ checkResult.unavailable }} unavailable
      </span>
      <span class="inline-flex rounded-full bg-[rgba(241,196,15,0.15)] px-2.5 py-1 text-chip font-semibold text-warning">
        {{ checkResult.unknown }} unknown
      </span>
      <span class="ml-auto text-text-muted">{{ checkResult.coinsChecked }} checked</span>
      <button
        class="px-1 text-xl leading-none text-text-muted transition hover:text-text-primary focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
        @click="checkResult = null"
      >
        &times;
      </button>
    </div>

    <div v-if="store.loading" class="loading-overlay">
      <div class="spinner"></div>
    </div>

    <div v-else-if="store.coins.length" class="coins-grid">
      <CoinCard
        v-for="coin in store.coins"
        :key="coin.id"
        :coin="coin"
        wishlist
        :active-reminder="remindersByCoinId.get(coin.id) ?? null"
        @purchase="openPurchaseModal"
        @dismiss-status="handleDismissStatus"
      />
    </div>

    <div v-else class="empty-state">
      <h3>Your wishlist is empty</h3>
      <p>Add coins to your wishlist to track what you're looking for</p>
      <div class="mt-3 flex flex-wrap justify-center gap-3">
        <button class="btn btn-primary focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" @click="showChat = true">
          <Bot :size="16" /> Search for Coins with AI
        </button>
        <router-link
          to="/wishlist/search-alerts"
          class="btn btn-secondary focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
          title="Add Wish List Finder Agent"
        >
          <CalendarClock :size="16" /> Add Wish List Finder Agent
        </router-link>
      </div>
    </div>

    <div v-if="store.coins.length && store.total > pageSize" class="mt-6 flex flex-wrap items-center justify-center gap-4 py-4">
      <button class="btn btn-secondary btn-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" :disabled="page <= 1" @click="page--">← Previous</button>
      <span class="text-body text-text-secondary">Page {{ page }} of {{ Math.ceil(store.total / pageSize) }}</span>
      <button class="btn btn-secondary btn-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" :disabled="page * pageSize >= store.total" @click="page++">Next →</button>
    </div>

    <PurchaseModal
      v-if="purchaseTarget"
      :coin="purchaseTarget"
      @close="purchaseTarget = null"
      @confirm="handlePurchaseConfirm"
    />

    <div
      v-if="showAddDialog"
      class="fixed inset-0 z-[1000] flex items-center justify-center bg-overlay p-4"
      @click.self="closeAddDialog"
      @keydown.esc="closeAddDialog"
    >
      <div
        ref="addDialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="wishlist-add-title"
        class="card max-h-[90vh] w-full max-w-2xl overflow-y-auto !p-6"
        tabindex="-1"
      >
        <div class="mb-4 flex items-center justify-between gap-3">
          <h2 id="wishlist-add-title" class="m-0 text-heading">Add to wishlist</h2>
          <button
            type="button"
            class="pwa-icon-btn"
            title="Close"
            aria-label="Close"
            @click="closeAddDialog"
          >
            <X :size="18" />
          </button>
        </div>

        <div v-if="addMethod === null" class="grid gap-3 sm:grid-cols-2">
          <button type="button" class="add-method" @click="addMethod = 'url'">
            <Link :size="24" />
            <span>
              <strong>Add from URL</strong>
              <small>Import and review a public dealer listing.</small>
            </span>
          </button>
          <router-link to="/lookup" class="add-method" @click="closeAddDialog">
            <ScanLine :size="24" />
            <span>
              <strong>Add from image analysis</strong>
              <small>Photograph or upload a coin for identification.</small>
            </span>
          </router-link>
        </div>

        <div v-else>
          <button type="button" class="btn btn-ghost btn-xs mb-4" @click="addMethod = null">
            <ChevronLeft :size="15" /> Add another way
          </button>
          <WishlistURLIntake embedded @created="handleURLCreated" />
        </div>
      </div>
    </div>

    <CoinSearchChat v-if="showChat" @close="showChat = false" @added="loadCoins" />
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch, onBeforeUnmount } from 'vue'
import { useCoinsStore } from '@/stores/coins'
import CoinCard from '@/components/CoinCard.vue'
import CoinSearchChat from '@/components/CoinSearchChat.vue'
import PurchaseModal from '@/components/PurchaseModal.vue'
import WishlistURLIntake from '@/components/wishlist/WishlistURLIntake.vue'
import { purchaseCoin, checkWishlistAvailability, updateListingStatus, listPurchaseReminders } from '@/api/client'
import type { Coin, AvailabilityRunSummary, PurchaseReminder } from '@/types'
import { CirclePlus, Bot, ShieldCheck, CalendarClock, History, ChevronLeft, Link, ScanLine, X } from 'lucide-vue-next'
import { usePwa } from '@/composables/usePwa'
import { useQuickAccess } from '@/composables/useQuickAccess'

const store = useCoinsStore()
const { isPwa } = usePwa()
const { refresh: refreshQuickAccess } = useQuickAccess()
const showChat = ref(false)
const showAddDialog = ref(false)
const addMethod = ref<'url' | null>(null)
const addDialog = ref<HTMLElement | null>(null)
const purchaseTarget = ref<Coin | null>(null)
const checking = ref(false)
const checkResult = ref<AvailabilityRunSummary | null>(null)
let dismissTimer: ReturnType<typeof setTimeout> | null = null
const page = ref(1)
const pageSize = 50

const remindersByCoinId = ref(new Map<number, PurchaseReminder>())

async function fetchReminderMap() {
  try {
    const res = await listPurchaseReminders()
    const map = new Map<number, PurchaseReminder>()
    for (const r of res.data.reminders ?? []) {
      map.set(r.coinId, r)
    }
    remindersByCoinId.value = map
  } catch {
    // non-critical: badge is best-effort
  }
}

function loadCoins() {
  store.fetchCoins({ wishlist: 'true', sort: 'updated_at', order: 'desc', page: page.value })
  fetchReminderMap()
}

function closeAddDialog() {
  showAddDialog.value = false
  addMethod.value = null
}

function handleURLCreated() {
  loadCoins()
  closeAddDialog()
}

watch(page, loadCoins)
watch(showAddDialog, async (open) => {
  if (!open) return
  await nextTick()
  addDialog.value?.focus()
})

function openPurchaseModal(coin: Coin) {
  purchaseTarget.value = coin
}

async function handleCheckAvailability() {
  checking.value = true
  checkResult.value = null
  if (dismissTimer) { clearTimeout(dismissTimer); dismissTimer = null }
  try {
    const res = await checkWishlistAvailability()
    checkResult.value = res.data
    loadCoins()
    dismissTimer = setTimeout(() => { checkResult.value = null }, 10000)
  } catch {
    // silently fail
  } finally {
    checking.value = false
  }
}

async function handleDismissStatus(coinId: number) {
  try {
    await updateListingStatus(coinId, '')
    loadCoins()
  } catch {
    // silently fail
  }
}

async function handlePurchaseConfirm(data: { purchasePrice?: number; purchaseDate?: string; purchaseLocation?: string }) {
  if (!purchaseTarget.value) return
  try {
    await purchaseCoin(purchaseTarget.value.id, data)
    await refreshQuickAccess()
    purchaseTarget.value = null
    loadCoins()
  } catch {
    purchaseTarget.value = null
  }
}

loadCoins()

onBeforeUnmount(() => {
  if (dismissTimer) clearTimeout(dismissTimer)
})
</script>

<style scoped>
.add-method {
  display: flex;
  min-height: 96px;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 1rem;
  text-align: left;
  color: var(--text-primary);
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  transition: var(--transition-fast);
}

.add-method:hover {
  background: var(--bg-card-hover);
  border-color: var(--border-accent);
}

.add-method svg {
  flex-shrink: 0;
  color: var(--accent-gold);
}

.add-method span {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.add-method strong {
  font-size: 0.9rem;
}

.add-method small {
  color: var(--text-secondary);
  font-size: 0.8rem;
  line-height: 1.5;
}
</style>
