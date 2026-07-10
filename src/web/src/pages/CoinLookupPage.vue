<template>
  <div class="container">
    <div class="page-header">
      <h1>Identify Coin</h1>
      <div class="pwa-actions">
        <RouterLink class="pwa-icon-btn" to="/quick-capture/drafts" title="All drafts" aria-label="All drafts">
          <List :size="22" />
        </RouterLink>
      </div>
    </div>

    <!-- Capture State -->
    <div v-if="state === 'capture'" class="lookup-capture">
      <p class="lookup-instructions card">
        Use the camera or upload an obverse image to start a quick AI draft. Reverse and slab detail photos are optional, but improve attribution and NGC number capture.
      </p>

      <InlineCameraCapturePanel
        ref="cameraPanel"
        filename-prefix="lookup"
        @captured="addCapturedFile"
        @upload="triggerFileUpload"
      />

      <!-- Image preview grid -->
      <div v-if="capturedImages.length > 0" class="captured-images">
        <div v-for="(img, idx) in capturedImages" :key="idx" class="captured-image-card">
          <span class="image-type-chip">{{ imageTypeLabel(idx) }}</span>
          <img :src="img.preview" alt="Captured coin" />
          <button class="remove-image-btn" @click="removeImage(idx)" title="Remove">
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
        class="btn btn-primary btn-submit"
        @click="handleSubmit"
        :disabled="submitting"
      >
        <span v-if="submitting" class="spinner-sm"></span>
        <Search v-else :size="20" />
        {{ submitting ? 'Analyzing...' : 'Create Quick AI Draft' }}
      </button>
    </div>

    <!-- Analyzing State -->
    <div v-if="state === 'analyzing'" class="lookup-analyzing">
      <div class="analyzing-spinner">
        <div class="spinner"></div>
      </div>
      <h3>Analyzing Images...</h3>
      <p>Extracting minimum draft details and checking for visible NGC data</p>
    </div>

    <!-- Results State -->
    <div v-if="state === 'results'" class="lookup-results">
      <div v-if="error" class="error-banner">
        <AlertCircle :size="20" />
        <span>{{ error }}</span>
      </div>

      <div v-if="results" class="results-content">
        <!-- NGC Certification Path -->
        <form v-if="ngcCertNumber" class="result-section card" @submit.prevent="handleSaveAsDraft">
          <h3>Review Coin Details</h3>
          <div class="review-grid">
            <label class="form-group full-width">
              <span class="section-label">Name</span>
              <input v-model="reviewForm.name" class="input" type="text" required>
            </label>

            <label class="form-group">
              <span class="section-label">Ruler</span>
              <input v-model="reviewForm.ruler" class="input" type="text">
            </label>

            <label class="form-group">
              <span class="section-label">Denomination</span>
              <input v-model="reviewForm.denomination" class="input" type="text">
            </label>

            <label class="form-group">
              <span class="section-label">Category</span>
              <input v-model="reviewForm.category" class="input" type="text">
            </label>

            <label class="form-group">
              <span class="section-label">Grade</span>
              <input v-model="reviewForm.grade" class="input" type="text">
            </label>
          </div>

          <div class="ngc-cert">
            <div class="ngc-cert-header">
              <ShieldCheck :size="20" />
              <span>NGC Certification: {{ ngcCertNumber }}</span>
            </div>
            <div v-if="ngcForm.grade" class="detail-item ngc-grade-display">
              <label>NGC Grade</label>
              <span>{{ ngcForm.grade }}</span>
            </div>
            <label class="form-group">
              <span class="section-label">NGC Coin Number</span>
              <input v-model="ngcForm.certNumber" class="input" type="text">
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

          <div v-if="aiObservations" class="ai-observations">
            <h4>AI Observations</h4>
            <div class="ai-observations-content markdown-rendered" v-html="renderedAiObservations"></div>
          </div>
        </form>

        <!-- Non-NGC Path (editable review form) -->
        <form v-else class="result-section card" @submit.prevent="handleSaveAsDraft">
          <h3>Review Coin Details</h3>

          <div class="review-grid">
            <label class="form-group full-width">
              <span class="section-label">Name</span>
              <input v-model="reviewForm.name" class="input" type="text" required>
            </label>

            <label class="form-group">
              <span class="section-label">Ruler</span>
              <input v-model="reviewForm.ruler" class="input" type="text">
            </label>

            <label class="form-group">
              <span class="section-label">Denomination</span>
              <input v-model="reviewForm.denomination" class="input" type="text">
            </label>

            <label class="form-group">
              <span class="section-label">Category</span>
              <input v-model="reviewForm.category" class="input" type="text">
            </label>

            <label class="form-group">
              <span class="section-label">Grade</span>
              <input v-model="reviewForm.grade" class="input" type="text">
            </label>

            <div v-if="aiObservations" class="form-group full-width ai-observations">
              <span class="section-label">AI Observations</span>
              <div class="ai-observations-content markdown-rendered" v-html="renderedAiObservations"></div>
            </div>
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
                <SafeExternalLink
                  :href="match.url"
                  class="numista-link"
                >
                  <ExternalLink :size="14" />
                  View on Numista
                </SafeExternalLink>
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
          <button class="btn btn-primary" :disabled="saving" @click="handleSaveAsDraft">
            <span v-if="saving" class="spinner-sm"></span>
            <Bookmark v-else :size="16" />
            {{ saving ? 'Saving...' : 'Save as Draft' }}
          </button>
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
.container {
  max-width: 900px;
  margin: 0 auto;
  padding: 1.5rem;
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
  border-radius: var(--radius-sm);
}

.captured-image-card img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-type-chip {
  position: absolute;
  left: 0.5rem;
  top: 0.5rem;
  z-index: 1;
  padding: 0.15rem 0.5rem;
  border-radius: var(--radius-full);
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  color: var(--text-secondary);
  font-size: 0.75rem;
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
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
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
.ai-observations {
  margin-top: 0.5rem;
}

.inscription-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1rem;
  margin-top: 0.5rem;
}

.inscription-side {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.inscription-side label {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  font-weight: 600;
}

.inscription-side p {
  color: var(--text-secondary);
  font-size: 0.85rem;
  line-height: 1.5;
}

.ai-observations-content {
  padding: 0.75rem;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: 0.85rem;
  line-height: 1.5;
}

/*
 * :deep() audit — markdown-rendered content
 * Target: HTML elements emitted by markdown-it inside .markdown-rendered.
 * Lookup results and AI observations are rendered from Markdown at runtime;
 * the generated HTML carries no Vue-scope hash and cannot be reached by
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

  .lookup-page-header {
    flex-direction: row;
    align-items: center;
  }

  .lookup-page-header .btn {
    flex-shrink: 0;
  }

  .details-grid {
    grid-template-columns: 1fr;
  }

  .inscription-grid {
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
