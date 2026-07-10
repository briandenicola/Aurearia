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
          to="/lookup"
          class="pwa-icon-btn focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
          title="Identify Coin"
        >
          <CirclePlus :size="22" />
        </router-link>
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
          to="/lookup"
          class="btn btn-secondary focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
        >
          <CirclePlus :size="16" /> Identify Coin
        </router-link>
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

    <CoinSearchChat v-if="showChat" @close="showChat = false" @added="loadCoins" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import { useCoinsStore } from '@/stores/coins'
import CoinCard from '@/components/CoinCard.vue'
import CoinSearchChat from '@/components/CoinSearchChat.vue'
import PurchaseModal from '@/components/PurchaseModal.vue'
import { purchaseCoin, checkWishlistAvailability, updateListingStatus } from '@/api/client'
import type { Coin, AvailabilityRunSummary } from '@/types'
import { CirclePlus, Bot, ShieldCheck, CalendarClock } from 'lucide-vue-next'
import { usePwa } from '@/composables/usePwa'

const store = useCoinsStore()
const { isPwa } = usePwa()
const showChat = ref(false)
const purchaseTarget = ref<Coin | null>(null)
const checking = ref(false)
const checkResult = ref<AvailabilityRunSummary | null>(null)
let dismissTimer: ReturnType<typeof setTimeout> | null = null
const page = ref(1)
const pageSize = 50

function loadCoins() {
  store.fetchCoins({ wishlist: 'true', sort: 'updated_at', order: 'desc', page: page.value })
}

watch(page, loadCoins)

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
