<template>
  <div class="container">
    <div class="form-wrapper">
      <div class="page-header">
        <h1>Add Coin</h1>
      </div>

      <div v-if="!isPwa" class="entry-mode-toggle">
        <button
          type="button"
          class="chip"
          :class="{ active: entryMode === 'manual' }"
          @click="entryMode = 'manual'"
        >
          Manual Mode
        </button>
        <button
          type="button"
          class="chip"
          :class="{ active: entryMode === 'agentic' }"
          @click="entryMode = 'agentic'"
        >
          AI Assist Mode
        </button>
      </div>

      <section v-if="entryMode === 'agentic'" class="agentic-layout">
        <!-- Loading overlay for AI analysis -->
        <div v-if="intakeLoading" class="intake-loading-overlay">
          <div class="loading-card">
            <div class="spinner-container">
              <div class="spinner"></div>
            </div>
            <p class="loading-text">Analyzing your coin…</p>
          </div>
        </div>

        <div v-if="isPwa" class="camera-first-card">
          <div class="camera-container">
            <video
              ref="cameraVideo"
              class="camera-preview"
              v-show="cameraStream !== null"
              autoplay
              playsinline
              muted
              @loadedmetadata="onVideoMetadataLoaded"
            />
            <div v-if="!cameraStream" class="camera-placeholder">
              <Camera :size="48" />
              <p>Start the camera when you're ready.</p>
              <button
                type="button"
                class="btn btn-secondary btn-sm camera-start-btn"
                @click="startCamera"
              >
                <Camera :size="16" />
                Start Camera
              </button>
            </div>
            <div v-if="cameraError" class="camera-error-banner">{{ cameraError }}</div>
            
            <!-- Circular focus-guide overlay (only when camera active) -->
            <div v-if="cameraStream !== null" class="focus-overlay">
              <div class="focus-mask"></div>
              <div class="focus-ring"></div>
              <p class="focus-instruction">Focus one coin in the circle</p>
            </div>
          </div>

          <div class="capture-slots">
            <div class="capture-tile" :class="{ filled: obverseFile, active: nextCaptureTarget === 'obverse' }">
              <div v-if="obverseFile" class="tile-thumbnail" :style="{ backgroundImage: `url(${getFileUrl(obverseFile)})` }">
                <button type="button" class="tile-clear-btn" @click="clearCapturedImage('obverse')" aria-label="Clear obverse">×</button>
              </div>
              <div v-else class="tile-empty">
                <span class="tile-dot"></span>
                <span class="tile-label">Obverse</span>
              </div>
            </div>

            <div class="capture-tile" :class="{ filled: reverseFile, active: nextCaptureTarget === 'reverse' }">
              <div v-if="reverseFile" class="tile-thumbnail" :style="{ backgroundImage: `url(${getFileUrl(reverseFile)})` }">
                <button type="button" class="tile-clear-btn" @click="clearCapturedImage('reverse')" aria-label="Clear reverse">×</button>
              </div>
              <div v-else class="tile-empty">
                <span class="tile-dot"></span>
                <span class="tile-label">Reverse</span>
              </div>
            </div>

            <div class="capture-tile" :class="{ filled: cardFile, active: nextCaptureTarget === 'card' }">
              <span class="tile-opt-badge">Opt</span>
              <div v-if="cardFile" class="tile-thumbnail" :style="{ backgroundImage: `url(${getFileUrl(cardFile)})` }">
                <button type="button" class="tile-clear-btn" @click="clearCapturedImage('card')" aria-label="Clear card">×</button>
              </div>
              <div v-else class="tile-empty">
                <span class="tile-dot"></span>
                <span class="tile-label">Card</span>
              </div>
            </div>
          </div>

          <div class="camera-actions">
            <button
              type="button"
              class="shutter-btn"
              :disabled="!cameraReady"
              @click="captureFromCamera()"
              aria-label="Capture photo"
            >
              <Camera :size="32" />
            </button>
            <button
              type="button"
              class="upload-icon-btn"
              @click="triggerFileInput(nextCaptureTarget)"
              aria-label="Upload from library"
            >
              <Images :size="20" />
            </button>
          </div>

          <div class="camera-footer">
            <button
              type="button"
              class="btn btn-primary"
              :disabled="intakeLoading || observationImages.length === 0"
              @click="generateDraft"
            >
              Generate Intake Draft
            </button>
            <button
              type="button"
              class="manual-mode-link"
              @click="switchToManualMode"
            >
              Use manual mode instead
            </button>
          </div>
          <p v-if="intakeError" class="status-text status-warning">{{ intakeError }}</p>
        </div>

        <div v-else class="intake-card">
          <h2 class="form-section-title">Upload Photos</h2>
          <p class="intake-copy">
            Add obverse and reverse photos to generate an intake draft you can review before saving.
          </p>
          <div class="upload-grid">
            <label class="upload-field">
              <span class="section-label">Obverse Image</span>
              <input type="file" accept="image/*" @change="onObservationFile('obverse', $event)">
              <span class="file-name">{{ obverseFile?.name ?? 'Not selected' }}</span>
            </label>
            <label class="upload-field">
              <span class="section-label">Reverse Image</span>
              <input type="file" accept="image/*" @change="onObservationFile('reverse', $event)">
              <span class="file-name">{{ reverseFile?.name ?? 'Not selected' }}</span>
            </label>
            <label class="upload-field full-width">
              <span class="section-label">Coin Card (Optional)</span>
              <input type="file" accept="image/*,.pdf" @change="onCardFile($event)">
              <span class="file-name">{{ cardFile?.name ?? 'Not selected' }}</span>
            </label>
          </div>
          <div class="draft-actions">
            <button
              type="button"
              class="btn btn-primary"
              :disabled="intakeLoading || observationImages.length === 0"
              @click="generateDraft"
            >
              {{ intakeLoading ? 'Generating Draft...' : 'Generate Intake Draft' }}
            </button>
          </div>
          <p v-if="intakeError" class="status-text status-warning">{{ intakeError }}</p>
        </div>

        <form v-if="draft" class="intake-card review-card" @submit.prevent="confirmDraft">
          <div class="review-header">
            <h2 class="form-section-title">Review Draft</h2>
            <span class="chip-sm confidence-chip" :class="confidenceClass">
              {{ draft.confidenceSummary.overall }} confidence
            </span>
          </div>

          <div class="review-grid">
            <label class="form-group">
              <span class="section-label">Name</span>
              <input v-model="reviewForm.name" class="input" type="text">
            </label>
            <label class="form-group">
              <span class="section-label">Category</span>
              <select v-model="reviewForm.category" class="input">
                <option v-for="category in categoryOptions" :key="category" :value="category">{{ category }}</option>
              </select>
            </label>
            <label class="form-group">
              <span class="section-label">Material</span>
              <select v-model="reviewForm.material" class="input">
                <option v-for="material in materialOptions" :key="material" :value="material">{{ material }}</option>
              </select>
            </label>
            <label class="form-group">
              <span class="section-label">Era</span>
              <select v-model="reviewForm.era" class="input">
                <option value="">Unknown</option>
                <option v-for="era in eraOptions" :key="era" :value="era">{{ era }}</option>
              </select>
            </label>
            <label class="form-group">
              <span class="section-label">Denomination</span>
              <input v-model="reviewForm.denomination" class="input" type="text">
            </label>
            <label class="form-group">
              <span class="section-label">Ruler</span>
              <input v-model="reviewForm.ruler" class="input" type="text">
            </label>
            <label class="form-group">
              <span class="section-label">Mint</span>
              <input v-model="reviewForm.mint" class="input" type="text">
            </label>
            <label class="form-group">
              <span class="section-label">Grade</span>
              <input v-model="reviewForm.grade" class="input" type="text">
            </label>
            <label class="form-group">
              <span class="section-label">Weight (g)</span>
              <input v-model.number="reviewForm.weightGrams" class="input" type="number" step="0.01" min="0">
            </label>
            <label class="form-group">
              <span class="section-label">Diameter (mm)</span>
              <input v-model.number="reviewForm.diameterMm" class="input" type="number" step="0.1" min="0">
            </label>
            <label class="form-group">
              <span class="section-label">Purchase Price</span>
              <input v-model.number="reviewForm.purchasePrice" class="input" type="number" step="0.01" min="0">
            </label>
            <label class="form-group">
              <span class="section-label">Current Value</span>
              <input v-model.number="reviewForm.currentValue" class="input" type="number" step="0.01" min="0">
            </label>
            <label class="form-group">
              <span class="section-label">Purchase Date</span>
              <input v-model="reviewForm.purchaseDate" class="input" type="date">
            </label>
            <label class="form-group">
              <span class="section-label">Purchase Location</span>
              <input v-model="reviewForm.purchaseLocation" class="input" type="text">
            </label>
            <label class="form-group full-width">
              <span class="section-label">Obverse Description</span>
              <textarea v-model="reviewForm.obverseDescription" class="input textarea" rows="2"></textarea>
            </label>
            <label class="form-group full-width">
              <span class="section-label">Reverse Description</span>
              <textarea v-model="reviewForm.reverseDescription" class="input textarea" rows="2"></textarea>
            </label>
            <label class="form-group full-width">
              <span class="section-label">Notes</span>
              <textarea v-model="reviewForm.notes" class="input textarea" rows="3"></textarea>
            </label>
          </div>

          <p v-if="draft.unresolvedFields.length > 0" class="status-text">
            Needs review: {{ draft.unresolvedFields.join(', ') }}
          </p>

          <div class="form-actions">
            <button type="button" class="btn btn-secondary" @click="switchToManualMode">
              Use Manual Mode
            </button>
            <button type="submit" class="btn btn-primary" :disabled="committingDraft">
              {{ committingDraft ? 'Saving...' : 'Confirm and Save Coin' }}
            </button>
          </div>
        </form>
      </section>

      <CoinForm
        v-else
        ref="coinFormRef"
        :form="form"
        submit-label="Add to Collection"
        :loading="saving"
        @submit="handleManualSubmit"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { Category, Coin, CoinMutationPayload, IntakeDraft, Material } from '@/types'
import {
  commitIntakeDraft,
  createIntakeDraft,
  extractText,
  updateCoin,
  uploadImage,
} from '@/api/client'
import { useCoinsStore } from '@/stores/coins'
import CoinForm from '@/components/CoinForm.vue'
import { useDialog } from '@/composables/useDialog'
import { usePwa } from '@/composables/usePwa'
import { Camera, Images } from 'lucide-vue-next'
import { useCoinOptions } from '@/composables/useCoinOptions'

type EntryMode = 'manual' | 'agentic'
type CaptureTarget = 'obverse' | 'reverse' | 'card'

const route = useRoute()
const router = useRouter()
const store = useCoinsStore()
const { showAlert } = useDialog()
const { isPwa } = usePwa()
const { categoryOptions, materialOptions, eraOptions, loadOptions } = useCoinOptions()

const wishlistDefault = route.query.wishlist === 'true'
const entryMode = ref<EntryMode>(isPwa ? 'agentic' : 'manual')

const saving = ref(false)
const intakeLoading = ref(false)
const committingDraft = ref(false)
const intakeError = ref('')

const obverseFile = ref<File | null>(null)
const reverseFile = ref<File | null>(null)
const cardFile = ref<File | null>(null)

// Track which images came from camera (for circleClip flag)
const obverseFromCamera = ref(false)
const reverseFromCamera = ref(false)

const cameraVideo = ref<HTMLVideoElement | null>(null)
const cameraStream = ref<MediaStream | null>(null)
const cameraError = ref('')
const videoReady = ref(false)
const nextCaptureTarget = ref<CaptureTarget>('obverse')

const draft = ref<IntakeDraft | null>(null)

const coinFormRef = ref<InstanceType<typeof CoinForm> | null>(null)

function createEmptyForm(category: Category, material: Material): Partial<Coin> {
  return {
    name: '',
    category,
    material,
    denomination: '',
    ruler: '',
    mint: '',
    era: '',
    weightGrams: undefined,
    diameterMm: undefined,
    grade: '',
    obverseInscription: '',
    reverseInscription: '',
    obverseDescription: '',
    reverseDescription: '',
    rarityRating: '',
    purchasePrice: undefined,
    currentValue: undefined,
    purchaseDate: '',
    purchaseLocation: '',
    storageLocationId: null,
    notes: '',
    referenceUrl: '',
    referenceText: 'Store Link',
    isWishlist: wishlistDefault,
  }
}

// Use first option from settings, or fallback to hardcoded defaults
const defaultCategory = computed(() => (categoryOptions.value?.[0] ?? 'Roman') as Category)
const defaultMaterial = computed(() => (materialOptions.value?.[0] ?? 'Silver') as Material)

const form = reactive<Partial<Coin>>(createEmptyForm(defaultCategory.value, defaultMaterial.value))
const reviewForm = reactive<Partial<Coin>>(createEmptyForm('Other', 'Other'))

const observationImages = computed(() => [obverseFile.value, reverseFile.value].filter(Boolean) as File[])
const cameraReady = computed(() => cameraStream.value !== null && videoReady.value)

const confidenceClass = computed(() => {
  const level = draft.value?.confidenceSummary?.overall ?? 'low'
  return `confidence-${level}`
})

function getFileUrl(file: File | null): string {
  return file ? URL.createObjectURL(file) : ''
}

function toRecord(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object') return {}
  return value as Record<string, unknown>
}

function readString(record: Record<string, unknown>, ...keys: string[]): string {
  for (const key of keys) {
    const value = record[key]
    if (typeof value === 'string') return value
  }
  return ''
}

function readNumber(record: Record<string, unknown>, ...keys: string[]): number | undefined {
  for (const key of keys) {
    const value = record[key]
    if (typeof value === 'number' && Number.isFinite(value)) return value
    if (typeof value === 'string' && value.trim() !== '') {
      const numeric = Number(value)
      if (Number.isFinite(numeric)) return numeric
    }
  }
  return undefined
}

function readBoolean(record: Record<string, unknown>, ...keys: string[]): boolean | undefined {
  for (const key of keys) {
    const value = record[key]
    if (typeof value === 'boolean') return value
  }
  return undefined
}

function readDateString(record: Record<string, unknown>, ...keys: string[]): string {
  for (const key of keys) {
    const value = record[key]
    if (typeof value === 'string' && value.length >= 10) return value.slice(0, 10)
  }
  return ''
}

function normalizeCategory(value: string): Category {
  return categoryOptions.value.includes(value as Category) ? (value as Category) : 'Other'
}

function normalizeMaterial(value: string): Material {
  return materialOptions.value.includes(value as Material) ? (value as Material) : 'Other'
}

function normalizeDraftCoin(coin: CoinMutationPayload): Partial<Coin> {
  const source = toRecord(coin)
  return {
    name: readString(source, 'name'),
    category: normalizeCategory(readString(source, 'category')),
    material: normalizeMaterial(readString(source, 'material')),
    denomination: readString(source, 'denomination'),
    ruler: readString(source, 'ruler'),
    mint: readString(source, 'mint'),
    era: readString(source, 'era'),
    weightGrams: readNumber(source, 'weightGrams', 'weight_grams'),
    diameterMm: readNumber(source, 'diameterMm', 'diameter_mm'),
    grade: readString(source, 'grade'),
    obverseInscription: readString(source, 'obverseInscription', 'obverse_inscription'),
    reverseInscription: readString(source, 'reverseInscription', 'reverse_inscription'),
    obverseDescription: readString(source, 'obverseDescription', 'obverse_description'),
    reverseDescription: readString(source, 'reverseDescription', 'reverse_description'),
    rarityRating: readString(source, 'rarityRating', 'rarity_rating'),
    purchasePrice: readNumber(source, 'purchasePrice', 'purchase_price'),
    currentValue: readNumber(source, 'currentValue', 'current_value'),
    purchaseDate: readDateString(source, 'purchaseDate', 'purchase_date'),
    purchaseLocation: readString(source, 'purchaseLocation', 'purchase_location'),
    storageLocationId: readNumber(source, 'storageLocationId', 'storage_location_id') ?? null,
    notes: readString(source, 'notes'),
    referenceUrl: readString(source, 'referenceUrl', 'reference_url'),
    referenceText: readString(source, 'referenceText', 'reference_text') || 'Store Link',
    isWishlist: readBoolean(source, 'isWishlist', 'is_wishlist') ?? wishlistDefault,
  }
}

function buildCoinPayload(source: Partial<Coin>): CoinMutationPayload {
  const payload: CoinMutationPayload = {
    name: source.name?.trim() || 'Untitled Coin',
    category: source.category || 'Other',
    material: source.material || 'Other',
    denomination: source.denomination?.trim() || undefined,
    ruler: source.ruler?.trim() || undefined,
    mint: source.mint?.trim() || undefined,
    era: source.era || undefined,
    weightGrams: source.weightGrams ?? undefined,
    diameterMm: source.diameterMm ?? undefined,
    grade: source.grade?.trim() || undefined,
    obverseInscription: source.obverseInscription?.trim() || undefined,
    reverseInscription: source.reverseInscription?.trim() || undefined,
    obverseDescription: source.obverseDescription?.trim() || undefined,
    reverseDescription: source.reverseDescription?.trim() || undefined,
    rarityRating: source.rarityRating?.trim() || undefined,
    purchasePrice: source.purchasePrice ?? undefined,
    currentValue: source.currentValue ?? undefined,
    purchaseDate: source.purchaseDate || undefined,
    purchaseLocation: source.purchaseLocation?.trim() || undefined,
    storageLocationId: source.storageLocationId ?? null,
    notes: source.notes?.trim() || undefined,
    referenceUrl: source.referenceUrl?.trim() || undefined,
    referenceText: source.referenceText?.trim() || undefined,
    isWishlist: source.isWishlist ?? wishlistDefault,
  }
  return payload
}

function applyCoinToTarget(target: Partial<Coin>, value: Partial<Coin>) {
  const defaults = target === form ? createEmptyForm(defaultCategory.value, defaultMaterial.value) : createEmptyForm('Other', 'Other')
  Object.assign(target, defaults, value)
}

function apiErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error !== null) {
    const e = error as {
      response?: { data?: { error?: string } }
      message?: string
    }
    if (typeof e.response?.data?.error === 'string' && e.response.data.error) return e.response.data.error
    if (typeof e.message === 'string' && e.message) return e.message
  }
  return fallback
}

async function startCamera() {
  if (!isPwa || entryMode.value !== 'agentic') return
  if (cameraStream.value) return
  if (!navigator.mediaDevices?.getUserMedia) {
    cameraError.value = 'Camera access is unavailable on this device.'
    return
  }
  try {
    const stream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: { ideal: 'environment' } },
      audio: false,
    })
    cameraStream.value = stream
    cameraError.value = ''
    videoReady.value = false
    
    // Wait for DOM to update before assigning srcObject
    await nextTick()
    
    if (cameraVideo.value) {
      cameraVideo.value.srcObject = stream
      await cameraVideo.value.play()
    }
  } catch (error) {
    const err = error as { name?: string }
    if (err.name === 'NotAllowedError') {
      cameraError.value = 'Camera permission was denied. You can still upload images.'
    } else if (err.name === 'NotFoundError') {
      cameraError.value = 'No camera found on this device.'
    } else {
      cameraError.value = 'Camera is unavailable. You can still upload images.'
    }
  }
}

function onVideoMetadataLoaded() {
  const video = cameraVideo.value
  if (video && video.videoWidth > 0 && video.videoHeight > 0) {
    videoReady.value = true
  }
}

function stopCamera() {
  if (!cameraStream.value) return
  for (const track of cameraStream.value.getTracks()) {
    track.stop()
  }
  cameraStream.value = null
  videoReady.value = false
}

/**
 * Compute the source rectangle from a raw video frame that corresponds
 * to the displayed portion when using object-fit: cover.
 * Returns { sx, sy, sw, sh } in native video coordinates.
 */
function computeCoverCropRect(
  videoWidth: number,
  videoHeight: number,
  displayWidth: number,
  displayHeight: number
): { sx: number; sy: number; sw: number; sh: number } {
  const videoAspect = videoWidth / (videoHeight || 1)
  const displayAspect = displayWidth / (displayHeight || 1)

  let sx: number, sy: number, sw: number, sh: number

  if (videoAspect > displayAspect) {
    // Video is wider than display → crop horizontally
    sh = videoHeight
    sw = sh * displayAspect
    sy = 0
    sx = (videoWidth - sw) / 2
  } else {
    // Video is taller than display → crop vertically
    sw = videoWidth
    sh = sw / displayAspect
    sx = 0
    sy = (videoHeight - sh) / 2
  }

  return { sx, sy, sw, sh }
}

async function captureFromCamera(target?: CaptureTarget) {
  const video = cameraVideo.value
  if (!video || !cameraReady.value || video.videoWidth === 0 || video.videoHeight === 0) {
    cameraError.value = 'Camera is not ready yet. Try again in a moment.'
    return
  }
  
  const actualTarget = target ?? nextCaptureTarget.value

  // Get displayed video box dimensions (object-fit: cover uses these)
  const displayWidth = video.clientWidth ?? 0
  const displayHeight = video.clientHeight ?? 0
  
  if (displayWidth === 0 || displayHeight === 0) {
    cameraError.value = 'Could not determine video display size.'
    return
  }

  // Compute the cover-crop region that matches what the user sees on screen
  const { sx, sy, sw, sh } = computeCoverCropRect(
    video.videoWidth,
    video.videoHeight,
    displayWidth,
    displayHeight
  )
  
  const canvas = document.createElement('canvas')
  canvas.width = sw
  canvas.height = sh
  const context = canvas.getContext('2d')
  if (!context) return

  // Draw only the displayed region (cover-cropped) to the canvas
  context.drawImage(video, sx, sy, sw, sh, 0, 0, sw, sh)
  
  const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.92))
  if (!blob) {
    cameraError.value = 'Could not capture image from camera.'
    return
  }
  const file = new File([blob], `${actualTarget}-${Date.now()}.jpg`, { type: 'image/jpeg' })
  
  // Store file and mark as camera-captured for circleClip flag
  if (actualTarget === 'obverse') {
    obverseFile.value = file
    obverseFromCamera.value = true
  }
  if (actualTarget === 'reverse') {
    reverseFile.value = file
    reverseFromCamera.value = true
  }
  if (actualTarget === 'card') {
    cardFile.value = file
  }
  
  // Update next capture target
  updateNextCaptureTarget()
}

function updateNextCaptureTarget() {
  if (!obverseFile.value) {
    nextCaptureTarget.value = 'obverse'
  } else if (!reverseFile.value) {
    nextCaptureTarget.value = 'reverse'
  } else {
    nextCaptureTarget.value = 'card'
  }
}

function onObservationFile(target: CaptureTarget, event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0] ?? null
  if (target === 'obverse') {
    obverseFile.value = file
    obverseFromCamera.value = false // Manual upload, not camera
  }
  if (target === 'reverse') {
    reverseFile.value = file
    reverseFromCamera.value = false // Manual upload, not camera
  }
  if (target === 'card') cardFile.value = file
  updateNextCaptureTarget()
}

function onCardFile(event: Event) {
  cardFile.value = (event.target as HTMLInputElement).files?.[0] ?? null
  updateNextCaptureTarget()
}

function triggerFileInput(target: CaptureTarget) {
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = 'image/*'
  input.onchange = (e) => onObservationFile(target, e)
  input.click()
}

function clearCapturedImage(target: CaptureTarget) {
  if (target === 'obverse') {
    obverseFile.value = null
    obverseFromCamera.value = false
  }
  if (target === 'reverse') {
    reverseFile.value = null
    reverseFromCamera.value = false
  }
  if (target === 'card') cardFile.value = null
  updateNextCaptureTarget()
}

function switchToManualMode() {
  if (draft.value) {
    applyCoinToTarget(form, reviewForm)
  }
  entryMode.value = 'manual'
}

async function generateDraft() {
  if (observationImages.value.length === 0) {
    intakeError.value = 'Add at least one coin image to continue.'
    return
  }
  intakeLoading.value = true
  intakeError.value = ''
  try {
    const response = await createIntakeDraft(observationImages.value, cardFile.value ?? undefined)
    draft.value = response.data
    applyCoinToTarget(reviewForm, normalizeDraftCoin(response.data.coin))
  } catch (error) {
    intakeError.value = apiErrorMessage(error, 'Failed to generate draft.')
  } finally {
    intakeLoading.value = false
  }
}

async function confirmDraft() {
  if (!draft.value) return
  committingDraft.value = true
  try {
    const response = await commitIntakeDraft({
      draftId: draft.value.draftId,
      confirm: true,
      overrides: buildCoinPayload(reviewForm),
    })
    const coinID = response.data.coinId
    if (obverseFile.value) {
      // Pass circleClip=true ONLY if this obverse was camera-captured
      await uploadImage(coinID, obverseFile.value, 'obverse', true, obverseFromCamera.value)
    }
    if (reverseFile.value) {
      // Pass circleClip=true ONLY if this reverse was camera-captured
      await uploadImage(coinID, reverseFile.value, 'reverse', false, reverseFromCamera.value)
    }
    router.push(`/coin/${coinID}`)
  } catch (error) {
    await showAlert(apiErrorMessage(error, 'Failed to save coin from draft.'), { title: 'Error' })
  } finally {
    committingDraft.value = false
  }
}

async function handleManualSubmit() {
  saving.value = true
  try {
    const coin = await store.addCoin(buildCoinPayload(form))
    const formComp = coinFormRef.value

    if (formComp?.obverseFile) {
      await uploadImage(coin.id, formComp.obverseFile, 'obverse', true)
    }
    if (formComp?.reverseFile) {
      await uploadImage(coin.id, formComp.reverseFile, 'reverse', false)
    }

    if (formComp?.cardFile) {
      try {
        const res = await extractText(formComp.cardFile)
        const extractedText = res.data.text
        if (extractedText) {
          const existingNotes = form.notes || ''
          const updatedNotes = existingNotes
            ? `${existingNotes}\n\n--- Store Card ---\n${extractedText}`
            : `--- Store Card ---\n${extractedText}`
          await updateCoin(coin.id, { notes: updatedNotes })
        }
      } catch {
        console.warn('Card text extraction failed – coin saved without card notes')
      }
    }

    router.push(`/coin/${coin.id}`)
  } catch {
    await showAlert('Failed to add coin', { title: 'Error' })
  } finally {
    saving.value = false
  }
}

watch(entryMode, (mode) => {
  if (!isPwa || mode !== 'agentic') stopCamera()
})

watch([obverseFile, reverseFile, cardFile], () => {
  updateNextCaptureTarget()
})

onMounted(async () => {
  // Load coin property options from settings
  await loadOptions()
  
  updateNextCaptureTarget()
})

onBeforeUnmount(() => {
  stopCamera()
})
</script>

<style scoped>
.entry-mode-toggle {
  display: flex;
  gap: 0.35rem;
  margin-bottom: 1rem;
}

.entry-mode-toggle .chip {
  border: 1px solid var(--border-subtle);
}

.entry-mode-toggle .chip.active {
  border-color: var(--accent-gold);
}

.agentic-layout {
  display: grid;
  gap: 1rem;
  position: relative;
}

.intake-loading-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--overlay-full);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}

.loading-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 2rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  max-width: 20rem;
}

.spinner-container {
  width: 3rem;
  height: 3rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.spinner {
  width: 2.5rem;
  height: 2.5rem;
  border: 3px solid var(--border-subtle);
  border-top-color: var(--accent-gold);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.loading-text {
  margin: 0;
  color: var(--text-primary);
  font-size: 0.9rem;
  text-align: center;
}

.camera-first-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.camera-container {
  position: relative;
  width: 100%;
  aspect-ratio: 4 / 3;
  border-radius: var(--radius-sm);
  overflow: hidden;
  background: #000;
  border: 1px solid var(--border-subtle);
}

.camera-preview {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.camera-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  color: var(--text-muted);
}

.camera-placeholder p {
  margin: 0;
}

.camera-start-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.camera-error-banner {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: var(--error-bg);
  color: #fff;
  padding: 0.5rem 0.75rem;
  font-size: 0.8rem;
  text-align: center;
  z-index: 10;
}

/* Circular focus-guide overlay */
.focus-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
  z-index: 5;
}

.focus-mask {
  position: absolute;
  inset: 0;
  background: radial-gradient(
    circle at 50% 52%,
    transparent 0%,
    transparent 36%,
    rgba(10, 12, 20, 0.2) 37%,
    rgba(10, 12, 20, 0.62) 100%
  );
}

.focus-ring {
  position: absolute;
  top: 52%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 74%;
  max-width: 360px;
  aspect-ratio: 1;
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, 0.55);
}

.focus-instruction {
  position: absolute;
  top: calc(env(safe-area-inset-top) + 20px);
  left: 50%;
  transform: translateX(-50%);
  color: #fff;
  font-size: 0.85rem;
  font-weight: 500;
  text-align: center;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.7);
  margin: 0;
}

.capture-slots {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.5rem;
}

.capture-tile {
  position: relative;
  min-height: 5rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
  overflow: hidden;
  background: var(--bg-card);
  transition: all var(--transition-fast);
}

.capture-tile.active {
  background: var(--accent-gold-glow);
  border-color: var(--accent-gold);
}

.capture-tile.filled {
  min-height: 6rem;
}

.tile-opt-badge {
  position: absolute;
  top: 0.25rem;
  right: 0.25rem;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  background: var(--bg-input);
  padding: 0.15rem 0.4rem;
  border-radius: var(--radius-sm);
  z-index: 2;
}

.tile-thumbnail {
  width: 100%;
  height: 100%;
  min-height: 6rem;
  background-size: cover;
  background-position: center;
  position: relative;
}

.tile-clear-btn {
  position: absolute;
  top: 0.35rem;
  right: 0.35rem;
  width: 1.5rem;
  height: 1.5rem;
  border-radius: 50%;
  background: var(--overlay-dark);
  color: #fff;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.2rem;
  line-height: 1;
  transition: background var(--transition-fast);
  z-index: 2;
}

.tile-clear-btn:hover {
  background: var(--error-bg);
}

.tile-empty {
  width: 100%;
  height: 100%;
  min-height: 5rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
}

.tile-dot {
  display: block;
  width: 0.5rem;
  height: 0.5rem;
  border-radius: 50%;
  background: var(--text-muted);
  transition: background var(--transition-fast);
}

.capture-tile.active .tile-dot {
  background: var(--accent-gold);
}

.tile-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
}

.camera-actions {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.5rem;
  align-items: center;
}

.shutter-btn {
  grid-column: 2;
  justify-self: center;
  width: 4rem;
  height: 4rem;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--accent-gold), var(--accent-bronze));
  border: 2px solid var(--border-white-dim);
  color: #000;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-fast);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.shutter-btn:hover:not(:disabled) {
  transform: scale(1.05);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.shutter-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.upload-icon-btn {
  grid-column: 3;
  justify-self: end;
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 50%;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.upload-icon-btn:hover {
  background: var(--bg-card-hover);
  border-color: var(--accent-gold);
  color: var(--accent-gold);
}

.camera-footer {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  align-items: center;
}

.camera-footer .btn-primary {
  width: 100%;
}

.manual-mode-link {
  background: transparent;
  border: none;
  color: var(--text-muted);
  font-size: 0.8rem;
  font-family: 'Inter', sans-serif;
  cursor: pointer;
  padding: 0.25rem 0;
  transition: color var(--transition-fast);
}

.manual-mode-link:hover {
  color: var(--text-secondary);
}

.intake-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1rem;
}

.intake-copy {
  margin: 0 0 0.75rem;
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.upload-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.upload-field {
  display: grid;
  gap: 0.35rem;
}

.upload-field.full-width {
  grid-column: 1 / -1;
}

.file-name {
  color: var(--text-secondary);
  font-size: 0.75rem;
}

.draft-actions {
  margin-top: 0.75rem;
}

.review-card {
  padding-bottom: 1.25rem;
}

.review-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.review-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.form-group {
  display: grid;
  gap: 0.35rem;
}

.form-group.full-width {
  grid-column: 1 / -1;
}

.section-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
}

.input {
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  padding: 0.55rem 0.65rem;
  font-size: 0.85rem;
}

.textarea {
  resize: vertical;
}

.status-text {
  margin: 0.6rem 0 0;
  color: var(--text-secondary);
  font-size: 0.8rem;
}

.status-warning {
  color: var(--text-warning);
}

.confidence-chip {
  border: 1px solid var(--border-subtle);
  text-transform: capitalize;
}

.confidence-high {
  border-color: var(--confidence-high);
  color: var(--confidence-high);
}

.confidence-medium {
  border-color: var(--confidence-medium);
  color: var(--confidence-medium);
}

.confidence-low {
  border-color: var(--confidence-low);
  color: var(--confidence-low);
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1rem;
}

@media (max-width: 768px) {
  .review-grid,
  .upload-grid {
    grid-template-columns: 1fr;
  }
}
</style>
