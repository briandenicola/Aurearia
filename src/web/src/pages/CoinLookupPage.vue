<template>
  <div class="container">
    <div class="mx-auto min-w-0 max-w-[900px]">
      <div class="page-header">
        <h1>Identify Coin</h1>
        <div class="pwa-actions">
          <RouterLink class="pwa-icon-btn" to="/quick-capture/drafts" title="All drafts" aria-label="All drafts">
            <List :size="22" />
          </RouterLink>
        </div>
      </div>

      <!-- Capture State -->
      <div v-if="state === 'capture'" class="flex flex-col gap-6">
        <CoinLookupCaptureWizard
          ref="captureWizard"
          :obverse="obverseImage"
          :reverse="reverseImage"
          :notes-image="notesImage"
          :notes="captureNotes"
          :submitting="submitting"
          :preparing-image="preparingImage"
          :upload-error="uploadError"
          :deep-analysis-enabled="deepAnalysisEnabled"
          @captured="handleCameraCapture"
          @selected="handleGallerySelection"
          @remove="removeCapturedImage"
          @update:notes="captureNotes = $event"
          @analyze="handleSubmit"
          @deep-analyze="showDeepAnalysisModal = true"
        />
      </div>

      <div v-if="showDeepAnalysisModal" class="fixed inset-0 z-[1000] flex items-center justify-center bg-overlay p-4" @click.self="showDeepAnalysisModal = false">
        <div class="card max-h-[90vh] w-full max-w-[640px] overflow-y-auto p-6">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-lg text-heading">Deep Analysis</h2>
            <AppIconButton title="Close" @click="showDeepAnalysisModal = false">
              <X :size="18" />
            </AppIconButton>
          </div>
          <DeepAnalysisStartPanel
            reuse-captured-evidence
            :initial-obverse-image="obverseImage?.file ?? null"
            :initial-reverse-image="reverseImage?.file ?? null"
            :initial-hint-images="notesImage ? [notesImage.file] : []"
            :initial-notes="captureNotes"
            :submitting="deepIdentification.starting.value"
            :submit-error="deepIdentification.error.value"
            @submit="onDeepAnalysisSubmit"
            @cancel="showDeepAnalysisModal = false"
          />
        </div>
      </div>

      <!-- Analyzing State -->
      <div v-if="state === 'analyzing'" class="flex flex-col items-center justify-center px-8 py-16 text-center">
        <div class="mb-6">
          <div class="spinner"></div>
        </div>
        <h3 class="mb-2 text-lg text-text-primary">Analyzing Images...</h3>
        <p class="text-base text-text-secondary">Extracting minimum draft details and checking for visible NGC data</p>
      </div>

      <!-- Results State -->
      <div v-if="state === 'results'" class="min-w-0 flex flex-col gap-6">
        <div v-if="error" class="flex items-center gap-3 rounded-md border border-[rgba(192,57,43,0.3)] bg-[rgba(192,57,43,0.2)] p-4 text-base text-byzantine">
          <AlertCircle :size="20" />
          <span>{{ error }}</span>
        </div>

        <div v-if="results" class="min-w-0 flex flex-col gap-6">
          <!-- NGC Certification Path -->
          <form v-if="ngcCertNumber" class="card min-w-0 overflow-hidden" @submit.prevent="handleSaveAsDraft">
            <h3 class="mb-4 text-lg text-text-primary">Review Coin Details</h3>
            <div class="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(220px,1fr))]">
              <label class="form-group col-span-full">
                <span class="section-label">Name</span>
                <input v-model="reviewForm.name" class="form-input" type="text" required>
              </label>

              <label class="form-group">
                <span class="section-label">Ruler</span>
                <input v-model="reviewForm.ruler" class="form-input" type="text">
              </label>

              <label class="form-group">
                <span class="section-label">Denomination</span>
                <input v-model="reviewForm.denomination" class="form-input" type="text">
              </label>

              <label class="form-group">
                <span class="section-label">Category</span>
                <input v-model="reviewForm.category" class="form-input" type="text">
              </label>

              <label class="form-group">
                <span class="section-label">Grade</span>
                <input v-model="reviewForm.grade" class="form-input" type="text">
              </label>
            </div>

            <div class="mt-5 flex flex-wrap items-center justify-between gap-4 rounded-sm border border-border-accent bg-input p-4">
              <div class="flex items-center gap-2 text-base font-medium text-gold">
                <ShieldCheck :size="20" />
                <span>NGC Certification: {{ ngcCertNumber }}</span>
              </div>
              <div v-if="ngcForm.grade" class="flex flex-col gap-1">
                <label class="section-label mb-0">NGC Grade</label>
                <span class="text-base text-text-primary">{{ ngcForm.grade }}</span>
              </div>
              <label class="form-group min-w-[220px] flex-1">
                <span class="section-label">NGC Coin Number</span>
                <input v-model="ngcForm.certNumber" class="form-input" type="text">
              </label>
              <SafeExternalLink
                :href="ngcLookupUrl"
                class="btn btn-secondary btn-sm"
              >
                <ExternalLink :size="16" />
                Verify on NGC
              </SafeExternalLink>
            </div>

            <!-- Inscriptions -->
            <div v-if="reviewForm.obverseInscription || reviewForm.reverseInscription" class="mt-2">
              <h4 class="section-label mb-3 block">Inscriptions</h4>
              <div class="mt-2 grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(250px,1fr))]">
                <div v-if="reviewForm.obverseInscription" class="flex flex-col gap-[0.35rem]">
                  <label class="section-label mb-0">Obverse</label>
                  <p class="text-body leading-6 text-text-secondary">{{ reviewForm.obverseInscription }}</p>
                </div>
                <div v-if="reviewForm.reverseInscription" class="flex flex-col gap-[0.35rem]">
                  <label class="section-label mb-0">Reverse</label>
                  <p class="text-body leading-6 text-text-secondary">{{ reviewForm.reverseInscription }}</p>
                </div>
              </div>
            </div>

            <div v-if="aiObservations" class="mt-2">
              <h4 class="section-label mb-3 block">AI Observations</h4>
              <div class="markdown-rendered min-w-0 overflow-hidden rounded-sm border border-border-subtle bg-input p-3 text-body leading-6 text-text-secondary [overflow-wrap:anywhere] [&_ol]:mb-3 [&_p]:mb-3 [&_p:last-child]:mb-0 [&_strong]:font-semibold [&_strong]:text-gold [&_ul]:mb-3" v-html="renderedAiObservations"></div>
            </div>
          </form>

          <!-- Non-NGC Path (editable review form) -->
          <form v-else class="card min-w-0 overflow-hidden" @submit.prevent="handleSaveAsDraft">
            <h3 class="mb-4 text-lg text-text-primary">Review Coin Details</h3>

            <div class="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(220px,1fr))]">
              <label class="form-group col-span-full">
                <span class="section-label">Name</span>
                <input v-model="reviewForm.name" class="form-input" type="text" required>
              </label>

              <label class="form-group">
                <span class="section-label">Ruler</span>
                <input v-model="reviewForm.ruler" class="form-input" type="text">
              </label>

              <label class="form-group">
                <span class="section-label">Denomination</span>
                <input v-model="reviewForm.denomination" class="form-input" type="text">
              </label>

              <label class="form-group">
                <span class="section-label">Category</span>
                <input v-model="reviewForm.category" class="form-input" type="text">
              </label>

              <label class="form-group">
                <span class="section-label">Grade</span>
                <input v-model="reviewForm.grade" class="form-input" type="text">
              </label>

              <div v-if="aiObservations" class="form-group col-span-full">
                <span class="section-label">AI Observations</span>
                <div class="markdown-rendered min-w-0 overflow-hidden rounded-sm border border-border-subtle bg-input p-3 text-body leading-6 text-text-secondary [overflow-wrap:anywhere] [&_ol]:mb-3 [&_p]:mb-3 [&_p:last-child]:mb-0 [&_strong]:font-semibold [&_strong]:text-gold [&_ul]:mb-3" v-html="renderedAiObservations"></div>
              </div>
            </div>
          </form>

          <div v-if="ngcCertNumber" class="card min-w-0 overflow-hidden">
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              :aria-expanded="ngcNumistaExpanded"
              aria-controls="ngc-numista-lookup"
              @click="toggleNgcNumista"
              @keydown.enter.prevent="toggleNgcNumista"
              @keydown.space.prevent="toggleNgcNumista"
            >
              Also search Numista
            </button>
            <div v-if="ngcNumistaExpanded" id="ngc-numista-lookup" class="mt-3 min-w-0 overflow-hidden">
              <NumistaLookupPanel
                :initial-query="photoNumistaQuery"
                :evidence="photoNumistaEvidence"
                path="photo"
                :is-admin="auth.isAdmin"
                :show-confirmation="false"
                @selection-changed="selectedNumistaCandidate = $event"
              />
            </div>
          </div>

          <div
            v-if="!ngcCertNumber && (hasPhotoNumistaProposalContract || numistaResults.length === 0)"
            class="card min-w-0 overflow-hidden"
          >
            <NumistaLookupPanel
              :initial-query="photoNumistaQuery"
              :evidence="photoNumistaEvidence"
              path="photo"
              :is-admin="auth.isAdmin"
              :show-confirmation="false"
              @selection-changed="selectedNumistaCandidate = $event"
            />
          </div>

          <!-- Deprecated result compatibility for older API deployments -->
          <div v-else-if="!ngcCertNumber && numistaResults.length > 0" class="card">
            <h3 class="mb-4 text-lg text-text-primary">Possible Matches</h3>
            <div class="flex flex-col gap-3">
              <div v-for="match in numistaResults" :key="match.id" class="flex flex-col gap-4 rounded-md border border-border-subtle bg-card p-4 transition-colors hover:border-border-accent md:flex-row md:items-start">
                <img
                  v-if="match.thumbnail"
                  :src="match.thumbnail"
                  :alt="match.title"
                  class="h-[200px] w-full object-cover md:h-20 md:w-20 md:shrink-0"
                />
                <div class="flex flex-1 flex-col gap-[0.35rem]">
                  <h4 class="m-0 text-base font-medium text-text-primary">{{ match.title }}</h4>
                  <p v-if="match.issuer" class="text-chip text-text-muted">{{ match.issuer }}</p>
                  <SafeExternalLink
                    :href="match.url"
                    class="mt-1 inline-flex items-center gap-[0.35rem] text-chip text-gold"
                  >
                    <ExternalLink :size="14" />
                    View on Numista
                  </SafeExternalLink>
                </div>
              </div>
            </div>
          </div>

          <!-- Quick Actions -->
          <div class="flex flex-col gap-3 pt-2 md:flex-row">
            <button class="btn btn-secondary min-w-[150px] flex-1 justify-center" @click="handleRetake">
              <RotateCcw :size="16" />
              Retake Photo
            </button>
            <button class="btn btn-secondary min-w-[150px] flex-1 justify-center" @click="handleCancel">
              <X :size="16" />
              Cancel
            </button>
            <button class="btn btn-primary min-w-[150px] flex-1 justify-center" :disabled="saving" @click="handleSaveAsDraft">
              <span v-if="saving" class="inline-block h-[14px] w-[14px] animate-spin rounded-full border-2 border-border-subtle border-t-gold"></span>
              <Bookmark v-else :size="16" />
              {{ saving ? 'Saving...' : 'Save as Draft' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, nextTick, onBeforeUnmount } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { createQuickCaptureDraft, getApiErrorMessage, lookupCoin } from '@/api/client'
import type { CoinLookupImageRole, CoinLookupResponse, CoinMutationPayload, CreateDeepIdentificationJobInput, NumistaCandidate, NumistaEvidence } from '@/types'
import { renderSafeMarkdown } from '@/composables/useMarkdown'
import { appendUniqueObservation, deriveAiObservations, normalizedEra, normalizeLookupDraft } from '@/utils/coinLookupDraft'
import {
  X,
  AlertCircle,
  ShieldCheck,
  ExternalLink,
  RotateCcw,
  Bookmark,
  List,
} from 'lucide-vue-next'
import CoinLookupCaptureWizard from '@/components/coin-lookup/CoinLookupCaptureWizard.vue'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import NumistaLookupPanel from '@/components/numista/NumistaLookupPanel.vue'
import DeepAnalysisStartPanel from '@/components/deep-identification/DeepAnalysisStartPanel.vue'
import AppIconButton from '@/components/ui/AppIconButton.vue'
import { useDeepIdentification } from '@/composables/useDeepIdentification'
import { useDeepIdentificationCapability } from '@/composables/useDeepIdentificationCapability'
import { selectedNumistaReferenceFromCandidate } from '@/utils/numistaLookup'
import { normalizeGalleryImage } from '@/utils/galleryImage'
import { useAuthStore } from '@/stores/auth'

interface CapturedImage {
  role: CoinLookupImageRole
  file: File
  preview: string
}

type LookupState = 'capture' | 'analyzing' | 'results'

const router = useRouter()
const auth = useAuthStore()

const state = ref<LookupState>('capture')
const capturedImages = ref<CapturedImage[]>([])
const captureWizard = ref<InstanceType<typeof CoinLookupCaptureWizard> | null>(null)
const captureNotes = ref('')
const submitting = ref(false)
const preparingImage = ref(false)
const saving = ref(false)
const error = ref('')
const uploadError = ref('')
const results = ref<CoinLookupResponse | null>(null)
const aiObservations = ref('')
const selectedNumistaCandidate = ref<NumistaCandidate | null>(null)
const ngcNumistaExpanded = ref(false)
const obverseImage = computed(() => capturedImages.value.find(image => image.role === 'obverse') ?? null)
const reverseImage = computed(() => capturedImages.value.find(image => image.role === 'reverse') ?? null)
const notesImage = computed(() => capturedImages.value.find(image => image.role === 'notes') ?? null)

const showDeepAnalysisModal = ref(false)
const { enabled: deepAnalysisEnabled } = useDeepIdentificationCapability()
const deepIdentification = useDeepIdentification()

async function onDeepAnalysisSubmit(input: CreateDeepIdentificationJobInput) {
  const job = await deepIdentification.start(input)
  if (job) {
    showDeepAnalysisModal.value = false
    await router.push(`/deep-analysis/${job.id}`)
  }
}


const reviewForm = reactive<CoinMutationPayload>({
  name: '',
  obverseDescription: '',
  reverseDescription: '',
  notes: '',
})

const ngcForm = reactive({
  certNumber: '',
  lookupUrl: '',
  grade: '',
  labelText: '',
  confidence: '',
})

const ngcCertNumber = computed(() => {
  return ngcForm.certNumber || results.value?.extractedData.ngc?.normalizedCert || null
})

const ngcLookupUrl = computed(() => {
  if (ngcForm.lookupUrl) return ngcForm.lookupUrl
  if (results.value?.extractedData.ngc?.lookupURL) return results.value.extractedData.ngc.lookupURL
  if (!ngcCertNumber.value) return ''
  const compactCert = ngcCertNumber.value.replace(/\D/g, '')
  return `https://www.ngccoin.com/certlookup/${encodeURIComponent(compactCert)}/NGCAncients/`
})

const numistaResults = computed(() => results.value?.numistaCandidates ?? [])
const photoNumistaQuery = computed(() => results.value?.proposedNumistaQuery?.trim() ?? '')
const hasPhotoNumistaProposalContract = computed(() => {
  return typeof results.value?.proposedNumistaQuery === 'string'
    || results.value?.numistaEvidence !== undefined
    || results.value?.numistaLookup !== undefined
})
const photoNumistaEvidence = computed<NumistaEvidence>(() => results.value?.numistaEvidence ?? {})
const renderedAiObservations = computed(() => renderSafeMarkdown(aiObservations.value))

async function toggleNgcNumista() {
  ngcNumistaExpanded.value = !ngcNumistaExpanded.value
  if (!ngcNumistaExpanded.value) return
  await nextTick()
  document.getElementById('numista-query')?.focus()
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
    grade: prefilled.grade,
    obverseInscription: prefilled.obverseInscription,
    reverseInscription: prefilled.reverseInscription,
    obverseDescription: prefilled.obverseDescription || '',
    reverseDescription: prefilled.reverseDescription || '',
    notes: prefilled.notes || prefilled.aiAnalysis || '',
  })
}

function applyLookupMetadata(lookup: CoinLookupResponse) {
  ngcForm.certNumber = lookup.extractedData.ngc?.normalizedCert ?? lookup.extractedData.ngc?.certNumber ?? ''
  ngcForm.lookupUrl = lookup.extractedData.ngc?.lookupURL ?? ''
  ngcForm.grade = lookup.extractedData.ngc?.grade ?? ''
  ngcForm.labelText = lookup.extractedData.labelText ?? ''
  ngcForm.confidence = lookup.extractedData.confidence ?? ''
}

function setCapturedImage(role: CoinLookupImageRole, file: File) {
  uploadError.value = ''
  const existingIndex = capturedImages.value.findIndex(image => image.role === role)
  if (existingIndex >= 0) {
    const existing = capturedImages.value[existingIndex]
    if (existing) URL.revokeObjectURL(existing.preview)
    capturedImages.value.splice(existingIndex, 1)
  }
  const preview = URL.createObjectURL(file)
  capturedImages.value.push({ role, file, preview })
}

function handleCameraCapture(role: CoinLookupImageRole, file: File) {
  setCapturedImage(role, file)
}

async function handleGallerySelection(role: CoinLookupImageRole, file: File) {
  uploadError.value = ''
  preparingImage.value = true
  try {
    setCapturedImage(role, await normalizeGalleryImage(file))
  } catch (err: unknown) {
    uploadError.value = getApiErrorMessage(err) || 'The selected image could not be prepared. Try a JPEG or PNG image.'
  } finally {
    preparingImage.value = false
  }
}

function removeCapturedImage(role: CoinLookupImageRole) {
  const index = capturedImages.value.findIndex(image => image.role === role)
  const image = capturedImages.value[index]
  if (!image) return
  URL.revokeObjectURL(image.preview)
  capturedImages.value.splice(index, 1)
}

async function handleSubmit() {
  if (!obverseImage.value || preparingImage.value) return

  submitting.value = true
  error.value = ''
  state.value = 'analyzing'
  captureWizard.value?.stopCamera()

  try {
    const selectedImages = [obverseImage.value, reverseImage.value, notesImage.value]
      .filter((image): image is CapturedImage => image !== null)
    const lookup = await lookupCoin(
      selectedImages.map(image => image.file),
      captureNotes.value,
      selectedImages.map(image => image.role),
    )
    const normalizedDraft = normalizeLookupDraft(lookup.data)
    results.value = lookup.data
    applyLookupMetadata(lookup.data)
    applyDraftToReviewForm(normalizedDraft)
    aiObservations.value = deriveAiObservations(lookup.data, normalizedDraft)

    state.value = 'results'
  } catch (err: unknown) {
    console.error('Lookup failed:', err)
    error.value = getApiErrorMessage(err) || 'Failed to analyze coin'
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
  captureNotes.value = ''
  results.value = null
  selectedNumistaCandidate.value = null
  ngcNumistaExpanded.value = false
  aiObservations.value = ''
  error.value = ''
  Object.assign(ngcForm, {
    certNumber: '',
    lookupUrl: '',
    grade: '',
    labelText: '',
    confidence: '',
  })

  applyDraftToReviewForm({})

  state.value = 'capture'
}

function handleCancel() {
  router.back()
}

function buildDraftNotes() {
  const parts: string[] = []
  appendUniqueObservation(parts, captureNotes.value, 'Collector notes')
  const extractedFields = [
    reviewForm.ruler ? `Ruler: ${reviewForm.ruler}` : '',
    reviewForm.denomination ? `Denomination: ${reviewForm.denomination}` : '',
    reviewForm.category ? `Category: ${reviewForm.category}` : '',
    reviewForm.grade ? `Grade: ${reviewForm.grade}` : '',
    reviewForm.mint ? `Mint: ${reviewForm.mint}` : '',
    reviewForm.material ? `Material: ${reviewForm.material}` : '',
  ].filter(Boolean)

  if (extractedFields.length > 0) {
    parts.push(`**Extracted fields**\n${extractedFields.join('\n')}`)
  }

  appendUniqueObservation(parts, aiObservations.value)
  if (!aiObservations.value.trim()) {
    appendUniqueObservation(parts, reviewForm.notes)
    appendUniqueObservation(parts, reviewForm.obverseDescription, 'Obverse')
    appendUniqueObservation(parts, reviewForm.reverseDescription, 'Reverse')
  }

  return parts.join('\n\n')
}

async function handleSaveAsDraft() {
  if (saving.value) return
  saving.value = true
  try {
    const selectedReference = selectedNumistaCandidate.value
      ? selectedNumistaReferenceFromCandidate(selectedNumistaCandidate.value)
      : null
    const draft = await createQuickCaptureDraft({
      workingTitle: reviewForm.name || 'Unidentified Coin',
      era: normalizedEra(reviewForm.era),
      notes: buildDraftNotes(),
      source: 'find_coin_ai',
      ngcCertNumber: ngcForm.certNumber,
      ngcLookupUrl: ngcLookupUrl.value,
      ngcGrade: ngcForm.grade || reviewForm.grade,
      labelText: ngcForm.labelText,
      aiConfidence: ngcForm.confidence,
      selectedNumistaId: selectedReference?.number,
      selectedNumistaUrl: selectedReference?.uri,
      obverseImage: obverseImage.value?.file ?? null,
      reverseImage: reverseImage.value?.file ?? null,
      detailImages: notesImage.value ? [notesImage.value.file] : [],
    })
    router.push(`/quick-capture/drafts/${draft.data.id}`)
  } catch (err: unknown) {
    console.error('Failed to save draft:', err)
    error.value = getApiErrorMessage(err) || 'Failed to save draft'
  } finally {
    saving.value = false
  }
}

onBeforeUnmount(() => {
  captureWizard.value?.stopCamera()
  for (const img of capturedImages.value) {
    URL.revokeObjectURL(img.preview)
  }
})
</script>

<style scoped>
/*
 * :deep() audit — markdown-rendered content
 * Target: HTML elements emitted by markdown-it inside .markdown-rendered.
 * Lookup results and AI observations are rendered from Markdown at runtime;
 * the generated nodes do not carry Vue scope attributes and cannot be styled by
 * scoped selectors or Tailwind utilities.
 */
.markdown-rendered :deep(p),
.markdown-rendered :deep(ul),
.markdown-rendered :deep(ol) {
  margin: 0 0 0.75rem;
}

.markdown-rendered :deep(p:last-child),
.markdown-rendered :deep(ul:last-child),
.markdown-rendered :deep(ol:last-child) {
  margin-bottom: 0;
}

.markdown-rendered :deep(strong) {
  color: var(--accent-gold);
  font-weight: 600;
}
</style>
