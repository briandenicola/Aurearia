<template>
  <div class="container">
    <div v-if="loading" class="loading-state">Loading showcase...</div>

    <div v-else-if="notFound" class="empty-state">
      <h3>Showcase not found</h3>
      <p>This showcase may have been removed or the link is incorrect.</p>
    </div>

    <template v-else-if="showcase">
      <div class="showcase-header">
        <h1>{{ showcase.title }}</h1>
        <p v-if="showcase.ownerName" class="owner">Curated by {{ showcase.ownerName }}</p>
        <p v-if="showcase.description" class="description">{{ showcase.description }}</p>
      </div>

      <div v-if="coins.length" class="coins-grid">
        <div v-for="coin in coins" :key="coin.id" class="card coin-card">
          <div class="coin-image-container">
            <img
              v-if="getPrimaryImage(coin)"
              :src="imageUrl(getPrimaryImage(coin)!)"
              :alt="coin.name ?? 'Coin'"
              class="coin-image"
            />
            <div v-else class="coin-image-placeholder">
              <Coins :size="32" />
            </div>
          </div>
          <div class="coin-details">
            <h3 class="coin-name">{{ coin.name ?? 'Untitled' }}</h3>
            <div class="coin-meta">
              <span v-if="coin.era" class="meta-tag">{{ coin.era }}</span>
              <span v-if="coin.category" class="meta-tag">{{ coin.category }}</span>
              <span v-if="coin.grade" class="meta-tag grade">{{ coin.grade }}</span>
            </div>
          </div>
        </div>
      </div>

      <div v-else class="empty-state">
        <p>This showcase has no coins yet.</p>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Coins } from 'lucide-vue-next'
import { getPublicShowcase } from '@/api/client'

const API_BASE = import.meta.env.VITE_API_BASE_URL || ''

interface CoinImage {
  id: number
  filePath: string
  imageType: string
}

interface PublicCoin {
  id: number
  name?: string
  era?: string
  category?: string
  grade?: string
  images?: CoinImage[]
}

interface PublicShowcase {
  title: string
  description?: string
  ownerName?: string
}

const route = useRoute()
const loading = ref(true)
const notFound = ref(false)
const showcase = ref<PublicShowcase | null>(null)
const coins = ref<PublicCoin[]>([])

function getPrimaryImage(coin: PublicCoin): CoinImage | undefined {
  if (!coin.images?.length) return undefined
  return coin.images?.[0]
}

function imageUrl(img: CoinImage): string {
  return `${API_BASE}/uploads/${img.filePath}`
}

async function loadShowcase() {
  loading.value = true
  const slug = route.params.slug as string
  try {
    const res = await getPublicShowcase(slug)
    showcase.value = res.data?.showcase ?? null
    coins.value = res.data?.coins ?? []
    if (!showcase.value) notFound.value = true
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
}

onMounted(loadShowcase)
</script>

<style scoped>
.container { max-width: 1200px; margin: 0 auto; padding: 2rem 1.5rem; }
.loading-state { text-align: center; padding: 2rem; color: var(--text-secondary); }
.empty-state { text-align: center; padding: 3rem; color: var(--text-secondary); }
.empty-state h3 { color: var(--text-primary); margin-bottom: 0.5rem; }

.showcase-header { text-align: center; margin-bottom: 2rem; }
.showcase-header h1 { font-size: 2rem; color: var(--text-primary); margin: 0 0 0.5rem; }
.owner { color: var(--accent-gold); font-size: 0.9rem; margin: 0 0 0.5rem; }
.description { color: var(--text-secondary); font-size: 1rem; max-width: 600px; margin: 0 auto; line-height: 1.5; }

.coins-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 1.25rem; }

.coin-card { background: var(--bg-card); border: 1px solid var(--border-subtle); border-radius: 12px; overflow: hidden; padding: 0; transition: border-color 0.2s; }
.coin-card:hover { border-color: var(--accent-gold); }

.coin-image-container { width: 100%; aspect-ratio: 1; overflow: hidden; background: rgba(0, 0, 0, 0.2); }
.coin-image { width: 100%; height: 100%; object-fit: cover; }
.coin-image-placeholder { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; color: var(--text-secondary); }

.coin-details { padding: 1rem; }
.coin-name { font-size: 1rem; color: var(--text-primary); margin: 0 0 0.5rem; line-height: 1.3; }
.coin-meta { display: flex; flex-wrap: wrap; gap: 0.35rem; }
.meta-tag { font-size: 0.75rem; padding: 0.15rem 0.5rem; border-radius: 4px; background: rgba(255, 255, 255, 0.06); color: var(--text-secondary); }
.meta-tag.grade { background: rgba(212, 175, 55, 0.12); color: var(--accent-gold); }
</style>
