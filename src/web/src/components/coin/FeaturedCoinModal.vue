<template>
  <Teleport to="body">
    <div class="modal-overlay" @click.self="close">
      <div class="modal-content card featured-modal">
        <div class="modal-header">
          <h2 class="featured-title">
            <Sparkles :size="20" class="featured-icon" />
            <span>Coin of the Day</span>
          </h2>
          <button class="modal-close" @click="close" title="Close">
            <X :size="18" />
          </button>
        </div>

        <div v-if="loading" class="featured-body featured-loading">
          Loading featured coin...
        </div>

        <div v-else-if="error || !featured" class="featured-body featured-error">
          {{ error || 'Unable to load featured coin' }}
        </div>

        <div v-else class="featured-body">
          <h3 class="featured-coin-name">{{ featured.coin?.name }}</h3>
          <div v-if="featured.coin?.ruler || featured.coin?.era" class="featured-subtitle">
            <span v-if="featured.coin?.ruler">{{ featured.coin?.ruler }}</span>
            <span v-if="featured.coin?.ruler && featured.coin?.era"> &middot; </span>
            <span v-if="featured.coin?.era">{{ featured.coin?.era }}</span>
          </div>

          <div v-if="images.length > 0" class="featured-images">
            <div v-for="img in images" :key="img.id" class="featured-image-wrap">
              <AuthenticatedImage :media-path="img.filePath" :alt="img.imageType" class="featured-image" />
              <span class="featured-image-label">{{ formatImageLabel(img.imageType) }}</span>
            </div>
          </div>

          <div v-if="featured.summary" class="featured-summary" v-html="renderedSummary"></div>

          <div class="featured-actions">
            <router-link
              v-if="featured.coin?.id"
              :to="`/coin/${featured.coin.id}`"
              class="btn btn-secondary btn-sm"
              @click="close"
            >
              View Full Coin
            </router-link>
            <button class="btn btn-primary btn-sm" @click="close">Close</button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { Sparkles, X } from 'lucide-vue-next'
import { getFeaturedCoin } from '@/api/client'
import type { FeaturedCoin } from '@/types'
import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'

const props = defineProps<{ featuredCoinId: number }>()
const emit = defineEmits<{ close: [] }>()

const md = new MarkdownIt({ html: false })

const featured = ref<FeaturedCoin | null>(null)
const loading = ref(false)
const error = ref('')

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

function close() {
  emit('close')
}

onMounted(load)
watch(() => props.featuredCoinId, load)
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.featured-modal {
  width: 100%;
  max-width: 640px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-accent);
  box-shadow: var(--shadow-glow);
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--border-subtle);
}

.featured-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  font-family: 'Cinzel', serif;
  font-size: 1.2rem;
  color: var(--accent-gold);
}

.featured-icon {
  color: var(--accent-gold);
}

.modal-close {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: var(--radius-sm);
  transition: color var(--transition-fast), background var(--transition-fast);
}

.modal-close:hover {
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.05);
}

.featured-body {
  padding: 1.25rem;
  overflow-y: auto;
}

.featured-loading,
.featured-error {
  text-align: center;
  color: var(--text-secondary);
  padding: 2rem 1rem;
}

.featured-coin-name {
  font-family: 'Cinzel', serif;
  font-size: 1.2rem;
  color: var(--text-heading);
  margin: 0 0 0.25rem;
}

.featured-subtitle {
  color: var(--text-secondary);
  font-size: 0.85rem;
  margin-bottom: 1rem;
}

.featured-images {
  display: flex;
  gap: 0.75rem;
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
  justify-content: center;
}

.featured-image-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.35rem;
}

.featured-image {
  width: 160px;
  height: 160px;
  object-fit: cover;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-subtle);
  background: var(--bg-input);
}

.featured-image-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
}

.featured-summary {
  color: var(--text-primary);
  font-size: 0.9rem;
  line-height: 1.55;
}

.featured-summary :deep(h1),
.featured-summary :deep(h2),
.featured-summary :deep(h3),
.featured-summary :deep(h4) {
  color: var(--accent-gold);
  margin-top: 1rem;
  margin-bottom: 0.5rem;
}

.featured-summary :deep(h1:first-child),
.featured-summary :deep(h2:first-child),
.featured-summary :deep(h3:first-child),
.featured-summary :deep(h4:first-child) {
  margin-top: 0;
}

.featured-summary :deep(strong) {
  color: var(--text-primary);
  font-weight: 600;
}

.featured-summary :deep(em) {
  font-style: italic;
}

.featured-summary :deep(p) {
  margin: 0 0 0.75rem;
}

.featured-summary :deep(p:last-child) {
  margin-bottom: 0;
}

.featured-summary :deep(ul),
.featured-summary :deep(ol) {
  padding-left: 1.25rem;
  margin: 0.5rem 0;
}

.featured-summary :deep(li) {
  margin-bottom: 0.25rem;
}

.featured-summary :deep(code) {
  background: var(--bg-input);
  padding: 0.1rem 0.3rem;
  border-radius: 3px;
  font-family: monospace;
  font-size: 0.85em;
}

.featured-summary :deep(pre) {
  background: var(--bg-input);
  padding: 0.75rem;
  border-radius: var(--radius-sm);
  overflow-x: auto;
  margin: 0.75rem 0;
}

.featured-summary :deep(blockquote) {
  border-left: 3px solid var(--accent-gold);
  padding-left: 1rem;
  margin: 0.75rem 0;
  color: var(--text-secondary);
  font-style: italic;
}

.featured-summary :deep(hr) {
  border: none;
  border-top: 1px solid var(--border-subtle);
  margin: 1rem 0;
}

.featured-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1.25rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-subtle);
}
</style>
