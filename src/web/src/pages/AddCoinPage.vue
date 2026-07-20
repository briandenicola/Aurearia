<template>
  <div class="container">
    <div class="form-wrapper">
      <div class="page-header">
        <h1>Add Coin</h1>
      </div>

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

        <InlineCameraCapturePanel
          v-if="isPwa"
          ref="cameraPanel"
          :filename-prefix="nextCaptureTarget"
          @captured="handleCameraCapture"
          @upload="triggerFileInput(nextCaptureTarget)"
        >
          <template #before-actions>
            <div class="grid grid-cols-3 gap-2">
              <div
                class="relative min-h-20 overflow-hidden rounded-md border border-border-subtle bg-card transition-colors"
                :class="{
                  'min-h-24': obverseFile,
                  'border-gold bg-gold-glow': nextCaptureTarget === 'obverse',
                }"
              >
                <div v-if="obverseFile" class="relative min-h-24 h-full w-full bg-cover bg-center" :style="{ backgroundImage: `url(${getFileUrl(obverseFile)})` }">
                  <button type="button" class="absolute top-[0.35rem] right-[0.35rem] z-[2] flex h-6 w-6 items-center justify-center rounded-full bg-overlay text-[1.2rem] leading-none text-white transition-colors hover:bg-error-bg" @click="clearCapturedImage('obverse')" aria-label="Clear obverse">×</button>
                </div>
                <div v-else class="flex min-h-20 h-full w-full flex-col items-center justify-center gap-[0.35rem]">
                  <span class="block h-2 w-2 rounded-full transition-colors" :class="nextCaptureTarget === 'obverse' ? 'bg-gold' : 'bg-text-muted'"></span>
                  <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Obverse</span>
                </div>
              </div>

              <div
                class="relative min-h-20 overflow-hidden rounded-md border border-border-subtle bg-card transition-colors"
                :class="{
                  'min-h-24': reverseFile,
                  'border-gold bg-gold-glow': nextCaptureTarget === 'reverse',
                }"
              >
                <div v-if="reverseFile" class="relative min-h-24 h-full w-full bg-cover bg-center" :style="{ backgroundImage: `url(${getFileUrl(reverseFile)})` }">
                  <button type="button" class="absolute top-[0.35rem] right-[0.35rem] z-[2] flex h-6 w-6 items-center justify-center rounded-full bg-overlay text-[1.2rem] leading-none text-white transition-colors hover:bg-error-bg" @click="clearCapturedImage('reverse')" aria-label="Clear reverse">×</button>
                </div>
                <div v-else class="flex min-h-20 h-full w-full flex-col items-center justify-center gap-[0.35rem]">
                  <span class="block h-2 w-2 rounded-full transition-colors" :class="nextCaptureTarget === 'reverse' ? 'bg-gold' : 'bg-text-muted'"></span>
                  <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Reverse</span>
                </div>
              </div>

              <div
                class="relative min-h-20 overflow-hidden rounded-md border border-border-subtle bg-card transition-colors"
                :class="{
                  'min-h-24': cardFile,
                  'border-gold bg-gold-glow': nextCaptureTarget === 'card',
                }"
              >
                <span class="absolute top-1 right-1 z-[2] rounded-sm bg-input px-[0.4rem] py-[0.15rem] text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Opt</span>
                <div v-if="cardFile" class="relative min-h-24 h-full w-full bg-cover bg-center" :style="{ backgroundImage: `url(${getFileUrl(cardFile)})` }">
                  <button type="button" class="absolute top-[0.35rem] right-[0.35rem] z-[2] flex h-6 w-6 items-center justify-center rounded-full bg-overlay text-[1.2rem] leading-none text-white transition-colors hover:bg-error-bg" @click="clearCapturedImage('card')" aria-label="Clear card">×</button>
                </div>
                <div v-else class="flex min-h-20 h-full w-full flex-col items-center justify-center gap-[0.35rem]">
                  <span class="block h-2 w-2 rounded-full transition-colors" :class="nextCaptureTarget === 'card' ? 'bg-gold' : 'bg-text-muted'"></span>
                  <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Card</span>
                </div>
              </div>
            </div>
          </template>

          <template #footer>
            <div class="flex flex-col items-center gap-2">
              <button
                type="button"
                class="btn btn-primary w-full"
                :disabled="intakeLoading || observationImages.length === 0"
                @click="generateDraft"
              >
                Generate Intake Draft
              </button>
              <button
                type="button"
                class="cursor-pointer border-0 bg-transparent px-0 py-1 text-chip text-text-muted transition-colors hover:text-text-secondary"
                @click="switchToManualMode"
              >
                Use manual mode instead
              </button>
            </div>
          </template>
          <p v-if="intakeError" class="mt-[0.6rem] text-chip text-warning">{{ intakeError }}</p>
        </InlineCameraCapturePanel>

        <div v-else class="rounded-md border border-border-subtle bg-card p-4">
          <h2 class="font-display text-xl font-medium text-heading">Upload Photos</h2>
          <p class="mb-3 text-body text-text-secondary">
            Add obverse and reverse photos to generate an intake draft you can review before saving.
          </p>
          <div class="grid gap-3 md:grid-cols-2">
            <label class="grid gap-[0.35rem]">
              <span class="section-label">Obverse Image</span>
              <input class="w-full rounded-sm border border-border-subtle bg-input text-base text-text-primary file:mr-3 file:border-0 file:bg-gold-glow file:px-3 file:py-[0.55rem] file:text-base file:font-medium file:text-text-primary" type="file" accept="image/*" @change="onObservationFile('obverse', $event)">
              <span class="text-sm text-text-secondary">{{ obverseFile?.name ?? 'Not selected' }}</span>
            </label>
            <label class="grid gap-[0.35rem]">
              <span class="section-label">Reverse Image</span>
              <input class="w-full rounded-sm border border-border-subtle bg-input text-base text-text-primary file:mr-3 file:border-0 file:bg-gold-glow file:px-3 file:py-[0.55rem] file:text-base file:font-medium file:text-text-primary" type="file" accept="image/*" @change="onObservationFile('reverse', $event)">
              <span class="text-sm text-text-secondary">{{ reverseFile?.name ?? 'Not selected' }}</span>
            </label>
            <label class="grid gap-[0.35rem] md:col-span-2">
              <span class="section-label">Coin Card (Optional)</span>
              <input class="w-full rounded-sm border border-border-subtle bg-input text-base text-text-primary file:mr-3 file:border-0 file:bg-gold-glow file:px-3 file:py-[0.55rem] file:text-base file:font-medium file:text-text-primary" type="file" accept="image/*,.pdf" @change="onCardFile($event)">
              <span class="text-sm text-text-secondary">{{ cardFile?.name ?? 'Not selected' }}</span>
            </label>
          </div>
          <div class="mt-3">
            <button
              type="button"
              class="btn btn-primary"
              :disabled="intakeLoading || observationImages.length === 0"
              @click="generateDraft"
            >
              {{ intakeLoading ? 'Generating Draft...' : 'Generate Intake Draft' }}
            </button>
          </div>
          <p v-if="intakeError" class="mt-[0.6rem] text-chip text-warning">{{ intakeError }}</p>
        </div>

        <form v-if="draft" class="rounded-md border border-border-subtle bg-card p-4 pb-5" @submit.prevent="confirmDraft">
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
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
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
import { useCoinOptions } from '@/composables/useCoinOptions'
import InlineCameraCapturePanel from '@/components/InlineCameraCapturePanel.vue'

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

const cameraPanel = ref<InstanceType<typeof InlineCameraCapturePanel> | null>(null)
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
    romanImperialFigureId: null,
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

function handleCameraCapture(file: File) {
  const actualTarget = nextCaptureTarget.value

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
  if (!isPwa || mode !== 'agentic') cameraPanel.value?.stopCamera()
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
  cameraPanel.value?.stopCamera()
})
</script>
