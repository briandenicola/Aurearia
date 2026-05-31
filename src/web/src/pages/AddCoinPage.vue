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
        <div class="intake-card">
          <h2 class="form-section-title">Observe Coin</h2>
          <p class="intake-copy">
            Add obverse and reverse photos to generate an intake draft you can review before saving.
          </p>

          <video
            v-if="isPwa && cameraReady"
            ref="cameraVideo"
            class="camera-preview"
            autoplay
            playsinline
            muted
          />
          <p v-if="isPwa && cameraError" class="status-text status-warning">{{ cameraError }}</p>

          <div v-if="isPwa" class="capture-actions">
            <button type="button" class="btn btn-secondary btn-sm" @click="captureFromCamera('obverse')">
              Capture Obverse
            </button>
            <button type="button" class="btn btn-secondary btn-sm" @click="captureFromCamera('reverse')">
              Capture Reverse
            </button>
          </div>

          <a
            v-if="isPwa"
            href="#"
            class="manual-mode-link"
            @click.prevent="switchToManualMode"
          >
            Use Manual Mode instead
          </a>
        </div>

        <div class="intake-card">
          <h2 class="form-section-title">Upload Photos</h2>
          <div class="upload-grid">
            <label class="upload-field">
              <span class="section-label">Obverse Image</span>
              <input type="file" accept="image/*" @change="onObservationFile('obverse', $event)">
              <span class="file-name">{{ obverseFile?.name || 'Not selected' }}</span>
            </label>
            <label class="upload-field">
              <span class="section-label">Reverse Image</span>
              <input type="file" accept="image/*" @change="onObservationFile('reverse', $event)">
              <span class="file-name">{{ reverseFile?.name || 'Not selected' }}</span>
            </label>
            <label class="upload-field full-width">
              <span class="section-label">Coin Card (Optional)</span>
              <input type="file" accept="image/*,.pdf" @change="onCardFile($event)">
              <span class="file-name">{{ cardFile?.name || 'Not selected' }}</span>
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
                <option v-for="category in CATEGORIES" :key="category" :value="category">{{ category }}</option>
              </select>
            </label>
            <label class="form-group">
              <span class="section-label">Material</span>
              <select v-model="reviewForm.material" class="input">
                <option v-for="material in MATERIALS" :key="material" :value="material">{{ material }}</option>
              </select>
            </label>
            <label class="form-group">
              <span class="section-label">Era</span>
              <select v-model="reviewForm.era" class="input">
                <option value="">Unknown</option>
                <option v-for="era in COIN_ERAS" :key="era" :value="era">{{ era }}</option>
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
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { COIN_ERAS, CATEGORIES, MATERIALS } from '@/types'
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

type EntryMode = 'manual' | 'agentic'
type CaptureTarget = 'obverse' | 'reverse'

const route = useRoute()
const router = useRouter()
const store = useCoinsStore()
const { showAlert } = useDialog()
const { isPwa } = usePwa()

const wishlistDefault = route.query.wishlist === 'true'
const entryMode = ref<EntryMode>(isPwa ? 'agentic' : 'manual')

const saving = ref(false)
const intakeLoading = ref(false)
const committingDraft = ref(false)
const intakeError = ref('')

const obverseFile = ref<File | null>(null)
const reverseFile = ref<File | null>(null)
const cardFile = ref<File | null>(null)

const cameraVideo = ref<HTMLVideoElement | null>(null)
const cameraStream = ref<MediaStream | null>(null)
const cameraError = ref('')

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
    notes: '',
    referenceUrl: '',
    referenceText: 'Store Link',
    isWishlist: wishlistDefault,
  }
}

const form = reactive<Partial<Coin>>(createEmptyForm('Roman', 'Silver'))
const reviewForm = reactive<Partial<Coin>>(createEmptyForm('Other', 'Other'))

const observationImages = computed(() => [obverseFile.value, reverseFile.value].filter(Boolean) as File[])
const cameraReady = computed(() => cameraStream.value !== null)

const confidenceClass = computed(() => {
  const level = draft.value?.confidenceSummary?.overall ?? 'low'
  return `confidence-${level}`
})

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
  return CATEGORIES.includes(value as Category) ? (value as Category) : 'Other'
}

function normalizeMaterial(value: string): Material {
  return MATERIALS.includes(value as Material) ? (value as Material) : 'Other'
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
    notes: source.notes?.trim() || undefined,
    referenceUrl: source.referenceUrl?.trim() || undefined,
    referenceText: source.referenceText?.trim() || undefined,
    isWishlist: source.isWishlist ?? wishlistDefault,
  }
  return payload
}

function applyCoinToTarget(target: Partial<Coin>, value: Partial<Coin>) {
  const defaults = target === form ? createEmptyForm('Roman', 'Silver') : createEmptyForm('Other', 'Other')
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
    if (cameraVideo.value) {
      cameraVideo.value.srcObject = stream
      await cameraVideo.value.play()
    }
  } catch {
    cameraError.value = 'Camera permission was denied. You can still upload images.'
  }
}

function stopCamera() {
  if (!cameraStream.value) return
  for (const track of cameraStream.value.getTracks()) {
    track.stop()
  }
  cameraStream.value = null
}

async function captureFromCamera(target: CaptureTarget) {
  const video = cameraVideo.value
  if (!video || !cameraReady.value || video.videoWidth === 0 || video.videoHeight === 0) {
    cameraError.value = 'Camera is not ready yet. Try again in a moment.'
    return
  }
  const canvas = document.createElement('canvas')
  canvas.width = video.videoWidth
  canvas.height = video.videoHeight
  const context = canvas.getContext('2d')
  if (!context) return
  context.drawImage(video, 0, 0, canvas.width, canvas.height)
  const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.92))
  if (!blob) {
    cameraError.value = 'Could not capture image from camera.'
    return
  }
  const file = new File([blob], `${target}-${Date.now()}.jpg`, { type: 'image/jpeg' })
  if (target === 'obverse') obverseFile.value = file
  if (target === 'reverse') reverseFile.value = file
}

function onObservationFile(target: CaptureTarget, event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0] ?? null
  if (target === 'obverse') obverseFile.value = file
  if (target === 'reverse') reverseFile.value = file
}

function onCardFile(event: Event) {
  cardFile.value = (event.target as HTMLInputElement).files?.[0] ?? null
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
      await uploadImage(coinID, obverseFile.value, 'obverse', true)
    }
    if (reverseFile.value) {
      await uploadImage(coinID, reverseFile.value, 'reverse', false)
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

watch(entryMode, async (mode) => {
  if (isPwa && mode === 'agentic') {
    await startCamera()
    return
  }
  stopCamera()
})

onMounted(async () => {
  if (isPwa && entryMode.value === 'agentic') {
    await startCamera()
  }
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

.camera-preview {
  width: 100%;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-subtle);
  background: #000;
  margin-bottom: 0.75rem;
}

.capture-actions {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.manual-mode-link {
  display: inline-block;
  margin-top: 0.75rem;
  color: var(--accent-gold);
  font-size: 0.8rem;
  text-decoration: underline;
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
  color: #f5c36a;
}

.confidence-chip {
  border: 1px solid var(--border-subtle);
  text-transform: capitalize;
}

.confidence-high {
  border-color: #69b77f;
  color: #69b77f;
}

.confidence-medium {
  border-color: #f0c261;
  color: #f0c261;
}

.confidence-low {
  border-color: #e08d8d;
  color: #e08d8d;
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
