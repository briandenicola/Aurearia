<template>
  <div class="fixed inset-0 z-[1000] flex items-center justify-center bg-overlay p-4" @click.self="emit('close')">
    <div class="card w-full max-w-[480px] !p-0">
      <div class="flex items-center justify-between border-b border-border-subtle px-6 py-5">
        <h3 class="m-0 text-lg font-medium text-heading">Add Auction Lot</h3>
        <button class="rounded-sm p-1 text-text-secondary transition-colors hover:text-text-primary" @click="emit('close')"><X :size="18" /></button>
      </div>

      <div class="p-6">
        <div class="form-group">
          <label class="form-label">Auction Lot URL</label>
          <input
            v-model="url"
            type="url"
            class="form-input"
            placeholder="https://www.numisbids.com/... or https://auctions.cngcoins.com/..."
            :disabled="importing"
          />
          <p class="mt-1.5 text-[0.78rem] text-text-muted">Paste a lot page URL from NumisBids or CNG Auctions</p>
        </div>

        <div v-if="error" class="mt-2 text-body text-loss">{{ error }}</div>

        <div v-if="preview" class="mt-5 overflow-hidden rounded-md border border-border-subtle">
          <div v-if="preview.imageUrl && proxiedImageUrl" class="max-h-[200px] overflow-hidden bg-surface">
            <img :src="proxiedImageUrl" :alt="preview.title" class="h-[200px] w-full object-contain" />
          </div>
          <div class="p-4">
            <h4 class="mb-1 text-[0.95rem] leading-[1.3] font-medium text-text-primary">{{ preview.title }}</h4>
            <p v-if="preview.auctionHouse" class="mb-1 text-[0.82rem] text-text-secondary">{{ preview.auctionHouse }}</p>
            <p v-if="preview.estimate" class="text-body text-text-secondary">Estimate: {{ formatCurrency(preview.estimate) }}</p>
            <p v-if="preview.currentBid" class="text-body font-semibold text-gold">Current Bid: {{ formatCurrency(preview.currentBid) }}</p>
          </div>
        </div>
      </div>

      <div class="flex justify-end gap-3 border-t border-border-subtle px-6 py-4">
        <button class="btn btn-secondary" @click="emit('close')" :disabled="importing">Cancel</button>
        <button class="btn btn-primary inline-flex items-center gap-2" @click="handleImport" :disabled="!url || importing">
          <Loader2 v-if="importing" :size="16" class="animate-spin" />
          {{ importing ? 'Adding...' : 'Add Lot' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { importAuctionLot, scrapeImage } from '@/api/client'
import { useProxiedImage } from '@/composables/useProxiedImage'
import type { AuctionLot } from '@/types'
import { X, Loader2 } from 'lucide-vue-next'
import { formatCurrency } from '@/utils/format'

const emit = defineEmits<{
  close: []
  imported: [lot: AuctionLot]
}>()

const url = ref('')
const importing = ref(false)
const error = ref('')
const preview = ref<AuctionLot | null>(null)
const previewSourceUrl = computed(() => preview.value?.imageUrl ?? '')
const { proxiedImageUrl } = useProxiedImage(previewSourceUrl)
const source = computed(() => {
  try {
    const hostname = new URL(url.value).hostname.toLowerCase()
    if (hostname === 'auctions.cngcoins.com' || hostname.endsWith('.auctions.cngcoins.com')) return 'cng'
  } catch {
    // Invalid URL input; fall back to default source
  }
  return 'numisbids'
})

async function handleImport() {
  if (!url.value) return
  error.value = ''
  importing.value = true

  try {
    let imageUrl = ''
    if (source.value === 'numisbids') {
      try {
        const scraped = await scrapeImage(url.value)
        imageUrl = scraped.data.imageUrl || ''
      } catch { /* scrape is best-effort */ }
    }

    const res = await importAuctionLot({ url: url.value, source: source.value, imageUrl })
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
