<template>
  <main class="viewer">
    <p v-if="failed" class="status">This host did not start the coin viewer.</p>
    <p v-else-if="!result" class="status">{{ pending ? 'Loading…' : 'Waiting for coin data…' }}</p>
    <p v-else-if="result.isError" class="status">The tool returned an error.</p>

    <template v-else-if="view === 'list'">
      <p v-if="coins.length === 0 && !filtered" class="status">No coins matched.</p>
      <template v-else>
        <div class="list-head">
          <p class="count">{{ countLabel }}</p>
          <form class="filter" role="search" @submit.prevent="applyFilter">
            <input v-model="filterText" type="search" placeholder="Filter coins" aria-label="Filter coins">
          </form>
          <div class="layout" role="group" aria-label="Layout">
            <button type="button" :aria-pressed="layout === 'grid'" aria-label="Grid" @click="layoutChoice = 'grid'">
              <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M1 1h6v6H1zM9 1h6v6H9zM1 9h6v6H1zM9 9h6v6H9z" /></svg>
            </button>
            <button type="button" :aria-pressed="layout === 'list'" aria-label="List" @click="layoutChoice = 'list'">
              <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M1 2h14v2H1zM1 7h14v2H1zM1 12h14v2H1z" /></svg>
            </button>
          </div>
        </div>

        <p v-if="listState === 'loading' && coins.length === 0" class="status">Searching…</p>
        <p v-else-if="coins.length === 0" class="status">No coins matched “{{ query }}”.</p>
        <div v-else-if="layout === 'grid'" class="grid">
          <CoinCard v-for="coin in coins" :key="coin.id" :coin="coin" :thumb="thumbs[coin.id]" @open="openCoin" />
        </div>
        <div v-else class="rows">
          <CoinRow v-for="coin in coins" :key="coin.id" :coin="coin" :thumb="thumbs[coin.id]" @open="openCoin" />
        </div>

        <p v-if="listState === 'error'" class="status">Could not load more coins.</p>
        <button v-if="remaining > 0" type="button" class="more" :disabled="listState === 'loading'" @click="showMore">
          {{ listState === 'loading' ? 'Loading…' : `Show ${Math.min(remaining, PAGE_SIZE)} more` }}
        </button>
      </template>
    </template>

    <template v-else-if="view === 'detail'">
      <div v-if="hasList" class="toolbar">
        <button type="button" class="back" @click="view = 'list'">← Results</button>
      </div>
      <p v-if="detailState === 'loading'" class="status">Loading coin…</p>
      <p v-else-if="detailState === 'error' || !detailCoin" class="status">Could not load that coin.</p>
      <CoinDetail v-else :coin="detailCoin" :images="detailImages" :gallery-state="galleryState" />
    </template>

    <p v-else class="status">Nothing to show.</p>
  </main>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import CoinCard from './CoinCard.vue'
import CoinDetail from './CoinDetail.vue'
import CoinRow from './CoinRow.vue'
import type { ViewerBridge } from './bridge'
import type { CallToolResult, CoinImage, CoinSummary } from './types'
import { coinFrom, imagesFrom, searchPageFrom } from './viewer-data'

const props = defineProps<{
  bridge: ViewerBridge
  result: CallToolResult | null
  pending?: boolean
  failed?: boolean
}>()

type LoadState = 'idle' | 'loading' | 'ready' | 'error'

// search_collection returns at most 20 per call; get_coin_images takes 20.
const PAGE_SIZE = 20
// Above this many results the grid gets too tall for a chat, so default to rows.
const GRID_MAX = 8

const view = ref<'list' | 'detail' | 'none'>('none')
const coins = ref<CoinSummary[]>([])
const total = ref(0)
const query = ref('')
const filterText = ref('')
const filtered = ref(false)
const hasList = ref(false)
const listState = ref<LoadState>('idle')
const layoutChoice = ref<'grid' | 'list' | null>(null)
// Keyed by coin id: a data URI, null for no photo, absent while loading.
// Kept across detail visits so returning to the results does not refetch.
const thumbs = reactive<Record<number, string | null>>({})
const detailCoin = ref<CoinSummary | null>(null)
const detailState = ref<LoadState>('loading')
const detailImages = ref<CoinImage[]>([])
const galleryState = ref<'loading' | 'ready' | 'error'>('loading')
let detailRequest = 0
let listRequest = 0

const layout = computed(() => layoutChoice.value ?? (coins.value.length > GRID_MAX ? 'list' : 'grid'))
const remaining = computed(() => Math.max(0, total.value - coins.value.length))
const countLabel = computed(() => {
  if (total.value > coins.value.length) return `Showing ${coins.value.length} of ${total.value} coins`
  return coins.value.length === 1 ? '1 coin' : `${coins.value.length} coins`
})

watch(
  () => props.result,
  (result) => {
    if (!result || result.isError) return
    const page = searchPageFrom(result)
    if (page) {
      setPage(page.coins, page.total, page.query)
      filterText.value = page.query
      hasList.value = true
      view.value = 'list'
      return
    }
    const coin = coinFrom(result)
    if (coin) {
      hasList.value = false
      void showDetail(coin)
      return
    }
    view.value = 'none'
  },
  { immediate: true },
)

function setPage(list: CoinSummary[], matches: number, text: string) {
  coins.value = list
  total.value = Math.max(matches, list.length)
  query.value = text
  listState.value = 'ready'
  void loadThumbs(list)
}

async function search(text: string, offset: number) {
  return searchPageFrom(
    await props.bridge.callTool('search_collection', { query: text, limit: PAGE_SIZE, offset }),
  )
}

async function showMore() {
  const request = ++listRequest
  listState.value = 'loading'
  try {
    const page = await search(query.value, coins.value.length)
    if (request !== listRequest) return
    if (!page) throw new Error('no page')
    const known = new Set(coins.value.map((coin) => coin.id))
    const added = page.coins.filter((coin) => !known.has(coin.id))
    coins.value = [...coins.value, ...added]
    // A page that adds nothing means the matches changed underneath us; stop
    // offering more rather than looping on an empty page.
    total.value = added.length === 0 ? coins.value.length : Math.max(page.total, coins.value.length)
    listState.value = 'ready'
    void loadThumbs(added)
  } catch {
    if (request === listRequest) listState.value = 'error'
  }
}

async function applyFilter() {
  const text = filterText.value.trim()
  const request = ++listRequest
  filtered.value = true
  listState.value = 'loading'
  coins.value = []
  total.value = 0
  query.value = text
  try {
    const page = await search(text, 0)
    if (request !== listRequest) return
    if (!page) throw new Error('no page')
    setPage(page.coins, page.total, text)
  } catch {
    if (request === listRequest) listState.value = 'error'
  }
}

async function loadThumbs(list: CoinSummary[]) {
  const missing = list.map((coin) => coin.id).filter((id) => !(id in thumbs))
  for (let start = 0; start < missing.length; start += PAGE_SIZE) {
    const batch = missing.slice(start, start + PAGE_SIZE)
    try {
      const images = imagesFrom(await props.bridge.callTool('get_coin_images', { coinIds: batch, size: 'thumb' }))
      for (const image of images) thumbs[image.coinId] = image.src
    } catch {
      // Fall through: coins without a thumbnail show "No photo".
    }
    for (const id of batch) {
      if (!(id in thumbs)) thumbs[id] = null
    }
  }
}

async function openCoin(coinId: number) {
  const request = ++detailRequest
  view.value = 'detail'
  detailState.value = 'loading'
  detailCoin.value = null
  try {
    const coin = coinFrom(await props.bridge.callTool('get_coin', { coinId }))
    if (request !== detailRequest) return
    if (!coin) {
      detailState.value = 'error'
      return
    }
    await showDetail(coin, request)
  } catch {
    if (request === detailRequest) detailState.value = 'error'
  }
}

async function showDetail(coin: CoinSummary, request = ++detailRequest) {
  view.value = 'detail'
  detailCoin.value = coin
  detailState.value = 'ready'
  detailImages.value = []
  galleryState.value = 'loading'
  try {
    const images = imagesFrom(await props.bridge.callTool('get_coin_images', { coinIds: [coin.id], size: 'medium' }))
    if (request !== detailRequest) return
    detailImages.value = images
    galleryState.value = 'ready'
  } catch {
    if (request === detailRequest) galleryState.value = 'error'
  }
}
</script>

<style scoped>
.viewer {
  padding: 12px;
}

.status {
  margin: 0;
  padding: 4px 0 8px;
  color: var(--viewer-muted);
}

.list-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  margin-bottom: 10px;
}

.count {
  flex: 1 1 auto;
  margin: 0;
  font-size: 13px;
  color: var(--viewer-muted);
}

.filter input {
  width: 180px;
  padding: 5px 10px;
  background: var(--viewer-surface);
  border: 1px solid var(--viewer-border);
  border-radius: 999px;
}

.filter input:focus-visible {
  border-color: var(--viewer-accent);
  outline: none;
}

.layout {
  display: inline-flex;
  overflow: hidden;
  border: 1px solid var(--viewer-border);
  border-radius: 6px;
}

.layout button {
  display: grid;
  place-items: center;
  width: 30px;
  height: 28px;
  padding: 0;
  cursor: pointer;
  color: var(--viewer-muted);
  background: none;
  border: 0;
}

.layout button[aria-pressed='true'] {
  color: var(--viewer-accent);
  background: var(--viewer-surface);
}

.layout svg {
  width: 14px;
  height: 14px;
  fill: currentColor;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(168px, 1fr));
  gap: 12px;
}

.rows {
  display: flex;
  flex-direction: column;
  border-top: 1px solid var(--viewer-border);
}

.rows > :deep(.row) {
  border-bottom: 1px solid var(--viewer-border);
  border-radius: 0;
}

.more {
  display: block;
  margin: 12px auto 0;
  padding: 6px 16px;
  cursor: pointer;
  color: var(--viewer-accent);
  background: none;
  border: 1px solid var(--viewer-border-strong);
  border-radius: 999px;
}

.more:disabled {
  cursor: default;
  opacity: 0.6;
}

.toolbar {
  margin-bottom: 12px;
}

.back {
  padding: 4px 10px;
  cursor: pointer;
  background: none;
  border: 1px solid var(--viewer-border);
  border-radius: 6px;
}

.back:hover {
  border-color: var(--viewer-border-strong);
}
</style>
