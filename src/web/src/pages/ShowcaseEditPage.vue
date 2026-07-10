<template>
  <div class="container overflow-x-hidden">
    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
      <p>Loading showcase...</p>
    </div>

    <template v-else-if="showcase">
      <div class="page-header items-start gap-4">
        <div class="min-w-0 flex-1">
          <router-link to="/showcases" class="mb-2 inline-flex items-center gap-1 text-body text-text-secondary no-underline transition-colors hover:text-gold"><ArrowLeft :size="16" /> Showcases</router-link>
          <div v-if="!editingTitle" class="group flex cursor-pointer items-center gap-2" @click="startEditTitle">
            <h1>{{ showcase.title }}</h1>
            <Pencil :size="14" class="text-text-secondary opacity-40 transition-opacity group-hover:opacity-100" />
          </div>
          <div v-else class="flex flex-wrap items-center gap-2">
            <input
              v-model="editTitle"
              type="text"
              class="form-input min-w-0 flex-1 text-lg font-bold"
              @keyup.enter="saveTitle"
              @keyup.escape="editingTitle = false"
              ref="titleInput"
            />
            <button class="btn btn-primary btn-sm" @click="saveTitle">Save</button>
            <button class="btn btn-secondary btn-sm" @click="editingTitle = false">Cancel</button>
          </div>
          <div v-if="!editingDesc" class="group mt-1 flex cursor-pointer items-center gap-1.5" @click="startEditDesc">
            <p class="m-0 text-body text-text-secondary">{{ showcase.description || 'No description' }}</p>
            <Pencil :size="12" class="text-text-secondary opacity-40 transition-opacity group-hover:opacity-100" />
          </div>
          <div v-else class="mt-1">
            <textarea v-model="editDesc" rows="2" class="form-input" @keyup.escape="editingDesc = false"></textarea>
            <div class="mt-1.5 flex gap-2">
              <button class="btn btn-primary btn-sm" @click="saveDesc">Save</button>
              <button class="btn btn-secondary btn-sm" @click="editingDesc = false">Cancel</button>
            </div>
          </div>
        </div>
        <button class="btn btn-primary" :disabled="saving" @click="saveCoins">
          <Save :size="16" /> {{ saving ? 'Saving...' : 'Save Coins' }}
        </button>
      </div>

      <div v-if="savedMessage" class="fixed bottom-8 left-1/2 z-[1000] -translate-x-1/2 rounded-sm bg-[var(--accent-gold)] px-5 py-2 text-body font-medium text-[var(--bg-primary)]">{{ savedMessage }}</div>

      <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
        <div class="card flex max-h-[70vh] flex-col overflow-hidden p-4">
          <div class="mb-3 flex items-center justify-between gap-3">
            <h2 class="m-0 text-[1.1rem] text-text-primary">Your Collection</h2>
            <span class="text-chip text-text-secondary">{{ availableCoins.length }} coins</span>
          </div>
          <div class="mb-3 flex items-center gap-2 rounded-sm border border-border-subtle bg-card px-3 py-2 text-text-secondary">
            <Search :size="16" />
            <input v-model="search" type="text" placeholder="Search coins..." class="min-w-0 flex-1 bg-transparent text-base text-text-primary outline-none" />
          </div>
          <div class="flex flex-1 flex-col gap-1 overflow-x-hidden overflow-y-auto">
            <div
              v-for="coin in filteredCollection"
              :key="coin.id"
              class="flex min-w-0 cursor-pointer items-center gap-2 rounded-sm px-2 py-2 transition-colors hover:bg-white/4"
              :class="{ 'opacity-40': selectedIds.has(coin.id) }"
              @click="addCoin(coin.id)"
            >
              <AuthenticatedImage
                v-if="getPrimaryImage(coin)"
                :media-path="imageUrl(getPrimaryImage(coin)!)"
                class="h-9 w-9 shrink-0 rounded-sm object-cover"
                alt=""
              />
              <div v-else class="flex h-9 w-9 shrink-0 items-center justify-center rounded-sm bg-white/5 text-text-secondary"><Coins :size="16" /></div>
              <div class="flex min-w-0 flex-1 flex-col overflow-hidden">
                <span class="truncate text-base font-medium text-text-primary">{{ coin.name ?? 'Untitled' }}</span>
                <span class="truncate text-sm text-text-secondary">{{ [coin.era, coin.category].filter(Boolean).join(' / ') }}</span>
              </div>
              <Plus :size="16" class="shrink-0 text-text-secondary" />
            </div>
            <div v-if="!filteredCollection.length" class="py-8 text-center text-base text-text-secondary">No matching coins</div>
          </div>
        </div>

        <div class="card flex max-h-[70vh] flex-col overflow-hidden p-4">
          <div class="mb-3 flex items-center justify-between gap-3">
            <h2 class="m-0 text-[1.1rem] text-text-primary">Showcase Coins</h2>
            <span class="text-chip text-text-secondary">{{ selectedCoinIds.length }} selected</span>
          </div>
          <div class="flex flex-1 flex-col gap-1 overflow-x-hidden overflow-y-auto">
            <div
              v-for="(coinId, idx) in selectedCoinIds"
              :key="coinId"
              class="flex min-w-0 items-center gap-2 rounded-sm px-2 py-2"
            >
              <span class="w-5 shrink-0 text-center text-sm text-text-secondary">{{ idx + 1 }}</span>
              <template v-if="coinMap.get(coinId)">
                <AuthenticatedImage
                  v-if="getPrimaryImage(coinMap.get(coinId)!)"
                  :media-path="imageUrl(getPrimaryImage(coinMap.get(coinId)!)!)"
                  class="h-9 w-9 shrink-0 rounded-sm object-cover"
                  alt=""
                />
                <div v-else class="flex h-9 w-9 shrink-0 items-center justify-center rounded-sm bg-white/5 text-text-secondary"><Coins :size="16" /></div>
                <div class="flex min-w-0 flex-1 flex-col overflow-hidden">
                  <span class="truncate text-base font-medium text-text-primary">{{ coinMap.get(coinId)?.name ?? 'Untitled' }}</span>
                  <span class="truncate text-sm text-text-secondary">{{ [coinMap.get(coinId)?.era, coinMap.get(coinId)?.category].filter(Boolean).join(' / ') }}</span>
                </div>
              </template>
              <span v-else class="min-w-0 flex-1 truncate text-base font-medium text-text-primary">Coin #{{ coinId }}</span>
              <button class="inline-flex shrink-0 rounded-sm p-1 text-text-secondary transition-colors hover:text-[var(--error-bg)]" @click="removeCoin(coinId)" title="Remove">
                <X :size="16" />
              </button>
            </div>
            <div v-if="!selectedCoinIds.length" class="py-8 text-center text-base text-text-secondary">
              Click coins from your collection to add them
            </div>
          </div>
        </div>
      </div>
    </template>

    <div v-else class="empty-state">
      <h3>Showcase not found</h3>
      <router-link to="/showcases" class="btn btn-secondary">Back to Showcases</router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowLeft, Pencil, Save, Search, Plus, X, Coins } from 'lucide-vue-next'
import { getShowcase, updateShowcase, setShowcaseCoins, getCoins } from '@/api/client'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'

interface CoinImage {
  id: number
  filePath: string
  imageType: string
  isPrimary?: boolean
}

interface Coin {
  id: number
  name?: string
  era?: string
  category?: string
  images?: CoinImage[]
}

interface ShowcaseData {
  id: number
  slug: string
  title: string
  description?: string
  isActive: boolean
  coinIds?: number[]
}

const route = useRoute()
const loading = ref(true)
const showcase = ref<ShowcaseData | null>(null)
const allCoins = ref<Coin[]>([])
const selectedCoinIds = ref<number[]>([])
const search = ref('')
const saving = ref(false)
const savedMessage = ref('')

const editingTitle = ref(false)
const editTitle = ref('')
const editingDesc = ref(false)
const editDesc = ref('')
const titleInput = ref<HTMLInputElement | null>(null)

const selectedIds = computed(() => new Set(selectedCoinIds.value))

const coinMap = computed(() => {
  const m = new Map<number, Coin>()
  for (const c of allCoins.value) {
    m.set(c.id, c)
  }
  return m
})

const availableCoins = computed(() =>
  allCoins.value.filter(c => !selectedIds.value.has(c.id))
)

const filteredCollection = computed(() => {
  if (!search.value.trim()) return availableCoins.value
  const q = search.value.toLowerCase()
  return availableCoins.value.filter(c =>
    (c.name?.toLowerCase()?.includes(q)) ||
    (c.era?.toLowerCase()?.includes(q)) ||
    (c.category?.toLowerCase()?.includes(q))
  )
})

function getPrimaryImage(coin: Coin): CoinImage | undefined {
  if (!coin.images?.length) return undefined
  return coin.images.find(i => i.isPrimary) ?? coin.images?.[0]
}

function imageUrl(img: CoinImage): string {
  return img.filePath
}

function addCoin(id: number) {
  if (!selectedIds.value.has(id)) {
    selectedCoinIds.value.push(id)
  }
}

function removeCoin(id: number) {
  selectedCoinIds.value = selectedCoinIds.value.filter(cid => cid !== id)
}

function startEditTitle() {
  editTitle.value = showcase.value?.title ?? ''
  editingTitle.value = true
  nextTick(() => titleInput.value?.focus())
}

function startEditDesc() {
  editDesc.value = showcase.value?.description ?? ''
  editingDesc.value = true
}

async function saveTitle() {
  if (!showcase.value || !editTitle.value.trim()) return
  await updateShowcase(showcase.value.id, { title: editTitle.value.trim() })
  showcase.value.title = editTitle.value.trim()
  editingTitle.value = false
}

async function saveDesc() {
  if (!showcase.value) return
  await updateShowcase(showcase.value.id, { description: editDesc.value.trim() })
  showcase.value.description = editDesc.value.trim()
  editingDesc.value = false
}

async function saveCoins() {
  if (!showcase.value) return
  saving.value = true
  try {
    await setShowcaseCoins(showcase.value.id, selectedCoinIds.value)
    savedMessage.value = 'Showcase coins saved'
    setTimeout(() => { savedMessage.value = '' }, 2000)
  } finally {
    saving.value = false
  }
}

async function fetchAllCoins(): Promise<Coin[]> {
  const all: Coin[] = []
  let page = 1
  const limit = 100
  while (true) {
    const res = await getCoins({ limit, page, wishlist: 'false', sold: 'false' })
    const coins: Coin[] = res.data?.coins ?? []
    all.push(...coins)
    if (coins.length < limit) break
    page++
  }
  return all
}

async function loadData() {
  loading.value = true
  try {
    const id = Number(route.params.id)
    const [scRes, collectionCoins] = await Promise.all([
      getShowcase(id),
      fetchAllCoins()
    ])
    showcase.value = scRes.data?.showcase ?? null
    const showcaseCoins: Coin[] = scRes.data?.coins ?? []

    // Merge coins from both sources
    const merged = new Map<number, Coin>()
    for (const c of collectionCoins) merged.set(c.id, c)
    for (const c of showcaseCoins) merged.set(c.id, c)
    allCoins.value = Array.from(merged.values())

    selectedCoinIds.value = showcase.value?.coinIds ?? []
  } catch {
    showcase.value = null
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>
