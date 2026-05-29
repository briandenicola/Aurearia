<template>
  <div class="suggestions-grid">
    <div v-for="(coin, j) in suggestions" :key="j" class="suggestion-card">
      <div class="suggestion-img" v-if="getSuggestionImageUrl(coin)">
        <img :src="getSuggestionImageUrl(coin)" :alt="coin.name" @error="handleImgError" />
      </div>
      <div class="suggestion-body">
        <h4>{{ coin.name }}</h4>
        <p class="suggestion-desc">{{ coin.description }}</p>
        <div class="suggestion-meta">
          <span v-if="coin.era" class="meta-tag">{{ coin.era }}</span>
          <span v-if="coin.material" class="meta-tag">{{ coin.material }}</span>
          <span v-if="coin.denomination" class="meta-tag">{{ coin.denomination }}</span>
        </div>
        <div class="suggestion-price" v-if="coin.estPrice">{{ coin.estPrice }}</div>
        <div class="suggestion-actions">
          <SafeExternalLink v-if="coin.sourceUrl" :href="coin.sourceUrl" target="_blank" rel="noopener" class="source-link">
            <ExternalLink :size="12" /> {{ coin.sourceName || 'Source' }}
          </SafeExternalLink>
          <button
            v-if="coin.era || coin.material || coin.denomination"
            class="btn btn-primary btn-sm add-btn"
            :disabled="addingIdx === `${messageIndex}-${j}`"
            @click="$emit('add-to-wishlist', coin, `${messageIndex}-${j}`)"
          >
            <CirclePlus :size="14" />
            {{ addedSet.has(`${messageIndex}-${j}`) ? 'Added!' : addingIdx === `${messageIndex}-${j}` ? 'Adding...' : 'Add to Wishlist' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import type { CoinSuggestion } from '@/types'
import { CirclePlus, ExternalLink } from 'lucide-vue-next'
import { proxyImage, scrapeImage } from '@/api/client'
import SafeExternalLink from '@/components/SafeExternalLink.vue'

defineProps<{
  suggestions: CoinSuggestion[]
  addedSet: Set<string>
  addingIdx: string | null
  messageIndex: number
}>()

defineEmits<{
  'add-to-wishlist': [coin: CoinSuggestion, key: string]
}>()

const scrapedImages = ref<Map<string, string>>(new Map())
const proxiedImages = ref<Map<string, string>>(new Map())
const pendingProxyLoads = new Set<string>()
const objectUrls = new Set<string>()

async function loadProxiedImage(url: string) {
  if (!url || pendingProxyLoads.has(url)) return
  pendingProxyLoads.add(url)
  try {
    const res = await proxyImage(url)
    const blob = res.data
    if (!(blob instanceof Blob) || blob.size === 0) {
      proxiedImages.value.set(url, '')
      return
    }
    const objectUrl = URL.createObjectURL(blob)
    objectUrls.add(objectUrl)
    proxiedImages.value.set(url, objectUrl)
  } catch (err) {
    console.warn('Failed to proxy suggestion image', err)
    proxiedImages.value.set(url, '')
  } finally {
    pendingProxyLoads.delete(url)
  }
}

function getOrLoadProxiedUrl(url: string): string {
  if (!url) return ''
  const cached = proxiedImages.value.get(url)
  if (cached !== undefined) return cached
  proxiedImages.value.set(url, '')
  void loadProxiedImage(url)
  return ''
}

function getSuggestionImageUrl(coin: CoinSuggestion): string {
  if (coin.sourceUrl) {
    const cached = scrapedImages.value.get(coin.sourceUrl)
    if (cached) return getOrLoadProxiedUrl(cached)
    if (cached === undefined) {
      scrapedImages.value.set(coin.sourceUrl, '')
      scrapeImage(coin.sourceUrl).then((res) => {
        if (res.data.imageUrl) {
          scrapedImages.value.set(coin.sourceUrl, res.data.imageUrl)
          void loadProxiedImage(res.data.imageUrl)
        } else if (coin.imageUrl) {
          scrapedImages.value.set(coin.sourceUrl, coin.imageUrl)
          void loadProxiedImage(coin.imageUrl)
        }
      }).catch(() => {
        if (coin.imageUrl) {
          scrapedImages.value.set(coin.sourceUrl, coin.imageUrl)
          void loadProxiedImage(coin.imageUrl)
        }
      })
    }
    return ''
  }
  if (coin.imageUrl) return getOrLoadProxiedUrl(coin.imageUrl)
  return ''
}

function handleImgError(e: Event) {
  const img = e.target as HTMLImageElement
  img.style.display = 'none'
}

onBeforeUnmount(() => {
  for (const objectUrl of objectUrls) {
    URL.revokeObjectURL(objectUrl)
  }
  objectUrls.clear()
  pendingProxyLoads.clear()
  proxiedImages.value.clear()
  scrapedImages.value.clear()
})
</script>

<style scoped>
.suggestions-grid {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
  width: 100%;
}

.suggestion-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
  display: flex;
  transition: border-color var(--transition-fast);
}

.suggestion-card:hover {
  border-color: var(--accent-gold);
}

.suggestion-img {
  width: 80px;
  min-height: 80px;
  flex-shrink: 0;
  overflow: hidden;
  background: var(--bg-body);
}

.suggestion-img img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.suggestion-body {
  padding: 0.6rem 0.75rem;
  flex: 1;
  min-width: 0;
}

.suggestion-body h4 {
  font-size: 0.85rem;
  margin: 0 0 0.25rem;
  color: var(--text-primary);
  line-height: 1.3;
}

.suggestion-desc {
  font-size: 0.78rem;
  color: var(--text-secondary);
  margin: 0 0 0.4rem;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.suggestion-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.3rem;
  margin-bottom: 0.4rem;
}

.meta-tag {
  font-size: 0.7rem;
  padding: 0.1rem 0.4rem;
  border-radius: var(--radius-full);
  background: var(--bg-body);
  color: var(--text-muted);
  border: 1px solid var(--border-subtle);
}

.suggestion-price {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--accent-gold);
  margin-bottom: 0.4rem;
}

.suggestion-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.source-link {
  font-size: 0.72rem;
  color: var(--text-muted);
  text-decoration: none;
  display: flex;
  align-items: center;
  gap: 0.2rem;
  transition: color var(--transition-fast);
}

.source-link:hover {
  color: var(--accent-gold);
}

.add-btn {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.72rem;
  padding: 0.3rem 0.6rem;
  flex-shrink: 0;
}

@media (max-width: 640px) {
  .suggestion-card {
    flex-direction: column;
  }

  .suggestion-img {
    width: 100%;
    height: 120px;
  }
}
</style>
