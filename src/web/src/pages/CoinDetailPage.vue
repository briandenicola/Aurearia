<template>
  <div class="container">
    <div v-if="store.loading && !coin" class="loading-overlay">
      <div class="spinner"></div>
    </div>

    <div v-else-if="coin" class="coin-detail">
      <CoinDetailHeaderActions
        :is-wishlist="coin.isWishlist"
        :is-sold="coin.isSold"
        :coin-id="coin.id"
        @purchase="showPurchaseModal = true"
        @sell="showSellModal = true"
        @delete="handleDelete"
      />

      <div class="detail-layout">
        <!-- Images -->
        <div class="detail-images">
          <ImageGallery :images="coin.images || []" :processing="removingBg" @remove-bg="handleRemoveBackground" @delete-image="handleDeleteImage" />
        </div>

        <!-- Info -->
        <div class="detail-info">
          <div class="detail-title-section">
            <h1>{{ coin.name }}</h1>
            <p v-if="coin.ruler" class="detail-ruler">{{ coin.ruler }}</p>
          </div>

          <div v-if="coin.obverseInscription || coin.reverseInscription" class="inscriptions-section">
            <h3>Inscriptions</h3>
            <div v-if="coin.obverseInscription" class="inscription">
              <span class="inscription-label">Obverse:</span>
              <span class="inscription-text">{{ coin.obverseInscription }}</span>
            </div>
            <div v-if="coin.reverseInscription" class="inscription">
              <span class="inscription-label">Reverse:</span>
              <span class="inscription-text">{{ coin.reverseInscription }}</span>
            </div>
          </div>

          <div class="detail-tags-section">
            <div class="detail-tags">
              <span class="badge" :class="`badge-${coin.category.toLowerCase()}`">{{ coin.category }}</span>
              <span
                v-for="tag in coin.tags ?? []"
                :key="tag.id"
                class="detail-tag-chip"
                :style="{ backgroundColor: tag.color + '22', color: tag.color, borderColor: tag.color + '44' }"
              >
                {{ tag.name }}
                <button class="tag-remove" @click="handleRemoveTag(tag.id)">x</button>
              </span>
              <button v-if="!showTagPicker" class="btn-tag-add" @click="showTagPicker = true">+ Tag</button>
            </div>
            <div v-if="showTagPicker" class="tag-picker">
              <select v-model="tagToAdd" class="tag-picker-select" @change="handleAddTag">
                <option value="" disabled>Select a tag...</option>
                <option
                  v-for="tag in availableTags"
                  :key="tag.id"
                  :value="tag.id"
                >{{ tag.name }}</option>
              </select>
              <button class="btn-tag-cancel" @click="showTagPicker = false">Cancel</button>
            </div>
          </div>

          <div v-if="coin.purchaseDate || coin.purchaseLocation || coin.referenceUrl" class="purchase-meta">
            <span v-if="coin.purchaseDate">Purchased {{ new Date(coin.purchaseDate).toLocaleDateString() }}</span>
            <span v-if="coin.purchaseDate && coin.purchaseLocation"> from </span>
            <span v-if="!coin.purchaseDate && coin.purchaseLocation">Purchased from </span>
            <span v-if="coin.purchaseLocation">{{ coin.purchaseLocation }}</span>
            <a v-if="coin.referenceUrl" :href="coin.referenceUrl" target="_blank" rel="noopener" class="store-link">
              {{ coin.referenceText || 'Store Link' }} ↗
            </a>
          </div>

          <div class="info-grid">
            <div class="info-card" v-if="coin.purchasePrice">
              <span class="info-label">Purchase Price</span>
              <span class="info-value">{{ formatCurrency(coin.purchasePrice) }}</span>
            </div>
            <div class="info-card" v-if="coin.currentValue">
              <span class="info-label">Current Value</span>
              <span class="info-value gold">{{ formatCurrency(coin.currentValue) }}</span>
            </div>
            <div class="info-card" v-if="coin.denomination">
              <span class="info-label">Denomination</span>
              <span class="info-value">{{ coin.denomination }}</span>
            </div>
            <div class="info-card" v-if="coin.era">
              <span class="info-label">Era</span>
              <span class="info-value">{{ coin.era }}</span>
            </div>
            <div class="info-card" v-if="coin.mint">
              <span class="info-label">Mint</span>
              <span class="info-value">{{ coin.mint }}</span>
            </div>
            <div class="info-card" v-if="coin.material">
              <span class="info-label">Material</span>
              <span class="info-value" :class="`material-${coin.material.toLowerCase()}`">{{ coin.material }}</span>
            </div>
            <div class="info-card" v-if="coin.weightGrams">
              <span class="info-label">Weight</span>
              <span class="info-value">{{ coin.weightGrams }}g</span>
            </div>
            <div class="info-card" v-if="coin.diameterMm">
              <span class="info-label">Diameter</span>
              <span class="info-value">{{ coin.diameterMm }}mm</span>
            </div>
            <div class="info-card" v-if="coin.grade">
              <span class="info-label">Grade</span>
              <span class="info-value gold">{{ coin.grade }}</span>
            </div>
            <div class="info-card" v-if="coin.rarityRating">
              <span class="info-label">Rarity / RIC</span>
              <span class="info-value">{{ coin.rarityRating }}</span>
            </div>
          </div>

          <div v-if="coin.obverseDescription || coin.reverseDescription" class="descriptions-section">
            <h3>Description</h3>
            <p v-if="coin.obverseDescription"><strong>Obverse:</strong> {{ coin.obverseDescription }}</p>
            <p v-if="coin.reverseDescription"><strong>Reverse:</strong> {{ coin.reverseDescription }}</p>
          </div>

          <div class="estimate-section">
            <div class="estimate-header">
              <h3>AI Value Estimate</h3>
              <button
                class="btn btn-secondary btn-sm"
                :disabled="estimating"
                @click="handleEstimateValue"
              >
                {{ estimating ? 'Researching...' : 'Estimate Value' }}
              </button>
            </div>
            <div v-if="estimating" class="estimate-loading">
              <div class="spinner" />
              <span>Researching current market value...</span>
            </div>
            <div v-if="estimateError" class="estimate-error">{{ estimateError }}</div>
            <div v-if="valueEstimate" class="estimate-result">
              <div class="estimate-value-row">
                <span class="estimate-value">{{ valueEstimate.estimatedValue ? formatCurrency(valueEstimate.estimatedValue) : 'N/A' }}</span>
                <span :class="['confidence-badge', `confidence-${valueEstimate.confidence}`]">
                  {{ valueEstimate.confidence }} confidence
                </span>
              </div>
              <p class="estimate-reasoning">{{ valueEstimate.reasoning }}</p>
              <div v-if="valueEstimate.comparables?.length" class="estimate-comparables">
                <h4>Comparable Listings</h4>
                <div v-for="(comp, i) in valueEstimate.comparables" :key="i" class="comparable-item">
                  <a v-if="comp.url" :href="comp.url" target="_blank" rel="noopener" class="comparable-source">{{ comp.source }}</a>
                  <span v-else class="comparable-source">{{ comp.source }}</span>
                  <span class="comparable-price">{{ comp.price }}</span>
                </div>
              </div>
              <div class="estimate-actions">
                <button class="btn btn-primary btn-sm" @click="applyEstimate">
                  Apply as Current Value
                </button>
                <button class="btn btn-ghost btn-sm" @click="valueEstimate = null">
                  Dismiss
                </button>
              </div>
            </div>
          </div>

          <CoinNumistaPanel
            :coin-name="coin.name"
            :coin-ruler="coin.ruler"
            :coin-denomination="coin.denomination"
          />

          <div v-if="coin.notes" class="notes-section">
            <h3>Notes</h3>
            <p>{{ coin.notes }}</p>
          </div>

          <CoinActivityJournal
            :entries="journalEntries"
            :coin-id="coin.id"
            @add="handleAddJournalEntry"
            @delete="handleDeleteJournalEntry"
          />

          <div v-if="coin.listingStatus" class="listing-status-card">
            <div class="listing-status-header">
              <span
                class="listing-status-badge"
                :class="{
                  'listing-available': coin.listingStatus === 'available',
                  'listing-unavailable': coin.listingStatus === 'unavailable',
                  'listing-unknown': coin.listingStatus === 'unknown',
                }"
              >{{ coin.listingStatus === 'available' ? 'Available' : coin.listingStatus === 'unavailable' ? 'Unavailable' : 'Unknown' }}</span>
              <button class="btn btn-ghost btn-xs" @click="dismissListingStatus">Dismiss</button>
            </div>
            <p v-if="coin.listingCheckReason" class="listing-reason">{{ coin.listingCheckReason }}</p>
            <p v-if="coin.listingCheckedAt" class="listing-checked-at">Last checked: {{ formatListingDate(coin.listingCheckedAt) }}</p>
          </div>
        </div>

        <!-- Upload Images -->
        <div class="detail-upload">
          <div class="image-upload-section">
            <h4>Upload Images</h4>
            <div class="upload-row">
              <select v-model="uploadType" class="form-select upload-select">
                <option value="obverse">Obverse</option>
                <option value="reverse">Reverse</option>
                <option value="detail">Detail</option>
                <option value="other">Other</option>
              </select>
              <label class="btn btn-secondary btn-sm upload-btn">
                Choose File
                <input type="file" accept="image/*" hidden @change="handleImageUpload" />
              </label>
              <label v-if="isPwa" class="btn btn-secondary btn-sm upload-btn camera-btn">
                <Camera :size="14" /> Photo
                <input type="file" accept="image/*" capture="environment" hidden @change="handleImageUpload" />
              </label>
            </div>

            <div class="url-upload-row">
              <input
                v-model="imageUrl"
                type="url"
                class="form-input url-input"
                placeholder="Or paste an image URL..."
                @keydown.enter="handleUrlUpload"
              />
              <button
                class="btn btn-secondary btn-sm"
                :disabled="!imageUrl || urlLoading"
                @click="handleUrlUpload"
              >
                {{ urlLoading ? 'Fetching...' : 'Fetch' }}
              </button>
            </div>

            <p v-if="uploadStatus" class="upload-status" :class="{ error: uploadError }">{{ uploadStatus }}</p>
          </div>
        </div>

        <!-- AI Analysis -->
        <div class="detail-ai">
          <div class="ai-analysis-section">
            <div class="ai-analysis-header">
              <h3>AI Analysis</h3>
              <div class="ai-buttons">
                <button
                  class="btn btn-primary btn-sm"
                  :disabled="analyzing || !hasObverse || !ollamaAvailable"
                  :title="!ollamaAvailable ? ollamaMessage : !hasObverse ? 'No obverse image' : ''"
                  @click="handleAnalyze('obverse')"
                >
                  {{ analyzingSide === 'obverse' ? 'Analyzing...' : 'Analyze Obverse' }}
                </button>
                <button
                  class="btn btn-primary btn-sm"
                  :disabled="analyzing || !hasReverse || !ollamaAvailable"
                  :title="!ollamaAvailable ? ollamaMessage : !hasReverse ? 'No reverse image' : ''"
                  @click="handleAnalyze('reverse')"
                >
                  {{ analyzingSide === 'reverse' ? 'Analyzing...' : 'Analyze Reverse' }}
                </button>
              </div>
            </div>
            <p v-if="!ollamaAvailable" class="ai-unavailable">AI unavailable — configure Ollama in Admin → AI Configuration</p>

            <div v-if="coin.obverseAnalysis" class="ai-result-section">
              <div class="ai-result-header">
                <h5 class="ai-result-heading">Obverse Analysis</h5>
                <button class="btn btn-ghost btn-xs" @click="handleDeleteAnalysis('obverse')">Remove</button>
              </div>
              <div class="ai-content" v-html="renderedObverse"></div>
            </div>

            <div v-if="coin.reverseAnalysis" class="ai-result-section">
              <div class="ai-result-header">
                <h5 class="ai-result-heading">Reverse Analysis</h5>
                <button class="btn btn-ghost btn-xs" @click="handleDeleteAnalysis('reverse')">Remove</button>
              </div>
              <div class="ai-content" v-html="renderedReverse"></div>
            </div>

            <div v-if="coin.aiAnalysis && !coin.obverseAnalysis && !coin.reverseAnalysis" class="ai-result-section">
              <div class="ai-content" v-html="renderedLegacy"></div>
            </div>

            <p v-if="!coin.obverseAnalysis && !coin.reverseAnalysis && !coin.aiAnalysis && ollamaAvailable" class="ai-empty">
              Upload images and click an analyze button to get an expert assessment.
            </p>
          </div>
        </div>
      </div>
    </div>

    <SellModal v-if="showSellModal && coin" :coin="coin" @close="showSellModal = false" @confirm="confirmSell" />
    <PurchaseModal v-if="showPurchaseModal && coin" :coin="coin" @close="showPurchaseModal = false" @confirm="confirmPurchase" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useCoinsStore } from '@/stores/coins'
import ImageGallery from '@/components/ImageGallery.vue'
import SellModal from '@/components/SellModal.vue'
import PurchaseModal from '@/components/PurchaseModal.vue'
import CoinDetailHeaderActions from '@/components/coin/CoinDetailHeaderActions.vue'
import CoinNumistaPanel from '@/components/coin/CoinNumistaPanel.vue'
import CoinActivityJournal from '@/components/coin/CoinActivityJournal.vue'
import { uploadImage, proxyImage, analyzeCoin, deleteAnalysis, deleteCoin, deleteImage, purchaseCoin, sellCoin, getOllamaStatus, getJournalEntries, addJournalEntry, deleteJournalEntry, estimateCoinValue, updateCoin, getCoinValueHistory, updateListingStatus, getTags, addTagToCoin, removeTagFromCoin } from '@/api/client'
import { removeBackground as removeBg } from '@imgly/background-removal'
import type { CoinImage, CoinJournal, ValueEstimate, CoinValueHistory as CoinValueHistoryType, Tag } from '@/types'
import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'
import { Camera } from 'lucide-vue-next'
import { useDialog } from '@/composables/useDialog'
import { usePwa } from '@/composables/usePwa'

const { showConfirm, showAlert } = useDialog()
const route = useRoute()
const router = useRouter()
const store = useCoinsStore()
const { isPwa } = usePwa()

const uploadType = ref('obverse')
const uploadStatus = ref('')
const uploadError = ref(false)
const imageUrl = ref('')
const urlLoading = ref(false)
const analyzing = ref(false)
const analyzingSide = ref<string | null>(null)
const ollamaAvailable = ref(true)
const ollamaMessage = ref('')
const removingBg = ref(false)
const showSellModal = ref(false)
const showPurchaseModal = ref(false)

// Journal
const journalEntries = ref<CoinJournal[]>([])

// Value Estimation
const estimating = ref(false)
const valueEstimate = ref<ValueEstimate | null>(null)
const estimateError = ref('')
const coinValueHistory = ref<CoinValueHistoryType[]>([])

// Tags
const userTags = ref<Tag[]>([])
const showTagPicker = ref(false)
const tagToAdd = ref('')
const availableTags = computed(() => {
  const coinTagIds = new Set((coin.value?.tags ?? []).map(t => t.id))
  return userTags.value.filter(t => !coinTagIds.has(t.id))
})

async function fetchTagList() {
  try {
    const res = await getTags()
    userTags.value = res.data?.tags ?? []
  } catch { /* ignore */ }
}

async function handleAddTag() {
  if (!tagToAdd.value || !coin.value) return
  try {
    await addTagToCoin(coin.value.id, Number(tagToAdd.value))
    await store.fetchCoin(coin.value.id)
  } catch { /* ignore */ }
  tagToAdd.value = ''
  showTagPicker.value = false
}

async function handleRemoveTag(tagId: number) {
  if (!coin.value) return
  try {
    await removeTagFromCoin(coin.value.id, tagId)
    await store.fetchCoin(coin.value.id)
  } catch { /* ignore */ }
}

const md = new MarkdownIt({ html: false })

const coin = computed(() => store.currentCoin)
const renderedObverse = computed(() => (coin.value?.obverseAnalysis ? DOMPurify.sanitize(md.render(coin.value.obverseAnalysis)) : ''))
const renderedReverse = computed(() => (coin.value?.reverseAnalysis ? DOMPurify.sanitize(md.render(coin.value.reverseAnalysis)) : ''))
const renderedLegacy = computed(() => (coin.value?.aiAnalysis ? DOMPurify.sanitize(md.render(coin.value.aiAnalysis)) : ''))
const hasObverse = computed(() => coin.value?.images?.some(i => i.imageType === 'obverse'))
const hasReverse = computed(() => coin.value?.images?.some(i => i.imageType === 'reverse'))

onMounted(async () => {
  const id = Number(route.params['id'])
  store.fetchCoin(id)
  loadJournal(id)
  loadValueHistory(id)
  fetchTagList()
  try {
    const res = await getOllamaStatus()
    ollamaAvailable.value = res.data.available
    ollamaMessage.value = res.data.message
  } catch {
    ollamaAvailable.value = false
    ollamaMessage.value = 'Unable to check Ollama status'
  }
})

async function loadValueHistory(coinId: number) {
  try {
    const res = await getCoinValueHistory(coinId)
    coinValueHistory.value = res.data || []
  } catch {
    coinValueHistory.value = []
  }
}

async function loadJournal(coinId: number) {
  try {
    const res = await getJournalEntries(coinId)
    journalEntries.value = res.data || []
  } catch {
    journalEntries.value = []
  }
}

async function handleAddJournalEntry(entry: string) {
  if (!coin.value || !entry) return
  try {
    await addJournalEntry(coin.value.id, entry)
    loadJournal(coin.value.id)
  } catch {
    await showAlert('Failed to add journal entry', { title: 'Error' })
  }
}

async function handleDeleteJournalEntry(entryId: number) {
  if (!coin.value) return
  try {
    await deleteJournalEntry(coin.value.id, entryId)
    loadJournal(coin.value.id)
  } catch {
    await showAlert('Failed to delete journal entry', { title: 'Error' })
  }
}

function formatListingDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString(undefined, {
    month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit',
  })
}

async function dismissListingStatus() {
  if (!coin.value) return
  try {
    await updateListingStatus(coin.value.id, '')
    store.fetchCoin(Number(route.params.id))
  } catch {
    // silently fail
  }
}

async function handleImageUpload(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file || !coin.value) return

  uploadStatus.value = 'Uploading...'
  uploadError.value = false

  try {
    await uploadImage(coin.value.id, file, uploadType.value, coin.value.images?.length === 0)
    uploadStatus.value = 'Upload complete!'
    store.fetchCoin(coin.value.id)
  } catch {
    uploadStatus.value = 'Upload failed'
    uploadError.value = true
  }
}

async function handleUrlUpload() {
  if (!imageUrl.value || !coin.value) return

  urlLoading.value = true
  uploadStatus.value = 'Fetching image...'
  uploadError.value = false

  try {
    const imgRes = await proxyImage(imageUrl.value)
    const blob = imgRes.data as Blob
    if (blob.size === 0) {
      uploadStatus.value = 'No image data received from URL'
      uploadError.value = true
      return
    }
    const ext = blob.type.includes('png') ? '.png' : '.jpg'
    const file = new File([blob], `${uploadType.value}${ext}`, { type: blob.type || 'image/jpeg' })
    await uploadImage(coin.value!.id, file, uploadType.value, coin.value!.images?.length === 0)
    uploadStatus.value = 'Image saved from URL!'
    imageUrl.value = ''
    store.fetchCoin(coin.value!.id)
  } catch {
    uploadStatus.value = 'Failed to fetch image from URL'
    uploadError.value = true
  } finally {
    urlLoading.value = false
  }
}

async function handleRemoveBackground(image: CoinImage) {
  if (!coin.value) return
  removingBg.value = true
  uploadStatus.value = ''
  uploadError.value = false

  try {
    // Fetch the original image
    const response = await fetch(`/uploads/${image.filePath}`)
    const srcBlob = await response.blob()

    // Remove background using @imgly/background-removal
    const resultBlob = await removeBg(srcBlob, {
      output: { format: 'image/png', quality: 1 },
    })

    // Upload the processed image with the same type and primary status
    const file = new File([resultBlob], `${image.imageType}-processed.png`, { type: 'image/png' })
    await uploadImage(coin.value.id, file, image.imageType, image.isPrimary)

    // Delete the old image
    await deleteImage(coin.value.id, image.id)

    uploadStatus.value = 'Background removed!'
    store.fetchCoin(coin.value.id)
  } catch (err) {
    console.error('Background removal failed:', err)
    uploadStatus.value = 'Background removal failed'
    uploadError.value = true
  } finally {
    removingBg.value = false
  }
}

async function handleDeleteImage(image: CoinImage) {
  if (!coin.value || !await showConfirm(`Delete this ${image.imageType} image?`, { title: 'Delete Image', variant: 'danger' })) return
  try {
    await deleteImage(coin.value.id, image.id)
    store.fetchCoin(coin.value.id)
  } catch {
    await showAlert('Failed to delete image', { title: 'Error' })
  }
}

async function handleAnalyze(side: 'obverse' | 'reverse') {
  if (!coin.value) return
  analyzing.value = true
  analyzingSide.value = side
  try {
    await analyzeCoin(coin.value.id, side)
    store.fetchCoin(coin.value.id)
  } catch {
    await showAlert(`AI analysis failed for ${side}. Ensure Ollama is running.`, { title: 'Analysis Failed' })
  } finally {
    analyzing.value = false
    analyzingSide.value = null
  }
}

async function confirmPurchase(data: { purchasePrice?: number; purchaseDate?: string; purchaseLocation?: string }) {
  if (!coin.value) return
  try {
    await purchaseCoin(coin.value.id, data)
    showPurchaseModal.value = false
    store.fetchCoin(coin.value.id)
  } catch {
    showPurchaseModal.value = false
  }
}

async function handleDelete() {
  if (!coin.value || !await showConfirm('Delete this coin from your collection?', { title: 'Delete Coin', variant: 'danger' })) return
  await deleteCoin(coin.value.id)
  router.push('/')
}

async function confirmSell(soldPrice: number | null, soldTo: string) {
  if (!coin.value) return
  try {
    await sellCoin(coin.value.id, soldPrice, soldTo)
    showSellModal.value = false
    router.push('/sold')
  } catch {
    await showAlert('Failed to mark as sold', { title: 'Error' })
    showSellModal.value = false
  }
}

async function handleDeleteAnalysis(side: 'obverse' | 'reverse') {
  if (!coin.value || !await showConfirm(`Delete the ${side} analysis?`, { title: 'Delete Analysis', variant: 'danger' })) return
  try {
    await deleteAnalysis(coin.value.id, side)
    store.fetchCoin(coin.value.id)
  } catch {
    await showAlert(`Failed to delete ${side} analysis`, { title: 'Error' })
  }
}

async function handleEstimateValue() {
  if (!coin.value) return
  estimating.value = true
  estimateError.value = ''
  valueEstimate.value = null
  try {
    const res = await estimateCoinValue(coin.value.id)
    const data = res.data
    if (!data || (!data.estimatedValue && !data.reasoning)) {
      estimateError.value = 'No estimate returned from AI'
      return
    }
    valueEstimate.value = data
    // Refresh value history and journal (backend auto-creates entries)
    loadValueHistory(coin.value.id)
    loadJournal(coin.value.id)
  } catch (err: unknown) {
    estimateError.value = err instanceof Error ? err.message : 'Failed to estimate value'
    if (typeof err === 'object' && err !== null && 'response' in err) {
      const axiosErr = err as { response?: { data?: { error?: string } } }
      estimateError.value = axiosErr.response?.data?.error || estimateError.value
    }
  } finally {
    estimating.value = false
  }
}

async function applyEstimate() {
  if (!coin.value || !valueEstimate.value) return
  try {
    await updateCoin(coin.value.id, { currentValue: valueEstimate.value.estimatedValue }, { source: 'estimate' })
    await store.fetchCoin(coin.value.id)
    valueEstimate.value = null
  } catch {
    await showAlert('Failed to update coin value', { title: 'Error' })
  }
}

function formatCurrency(value: number) {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(value)
}
</script>

<style scoped>
.detail-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2rem;
  align-items: start;
  max-width: 1000px;
  margin-left: auto;
  margin-right: auto;
}

.detail-ai {
  grid-column: 1 / -1;
}

.detail-upload {
  grid-column: 1 / -1;
}

.detail-title-section {
  margin-bottom: 1.5rem;
}

.detail-title-section h1 {
  margin-top: 0.5rem;
}

.detail-ruler {
  color: var(--text-secondary);
  font-size: 1.1rem;
  margin-top: 0.25rem;
}

.detail-tags-section {
  margin-bottom: 1rem;
}

.detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  align-items: center;
}

.detail-tag-chip {
  font-size: 0.75rem;
  padding: 0.15rem 0.5rem;
  border-radius: var(--radius-full);
  border: 1px solid;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

.tag-remove {
  background: none;
  border: none;
  color: inherit;
  cursor: pointer;
  font-size: 0.7rem;
  padding: 0;
  opacity: 0.6;
  line-height: 1;
}

.tag-remove:hover {
  opacity: 1;
}

.btn-tag-add {
  background: none;
  border: 1px dashed var(--border-subtle);
  border-radius: var(--radius-full);
  color: var(--text-secondary);
  font-size: 0.75rem;
  padding: 0.15rem 0.5rem;
  cursor: pointer;
}

.btn-tag-add:hover {
  color: var(--text-primary);
  border-color: var(--text-primary);
}

.tag-picker {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  margin-top: 0.5rem;
}

.tag-picker-select {
  padding: 0.3rem 0.5rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text-primary);
  font-size: 0.8rem;
}

.btn-tag-cancel {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 0.75rem;
  cursor: pointer;
}

.purchase-meta {
  margin-bottom: 0.75rem;
  color: var(--text-secondary);
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0.25rem;
}

.store-link {
  color: var(--accent-gold);
  font-size: 0.85rem;
  text-decoration: none;
  margin-left: 0.5rem;
  white-space: nowrap;
}

.store-link:hover {
  text-decoration: underline;
}

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
}

.info-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 0.75rem;
}

.info-label {
  display: block;
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  margin-bottom: 0.2rem;
}

.info-value {
  font-size: 0.95rem;
  font-weight: 500;
}

.info-value.gold {
  color: var(--accent-gold);
}

.inscriptions-section,
.descriptions-section,
.value-section,
.notes-section {
  margin-bottom: 1.5rem;
}

.inscriptions-section h3,
.descriptions-section h3,
.value-section h3,
.notes-section h3 {
  margin-bottom: 0.75rem;
  font-size: 1rem;
}

.inscription {
  margin-bottom: 0.4rem;
}

.inscription-label {
  font-size: 0.8rem;
  color: var(--text-muted);
  margin-right: 0.4rem;
}

.inscription-text {
  font-style: italic;
  color: var(--text-secondary);
}

.descriptions-section p {
  font-size: 0.9rem;
  color: var(--text-secondary);
  margin-bottom: 0.4rem;
}

.value-grid {
  display: flex;
  gap: 1.5rem;
}

.value-item {
  display: flex;
  flex-direction: column;
}

.value-label {
  font-size: 0.75rem;
  color: var(--text-muted);
  text-transform: uppercase;
}

.value-amount {
  font-size: 1.2rem;
  font-weight: 600;
}

.value-amount.gold {
  color: var(--accent-gold);
}

.value-meta {
  margin-top: 0.5rem;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.notes-section p {
  color: var(--text-secondary);
  font-size: 0.9rem;
  white-space: pre-wrap;
}


/* Listing Status Card */
.listing-status-card {
  margin-top: 1rem;
  padding: 0.75rem 1rem;
  background: var(--bg-primary);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}

.listing-status-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.35rem;
}

.listing-status-badge {
  display: inline-block;
  padding: 0.2rem 0.6rem;
  border-radius: var(--radius-full);
  font-size: 0.75rem;
  font-weight: 600;
}

.listing-available {
  background: rgba(46, 204, 113, 0.15);
  color: #2ecc71;
}

.listing-unavailable {
  background: rgba(231, 76, 60, 0.15);
  color: #e74c3c;
}

.listing-unknown {
  background: rgba(241, 196, 15, 0.15);
  color: #f1c40f;
}

.listing-reason {
  font-size: 0.82rem;
  color: var(--text-secondary);
  margin: 0.25rem 0;
}

.listing-checked-at {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin: 0;
}

/* AI Value Estimate — matches Numista section */
.estimate-section {
  margin-bottom: 1.5rem;
}

.estimate-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.estimate-header h3 {
  margin: 0;
  font-size: 1rem;
}

.estimate-loading {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
}

.estimate-loading .spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border-subtle);
  border-top-color: var(--accent-gold);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.estimate-error {
  color: #e74c3c;
  padding: 0.5rem;
  font-size: 0.9rem;
}

.estimate-result {
  padding: 1rem;
  background: var(--bg-card);
  border: 1px solid var(--accent-gold-dim, var(--border-subtle));
  border-radius: var(--radius-sm);
}

.estimate-value-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.estimate-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--accent-gold);
}

.confidence-badge {
  font-size: 0.75rem;
  padding: 0.2rem 0.6rem;
  border-radius: 12px;
  text-transform: uppercase;
  font-weight: 600;
  letter-spacing: 0.03em;
}

.confidence-high {
  background: rgba(46, 204, 113, 0.15);
  color: #2ecc71;
}

.confidence-medium {
  background: rgba(241, 196, 15, 0.15);
  color: #f1c40f;
}

.confidence-low {
  background: rgba(231, 76, 60, 0.15);
  color: #e74c3c;
}

.estimate-reasoning {
  color: var(--text-secondary);
  font-size: 0.9rem;
  line-height: 1.5;
  margin-bottom: 0.75rem;
}

.estimate-comparables {
  margin-bottom: 0.75rem;
}

.estimate-comparables h4 {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin: 0 0 0.5rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.comparable-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.4rem 0;
  border-bottom: 1px solid var(--border-subtle);
}

.comparable-item:last-child {
  border-bottom: none;
}

.comparable-source {
  color: var(--accent-gold);
  text-decoration: none;
  font-size: 0.85rem;
}

.comparable-source:hover {
  text-decoration: underline;
}

.comparable-price {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.85rem;
}

.estimate-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.75rem;
}

/* Image upload */
.image-upload-section {
  margin-top: 1.25rem;
  padding: 1rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}

.image-upload-section h4 {
  font-size: 0.9rem;
  margin-bottom: 0.75rem;
}

.upload-row {
  display: flex;
  gap: 0.5rem;
}

.upload-select {
  flex: 1;
}

.upload-btn {
  white-space: nowrap;
  cursor: pointer;
}

.camera-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
}

.upload-status {
  margin-top: 0.5rem;
  font-size: 0.8rem;
  color: var(--accent-gold);
}

.upload-status.error {
  color: #e74c3c;
}

.url-upload-row {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.url-input {
  flex: 1;
  min-width: 0;
  font-size: 0.82rem;
}

/* AI Analysis — standard section matching Description, Notes, Journal */
.ai-analysis-section {
  margin-bottom: 1.5rem;
}

.ai-analysis-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.ai-analysis-header h3 {
  margin: 0;
  font-size: 1rem;
}

.ai-buttons {
  display: flex;
  gap: 0.4rem;
}

.ai-result-section {
  margin-bottom: 1.25rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-subtle);
}

.ai-result-section:last-of-type {
  border-bottom: none;
  margin-bottom: 0;
  padding-bottom: 0;
}

.ai-result-heading {
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--accent-gold);
  margin-bottom: 0.5rem;
}

.ai-result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.ai-result-header .ai-result-heading {
  margin-bottom: 0;
}

/* btn-ghost and btn-xs are now global in main.css */

.ai-content {
  font-size: 0.85rem;
  line-height: 1.7;
  color: var(--text-secondary);
}

.ai-content :deep(h1),
.ai-content :deep(h2),
.ai-content :deep(h3) {
  color: var(--accent-gold);
  margin-top: 1rem;
  margin-bottom: 0.5rem;
}

.ai-content :deep(strong) {
  color: var(--text-primary);
}

.ai-content :deep(ul),
.ai-content :deep(ol) {
  padding-left: 1.25rem;
}

.ai-empty {
  font-size: 0.85rem;
  color: var(--text-muted);
  font-style: italic;
}

.ai-unavailable {
  font-size: 0.85rem;
  color: #e67e22;
  font-style: italic;
  margin-bottom: 0.5rem;
}

@media (max-width: 768px) {
  .detail-layout {
    grid-template-columns: 1fr;
  }
  .detail-images { order: 1; }
  .detail-info { order: 2; }
  .detail-upload { order: 3; }
  .detail-ai { order: 4; }
  .info-grid {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
