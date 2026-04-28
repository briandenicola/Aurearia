<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal card">
      <div class="modal-header">
        <h3>Add from NumisBids</h3>
        <button class="btn-close" @click="emit('close')"><X :size="18" /></button>
      </div>

      <div class="modal-body">
        <div class="form-group">
          <label class="form-label">NumisBids Lot URL</label>
          <input
            v-model="url"
            type="url"
            class="form-input"
            placeholder="https://www.numisbids.com/n.php?p=lot&sid=..."
            :disabled="importing"
          />
          <p class="form-hint">Paste the URL of a lot page from numisbids.com</p>
        </div>

        <div v-if="error" class="error-msg">{{ error }}</div>

        <div v-if="preview" class="import-preview">
          <div class="preview-image-container" v-if="preview.imageUrl">
            <img :src="proxiedUrl(preview.imageUrl)" :alt="preview.title" class="preview-image" />
          </div>
          <div class="preview-details">
            <h4>{{ preview.title }}</h4>
            <p v-if="preview.auctionHouse" class="preview-meta">{{ preview.auctionHouse }}</p>
            <p v-if="preview.estimate" class="preview-estimate">Estimate: {{ formatCurrency(preview.estimate) }}</p>
            <p v-if="preview.currentBid" class="preview-bid">Current Bid: {{ formatCurrency(preview.currentBid) }}</p>
          </div>
        </div>
      </div>

      <div class="modal-actions">
        <button class="btn btn-secondary" @click="emit('close')" :disabled="importing">Cancel</button>
        <button class="btn btn-primary" @click="handleImport" :disabled="!url || importing">
          <Loader2 v-if="importing" :size="16" class="spin" />
          {{ importing ? 'Adding...' : 'Add Lot' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { importAuctionLot, scrapeImage } from '@/api/client'
import type { AuctionLot } from '@/types'
import { X, Loader2 } from 'lucide-vue-next'
import { formatCurrency } from '@/utils/format'

const emit = defineEmits<{
  close: []
  imported: [lot: AuctionLot]
}>()

const API_BASE = import.meta.env.VITE_API_BASE_URL || ''
const url = ref('')
const importing = ref(false)
const error = ref('')
const preview = ref<AuctionLot | null>(null)


function proxiedUrl(imageUrl: string): string {
  if (!imageUrl) return ''
  const token = localStorage.getItem('token') ?? ''
  return `${API_BASE}/api/proxy-image?url=${encodeURIComponent(imageUrl)}&token=${encodeURIComponent(token)}`
}

async function handleImport() {
  if (!url.value) return
  error.value = ''
  importing.value = true

  try {
    // Scrape the lot page for an image first
    let imageUrl = ''
    try {
      const scraped = await scrapeImage(url.value)
      imageUrl = scraped.data.imageUrl || ''
    } catch { /* scrape is best-effort */ }

    const res = await importAuctionLot({ url: url.value, imageUrl })
    preview.value = res.data
    emit('imported', res.data)
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    error.value = msg ?? 'Failed to add lot. Please check the URL and try again.'
  } finally {
    importing.value = false
  }
}
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
}

.modal {
  max-width: 480px;
  width: 90%;
  padding: 0;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--border-subtle);
}

.modal-header h3 {
  margin: 0;
  font-size: 1.1rem;
}

.btn-close {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: var(--radius-sm);
  transition: color var(--transition-fast);
}

.btn-close:hover {
  color: var(--text-primary);
}

.modal-body {
  padding: 1.5rem;
}

.form-hint {
  font-size: 0.78rem;
  color: var(--text-muted);
  margin-top: 0.4rem;
}

.error-msg {
  color: #f87171;
  font-size: 0.85rem;
  margin-top: 0.5rem;
}

.import-preview {
  margin-top: 1.25rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.preview-image-container {
  width: 100%;
  max-height: 200px;
  overflow: hidden;
  background: var(--bg-primary);
}

.preview-image {
  width: 100%;
  height: 200px;
  object-fit: contain;
}

.preview-details {
  padding: 1rem;
}

.preview-details h4 {
  font-size: 0.95rem;
  margin-bottom: 0.4rem;
  line-height: 1.3;
}

.preview-meta {
  font-size: 0.82rem;
  color: var(--text-secondary);
  margin-bottom: 0.25rem;
}

.preview-estimate {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.preview-bid {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--accent-gold);
}

.modal-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--border-subtle);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
