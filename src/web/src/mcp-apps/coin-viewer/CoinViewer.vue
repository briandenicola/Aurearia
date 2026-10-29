<template>
  <main class="viewer">
    <p v-if="failed" class="status">This host did not start the coin viewer.</p>
    <p v-else-if="!result" class="status">{{ pending ? 'Loading…' : 'Waiting for coin data…' }}</p>
    <p v-else-if="result.isError" class="status">The tool returned an error.</p>

    <template v-else-if="view === 'list' && coins">
      <p v-if="coins.length === 0" class="status">No coins matched.</p>
      <template v-else>
        <p class="status">{{ coins.length === 1 ? '1 coin' : `${coins.length} coins` }}</p>
        <div class="grid">
          <CoinCard
            v-for="coin in coins"
            :key="coin.id"
            :coin="coin"
            :thumb="thumbs[coin.id]"
            @open="openCoin"
          />
        </div>
      </template>
    </template>

    <template v-else-if="view === 'detail'">
      <div v-if="coins" class="toolbar">
        <button type="button" class="back" @click="view = 'list'">← Results</button>
      </div>
      <p v-if="detailState === 'loading'" class="status">Loading coin…</p>
      <p v-else-if="detailState === 'error' || !detailCoin" class="status">Could not load that coin.</p>
      <div v-else class="detail">
        <div class="gallery">
          <p v-if="galleryState === 'loading'" class="status">Loading photos…</p>
          <p v-else-if="galleryState === 'error'" class="status">Photos unavailable.</p>
          <p v-else-if="detailImages.length === 0" class="status">No photos.</p>
          <template v-else>
            <figure v-for="image in detailImages" :key="image.imageId">
              <img :src="image.src" :alt="`${detailCoin.name} ${image.imageType}`">
              <figcaption>{{ image.imageType }}</figcaption>
            </figure>
          </template>
        </div>
        <div>
          <h1>{{ detailCoin.name }}</h1>
          <dl>
            <template v-for="[label, value] in rows" :key="label">
              <dt>{{ label }}</dt>
              <dd>{{ value }}</dd>
            </template>
          </dl>
          <p v-if="missing.length" class="missing">
            Missing: {{ missing.join(', ') }}
          </p>
        </div>
      </div>
    </template>

    <p v-else class="status">Nothing to show.</p>
  </main>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import CoinCard from './CoinCard.vue'
import type { ViewerBridge } from './bridge'
import type { CallToolResult, CoinImage, CoinSummary } from './types'
import { coinFrom, coinsFrom, detailRows, imagesFrom, missingLabels } from './viewer-data'

const props = defineProps<{
  bridge: ViewerBridge
  result: CallToolResult | null
  pending?: boolean
  failed?: boolean
}>()

type LoadState = 'loading' | 'ready' | 'error'

const view = ref<'list' | 'detail' | 'none'>('none')
const coins = ref<CoinSummary[] | null>(null)
// Keyed by coin id: a data URI, null for no photo, absent while loading.
// Kept across detail visits so returning to the results does not refetch.
const thumbs = reactive<Record<number, string | null>>({})
const detailCoin = ref<CoinSummary | null>(null)
const detailState = ref<LoadState>('loading')
const detailImages = ref<CoinImage[]>([])
const galleryState = ref<LoadState>('loading')
let detailRequest = 0

const rows = computed(() => (detailCoin.value ? detailRows(detailCoin.value) : []))
const missing = computed(() => (detailCoin.value ? missingLabels(detailCoin.value) : []))

watch(
  () => props.result,
  (result) => {
    if (!result || result.isError) return
    const list = coinsFrom(result)
    if (list) {
      coins.value = list
      view.value = 'list'
      void loadThumbs(list)
      return
    }
    const coin = coinFrom(result)
    if (coin) {
      coins.value = null
      void showDetail(coin)
      return
    }
    view.value = 'none'
  },
  { immediate: true },
)

async function loadThumbs(list: CoinSummary[]) {
  const missing = list.map((coin) => coin.id).filter((id) => !(id in thumbs)).slice(0, 20)
  if (missing.length === 0) return
  try {
    const images = imagesFrom(await props.bridge.callTool('get_coin_images', { coinIds: missing, size: 'thumb' }))
    for (const image of images) thumbs[image.coinId] = image.src
  } catch {
    // Fall through: coins without a thumbnail show "No photo".
  }
  for (const id of missing) {
    if (!(id in thumbs)) thumbs[id] = null
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

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 10px;
}

.toolbar {
  margin-bottom: 10px;
}

.back {
  padding: 4px 10px;
  font: inherit;
  color: inherit;
  cursor: pointer;
  background: none;
  border: 1px solid var(--viewer-border);
  border-radius: 6px;
}

.back:hover,
.back:focus-visible {
  border-color: var(--viewer-accent);
  outline: none;
}

.detail {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 16px;
}

@media (max-width: 520px) {
  .detail {
    grid-template-columns: 1fr;
  }
}

.gallery {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  align-content: start;
  gap: 8px;
}

.gallery .status {
  grid-column: 1 / -1;
}

figure {
  margin: 0;
}

/* Square frames keep obverse and reverse aligned whatever each photo's crop;
   contain keeps the whole coin visible. */
figure img {
  display: block;
  width: 100%;
  aspect-ratio: 1;
  object-fit: contain;
  background: var(--viewer-placeholder);
  border-radius: 8px;
}

figcaption {
  font-size: 12px;
  color: var(--viewer-muted);
  text-transform: capitalize;
}

h1 {
  margin: 0 0 8px;
  font-size: 18px;
  color: var(--viewer-heading);
}

dl {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 4px 12px;
  margin: 0;
}

dt {
  color: var(--viewer-muted);
}

dd {
  margin: 0;
}

.missing {
  margin: 12px 0 0;
  font-size: 12px;
  color: var(--viewer-muted);
}
</style>
