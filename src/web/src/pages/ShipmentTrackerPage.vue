<template>
  <div class="container pb-6">
    <div class="page-header">
      <h1>Shipment Tracker</h1>
    </div>

    <section class="mb-6 rounded-md border border-border-subtle bg-card p-4 shadow-[var(--shadow-card)]">
      <p class="m-0 mb-3 text-body text-text-secondary">Select a coin to add or update shipment tracking.</p>
      <div class="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto] md:items-end">
        <label class="form-group mb-0 block">
          <span class="form-label">Find Coin</span>
          <input
            v-model="search"
            class="form-input"
            placeholder="Search by coin name or ruler"
            @keydown.enter.prevent="loadCoins"
          />
        </label>
        <button class="btn btn-secondary btn-sm" :disabled="loadingCoins" @click="loadCoins">
          {{ loadingCoins ? 'Searching...' : 'Search' }}
        </button>
      </div>

      <label class="form-group mt-3 mb-0 block">
        <span class="form-label">Coin</span>
        <select v-model.number="selectedCoinId" class="form-select">
          <option :value="0">Select a coin</option>
          <option v-for="coin in coins" :key="coin.id" :value="coin.id">{{ coinOptionLabel(coin) }}</option>
        </select>
      </label>

      <p v-if="loadError" class="mt-3 mb-0 text-body text-[var(--color-negative)]">{{ loadError }}</p>
    </section>

    <CoinShipmentSection v-if="selectedCoinId > 0" :coin-id="selectedCoinId" />

    <div v-else class="empty-state">
      <h3>Select a coin to manage shipment tracking</h3>
      <p>Use the search above, then choose a coin from the list.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { getCoins } from '@/api/client'
import CoinShipmentSection from '@/components/coin/CoinShipmentSection.vue'
import type { Coin } from '@/types'

const route = useRoute()
const search = ref('')
const coins = ref<Coin[]>([])
const loadingCoins = ref(false)
const loadError = ref('')
const selectedCoinId = ref(0)

onMounted(async () => {
  const initialCoinID = Number(route.query.coinId ?? 0)
  if (Number.isFinite(initialCoinID) && initialCoinID > 0) {
    selectedCoinId.value = initialCoinID
  }
  await loadCoins()
})

async function loadCoins() {
  loadingCoins.value = true
  loadError.value = ''
  try {
    const res = await getCoins({
      search: search.value.trim() || undefined,
      page: 1,
      limit: 200,
      sort: 'updated_at',
      order: 'desc',
    })
    coins.value = res.data.coins ?? []
  } catch {
    loadError.value = 'Unable to load coins right now.'
  } finally {
    loadingCoins.value = false
  }
}

function coinOptionLabel(coin: Coin): string {
  const labels: string[] = [coin.name]
  if (coin.ruler) labels.push(coin.ruler)
  if (coin.category) labels.push(coin.category)
  return labels.join(' · ')
}
</script>
