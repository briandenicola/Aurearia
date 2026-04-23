<template>
  <div class="container">
    <div v-if="loading" class="loading-state">Loading showcase...</div>

    <template v-else-if="showcase">
      <div class="page-header">
        <div class="title-section">
          <router-link to="/showcases" class="back-link"><ArrowLeft :size="16" /> Showcases</router-link>
          <div v-if="!editingTitle" class="title-row" @click="startEditTitle">
            <h1>{{ showcase.title }}</h1>
            <Pencil :size="14" class="edit-icon" />
          </div>
          <div v-else class="title-edit-row">
            <input
              v-model="editTitle"
              type="text"
              class="title-input"
              @keyup.enter="saveTitle"
              @keyup.escape="editingTitle = false"
              ref="titleInput"
            />
            <button class="btn btn-primary btn-sm" @click="saveTitle">Save</button>
            <button class="btn btn-secondary btn-sm" @click="editingTitle = false">Cancel</button>
          </div>
          <div v-if="!editingDesc" class="desc-row" @click="startEditDesc">
            <p class="showcase-desc">{{ showcase.description || 'No description' }}</p>
            <Pencil :size="12" class="edit-icon" />
          </div>
          <div v-else class="desc-edit-row">
            <textarea v-model="editDesc" rows="2" class="desc-input" @keyup.escape="editingDesc = false"></textarea>
            <div class="inline-actions">
              <button class="btn btn-primary btn-sm" @click="saveDesc">Save</button>
              <button class="btn btn-secondary btn-sm" @click="editingDesc = false">Cancel</button>
            </div>
          </div>
        </div>
        <button class="btn btn-primary" :disabled="saving" @click="saveCoins">
          <Save :size="16" /> {{ saving ? 'Saving...' : 'Save Coins' }}
        </button>
      </div>

      <div v-if="savedMessage" class="toast">{{ savedMessage }}</div>

      <div class="columns">
        <!-- Left: Your Collection -->
        <div class="column">
          <div class="column-header">
            <h2>Your Collection</h2>
            <span class="count-label">{{ availableCoins.length }} coins</span>
          </div>
          <div class="search-bar">
            <Search :size="16" />
            <input v-model="search" type="text" placeholder="Search coins..." />
          </div>
          <div class="coin-list">
            <div
              v-for="coin in filteredCollection"
              :key="coin.id"
              class="coin-row"
              :class="{ selected: selectedIds.has(coin.id) }"
              @click="addCoin(coin.id)"
            >
              <img
                v-if="getPrimaryImage(coin)"
                :src="imageUrl(getPrimaryImage(coin)!)"
                class="coin-thumb"
                alt=""
              />
              <div v-else class="coin-thumb placeholder"><Coins :size="16" /></div>
              <div class="coin-info">
                <span class="coin-name">{{ coin.name ?? 'Untitled' }}</span>
                <span class="coin-meta">{{ [coin.era, coin.category].filter(Boolean).join(' / ') }}</span>
              </div>
              <Plus :size="16" class="add-icon" />
            </div>
            <div v-if="!filteredCollection.length" class="empty-list">No matching coins</div>
          </div>
        </div>

        <!-- Right: Showcase Coins -->
        <div class="column">
          <div class="column-header">
            <h2>Showcase Coins</h2>
            <span class="count-label">{{ selectedCoinIds.length }} selected</span>
          </div>
          <div class="coin-list">
            <div
              v-for="(coinId, idx) in selectedCoinIds"
              :key="coinId"
              class="coin-row showcase-row"
            >
              <span class="order-num">{{ idx + 1 }}</span>
              <template v-if="coinMap.get(coinId)">
                <img
                  v-if="getPrimaryImage(coinMap.get(coinId)!)"
                  :src="imageUrl(getPrimaryImage(coinMap.get(coinId)!)!)"
                  class="coin-thumb"
                  alt=""
                />
                <div v-else class="coin-thumb placeholder"><Coins :size="16" /></div>
                <div class="coin-info">
                  <span class="coin-name">{{ coinMap.get(coinId)?.name ?? 'Untitled' }}</span>
                  <span class="coin-meta">{{ [coinMap.get(coinId)?.era, coinMap.get(coinId)?.category].filter(Boolean).join(' / ') }}</span>
                </div>
              </template>
              <span v-else class="coin-info"><span class="coin-name">Coin #{{ coinId }}</span></span>
              <button class="btn-remove" @click="removeCoin(coinId)" title="Remove">
                <X :size="16" />
              </button>
            </div>
            <div v-if="!selectedCoinIds.length" class="empty-list">
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

const API_BASE = import.meta.env.VITE_API_BASE_URL || ''

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
  return `${API_BASE}/uploads/${img.filePath}`
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

async function loadData() {
  loading.value = true
  try {
    const id = Number(route.params.id)
    const [scRes, coinsRes] = await Promise.all([
      getShowcase(id),
      getCoins({ limit: 500, wishlist: 'false', sold: 'false' })
    ])
    showcase.value = scRes.data?.showcase ?? null
    const showcaseCoins: Coin[] = scRes.data?.coins ?? []
    const collectionCoins: Coin[] = coinsRes.data?.coins ?? []

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

<style scoped>
.container { max-width: 1200px; margin: 0 auto; padding: 1.5rem; overflow-x: hidden; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 1.5rem; gap: 1rem; flex-wrap: wrap; }
.page-header h1 { font-size: 1.75rem; color: var(--text-primary); margin: 0; }
.btn { display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.5rem 1rem; border-radius: 8px; border: none; cursor: pointer; font-weight: 500; font-size: 0.875rem; white-space: nowrap; }
.btn-primary { background: var(--accent-gold); color: #1e1e1e; }
.btn-secondary { background: var(--bg-card); color: var(--text-primary); border: 1px solid var(--border-subtle); }
.btn-sm { padding: 0.35rem 0.65rem; font-size: 0.8rem; }
.loading-state { text-align: center; padding: 2rem; color: var(--text-secondary); }
.empty-state { text-align: center; padding: 3rem; color: var(--text-secondary); }
.empty-state h3 { color: var(--text-primary); }

.back-link { display: inline-flex; align-items: center; gap: 0.25rem; color: var(--text-secondary); font-size: 0.85rem; text-decoration: none; margin-bottom: 0.5rem; }
.back-link:hover { color: var(--accent-gold); }

.title-section { flex: 1; min-width: 0; }
.title-row { display: flex; align-items: center; gap: 0.5rem; cursor: pointer; }
.title-row:hover .edit-icon { opacity: 1; }
.edit-icon { color: var(--text-secondary); opacity: 0.4; transition: opacity 0.2s; }
.title-edit-row { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.title-input { font-size: 1.25rem; font-weight: 700; background: var(--bg-card); color: var(--text-primary); border: 1px solid var(--border-subtle); border-radius: 8px; padding: 0.25rem 0.5rem; flex: 1; min-width: 0; }
.desc-row { display: flex; align-items: center; gap: 0.35rem; cursor: pointer; margin-top: 0.25rem; }
.desc-row:hover .edit-icon { opacity: 1; }
.showcase-desc { color: var(--text-secondary); font-size: 0.875rem; margin: 0; }
.desc-edit-row { margin-top: 0.25rem; }
.desc-input { width: 100%; background: var(--bg-card); color: var(--text-primary); border: 1px solid var(--border-subtle); border-radius: 8px; padding: 0.5rem 0.75rem; font-size: 0.875rem; box-sizing: border-box; }
.inline-actions { display: flex; gap: 0.5rem; margin-top: 0.35rem; }

.toast { position: fixed; bottom: 2rem; left: 50%; transform: translateX(-50%); background: var(--accent-gold); color: #1e1e1e; padding: 0.5rem 1.25rem; border-radius: 8px; font-weight: 500; font-size: 0.875rem; z-index: 1000; }

.columns { display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; }
@media (max-width: 768px) {
  .columns { grid-template-columns: 1fr; }
  .container { padding: 1rem; }
  .page-header h1 { font-size: 1.35rem; }
}

.column { background: var(--bg-card); border: 1px solid var(--border-subtle); border-radius: 12px; padding: 1rem; display: flex; flex-direction: column; max-height: 70vh; overflow: hidden; }
.column-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem; }
.column-header h2 { font-size: 1.1rem; color: var(--text-primary); margin: 0; }
.count-label { font-size: 0.8rem; color: var(--text-secondary); }

.search-bar { display: flex; align-items: center; gap: 0.5rem; background: var(--bg-card); border: 1px solid var(--border-subtle); border-radius: 8px; padding: 0.4rem 0.75rem; margin-bottom: 0.75rem; color: var(--text-secondary); }
.search-bar input { flex: 1; background: transparent; border: none; color: var(--text-primary); outline: none; font-size: 0.875rem; min-width: 0; }

.coin-list { flex: 1; overflow-y: auto; overflow-x: hidden; display: flex; flex-direction: column; gap: 0.25rem; }
.coin-row { display: flex; align-items: center; gap: 0.5rem; padding: 0.5rem; border-radius: 8px; cursor: pointer; transition: background 0.15s; min-width: 0; }
.coin-row:hover { background: rgba(255, 255, 255, 0.04); }
.coin-row.selected { opacity: 0.4; }
.showcase-row { cursor: default; }

.coin-thumb { width: 36px; height: 36px; border-radius: 6px; object-fit: cover; flex-shrink: 0; }
.coin-thumb.placeholder { display: flex; align-items: center; justify-content: center; background: rgba(255, 255, 255, 0.05); color: var(--text-secondary); }
.coin-info { flex: 1; display: flex; flex-direction: column; min-width: 0; overflow: hidden; }
.coin-name { color: var(--text-primary); font-size: 0.875rem; font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.coin-meta { color: var(--text-secondary); font-size: 0.75rem; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.add-icon { color: var(--text-secondary); flex-shrink: 0; }
.order-num { color: var(--text-secondary); font-size: 0.75rem; width: 1.25rem; text-align: center; flex-shrink: 0; }
.btn-remove { background: none; border: none; color: var(--text-secondary); cursor: pointer; padding: 0.25rem; border-radius: 4px; flex-shrink: 0; }
.btn-remove:hover { color: #dc3545; }
.empty-list { text-align: center; padding: 2rem; color: var(--text-secondary); font-size: 0.875rem; }
</style>
