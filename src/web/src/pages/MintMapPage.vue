<template>
  <div class="container mint-map-page">
    <header class="page-header mint-map-header">
      <div>
        <p class="section-label">Collection Insights</p>
        <h1>Map of Coins</h1>
      </div>
      <router-link class="back-button" to="/stats" aria-label="Back to Stats">
        <ArrowLeft :size="20" />
      </router-link>
    </header>

    <div v-if="loading" class="loading-card card" role="status">
      <div class="spinner"></div>
      <span>Loading collection mints...</span>
    </div>

    <div v-else-if="errorMessage" class="state-card card" role="alert">
      <h2>Mint map unavailable</h2>
      <p>{{ errorMessage }}</p>
      <button class="btn btn-primary" type="button" @click="loadMapData">Try Again</button>
    </div>

    <div v-else-if="!mintLocations.length" class="state-card card" role="alert">
      <h2>No mint locations configured</h2>
      <p>Ask an administrator to add global mint locations before using the map.</p>
    </div>

    <div v-else-if="!collectionCoins.length" class="state-card card">
      <h2>No active coins to map</h2>
      <p>Add coins with mint values, or return after your collection has loaded.</p>
      <router-link class="btn btn-primary" to="/add">Add Coin</router-link>
    </div>

    <template v-else>
      <section class="summary-row" aria-label="Mint map summary">
        <span class="summary-label">Mapped Coins:</span>
        <strong class="mapped-count">{{ mappedCoinCount }}</strong>
      </section>

      <MintMapLeaflet
        :groups="aggregation.matched"
        :selected-mint-id="selectedMintId"
        @select-mint="selectMint"
      />

      <UnattributedMintBucket
        v-if="unattributedCount > 0"
        v-model:expanded="unattributedExpanded"
        :unknown="aggregation.unknown"
        :unmatched="aggregation.unmatched"
      />

      <MintCoinDrawer
        :open="selectedGroup !== null"
        :group="selectedGroup"
        @close="selectedMintId = null"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ArrowLeft } from 'lucide-vue-next'
import { getCoins, getMintLocations, type MintLocationsResponse } from '@/api/client'
import MintMapLeaflet from '@/components/map/MintMapLeaflet.vue'
import MintCoinDrawer from '@/components/map/MintCoinDrawer.vue'
import UnattributedMintBucket from '@/components/map/UnattributedMintBucket.vue'
import { groupCoinsByMint, type MintGroup } from '@/utils/mintMap'
import type { Coin, MintLocation } from '@/types'

const MAP_PAGE_LIMIT = 100

const selectedMintId = ref<number | null>(null)
const unattributedExpanded = ref(false)
const errorMessage = ref('')
const mintLocations = ref<MintLocation[]>([])
const collectionCoins = ref<Coin[]>([])
const mintLocationsLoading = ref(false)
const coinsLoading = ref(false)

const loading = computed(() => mintLocationsLoading.value || coinsLoading.value)
const aggregation = computed(() => groupCoinsByMint(collectionCoins.value, mintLocations.value))
const selectedGroup = computed(() =>
  aggregation.value.matched.find((group) => group.mint.id === selectedMintId.value) ?? null,
)
const mappedCoinCount = computed(() =>
  aggregation.value.matched.reduce((total, group) => total + group.count, 0),
)
const unattributedCount = computed(() =>
  aggregation.value.unknown.length + aggregation.value.unmatched.reduce((total, group) => total + group.coins.length, 0),
)

function selectMint(group: MintGroup) {
  selectedMintId.value = group.mint.id
}

function unwrapMintLocations(data: MintLocationsResponse): MintLocation[] {
  return Array.isArray(data) ? data : data.mintLocations ?? []
}

async function fetchAllActiveCollectionCoins(): Promise<Coin[]> {
  const allCoins: Coin[] = []
  let page = 1

  while (true) {
    const res = await getCoins({
      wishlist: 'false',
      sold: 'false',
      page,
      limit: MAP_PAGE_LIMIT,
    })
    const pageCoins = res.data.coins ?? []
    allCoins.push(...pageCoins)
    const total = res.data.total ?? allCoins.length

    if (!pageCoins.length || allCoins.length >= total) break
    page += 1
  }

  return allCoins
}

async function loadMapData() {
  errorMessage.value = ''
  selectedMintId.value = null
  mintLocationsLoading.value = true
  try {
    const locationsRes = await getMintLocations()
    mintLocations.value = unwrapMintLocations(locationsRes.data)
  } catch {
    mintLocations.value = []
    errorMessage.value = 'Mint locations could not be loaded. Check your connection and try again.'
  } finally {
    mintLocationsLoading.value = false
  }

  if (errorMessage.value) return

  try {
    coinsLoading.value = true
    collectionCoins.value = await fetchAllActiveCollectionCoins()
  } catch {
    collectionCoins.value = []
    errorMessage.value = 'The active collection could not be loaded. Check your connection and try again.'
  } finally {
    coinsLoading.value = false
  }
}

onMounted(() => {
  loadMapData()
})
</script>

<style scoped>
.mint-map-page {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.mint-map-header {
  flex-direction: row;
  align-items: center;
}

.back-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.4rem;
  color: var(--text-secondary);
  background: transparent;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  text-decoration: none;
  transition: var(--transition-fast);
  flex-shrink: 0;
}

.back-button:hover {
  color: var(--accent-gold);
  border-color: var(--border-accent);
  background: var(--accent-gold-glow);
}

.summary-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.6rem 1rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
}

.summary-label {
  color: var(--text-secondary);
  font-size: 0.9rem;
  font-weight: 500;
}

.mapped-count {
  color: var(--accent-gold);
  font-family: 'Cinzel', serif;
  font-size: 1.2rem;
  font-weight: 600;
}

.loading-card,
.state-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 1.5rem;
}

.loading-card {
  align-items: center;
  color: var(--text-secondary);
}

.state-card p {
  margin: 0;
  color: var(--text-secondary);
}
</style>
