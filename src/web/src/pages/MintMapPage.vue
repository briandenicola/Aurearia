<template>
  <div class="container mint-map-page">
    <header class="page-header mint-map-header">
      <div>
        <p class="section-label">Collection Geography</p>
        <h1>Mint Map</h1>
        <p class="page-intro">Explore where the currently loaded active collection was struck.</p>
      </div>
      <router-link class="btn btn-secondary" to="/">Back to Collection</router-link>
    </header>

    <div v-if="store.loading" class="loading-card card" role="status">
      <div class="spinner"></div>
      <span>Loading collection mints...</span>
    </div>

    <div v-else-if="errorMessage" class="state-card card" role="alert">
      <h2>Mint map unavailable</h2>
      <p>{{ errorMessage }}</p>
      <button class="btn btn-primary" type="button" @click="loadDefaultCollection">Try Again</button>
    </div>

    <div v-else-if="!store.coins.length" class="state-card card">
      <h2>No active coins to map</h2>
      <p>Add coins with mint values, or return after your collection has loaded.</p>
      <router-link class="btn btn-primary" to="/add">Add Coin</router-link>
    </div>

    <template v-else>
      <section class="summary-grid" aria-label="Mint map summary">
        <div class="summary-card card">
          <span class="section-label">Matched Mints</span>
          <strong>{{ aggregation.matched.length }}</strong>
        </div>
        <div class="summary-card card">
          <span class="section-label">Mapped Coins</span>
          <strong>{{ mappedCoinCount }}</strong>
        </div>
        <div class="summary-card card">
          <span class="section-label">Unmatched Names</span>
          <strong>{{ aggregation.unmatched.length }}</strong>
        </div>
        <div class="summary-card card">
          <span class="section-label">Unknown Mint Coins</span>
          <strong>{{ aggregation.unknown.length }}</strong>
        </div>
      </section>

      <MintMapSvg
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
import { useCoinsStore } from '@/stores/coins'
import MintMapSvg from '@/components/map/MintMapSvg.vue'
import MintCoinDrawer from '@/components/map/MintCoinDrawer.vue'
import UnattributedMintBucket from '@/components/map/UnattributedMintBucket.vue'
import { groupCoinsByMint, type MintGroup } from '@/utils/mintMap'

const store = useCoinsStore()
const selectedMintId = ref<string | null>(null)
const unattributedExpanded = ref(false)
const errorMessage = ref('')

const aggregation = computed(() => groupCoinsByMint(store.coins))
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

async function loadDefaultCollection() {
  errorMessage.value = ''
  try {
    await store.fetchCoins({ wishlist: 'false', sold: 'false' })
  } catch {
    errorMessage.value = 'The active collection could not be loaded. Check your connection and try again.'
  }
}

onMounted(() => {
  if (!store.coins.length) {
    loadDefaultCollection()
  }
})
</script>

<style scoped>
.mint-map-page {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.mint-map-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.page-intro {
  margin: 0.35rem 0 0;
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}

.summary-card {
  padding: 1rem;
}

.summary-card strong {
  display: block;
  margin-top: 0.35rem;
  color: var(--accent-gold);
  font-family: 'Cinzel', serif;
  font-size: 1.5rem;
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

@media (max-width: 768px) {
  .mint-map-header {
    align-items: stretch;
    flex-direction: column;
  }

  .summary-grid {
    grid-template-columns: 1fr;
  }
}
</style>
