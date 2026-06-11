<template>
  <div class="container">
    <div class="page-header">
      <h1>Find Coin</h1>
      <button class="btn btn-secondary" @click="handleBack">
        <ArrowLeft :size="16" />
        Back
      </button>
    </div>

    <!-- Capture State -->
    <div v-if="state === 'capture'" class="lookup-capture">
      <p class="lookup-instructions card">
        Take a photo of the coin or certification slab to identify it. Save only when you want to add the result to your wishlist.
      </p>

      <!-- Image preview grid -->
      <div v-if="capturedImages.length > 0" class="captured-images">
        <div v-for="(img, idx) in capturedImages" :key="idx" class="captured-image-card">
          <img :src="img.preview" alt="Captured coin" />
          <button class="remove-image-btn" @click="removeImage(idx)" title="Remove">
            <X :size="16" />
          </button>
        </div>
      </div>

      <!-- Capture controls -->
      <div class="capture-controls">
        <button class="btn btn-primary btn-large" @click="openCamera">
          <Camera :size="20" />
          Take Photo
        </button>
        <label class="btn btn-secondary btn-large">
          <Upload :size="20" />
          Upload Image
          <input
            ref="fileInput"
            type="file"
            accept="image/*"
            multiple
            style="display: none"
            @change="handleFileUpload"
          />
        </label>
      </div>

      <button
        v-if="capturedImages.length > 0"
        class="btn btn-primary btn-submit"
        @click="handleSubmit"
        :disabled="submitting"
      >
        <span v-if="submitting" class="spinner-sm"></span>
        <Search v-else :size="20" />
        {{ submitting ? 'Analyzing...' : 'Analyze Coin' }}
      </button>
    </div>

    <!-- Analyzing State -->
    <div v-if="state === 'analyzing'" class="lookup-analyzing">
      <div class="analyzing-spinner">
        <div class="spinner"></div>
      </div>
      <h3>Analyzing Images...</h3>
      <p>Extracting coin details and searching for matches</p>
    </div>

    <!-- Results State -->
    <div v-if="state === 'results'" class="lookup-results">
      <div v-if="error" class="error-banner">
        <AlertCircle :size="20" />
        <span>{{ error }}</span>
      </div>

      <div v-if="results" class="results-content">
        <!-- NGC Certification Path (read-only display) -->
        <div v-if="ngcCertNumber" class="result-section card">
          <h3>Extracted Details</h3>
          <div class="details-grid">
            <div v-if="reviewForm.name" class="detail-item">
              <label>Name</label>
              <span>{{ reviewForm.name }}</span>
            </div>
            <div v-if="reviewForm.ruler" class="detail-item">
              <label>Ruler</label>
              <span>{{ reviewForm.ruler }}</span>
            </div>
            <div v-if="reviewForm.denomination" class="detail-item">
              <label>Denomination</label>
              <span>{{ reviewForm.denomination }}</span>
            </div>
            <div v-if="reviewForm.era" class="detail-item">
              <label>Era</label>
              <span>{{ reviewForm.era }}</span>
            </div>
            <div v-if="reviewForm.mint" class="detail-item">
              <label>Mint</label>
              <span>{{ reviewForm.mint }}</span>
            </div>
            <div v-if="reviewForm.material" class="detail-item">
              <label>Material</label>
              <span>{{ reviewForm.material }}</span>
            </div>
          </div>

          <div class="ngc-cert">
            <div class="ngc-cert-header">
              <ShieldCheck :size="20" />
              <span>NGC Certification: {{ ngcCertNumber }}</span>
            </div>
            <a
              :href="ngcLookupUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="btn btn-secondary btn-sm"
            >
              <ExternalLink :size="16" />
              Verify on NGC
            </a>
          </div>

          <!-- Inscriptions -->
          <div v-if="reviewForm.obverseInscription || reviewForm.reverseInscription" class="inscriptions">
            <h4>Inscriptions</h4>
            <div class="inscription-grid">
              <div v-if="reviewForm.obverseInscription" class="inscription-side">
                <label>Obverse</label>
                <p>{{ reviewForm.obverseInscription }}</p>
              </div>
              <div v-if="reviewForm.reverseInscription" class="inscription-side">
                <label>Reverse</label>
                <p>{{ reviewForm.reverseInscription }}</p>
              </div>
            </div>
          </div>

          <!-- Descriptions -->
          <div v-if="reviewForm.obverseDescription || reviewForm.reverseDescription" class="descriptions">
            <h4>Descriptions</h4>
            <div class="description-grid">
              <div v-if="reviewForm.obverseDescription" class="description-side">
                <label>Obverse</label>
                <p>{{ reviewForm.obverseDescription }}</p>
              </div>
              <div v-if="reviewForm.reverseDescription" class="description-side">
                <label>Reverse</label>
                <p>{{ reviewForm.reverseDescription }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Non-NGC Path (editable review form) -->
        <form v-else class="result-section card" @submit.prevent="handleSaveToWishlist">
          <h3>Review Coin Details</h3>

          <div class="review-grid">
            <label class="form-group full-width">
              <span class="section-label">Name</span>
              <input v-model="reviewForm.name" class="input" type="text" required>
            </label>

            <label class="form-group full-width">
              <span class="section-label">Obverse Description</span>
              <textarea v-model="reviewForm.obverseDescription" class="input textarea" rows="3"></textarea>
            </label>

            <label class="form-group full-width">
              <span class="section-label">Reverse Description</span>
              <textarea v-model="reviewForm.reverseDescription" class="input textarea" rows="3"></textarea>
            </label>

            <label class="form-group full-width">
              <span class="section-label">Notes</span>
              <textarea v-model="reviewForm.notes" class="input textarea" rows="3"></textarea>
            </label>
          </div>
        </form>

        <!-- Numista matches -->
        <div v-if="numistaResults && numistaResults.length > 0" class="result-section card">
          <h3>Possible Matches</h3>
          <div class="numista-results">
            <div v-for="match in numistaResults" :key="match.id" class="numista-card card">
              <img
                v-if="match.thumbnail"
                :src="match.thumbnail"
                :alt="match.title"
                class="numista-thumbnail"
              />
              <div class="numista-info">
                <h4>{{ match.title }}</h4>
                <p v-if="match.issuer" class="numista-issuer">{{ match.issuer }}</p>
                <a
                  :href="match.url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="numista-link"
                >
                  <ExternalLink :size="14" />
                  View on Numista
                </a>
              </div>
            </div>
          </div>
        </div>

        <!-- Quick Actions -->
        <div class="result-actions">
          <button class="btn btn-secondary" @click="handleRetake">
            <RotateCcw :size="16" />
            Retake Photo
          </button>
          <button class="btn btn-secondary" @click="handleCancel">
            <X :size="16" />
            Cancel
          </button>
          <button class="btn btn-primary" :disabled="saving" @click="handleSaveToWishlist">
            <span v-if="saving" class="spinner-sm"></span>
            <Bookmark v-else :size="16" />
            {{ saving ? 'Saving...' : 'Save to Wishlist' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Camera Modal -->
    <CameraCaptureModal
      :is-open="showCamera"
      instruction="Center the coin or slab in the frame"
      @close="showCamera = false"
      @captured="handleCameraCaptured"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { createCoin, createCoinReference, createIntakeDraft, lookupCoin, uploadImage } from '@/api/client'
import type { CoinLookupResponse, CoinMutationPayload } from '@/types'
import {
  Camera,
  Upload,
  Search,
  ArrowLeft,
  X,
  AlertCircle,
  ShieldCheck,
  ExternalLink,
  RotateCcw,
  Bookmark,
} from 'lucide-vue-next'
import CameraCaptureModal from '@/components/CameraCaptureModal.vue'

interface CapturedImage {
  file: File
  preview: string
}

type LookupState = 'capture' | 'analyzing' | 'results'

const router = useRouter()

const state = ref<LookupState>('capture')
const capturedImages = ref<CapturedImage[]>([])
const showCamera = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const submitting = ref(false)
const saving = ref(false)
const error = ref('')
const results = ref<CoinLookupResponse | null>(null)

const reviewForm = reactive<CoinMutationPayload>({
  name: '',
  obverseDescription: '',
  reverseDescription: '',
  notes: '',
})

const ngcCertNumber = computed(() => {
  return results.value?.extractedData.ngc?.normalizedCert ?? null
})

const ngcLookupUrl = computed(() => {
  if (results.value?.extractedData.ngc?.lookupURL) return results.value.extractedData.ngc.lookupURL
  if (!ngcCertNumber.value) return ''
  const compactCert = ngcCertNumber.value.replace(/\D/g, '')
  return `https://www.ngccoin.com/certlookup/${encodeURIComponent(compactCert)}/NGCAncients/`
})

const numistaResults = computed(() => results.value?.numistaCandidates ?? [])

function normalizedEra(value: unknown): 'ancient' | 'medieval' | 'modern' | undefined {
  if (typeof value !== 'string') return undefined
  const normalized = value.trim().toLowerCase()
  if (normalized === 'ancient' || normalized === 'medieval' || normalized === 'modern') return normalized
  return undefined
}

function applyDraftToReviewForm(prefilled: CoinMutationPayload) {
  Object.assign(reviewForm, {
    name: prefilled.name || '',
    ruler: prefilled.ruler,
    denomination: prefilled.denomination,
    era: prefilled.era,
    mint: prefilled.mint,
    material: prefilled.material,
    category: prefilled.category,
    obverseInscription: prefilled.obverseInscription,
    reverseInscription: prefilled.reverseInscription,
    obverseDescription: prefilled.obverseDescription || '',
    reverseDescription: prefilled.reverseDescription || '',
    notes: prefilled.notes || prefilled.aiAnalysis || '',
  })
}

function handleBack() {
  if (state.value === 'results') {
    state.value = 'capture'
    results.value = null
    error.value = ''
  } else {
    router.back()
  }
}

function openCamera() {
  showCamera.value = true
}

function handleCameraCaptured(file: File) {
  const preview = URL.createObjectURL(file)
  capturedImages.value.push({ file, preview })
  showCamera.value = false
}

function handleFileUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const files = input.files
  if (!files || files.length === 0) return

  for (let i = 0; i < files.length; i++) {
    const file = files[i]
    if (!file) continue
    const preview = URL.createObjectURL(file)
    capturedImages.value.push({ file, preview })
  }

  // Reset input
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

function removeImage(index: number) {
  const img = capturedImages.value[index]
  if (img) {
    URL.revokeObjectURL(img.preview)
    capturedImages.value.splice(index, 1)
  }
}

async function handleSubmit() {
  if (capturedImages.value.length === 0) return

  submitting.value = true
  error.value = ''
  state.value = 'analyzing'

  try {
    const files = capturedImages.value.map(img => img.file)
    const lookup = await lookupCoin(files)
    results.value = lookup.data

    if (lookup.data.extractedData.ngc) {
      applyDraftToReviewForm(lookup.data.prefilledDraft ?? {})
    } else {
      const intake = await createIntakeDraft(files)
      results.value = {
        ...lookup.data,
        prefilledDraft: intake.data.coin,
      }
      applyDraftToReviewForm(intake.data.coin)
    }

    state.value = 'results'
  } catch (err: unknown) {
    console.error('Lookup failed:', err)
    error.value = err instanceof Error ? err.message : 'Failed to analyze coin'
    state.value = 'results'
  } finally {
    submitting.value = false
  }
}

function handleRetake() {
  // Clean up previews
  for (const img of capturedImages.value) {
    URL.revokeObjectURL(img.preview)
  }
  capturedImages.value = []
  results.value = null
  error.value = ''

  applyDraftToReviewForm({})

  state.value = 'capture'
}

function handleCancel() {
  router.back()
}

async function createWishlistCoinFromLookup() {
  if (!results.value) return

  const payload: CoinMutationPayload = {
    ...reviewForm,
    name: reviewForm.name || 'Untitled Coin',
    category: reviewForm.category || 'Other',
    material: reviewForm.material || 'Other',
    era: normalizedEra(reviewForm.era),
    isWishlist: true,
  }
  const created = await createCoin(payload)
  const coinId = created.data.id

  for (let index = 0; index < capturedImages.value.length; index += 1) {
    const image = capturedImages.value[index]?.file
    if (!image) continue
    const imageType = index === 0 ? 'obverse' : index === 1 ? 'reverse' : 'detail'
    await uploadImage(coinId, image, imageType, index === 0, false)
  }

  for (const reference of results.value.candidateReferences ?? []) {
    await createCoinReference(coinId, reference)
  }
}

async function handleSaveToWishlist() {
  if (saving.value) return
  saving.value = true
  try {
    await createWishlistCoinFromLookup()
    router.push('/wishlist')
  } catch (err: unknown) {
    console.error('Failed to save to wishlist:', err)
    error.value = err instanceof Error ? err.message : 'Failed to save to wishlist'
  } finally {
    saving.value = false
  }
}

onBeforeUnmount(() => {
  for (const img of capturedImages.value) {
    URL.revokeObjectURL(img.preview)
  }
})
</script>

<style scoped>
.container {
  max-width: 900px;
  margin: 0 auto;
  padding: 1.5rem;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.5rem;
}

/* Capture State */
.lookup-capture {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.lookup-instructions {
  color: var(--text-secondary);
  font-size: 0.9rem;
  line-height: 1.5;
}

.captured-images {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 0.75rem;
}

.captured-image-card {
  position: relative;
  aspect-ratio: 1;
  overflow: hidden;
  border: 1px solid var(--border-subtle);
}

.captured-image-card img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.remove-image-btn {
  position: absolute;
  top: 0.5rem;
  right: 0.5rem;
  background: rgba(0, 0, 0, 0.7);
  border: none;
  color: var(--text-primary);
  cursor: pointer;
  padding: 0.35rem;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background var(--transition-fast);
}

.remove-image-btn:hover {
  background: rgba(192, 57, 43, 0.8);
}

.capture-controls {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.btn-large {
  padding: 0.85rem 1.5rem;
  font-size: 1rem;
  flex: 1;
  min-width: 200px;
  justify-content: center;
}

.btn-submit {
  width: 100%;
  padding: 0.85rem 1.5rem;
  font-size: 1rem;
  justify-content: center;
}

/* Analyzing State */
.lookup-analyzing {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  text-align: center;
}

.analyzing-spinner {
  margin-bottom: 1.5rem;
}

.lookup-analyzing h3 {
  margin-bottom: 0.5rem;
}

.lookup-analyzing p {
  color: var(--text-secondary);
  font-size: 0.9rem;
}

/* Results State */
.lookup-results {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.error-banner {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem;
  background: rgba(192, 57, 43, 0.2);
  border: 1px solid rgba(192, 57, 43, 0.3);
  color: var(--cat-byzantine);
  font-size: 0.9rem;
}

.results-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.result-section {
}

.result-section h3 {
  margin-bottom: 1rem;
}

.result-section h4 {
  margin-top: 1.25rem;
  margin-bottom: 0.75rem;
  text-transform: uppercase;
  font-size: 0.7rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: var(--text-muted);
}

.review-grid {
  display: grid;
  gap: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.form-group.full-width {
  grid-column: 1 / -1;
}

.input {
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  color: var(--text-primary);
  padding: 0.6rem;
  font-size: 0.9rem;
  font-family: inherit;
  transition: border-color var(--transition-fast);
}

.input:focus {
  outline: none;
  border-color: var(--accent-gold);
}

.textarea {
  resize: vertical;
  min-height: 4rem;
  font-family: inherit;
  line-height: 1.5;
}

.details-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 1rem;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.detail-item label {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  font-weight: 600;
}

.detail-item span {
  color: var(--text-primary);
  font-size: 0.9rem;
}

.ngc-cert {
  margin-top: 1.25rem;
  padding: 1rem;
  background: var(--bg-input);
  border: 1px solid var(--border-accent);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}

.ngc-cert-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--accent-gold);
  font-weight: 500;
  font-size: 0.9rem;
}

.inscriptions,
.descriptions {
  margin-top: 0.5rem;
}

.inscription-grid,
.description-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1rem;
  margin-top: 0.5rem;
}

.inscription-side,
.description-side {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.inscription-side label,
.description-side label {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  font-weight: 600;
}

.inscription-side p,
.description-side p {
  color: var(--text-secondary);
  font-size: 0.85rem;
  line-height: 1.5;
}

.numista-results {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.numista-card {
  display: flex;
  gap: 1rem;
  transition: border-color var(--transition-fast);
}

.numista-card:hover {
  border-color: var(--border-accent);
}

.numista-thumbnail {
  width: 80px;
  height: 80px;
  object-fit: cover;
  flex-shrink: 0;
}

.numista-info {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  flex: 1;
}

.numista-info h4 {
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0;
  text-transform: none;
  letter-spacing: normal;
}

.numista-issuer {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.numista-link {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.8rem;
  color: var(--accent-gold);
  margin-top: 0.25rem;
}

.result-actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
  padding-top: 0.5rem;
}

.result-actions .btn {
  flex: 1;
  min-width: 150px;
  justify-content: center;
}

/* Mobile responsive */
@media (max-width: 768px) {
  .container {
    padding: 1rem;
  }

  .capture-controls {
    flex-direction: column;
  }

  .btn-large {
    min-width: unset;
    width: 100%;
  }

  .details-grid {
    grid-template-columns: 1fr;
  }

  .inscription-grid,
  .description-grid {
    grid-template-columns: 1fr;
  }

  .ngc-cert {
    flex-direction: column;
    align-items: flex-start;
  }

  .numista-card {
    flex-direction: column;
  }

  .numista-thumbnail {
    width: 100%;
    height: 200px;
  }

  .result-actions {
    flex-direction: column;
  }

  .result-actions .btn {
    width: 100%;
  }
}
</style>
