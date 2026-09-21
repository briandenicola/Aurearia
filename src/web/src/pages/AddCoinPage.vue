<template>
  <div class="container">
    <div class="mx-auto min-w-0 max-w-[900px]">
      <header class="page-header">
        <h1>Add Coin</h1>
      </header>

      <fieldset :disabled="saving || committingDraft || intakeLoading || preparingImage || savedCoinId !== null" class="min-w-0 border-0 p-0">
        <div v-if="!isPwa" class="mb-4 flex gap-[0.35rem]">
          <button
            type="button"
            class="chip border border-border-subtle"
            :class="{ 'border-gold': entryMode === 'manual' }"
            @click="entryMode = 'manual'"
          >
            Manual Mode
          </button>
          <button
            type="button"
            class="chip border border-border-subtle"
            :class="{ 'border-gold': entryMode === 'agentic' }"
            @click="entryMode = 'agentic'"
          >
            AI Assist Mode
          </button>
        </div>

        <section v-if="entryMode === 'agentic'" class="relative grid gap-4">
          <!-- Loading overlay for AI analysis -->
          <div v-if="intakeLoading" class="fixed inset-0 z-[1000] flex items-center justify-center bg-overlay-full backdrop-blur-[4px]">
            <div class="mx-4 flex max-w-[20rem] flex-col items-center gap-4 rounded-md border border-border-subtle bg-card p-8">
              <div class="flex h-12 w-12 items-center justify-center">
                <div class="h-10 w-10 animate-spin rounded-full border-[3px] border-border-subtle border-t-gold"></div>
              </div>
              <p class="m-0 text-center text-base text-text-primary">Analyzing your coin…</p>
            </div>
          </div>

          <CoinLookupCaptureWizard
            v-if="!isPwa || !draft"
            ref="captureWizard"
            purpose="intake"
            :obverse="captureImages.obverse"
            :reverse="captureImages.reverse"
            :notes-image="captureImages.notes"
            notes=""
            :submitting="intakeLoading"
            :preparing-image="preparingImage"
            :upload-error="intakeError"
            @captured="handleCameraCapture"
            @selected="handleGallerySelection"
            @remove="clearCapturedImage"
            @analyze="generateDraft"
            @manual="switchToManualMode"
          />
          <form v-if="draft" class="rounded-md border border-border-subtle bg-card p-4 pb-5" @submit.prevent="confirmDraft">
            <p v-if="intakeWarning" role="status" class="mb-3 text-sm text-warning">{{ intakeWarning }}</p>
            <div class="mb-3 flex items-center justify-between gap-3">
              <h2 class="font-display text-xl font-medium text-heading">Review Draft</h2>
              <span
                class="chip-sm border border-border-subtle capitalize"
                :class="confidenceClass === 'confidence-high' ? 'border-confidence-high text-confidence-high' : confidenceClass === 'confidence-medium' ? 'border-confidence-medium text-confidence-medium' : 'border-confidence-low text-confidence-low'"
              >
                {{ draft.confidenceSummary.overall }} confidence
              </span>
            </div>

            <div class="grid gap-3 md:grid-cols-2">
              <label class="grid gap-[0.35rem]">
                <span class="section-label">Name</span>
                <input v-model="reviewForm.name" class="form-input" type="text">
              </label>
              <label class="grid gap-[0.35rem]">
                <span class="section-label">Category</span>
                <select v-model="reviewForm.category" class="form-select">
                  <option v-for="category in categoryOptions" :key="category" :value="category">{{ category }}</option>
                </select>
              </label>
              <label class="grid gap-[0.35rem]">
                <span class="section-label">Material</span>
                <select v-model="reviewForm.material" class="form-select">
                  <option v-for="material in materialOptions" :key="material" :value="material">{{ material }}</option>
                </select>
              </label>
              <label class="grid gap-[0.35rem]">
                <span class="section-label">Era</span>
                <select v-model="reviewForm.era" class="form-select">
                  <option value="">Unknown</option>
                  <option v-for="era in eraOptions" :key="era" :value="era">{{ era }}</option>
                </select>
              </label>
              <label class="grid gap-[0.35rem]">
                <span class="section-label">Denomination</span>
                <input v-model="reviewForm.denomination" class="form-input" type="text">
              </label>
              <label class="grid gap-[0.35rem]">
                <span class="section-label">Ruler</span>
                <input v-model="reviewForm.ruler" class="form-input" type="text">
              </label>
              <label class="grid gap-[0.35rem]">
                <span class="section-label">Mint</span>
                <input v-model="reviewForm.mint" class="form-input" type="text">
              </label>
              <label class="grid gap-[0.35rem]">
                <span class="section-label">Grade</span>
                <input v-model="reviewForm.grade" class="form-input" type="text">
              </label>
              <label class="grid gap-[0.35rem]">
                <span class="section-label">Weight (g)</span>
                <input v-model.number="reviewForm.weightGrams" class="form-input" type="number" step="0.01" min="0">
              </label>
              <label class="grid gap-[0.35rem]">
                <span class="section-label">Diameter (mm)</span>
                <input v-model.number="reviewForm.diameterMm" class="form-input" type="number" step="0.1" min="0">
              </label>
              <label class="grid gap-[0.35rem]">
                <span class="section-label">Purchase Price</span>
                <input v-model.number="reviewForm.purchasePrice" class="form-input" type="number" step="0.01" min="0">
              </label>
              <label class="grid gap-[0.35rem]">
                <span class="section-label">Current Value</span>
                <input v-model.number="reviewForm.currentValue" class="form-input" type="number" step="0.01" min="0">
              </label>
              <label class="grid gap-[0.35rem]">
                <span class="section-label">Purchase Date</span>
                <input v-model="reviewForm.purchaseDate" class="form-input" type="date">
              </label>
              <label class="grid gap-[0.35rem]">
                <span class="section-label">Purchase Location</span>
                <input v-model="reviewForm.purchaseLocation" class="form-input" type="text">
              </label>
              <label class="grid gap-[0.35rem] md:col-span-2">
                <span class="section-label">Obverse Description</span>
                <textarea v-model="reviewForm.obverseDescription" class="form-input min-h-20 resize-y" rows="2"></textarea>
              </label>
              <label class="grid gap-[0.35rem] md:col-span-2">
                <span class="section-label">Reverse Description</span>
                <textarea v-model="reviewForm.reverseDescription" class="form-input min-h-20 resize-y" rows="2"></textarea>
              </label>
              <label class="grid gap-[0.35rem] md:col-span-2">
                <span class="section-label">Notes</span>
                <textarea v-model="reviewForm.notes" class="form-input min-h-20 resize-y" rows="3"></textarea>
              </label>
            </div>

            <p v-if="draft.unresolvedFields.length > 0" class="mt-[0.6rem] text-chip text-text-secondary">
              Needs review: {{ draft.unresolvedFields.join(', ') }}
            </p>

            <div class="mt-4 flex flex-col-reverse gap-3 md:flex-row md:justify-end">
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
      </fieldset>
      <div v-if="savedCoinId && saveCompletionError" role="alert" class="mt-4 grid gap-3 rounded-md border border-border-subtle bg-card p-4">
        <p>Coin saved. {{ saveCompletionError }}</p>
        <div class="flex flex-wrap gap-3">
          <button class="btn btn-primary" :disabled="saving" @click="retrySavedCoin">
            {{ saving ? 'Uploading...' : 'Retry remaining uploads' }}
          </button>
          <RouterLink class="btn btn-secondary" :to="`/coin/${savedCoinId}`">Open saved coin</RouterLink>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { Category, Coin, CoinLookupImageRole, CoinMutationPayload, IntakeDraft, Material } from '@/types'
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
import { useCoinOptions } from '@/composables/useCoinOptions'
import CoinLookupCaptureWizard from '@/components/coin-lookup/CoinLookupCaptureWizard.vue'
import { normalizeGalleryImage } from '@/utils/galleryImage'

type EntryMode = 'manual' | 'agentic'

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
const intakeWarning = ref('')
const savedCoinId = ref<number | null>(null)
const saveCompletionError = ref('')
type PendingImage = { file: File; face: 'obverse' | 'reverse'; primary: boolean; circleClip: boolean }
const pendingImages: PendingImage[] = []
let pendingCard: File | null = null
let savedNotes = ''

const obverseFile = ref<File | null>(null)
const reverseFile = ref<File | null>(null)
const cardFile = ref<File | null>(null)

// Track which images came from camera (for circleClip flag)
const obverseFromCamera = ref(false)
const reverseFromCamera = ref(false)

const captureWizard = ref<InstanceType<typeof CoinLookupCaptureWizard> | null>(null)
const preparingImage = ref(false)
let imageGeneration = 0
const captureImages = reactive<Record<CoinLookupImageRole, { file: File; preview: string } | null>>({
  obverse: null, reverse: null, notes: null,
})

const draft = ref<IntakeDraft | null>(null)

const coinFormRef = ref<InstanceType<typeof CoinForm> | null>(null)

function createEmptyForm(category: Category, material: Material): Partial<Coin> {
  return {
    name: '',
    category,
    material,
    denomination: '',
    ruler: '',
    romanImperialFigureId: null,
    mint: '',
    era: '',
    dateRange: '',
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
    vendorSku: '',
    vendorInvoice: '',
    storageLocationId: null,
    storageSlot: null,
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
  return categoryOptions.value.includes(value as Category) ? (value as Category) : 'Other'
}

function normalizeMaterial(value: string): Material {
  return materialOptions.value.includes(value as Material) ? (value as Material) : 'Other'
}

function normalizeDraftCoin(coin: CoinMutationPayload): Partial<Coin> {
  const source = toRecord(coin)
  const suggestedEra = readString(source, 'era').trim()
  const era = eraOptions.value.find(option => option.toLowerCase() === suggestedEra.toLowerCase()) ?? ''
  const notes = readString(source, 'notes')
  const unsupportedEra = suggestedEra && !era
  intakeWarning.value = unsupportedEra
    ? `AI suggested era "${suggestedEra}", which is not a configured option. Choose an era or leave it Unknown; the suggestion is preserved in Notes.`
    : ''
  return {
    name: readString(source, 'name'),
    category: normalizeCategory(readString(source, 'category')),
    material: normalizeMaterial(readString(source, 'material')),
    denomination: readString(source, 'denomination'),
    ruler: readString(source, 'ruler'),
    mint: readString(source, 'mint'),
    era,
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
    vendorSku: readString(source, 'vendorSku', 'vendor_sku'),
    vendorInvoice: readString(source, 'vendorInvoice', 'vendor_invoice'),
    storageLocationId: readNumber(source, 'storageLocationId', 'storage_location_id') ?? null,
    storageSlot: readNumber(source, 'storageSlot', 'storage_slot') ?? null,
    notes: unsupportedEra ? [notes, `AI-suggested era: ${suggestedEra}`].filter(Boolean).join('\n\n') : notes,
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
    mintLocationId: source.mintLocationId ?? null,
    era: source.era?.trim() ?? '',
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
    vendorSku: source.vendorSku?.trim() || undefined,
    vendorInvoice: source.vendorInvoice?.trim() || undefined,
    storageLocationId: source.storageLocationId ?? null,
    storageSlot: source.storageSlot ?? null,
    romanImperialFigureId: source.category === 'Roman' ? (source.romanImperialFigureId ?? null) : null,
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

function setCapturedImage(role: CoinLookupImageRole, file: File | null, fromCamera = false) {
  const previous = captureImages[role]
  if (previous) URL.revokeObjectURL(previous.preview)
  captureImages[role] = file ? { file, preview: URL.createObjectURL(file) } : null
  if (role === 'obverse') {
    obverseFile.value = file
    obverseFromCamera.value = fromCamera
  }
  if (role === 'reverse') {
    reverseFile.value = file
    reverseFromCamera.value = fromCamera
  }
  if (role === 'notes') cardFile.value = file
}

function handleCameraCapture(role: CoinLookupImageRole, file: File) {
  imageGeneration += 1
  preparingImage.value = false
  intakeError.value = ''
  setCapturedImage(role, file, true)
}

async function handleGallerySelection(role: CoinLookupImageRole, file: File) {
  const generation = ++imageGeneration
  preparingImage.value = true
  intakeError.value = ''
  try {
    const normalized = await normalizeGalleryImage(file)
    if (generation === imageGeneration) setCapturedImage(role, normalized)
  } catch (error) {
    if (generation === imageGeneration) intakeError.value = apiErrorMessage(error, 'Unable to prepare this image. Try a JPEG or PNG.')
  } finally {
    if (generation === imageGeneration) preparingImage.value = false
  }
}

function clearCapturedImage(role: CoinLookupImageRole) {
  imageGeneration += 1
  preparingImage.value = false
  setCapturedImage(role, null)
}

function switchToManualMode() {
  if (draft.value) {
    applyCoinToTarget(form, reviewForm)
  }
  entryMode.value = 'manual'
}

async function generateDraft() {
  if (intakeLoading.value || preparingImage.value) return
  if (!obverseFile.value) {
    intakeError.value = 'Add an obverse image to continue.'
    return
  }
  intakeLoading.value = true
  intakeError.value = ''
  captureWizard.value?.stopCamera()
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
  if (!draft.value || committingDraft.value || savedCoinId.value !== null) return
  committingDraft.value = true
  try {
    const response = await commitIntakeDraft({
      draftId: draft.value.draftId,
      confirm: true,
      overrides: buildCoinPayload(reviewForm),
    })
    savedCoinId.value = response.data.coinId
    if (obverseFile.value) {
      pendingImages.push({ file: obverseFile.value, face: 'obverse', primary: true, circleClip: obverseFromCamera.value })
    }
    if (reverseFile.value) {
      pendingImages.push({ file: reverseFile.value, face: 'reverse', primary: false, circleClip: reverseFromCamera.value })
    }
    await completeSavedCoin()
  } catch (error) {
    if (savedCoinId.value !== null) {
      saveCompletionError.value = apiErrorMessage(error, 'Image upload failed. Retry the remaining uploads or open the saved coin.')
    } else {
      await showAlert(apiErrorMessage(error, 'Failed to save coin from draft.'), { title: 'Error' })
    }
  } finally {
    committingDraft.value = false
  }
}

async function handleManualSubmit() {
  if (saving.value || savedCoinId.value !== null) return
  saving.value = true
  try {
    const coin = await store.addCoin(buildCoinPayload(form))
    savedCoinId.value = coin.id
    const formComp = coinFormRef.value

    if (formComp?.obverseFile) {
      pendingImages.push({ file: formComp.obverseFile, face: 'obverse', primary: true, circleClip: false })
    }
    if (formComp?.reverseFile) {
      pendingImages.push({ file: formComp.reverseFile, face: 'reverse', primary: false, circleClip: false })
    }

    pendingCard = formComp?.cardFile ?? null
    savedNotes = form.notes || ''
    await completeSavedCoin()
  } catch (error: unknown) {
    const code = (error as { response?: { data?: { code?: string } } })?.response?.data?.code
    if (savedCoinId.value !== null) {
      saveCompletionError.value = apiErrorMessage(error, 'Image or card upload failed. Retry the remaining uploads or open the saved coin.')
    } else if (code === 'slot_occupied') {
      await coinFormRef.value?.refreshStorageOccupancy()
      await showAlert('That tray slot was just taken. Choose another available slot; your form has been preserved.', { title: 'Slot unavailable' })
    } else {
      await showAlert(apiErrorMessage(error, 'Failed to add coin'), { title: 'Error' })
    }
  } finally {
    saving.value = false
  }
}

async function completeSavedCoin() {
  const coinID = savedCoinId.value
  if (coinID === null) return
  for (let image = pendingImages[0]; image; image = pendingImages[0]) {
    await uploadImage(coinID, image.file, image.face, image.primary, image.circleClip)
    pendingImages.shift()
  }
  if (pendingCard) {
    const res = await extractText(pendingCard)
    if (res.data.text) {
      const notes = [savedNotes, `--- Store Card ---\n${res.data.text}`].filter(Boolean).join('\n\n')
      await updateCoin(coinID, { notes })
    }
    pendingCard = null
  }
  saveCompletionError.value = ''
  await router.push(`/coin/${coinID}`)
}

async function retrySavedCoin() {
  if (saving.value) return
  saving.value = true
  try {
    await completeSavedCoin()
  } catch (error) {
    saveCompletionError.value = apiErrorMessage(error, 'Upload failed. Your coin is already saved; retry the remaining uploads.')
  } finally {
    saving.value = false
  }
}

watch(entryMode, (mode) => {
  if (mode !== 'agentic') captureWizard.value?.stopCamera()
})

onMounted(async () => {
  // Load coin property options from settings
  await loadOptions()
})

onBeforeUnmount(() => {
  imageGeneration += 1
  captureWizard.value?.stopCamera()
  for (const image of Object.values(captureImages)) {
    if (image) URL.revokeObjectURL(image.preview)
  }
})
</script>
