<template>
  <div class="flex w-full flex-col gap-2.5">
    <div
      v-for="(coin, j) in suggestions"
      :key="j"
      class="flex overflow-hidden rounded-md border border-border-subtle bg-card transition-colors hover:border-border-accent max-sm:flex-col"
    >
      <div v-if="getSuggestionImageUrl(coin)" class="h-20 w-20 shrink-0 overflow-hidden bg-surface-secondary max-sm:h-[120px] max-sm:w-full">
        <img :src="getSuggestionImageUrl(coin)" :alt="coin.name" class="h-full w-full object-cover" @error="handleImgError" />
      </div>
      <div class="min-w-0 flex-1 p-3">
        <h4 class="mb-1 text-[0.85rem] leading-snug text-text-primary">{{ coin.name }}</h4>
        <p class="mb-1.5 line-clamp-2 text-sm text-text-secondary">{{ coin.description }}</p>
        <div class="mb-1.5 flex flex-wrap gap-1">
          <span v-if="coin.era" class="chip-sm">{{ coin.era }}</span>
          <span v-if="coin.material" class="chip-sm">{{ coin.material }}</span>
          <span v-if="coin.denomination" class="chip-sm">{{ coin.denomination }}</span>
        </div>
        <div v-if="coin.estPrice" class="mb-1.5 text-chip font-semibold text-gold">{{ coin.estPrice }}</div>
        <div class="flex items-center justify-between gap-2 max-sm:flex-wrap">
          <SafeExternalLink
            v-if="coin.sourceUrl"
            :href="coin.sourceUrl"
            target="_blank"
            rel="noopener"
            class="flex items-center gap-1 text-sm text-text-muted no-underline transition-colors hover:text-gold"
          >
            <ExternalLink :size="12" />
            {{ coin.sourceName || 'Source' }}
          </SafeExternalLink>
          <button
            v-if="coin.era || coin.material || coin.denomination"
            class="btn btn-primary btn-sm shrink-0 disabled:cursor-default disabled:opacity-60"
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
