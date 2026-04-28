<template>
  <div class="detail-actions">
    <h2>Actions</h2>

    <div class="section-content-card">
      <div class="upload-section">
        <h3>Upload Images</h3>
        <div class="upload-content">
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
            <button class="btn btn-primary btn-sm" @click="handleApplyEstimate">
              Apply as Current Value
            </button>
            <button class="btn btn-ghost btn-sm" @click="valueEstimate = null">
              Dismiss
            </button>
          </div>
        </div>
      </div>

      <CoinNumistaPanel
        :coin-name="coinName"
        :coin-ruler="coinRuler ?? ''"
        :coin-denomination="coinDenomination ?? ''"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { uploadImage, proxyImage, estimateCoinValue, updateCoin } from '@/api/client'
import { formatCurrency } from '@/utils/format'
import CoinNumistaPanel from '@/components/coin/CoinNumistaPanel.vue'
import { Camera } from 'lucide-vue-next'
import { useDialog } from '@/composables/useDialog'
import type { ValueEstimate } from '@/types'

const props = defineProps<{
  coinId: number
  coinName: string
  coinRuler?: string | null
  coinDenomination?: string | null
  imageCount: number
  isPwa: boolean
}>()

const emit = defineEmits<{
  imagesChanged: []
  estimateApplied: []
}>()

const { showAlert } = useDialog()

const uploadType = ref('obverse')
const uploadStatus = ref('')
const uploadError = ref(false)
const imageUrl = ref('')
const urlLoading = ref(false)
const estimating = ref(false)
const valueEstimate = ref<ValueEstimate | null>(null)
const estimateError = ref('')

async function handleImageUpload(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return

  uploadStatus.value = 'Uploading...'
  uploadError.value = false

  try {
    await uploadImage(props.coinId, file, uploadType.value, props.imageCount === 0)
    uploadStatus.value = 'Upload complete!'
    emit('imagesChanged')
  } catch {
    uploadStatus.value = 'Upload failed'
    uploadError.value = true
  }
}

async function handleUrlUpload() {
  if (!imageUrl.value) return

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
    await uploadImage(props.coinId, file, uploadType.value, props.imageCount === 0)
    uploadStatus.value = 'Image saved from URL!'
    imageUrl.value = ''
    emit('imagesChanged')
  } catch {
    uploadStatus.value = 'Failed to fetch image from URL'
    uploadError.value = true
  } finally {
    urlLoading.value = false
  }
}

async function handleEstimateValue() {
  estimating.value = true
  estimateError.value = ''
  valueEstimate.value = null
  try {
    const res = await estimateCoinValue(props.coinId)
    const data = res.data
    if (!data || (!data.estimatedValue && !data.reasoning)) {
      estimateError.value = 'No estimate returned from AI'
      return
    }
    valueEstimate.value = data
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

async function handleApplyEstimate() {
  if (!valueEstimate.value) return
  try {
    await updateCoin(props.coinId, { currentValue: valueEstimate.value.estimatedValue }, { source: 'estimate' })
    valueEstimate.value = null
    emit('estimateApplied')
  } catch {
    await showAlert('Failed to update coin value', { title: 'Error' })
  }
}
</script>

<style scoped>
.detail-actions h2 {
  margin-bottom: 1.25rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid var(--border-subtle);
}

.upload-section {
  margin-bottom: 1.5rem;
}

.upload-section h3 {
  margin-bottom: 0.75rem;
  font-size: 1rem;
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
</style>
