<template>
  <PullToRefresh :on-refresh="loadCoins">
  <div class="container">
    <div class="page-header">
      <h1><Clock :size="24" /> Collection Timeline</h1>
      <div class="header-controls">
        <select v-model="filterType" class="form-input form-select">
          <option value="all">All Coins</option>
          <option value="collection">Collection Only</option>
          <option value="sold">Sold Only</option>
        </select>
      </div>
    </div>

    <div v-if="loading" class="loading-state">
      <div class="spinner" />
      <p>Loading timeline...</p>
    </div>

    <div v-else-if="timelineGroups.length === 0" class="empty-state">
      <Clock :size="48" />
      <h3>No timeline data</h3>
      <p>Add purchase dates to your coins to see them on the timeline.</p>
    </div>

    <div v-else class="timeline-container">
      <!-- Summary bar -->
      <div class="timeline-summary card">
        <div class="summary-item">
          <span class="summary-value">{{ totalCoins }}</span>
          <span class="summary-label">Coins</span>
        </div>
        <div class="summary-item">
          <span class="summary-value">{{ yearSpan }}</span>
          <span class="summary-label">Year Span</span>
        </div>
        <div class="summary-item">
          <span class="summary-value">${{ totalInvested.toLocaleString() }}</span>
          <span class="summary-label">Invested</span>
        </div>
        <div class="summary-item">
          <span class="summary-value">${{ totalValue.toLocaleString() }}</span>
          <span class="summary-label">Current Value</span>
        </div>
      </div>

      <!-- Timeline -->
      <div class="timeline">
        <div v-for="group in timelineGroups" :key="group.label" class="timeline-group">
          <div class="timeline-marker">
            <div class="marker-dot" />
            <div class="marker-label">{{ group.label }}</div>
            <div class="marker-count">{{ group.coins.length }} {{ group.coins.length === 1 ? 'coin' : 'coins' }}</div>
          </div>
          <div class="timeline-cards">
            <router-link
              v-for="coin in group.coins"
              :key="coin.id"
              :to="`/coin/${coin.id}`"
              class="timeline-card card"
            >
              <img
                v-if="getPrimaryImage(coin)"
                :src="`/uploads/${getPrimaryImage(coin)}`"
                :alt="coin.name"
                class="card-image"
              />
              <div v-else class="card-image card-placeholder">
                <ImageIcon :size="24" />
              </div>
              <div class="card-body">
                <span class="card-name">{{ coin.name }}</span>
                <span class="card-meta">
                  <span class="card-category" :style="{ color: categoryColor(coin.category) }">{{ coin.category }}</span>
                  <span v-if="coin.ruler" class="card-ruler">{{ coin.ruler }}</span>
                </span>
                <span v-if="coin.purchaseDate" class="card-date">{{ formatDate(coin.purchaseDate) }}</span>
                <div class="card-values">
                  <span v-if="coin.purchasePrice" class="card-price">
                    ${{ coin.purchasePrice.toLocaleString() }}
                  </span>
                  <span v-if="coin.isSold" class="card-sold-badge">Sold</span>
                  <span v-if="coin.grade" class="card-grade">{{ coin.grade }}</span>
                </div>
              </div>
            </router-link>
          </div>
        </div>
      </div>
    </div>
  </div>
  </PullToRefresh>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Clock, Image as ImageIcon } from 'lucide-vue-next'
import { getCoins } from '@/api/client'
import type { Coin } from '@/types'
import { CATEGORY_COLORS } from '@/types'
import PullToRefresh from '@/components/PullToRefresh.vue'

const loading = ref(true)
const allCoins = ref<Coin[]>([])
const filterType = ref<'all' | 'collection' | 'sold'>('all')

function categoryColor(cat: string): string {
  return (CATEGORY_COLORS as Record<string, string>)[cat] || '#888'
}

function getPrimaryImage(coin: Coin): string | null {
  if (!coin.images || coin.images.length === 0) return null
  const primary = coin.images.find(i => i.isPrimary)
  const img = primary ?? coin.images[0]
  return img ? img.filePath : null
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr)
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

function formatMonthYear(dateStr: string): string {
  const d = new Date(dateStr)
  return d.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })
}

function monthYearKey(dateStr: string): string {
  const d = new Date(dateStr)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

const filteredCoins = computed(() => {
  let coins = allCoins.value.filter(c => !c.isWishlist && c.purchaseDate)
  if (filterType.value === 'collection') {
    coins = coins.filter(c => !c.isSold)
  } else if (filterType.value === 'sold') {
    coins = coins.filter(c => c.isSold)
  }
  return coins.sort((a, b) => {
    const da = new Date(b.purchaseDate!).getTime()
    const db = new Date(a.purchaseDate!).getTime()
    return da - db
  })
})

const timelineGroups = computed(() => {
  const groups: { label: string; key: string; coins: Coin[] }[] = []
  const map = new Map<string, Coin[]>()

  for (const coin of filteredCoins.value) {
    const key = monthYearKey(coin.purchaseDate!)
    if (!map.has(key)) map.set(key, [])
    map.get(key)!.push(coin)
  }

  for (const [key, coins] of map) {
    groups.push({
      key,
      label: formatMonthYear(coins[0]?.purchaseDate ?? ''),
      coins,
    })
  }

  return groups
})

const totalCoins = computed(() => filteredCoins.value.length)

const yearSpan = computed(() => {
  if (filteredCoins.value.length === 0) return '0'
  const dates = filteredCoins.value.map(c => new Date(c.purchaseDate!).getFullYear())
  const min = Math.min(...dates)
  const max = Math.max(...dates)
  return min === max ? String(min) : `${min}–${max}`
})

const totalInvested = computed(() =>
  filteredCoins.value.reduce((sum, c) => sum + (c.purchasePrice || 0), 0)
)

const totalValue = computed(() =>
  filteredCoins.value.reduce((sum, c) => sum + (c.currentValue || c.purchasePrice || 0), 0)
)

async function loadCoins() {
  loading.value = true
  try {
    // Fetch all non-wishlist coins (collection + sold)
    const [collectionRes, soldRes] = await Promise.all([
      getCoins({ limit: 9999, sort: 'purchaseDate', order: 'desc' }),
      getCoins({ limit: 9999, sold: 'true', sort: 'purchaseDate', order: 'desc' }),
    ])
    const coinMap = new Map<number, Coin>()
    for (const c of collectionRes.data.coins) coinMap.set(c.id, c)
    for (const c of soldRes.data.coins) coinMap.set(c.id, c)
    allCoins.value = Array.from(coinMap.values())
  } catch {
    allCoins.value = []
  } finally {
    loading.value = false
  }
}

onMounted(loadCoins)
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.page-header h1 {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1.4rem;
  margin: 0;
}

.header-controls {
  display: flex;
  gap: 0.5rem;
}

.form-select {
  min-width: 150px;
  padding: 0.4rem 0.75rem;
  font-size: 0.85rem;
}

/* Summary */
.timeline-summary {
  display: flex;
  justify-content: space-around;
  padding: 1rem;
  margin-bottom: 2rem;
  flex-wrap: wrap;
  gap: 1rem;
}

.summary-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.2rem;
}

.summary-value {
  font-size: 1.2rem;
  font-weight: 700;
  color: var(--accent-gold);
  font-family: 'Cinzel', serif;
}

.summary-label {
  font-size: 0.75rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* Timeline */
.timeline {
  position: relative;
  padding-left: 2rem;
  max-width: 100%;
  overflow-x: hidden;
}

.timeline::before {
  content: '';
  position: absolute;
  left: 0.55rem;
  top: 0;
  bottom: 0;
  width: 2px;
  background: var(--border-subtle);
}

.timeline-group {
  position: relative;
  margin-bottom: 2rem;
}

.timeline-group:last-child {
  margin-bottom: 0;
}

.timeline-marker {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
  position: relative;
}

.marker-dot {
  position: absolute;
  left: -1.55rem;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--accent-gold);
  border: 2px solid var(--bg-primary);
  box-shadow: 0 0 0 2px var(--accent-gold-dim);
  z-index: 1;
}

.marker-label {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
  font-family: 'Cinzel', serif;
}

.marker-count {
  font-size: 0.75rem;
  color: var(--text-muted);
  background: var(--accent-gold-dim);
  padding: 0.1rem 0.5rem;
  border-radius: var(--radius-full, 999px);
}

/* Cards */
.timeline-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 0.75rem;
}

.timeline-card {
  display: flex;
  gap: 0.75rem;
  padding: 0.75rem;
  text-decoration: none;
  color: inherit;
  transition: all var(--transition-fast);
  cursor: pointer;
}

.timeline-card:hover {
  background: var(--bg-card-hover);
  transform: translateY(-1px);
  box-shadow: var(--shadow-glow);
}

.card-image {
  width: 64px;
  height: 64px;
  border-radius: var(--radius-sm);
  object-fit: cover;
  flex-shrink: 0;
  border: 1px solid var(--border-subtle);
}

.card-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-secondary);
  color: var(--text-muted);
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  min-width: 0;
}

.card-name {
  font-weight: 600;
  font-size: 0.9rem;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-meta {
  display: flex;
  gap: 0.5rem;
  font-size: 0.78rem;
}

.card-category {
  font-weight: 500;
}

.card-ruler {
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-date {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.card-values {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  margin-top: 0.1rem;
}

.card-price {
  font-size: 0.8rem;
  color: var(--accent-gold);
  font-weight: 600;
}

.card-sold-badge {
  font-size: 0.65rem;
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
  padding: 0.1rem 0.4rem;
  border-radius: var(--radius-full, 999px);
  font-weight: 500;
}

.card-grade {
  font-size: 0.7rem;
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
  padding: 0.1rem 0.4rem;
  border-radius: var(--radius-full, 999px);
  font-weight: 500;
}

/* States */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  padding: 3rem 1rem;
  color: var(--text-secondary);
}

.spinner {
  width: 28px;
  height: 28px;
  border: 3px solid var(--border-subtle);
  border-top-color: var(--accent-gold);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 3rem 1rem;
  color: var(--text-secondary);
  text-align: center;
}

.empty-state h3 {
  margin: 0;
  font-size: 1rem;
  color: var(--text-primary);
}

.empty-state p {
  margin: 0;
  font-size: 0.85rem;
  color: var(--text-muted);
}

/* Responsive */
@media (max-width: 640px) {
  .container {
    overflow-x: hidden;
  }

  .timeline {
    padding-left: 1.5rem;
  }

  .timeline-cards {
    grid-template-columns: 1fr;
  }

  .timeline-card {
    padding: 0.5rem;
  }

  .card-image {
    width: 48px;
    height: 48px;
  }

  .page-header h1 {
    font-size: 1.2rem;
  }

  .timeline-summary {
    gap: 0.5rem;
    padding: 0.75rem;
  }

  .summary-value {
    font-size: 0.95rem;
  }

  .summary-label {
    font-size: 0.65rem;
  }

  .marker-dot {
    left: -1.15rem;
    width: 10px;
    height: 10px;
  }

  .marker-label {
    font-size: 0.9rem;
  }
}
</style>
