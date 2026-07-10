<template>
  <PullToRefresh :on-refresh="loadCoins">
    <div class="container">
      <div class="page-header max-sm:flex-wrap">
        <h1 class="flex items-center gap-2 text-lg text-heading sm:text-xl">
          <Clock :size="24" />
          Collection Timeline
        </h1>
        <div class="flex gap-2 max-sm:w-full sm:justify-end">
          <select v-model="filterType" class="form-input form-select min-w-[150px] max-sm:w-full sm:w-auto">
            <option value="all">All Coins</option>
            <option value="collection">Collection Only</option>
            <option value="sold">Sold Only</option>
          </select>
        </div>
      </div>

      <div v-if="loading" class="loading-overlay">
        <div class="spinner" />
        <p>Loading timeline...</p>
      </div>

      <div v-else-if="timelineGroups.length === 0" class="empty-state">
        <Clock :size="48" />
        <h3>No timeline data</h3>
        <p>Add purchase dates to your coins to see them on the timeline.</p>
      </div>

      <div v-else>
        <div class="mb-8 flex flex-wrap justify-around gap-4 rounded-md border border-border-subtle bg-card p-4 shadow-[var(--shadow-card)]">
          <div class="flex flex-col items-center gap-1">
            <span class="font-['Cinzel',serif] text-lg font-bold text-gold">{{ totalCoins }}</span>
            <span class="section-label">Coins</span>
          </div>
          <div class="flex flex-col items-center gap-1">
            <span class="font-['Cinzel',serif] text-lg font-bold text-gold">{{ yearSpan }}</span>
            <span class="section-label">Year Span</span>
          </div>
          <div class="flex flex-col items-center gap-1">
            <span class="font-['Cinzel',serif] text-lg font-bold text-gold">${{ totalInvested.toLocaleString() }}</span>
            <span class="section-label">Invested</span>
          </div>
          <div class="flex flex-col items-center gap-1">
            <span class="font-['Cinzel',serif] text-lg font-bold text-gold">${{ totalValue.toLocaleString() }}</span>
            <span class="section-label">Current Value</span>
          </div>
        </div>

        <div class="relative max-w-full overflow-x-hidden pl-8 max-sm:pl-5">
          <div class="absolute left-[0.55rem] top-0 bottom-0 w-px bg-border-subtle" />
          <div v-for="group in timelineGroups" :key="group.label" class="relative mb-8 last:mb-0">
            <div class="relative mb-3 flex items-center gap-3">
              <div class="absolute -left-6 z-10 h-3 w-3 rounded-full border-2 border-surface bg-gold shadow-[0_0_0_2px_var(--accent-gold-dim)] max-sm:-left-[0.95rem] max-sm:h-2.5 max-sm:w-2.5" />
              <div class="font-['Cinzel',serif] text-base font-semibold text-text-primary max-sm:text-[0.9rem]">{{ group.label }}</div>
              <div class="rounded-full bg-surface-secondary px-2 py-0.5 text-sm text-text-muted">
                {{ group.coins.length }} {{ group.coins.length === 1 ? 'coin' : 'coins' }}
              </div>
            </div>
            <div class="grid grid-cols-1 gap-3 sm:[grid-template-columns:repeat(auto-fill,minmax(280px,1fr))]">
              <router-link
                v-for="coin in group.coins"
                :key="coin.id"
                :to="`/coin/${coin.id}`"
                class="flex min-w-0 cursor-pointer gap-3 overflow-hidden rounded-md border border-border-subtle bg-card p-3 text-inherit no-underline transition-all hover:-translate-y-px hover:bg-card-hover hover:border-border-accent"
              >
                <AuthenticatedImage
                  v-if="getPrimaryImage(coin)"
                  :media-path="getPrimaryImage(coin)"
                  :alt="coin.name"
                  class="h-12 w-12 shrink-0 rounded-sm border border-border-subtle object-cover sm:h-16 sm:w-16"
                />
                <div v-else class="flex h-12 w-12 shrink-0 items-center justify-center rounded-sm border border-border-subtle bg-surface-secondary text-text-muted sm:h-16 sm:w-16">
                  <ImageIcon :size="24" />
                </div>
                <div class="min-w-0 flex flex-1 flex-col gap-1">
                  <span class="truncate text-base font-semibold text-text-primary">{{ coin.name }}</span>
                  <span class="flex min-w-0 gap-2 text-sm">
                    <span class="font-medium" :style="{ color: categoryColor(coin.category) }">{{ coin.category }}</span>
                    <span v-if="coin.ruler" class="truncate text-text-secondary">{{ coin.ruler }}</span>
                  </span>
                  <span v-if="coin.purchaseDate" class="text-sm text-text-muted">{{ formatDate(coin.purchaseDate) }}</span>
                  <div class="mt-0.5 flex items-center gap-2">
                    <span v-if="coin.purchasePrice" class="text-chip font-semibold text-gold">
                      ${{ coin.purchasePrice.toLocaleString() }}
                    </span>
                    <span v-if="coin.isSold" class="rounded-full border border-border-subtle bg-surface-secondary px-2 py-0.5 text-sm font-medium text-warning">
                      Sold
                    </span>
                    <span v-if="coin.grade" class="rounded-full bg-surface-secondary px-2 py-0.5 text-sm font-medium text-gold">
                      {{ coin.grade }}
                    </span>
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
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'

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
    // Fetch all non-wishlist coins (collection + sold) in pages
    const allFetched = new Map<number, Coin>()
    for (const soldParam of [undefined, 'true'] as const) {
      let page = 1
      let hasMore = true
      while (hasMore) {
        const params: Record<string, unknown> = { limit: 100, page, sort: 'purchase_date', order: 'desc' }
        if (soldParam) params.sold = soldParam
        const res = await getCoins(params as Parameters<typeof getCoins>[0])
        for (const c of res.data.coins) allFetched.set(c.id, c)
        hasMore = res.data.coins.length === 100
        page++
      }
    }
    allCoins.value = Array.from(allFetched.values())
  } catch {
    allCoins.value = []
  } finally {
    loading.value = false
  }
}

onMounted(loadCoins)
</script>
