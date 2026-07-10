<template>
  <div class="container">
    <div class="mx-auto max-w-[900px]">
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
        <p class="rounded-md border border-border-subtle bg-card p-6 text-base leading-6 text-text-secondary shadow-[var(--shadow-card)]">
          Use the camera or upload an obverse image to start a quick AI draft. Reverse and slab detail photos are optional, but improve attribution and NGC number capture.
        </p>

        <InlineCameraCapturePanel
          ref="cameraPanel"
          filename-prefix="lookup"
          @captured="addCapturedFile"
          @upload="triggerFileUpload"
        />

        <!-- Image preview grid -->
        <div v-if="capturedImages.length > 0" class="grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(150px,1fr))]">
          <div v-for="(img, idx) in capturedImages" :key="idx" class="relative aspect-square overflow-hidden rounded-sm border border-border-subtle">
            <span class="absolute left-2 top-2 z-[1] rounded-full border border-border-subtle bg-card px-2 py-[0.15rem] text-sm text-text-secondary">{{ imageTypeLabel(idx) }}</span>
            <img :src="img.preview" alt="Captured coin" class="h-full w-full object-cover" />
            <button class="absolute right-2 top-2 flex items-center justify-center rounded-sm bg-[rgba(0,0,0,0.7)] p-[0.35rem] text-text-primary transition-colors hover:bg-[rgba(192,57,43,0.8)]" @click="removeImage(idx)" title="Remove">
              <X :size="16" />
            </button>
          </div>
        </div>

        <input
          ref="fileInput"
          type="file"
          accept="image/*"
          multiple
          style="display: none"
          @change="handleFileUpload"
        />

        <button
          v-if="capturedImages.length > 0"
          class="btn btn-primary w-full justify-center px-6 py-[0.85rem] text-base"
          @click="handleSubmit"
          :disabled="submitting"
        >
          <span v-if="submitting" class="inline-block h-[14px] w-[14px] animate-spin rounded-full border-2 border-border-subtle border-t-gold"></span>
          <Search v-else :size="20" />
          {{ submitting ? 'Analyzing...' : 'Create Quick AI Draft' }}
        </button>
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
      <div v-if="state === 'results'" class="flex flex-col gap-6">
        <div v-if="error" class="flex items-center gap-3 rounded-md border border-[rgba(192,57,43,0.3)] bg-[rgba(192,57,43,0.2)] p-4 text-base text-byzantine">
          <AlertCircle :size="20" />
          <span>{{ error }}</span>
        </div>

        <div v-if="results" class="flex flex-col gap-6">
          <!-- NGC Certification Path -->
          <form v-if="ngcCertNumber" class="card" @submit.prevent="handleSaveAsDraft">
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
              <div class="markdown-rendered rounded-sm border border-border-subtle bg-input p-3 text-body leading-6 text-text-secondary [&_ol]:mb-3 [&_p]:mb-3 [&_p:last-child]:mb-0 [&_strong]:font-semibold [&_strong]:text-gold [&_ul]:mb-3" v-html="renderedAiObservations"></div>
            </div>
          </form>

          <!-- Non-NGC Path (editable review form) -->
          <form v-else class="card" @submit.prevent="handleSaveAsDraft">
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
                <div class="markdown-rendered rounded-sm border border-border-subtle bg-input p-3 text-body leading-6 text-text-secondary [&_ol]:mb-3 [&_p]:mb-3 [&_p:last-child]:mb-0 [&_strong]:font-semibold [&_strong]:text-gold [&_ul]:mb-3" v-html="renderedAiObservations"></div>
              </div>
            </div>
          </form>

          <!-- Numista matches -->
          <div v-if="numistaResults && numistaResults.length > 0" class="card">
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
import { ref, computed, reactive, onBeforeUnmount } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { createQuickCaptureDraft, lookupCoin } from '@/api/client'
import type { CoinLookupResponse, CoinMutationPayload } from '@/types'
import { renderSafeMarkdown } from '@/composables/useMarkdown'
import { appendUniqueObservation, deriveAiObservations, normalizedEra, normalizeLookupDraft } from '@/utils/coinLookupDraft'
import {
  Search,
  X,
  AlertCircle,
  ShieldCheck,
  ExternalLink,
  RotateCcw,
  Bookmark,
  List,
} from 'lucide-vue-next'
import InlineCameraCapturePanel from '@/components/InlineCameraCapturePanel.vue'
import SafeExternalLink from '@/components/SafeExternalLink.vue'

interface CapturedImage {
  file: File
  preview: string
}

type LookupState = 'capture' | 'analyzing' | 'results'

const router = useRouter()

const state = ref<LookupState>('capture')
const capturedImages = ref<CapturedImage[]>([])
const fileInput = ref<HTMLInputElement | null>(null)
const cameraPanel = ref<InstanceType<typeof InlineCameraCapturePanel> | null>(null)
const submitting = ref(false)
const saving = ref(false)
const error = ref('')
const results = ref<CoinLookupResponse | null>(null)
const aiObservations = ref('')

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
const renderedAiObservations = computed(() => renderSafeMarkdown(aiObservations.value))

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

function addCapturedFile(file: File) {
  const preview = URL.createObjectURL(file)
  capturedImages.value.push({ file, preview })
}

function imageTypeLabel(index: number) {
  if (index === 0) return 'Obverse'
  if (index === 1) return 'Reverse optional'
  return 'Detail'
}

function triggerFileUpload() {
  fileInput.value?.click()
}

function handleFileUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const files = input.files
  if (!files || files.length === 0) return

  for (let i = 0; i < files.length; i++) {
    const file = files[i]
    if (!file) continue
    addCapturedFile(file)
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
  cameraPanel.value?.stopCamera()

  try {
    const files = capturedImages.value.map(img => img.file)
    const lookup = await lookupCoin(files)
    const normalizedDraft = normalizeLookupDraft(lookup.data)
    results.value = lookup.data
    applyLookupMetadata(lookup.data)
    applyDraftToReviewForm(normalizedDraft)
    aiObservations.value = deriveAiObservations(lookup.data, normalizedDraft)

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
      obverseImage: capturedImages.value[0]?.file ?? null,
      reverseImage: capturedImages.value[1]?.file ?? null,
      detailImages: capturedImages.value.slice(2).map(img => img.file),
    })
    router.push(`/quick-capture/drafts/${draft.data.id}`)
  } catch (err: unknown) {
    console.error('Failed to save draft:', err)
    error.value = err instanceof Error ? err.message : 'Failed to save draft'
  } finally {
    saving.value = false
  }
}

onBeforeUnmount(() => {
  cameraPanel.value?.stopCamera()
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
