<template>
  <div class="container">
    <div class="page-header">
      <h1>Wishlist</h1>
      <!-- PWA: icon-only buttons inline with title -->
      <div v-if="isPwa" class="pwa-actions">
        <button class="pwa-icon-btn" :disabled="checking" @click="handleCheckAvailability" title="Check Availability">
          <span v-if="checking" class="spinner-sm"></span>
          <ShieldCheck v-else :size="22" />
        </button>
        <router-link to="/add?wishlist=true" class="pwa-icon-btn" title="Add Coin">
          <CirclePlus :size="22" />
        </router-link>
      </div>
      <!-- Desktop: full text buttons -->
      <div v-else class="header-actions">
        <button
          class="btn btn-secondary"
          :disabled="checking"
          @click="handleCheckAvailability"
        >
          <span v-if="checking" class="spinner-sm"></span>
          <ShieldCheck v-else :size="16" />
          {{ checking ? 'Checking...' : 'Check Availability' }}
        </button>
        <router-link to="/add?wishlist=true" class="btn btn-secondary"><CirclePlus :size="16" /> Add Coin</router-link>
      </div>
    </div>

    <div v-if="checkResult" class="availability-banner">
      <span class="banner-count banner-available">{{ checkResult.available }} available</span>
      <span class="banner-count banner-unavailable">{{ checkResult.unavailable }} unavailable</span>
      <span class="banner-count banner-unknown">{{ checkResult.unknown }} unknown</span>
      <span class="banner-total">{{ checkResult.coinsChecked }} checked</span>
      <button class="banner-dismiss" @click="checkResult = null">&times;</button>
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
      <button class="btn btn-primary" @click="showChat = true" style="margin-top: 0.75rem">
        <Bot :size="16" /> Search for Coins with AI
      </button>
    </div>

    <div v-if="store.coins.length && store.total > pageSize" class="pagination">
      <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="page--">← Previous</button>
      <span class="page-info">Page {{ page }} of {{ Math.ceil(store.total / pageSize) }}</span>
      <button class="btn btn-secondary btn-sm" :disabled="page * pageSize >= store.total" @click="page++">Next →</button>
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
import { CirclePlus, Bot, ShieldCheck } from 'lucide-vue-next'
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

<style scoped>
.header-actions {
  display: flex;
  gap: 0.75rem;
  align-items: center;
  flex-wrap: wrap;
}

.spinner-sm {
  display: inline-block;
  width: 14px;
  height: 14px;
  border: 2px solid var(--border-subtle);
  border-top-color: var(--accent-gold);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.availability-banner {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 1rem;
  margin-bottom: 1rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  font-size: 0.85rem;
  flex-wrap: wrap;
}

.banner-count {
  font-weight: 600;
  padding: 0.2rem 0.6rem;
  border-radius: var(--radius-full);
  font-size: 0.8rem;
}

.banner-available {
  background: rgba(46, 204, 113, 0.15);
  color: #2ecc71;
}

.banner-unavailable {
  background: rgba(231, 76, 60, 0.15);
  color: #e74c3c;
}

.banner-unknown {
  background: rgba(241, 196, 15, 0.15);
  color: #f1c40f;
}

.banner-total {
  color: var(--text-muted);
  margin-left: auto;
}

.banner-dismiss {
  background: none;
  border: none;
  color: var(--text-muted);
  font-size: 1.2rem;
  cursor: pointer;
  padding: 0 0.25rem;
  line-height: 1;
}

.banner-dismiss:hover {
  color: var(--text-primary);
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 1rem;
  margin-top: 1.5rem;
  padding: 1rem 0;
}

.page-info {
  font-size: 0.85rem;
  color: var(--text-secondary);
}
</style>
