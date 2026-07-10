<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-[1000] flex items-center justify-center bg-overlay p-4" @click.self="close">
      <div class="flex max-h-[90vh] w-full max-w-[640px] flex-col overflow-hidden rounded-md border border-border-accent bg-card shadow-glow">
        <div class="flex items-center justify-between border-b border-border-subtle px-5 py-4">
          <h2 class="flex items-center gap-2 font-display text-lg text-gold">
            <Sparkles :size="20" class="text-gold" />
            <span>Coin of the Day</span>
          </h2>
          <button class="rounded-sm p-1 text-text-muted transition-colors hover:bg-white/5 hover:text-text-primary" @click="close" title="Close">
            <X :size="18" />
          </button>
        </div>

        <div v-if="loading" class="overflow-y-auto px-5 py-8 text-center text-text-secondary">
          Loading featured coin...
        </div>

        <div v-else-if="error || !featured" class="overflow-y-auto px-5 py-8 text-center text-text-secondary">
          {{ error || 'Unable to load featured coin' }}
        </div>

        <div v-else class="overflow-y-auto p-5">
          <h3 class="mb-1 font-display text-lg text-heading">{{ featured.coin?.name }}</h3>
          <div v-if="featured.coin?.ruler || featured.coin?.era" class="mb-4 text-body text-text-secondary">
            <span v-if="featured.coin?.ruler">{{ featured.coin?.ruler }}</span>
            <span v-if="featured.coin?.ruler && featured.coin?.era"> &middot; </span>
            <span v-if="featured.coin?.era">{{ featured.coin?.era }}</span>
          </div>

          <div v-if="images.length > 0" class="mb-5 flex flex-wrap justify-center gap-3">
            <div v-for="img in images" :key="img.id" class="flex flex-col items-center gap-[0.35rem]">
              <AuthenticatedImage :media-path="img.filePath" :alt="img.imageType" class="h-40 w-40 rounded-sm border border-border-subtle bg-input object-cover" />
              <span class="section-label !mb-0">{{ formatImageLabel(img.imageType) }}</span>
            </div>
          </div>

          <div
            v-if="featured.summary"
            class="text-base leading-[1.55] text-text-primary [&_blockquote]:my-3 [&_blockquote]:border-l-[3px] [&_blockquote]:border-l-gold [&_blockquote]:pl-4 [&_blockquote]:italic [&_blockquote]:text-text-secondary [&_code]:rounded-[3px] [&_code]:bg-input [&_code]:px-[0.3rem] [&_code]:py-[0.1rem] [&_code]:font-mono [&_code]:text-[0.85em] [&_em]:italic [&_h1]:mb-2 [&_h1]:mt-4 [&_h1]:text-gold [&_h2]:mb-2 [&_h2]:mt-4 [&_h2]:text-gold [&_h3]:mb-2 [&_h3]:mt-4 [&_h3]:text-gold [&_h4]:mb-2 [&_h4]:mt-4 [&_h4]:text-gold [&_hr]:my-4 [&_hr]:border-0 [&_hr]:border-t [&_hr]:border-border-subtle [&_li]:mb-1 [&_ol]:my-2 [&_ol]:pl-5 [&_p]:mb-3 [&_pre]:my-3 [&_pre]:overflow-x-auto [&_pre]:rounded-sm [&_pre]:bg-input [&_pre]:p-3 [&_strong]:font-semibold [&_strong]:text-text-primary [&_ul]:my-2 [&_ul]:pl-5"
            v-html="renderedSummary"
          ></div>

          <div class="mt-5 flex flex-wrap justify-end gap-2 border-t border-border-subtle pt-4">
            <router-link
              v-if="featured.coin?.id"
              :to="`/coin/${featured.coin.id}`"
              class="btn btn-secondary btn-sm"
              @click="close"
            >
              View Full Coin
            </router-link>
            <button
              v-if="featured.coin"
              class="btn btn-secondary btn-sm inline-flex items-center gap-[0.35rem]"
              :disabled="sharing"
              @click="handleShare"
            >
              <Share2 :size="14" />
              {{ sharing ? 'Sharing...' : 'Share' }}
            </button>
            <button class="btn btn-primary btn-sm" @click="close">Close</button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { Share2, Sparkles, X } from 'lucide-vue-next'
import { getFeaturedCoin } from '@/api/client'
import type { FeaturedCoin } from '@/types'
import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'
import { useCoinShareCard } from '@/composables/useCoinShareCard'

const props = defineProps<{ featuredCoinId: number }>()
const emit = defineEmits<{ close: [] }>()

const md = new MarkdownIt({ html: false })

const featured = ref<FeaturedCoin | null>(null)
const loading = ref(false)
const error = ref('')
const { sharing, shareCoinCard } = useCoinShareCard()

const images = computed(() => {
  const all = featured.value?.coin?.images || []
  // Prefer obverse, reverse first
  const order = ['obverse', 'reverse']
  return [...all].sort((a, b) => {
    const ai = order.indexOf(a.imageType)
    const bi = order.indexOf(b.imageType)
    if (ai === -1 && bi === -1) return 0
    if (ai === -1) return 1
    if (bi === -1) return -1
    return ai - bi
  })
})

const renderedSummary = computed(() => {
  const s = featured.value?.summary || ''
  if (!s) return ''
  return DOMPurify.sanitize(md.render(s))
})

function formatImageLabel(type: string) {
  if (!type) return ''
  return type.charAt(0).toUpperCase() + type.slice(1)
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await getFeaturedCoin(props.featuredCoinId)
    featured.value = res.data
  } catch {
    error.value = 'Failed to load featured coin'
  } finally {
    loading.value = false
  }
}

async function handleShare() {
  const coin = featured.value?.coin
  if (!coin) return

  await shareCoinCard(coin, {
    context: {
      heading: 'Coin of the Day',
      summary: featured.value?.summary ?? '',
    },
  })
}

function close() {
  emit('close')
}

onMounted(load)
watch(() => props.featuredCoinId, load)
</script>
