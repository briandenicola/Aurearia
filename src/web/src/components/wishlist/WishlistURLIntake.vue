<template>
  <section class="url-intake">
    <button
      v-if="!embedded"
      type="button"
      class="btn btn-secondary btn-sm"
      :aria-expanded="expanded"
      @click="expanded = !expanded"
    >
      <Link :size="16" /> Add from URL
    </button>

    <div v-if="expanded" class="url-intake-panel" :class="{ embedded }">
      <form class="url-row" @submit.prevent="analyze">
        <label for="wishlist-url" class="sr-only">Public coin listing URL</label>
        <input
          id="wishlist-url"
          v-model.trim="sourceURL"
          type="url"
          class="form-input min-w-0 flex-1"
          placeholder="https://dealer.example/coin"
          required
          :disabled="status === 'pending'"
        />
        <button type="submit" class="btn btn-primary btn-sm" :disabled="status === 'pending' || !sourceURL">
          {{ status === 'pending' ? 'Analyzing...' : 'Analyze' }}
        </button>
        <button v-if="status === 'pending'" type="button" class="btn btn-ghost btn-sm" @click="cancel">
          Cancel
        </button>
      </form>

      <p v-if="message" class="status-message" role="status">{{ message }}</p>

      <div v-if="analysis?.outcome === 'duplicate' && analysis.existingCoinId" class="review-actions">
        <router-link :to="`/coin/${analysis.existingCoinId}`" class="btn btn-secondary btn-sm">
          View existing wishlist coin
        </router-link>
      </div>

      <form v-else-if="analysis?.hypothesis" class="proposal" @submit.prevent="confirmCreate">
        <div class="proposal-heading">
          <div>
            <span class="section-label">Review proposal</span>
            <p>Verify and edit every field before adding it to your wishlist.</p>
          </div>
          <span v-if="analysis.hypothesis.listingStatus" class="chip-sm">
            {{ analysis.hypothesis.listingStatus.value }}
          </span>
        </div>

        <div class="proposal-grid">
          <label class="form-group proposal-wide">
            <span>Name *</span>
            <input v-model.trim="draft.name" class="form-input" maxlength="200" required />
          </label>
          <label class="form-group">
            <span>Category</span>
            <select v-model="draft.category" class="form-input">
              <option v-for="category in categoryOptions" :key="category" :value="category">{{ category }}</option>
            </select>
          </label>
          <label class="form-group">
            <span>Era</span>
            <select v-model="draft.era" class="form-input">
              <option value="">Unknown</option>
              <option v-for="era in eraOptions" :key="era" :value="era">{{ era }}</option>
            </select>
          </label>
          <label class="form-group">
            <span>Material</span>
            <select v-model="draft.material" class="form-input">
              <option v-for="material in materialOptions" :key="material" :value="material">{{ material }}</option>
            </select>
          </label>
          <label v-for="field in textFields" :key="field.key" class="form-group">
            <span>{{ field.label }}</span>
            <input v-model.trim="draft[field.key]" class="form-input" :maxlength="field.maxlength" />
          </label>
          <label class="form-group">
            <span>Weight (g)</span>
            <input v-model="draft.weightGrams" class="form-input" type="number" min="0" step="0.01" />
          </label>
          <label class="form-group">
            <span>Diameter (mm)</span>
            <input v-model="draft.diameterMm" class="form-input" type="number" min="0" step="0.01" />
          </label>
          <label class="form-group">
            <span>Listed value</span>
            <input v-model="draft.currentValue" class="form-input" type="number" min="0" step="0.01" />
          </label>
          <label class="form-group">
            <span>Listing status</span>
            <select v-model="draft.listingStatus" class="form-input">
              <option value="">Unknown</option>
              <option value="available">Available</option>
              <option value="sold">Sold</option>
              <option value="reserved">Reserved</option>
              <option value="withdrawn">Withdrawn</option>
            </select>
          </label>
          <label class="form-group proposal-wide">
            <span>Obverse legend</span>
            <input v-model.trim="draft.obverseInscription" class="form-input" maxlength="1000" />
          </label>
          <label class="form-group proposal-wide">
            <span>Obverse description</span>
            <textarea v-model.trim="draft.obverseDescription" class="form-input" rows="2" maxlength="2000"></textarea>
          </label>
          <label class="form-group proposal-wide">
            <span>Reverse legend</span>
            <input v-model.trim="draft.reverseInscription" class="form-input" maxlength="1000" />
          </label>
          <label class="form-group proposal-wide">
            <span>Reverse description</span>
            <textarea v-model.trim="draft.reverseDescription" class="form-input" rows="2" maxlength="2000"></textarea>
          </label>
          <label class="form-group proposal-wide">
            <span>Notes</span>
            <textarea v-model.trim="draft.notes" class="form-input" rows="3" maxlength="10000"></textarea>
          </label>
        </div>

        <div v-if="previewURL" class="image-review">
          <img :src="previewURL" alt="Listing image preview" />
          <label class="image-choice">
            <input v-model="attachImage" type="checkbox" />
            Attach this image as the obverse
          </label>
        </div>

        <ul v-if="analysis.warnings?.length" class="warnings">
          <li v-for="warning in analysis.warnings" :key="warning">{{ warning }}</li>
        </ul>

        <div class="review-actions">
          <button type="submit" class="btn btn-primary btn-sm" :disabled="status === 'creating' || !draft.name">
            {{ status === 'creating' ? 'Adding...' : 'Confirm and add' }}
          </button>
          <button type="button" class="btn btn-ghost btn-sm" :disabled="status === 'creating'" @click="resetProposal">
            Cancel
          </button>
        </div>
      </form>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { Link } from 'lucide-vue-next'
import { analyzeWishlistURL, createCoin, getApiErrorMessage, proxyImage, uploadImage } from '@/api/client'
import type { Category, CoinMutationPayload, Material, WishlistURLAnalysis, WishlistURLHypothesis } from '@/types'
import { useCoinOptions } from '@/composables/useCoinOptions'

const props = withDefaults(defineProps<{
  embedded?: boolean
}>(), {
  embedded: false,
})

const emit = defineEmits<{ created: [] }>()

const textFields = [
  { key: 'ruler', label: 'Ruler or issuer', maxlength: 200 },
  { key: 'denomination', label: 'Denomination', maxlength: 200 },
  { key: 'mint', label: 'Mint', maxlength: 200 },
  { key: 'dateRange', label: 'Date range', maxlength: 100 },
  { key: 'grade', label: 'Grade', maxlength: 100 },
  { key: 'rarityRating', label: 'Rarity', maxlength: 100 },
  { key: 'references', label: 'References', maxlength: 1000 },
] as const

type DraftKey = typeof textFields[number]['key']
type Draft = Record<DraftKey, string> & {
  name: string
  category: Category
  era: string
  material: string
  weightGrams: string
  diameterMm: string
  currentValue: string
  listingStatus: string
  obverseInscription: string
  obverseDescription: string
  reverseInscription: string
  reverseDescription: string
  notes: string
}

const expanded = ref(props.embedded)
const { categoryOptions, eraOptions, materialOptions, loadOptions } = useCoinOptions()
const sourceURL = ref('')
const status = ref<'idle' | 'pending' | 'cancelled' | 'creating' | 'created' | 'image-warning' | 'failed'>('idle')
const message = ref('')
const analysis = ref<WishlistURLAnalysis | null>(null)
const attachImage = ref(true)
const previewURL = ref('')
const previewFile = ref<File | null>(null)
let controller: AbortController | null = null

const draft = reactive<Draft>(emptyDraft())

function emptyDraft(): Draft {
  return {
    name: '', category: 'Other', era: '', ruler: '', denomination: '', material: 'Other',
    mint: '', dateRange: '', grade: '', rarityRating: '', weightGrams: '', diameterMm: '',
    currentValue: '', listingStatus: '', references: '',
    obverseInscription: '', obverseDescription: '', reverseInscription: '',
    reverseDescription: '', notes: '',
  }
}

function value(hypothesis: WishlistURLHypothesis, key: keyof WishlistURLHypothesis): string {
  const field = hypothesis[key]
  return typeof field === 'object' && field !== null && 'value' in field ? field.value : ''
}

function fillDraft(hypothesis: WishlistURLHypothesis) {
  Object.assign(draft, {
    name: value(hypothesis, 'name'),
    category: categoryOptions.value.includes(value(hypothesis, 'category'))
      ? value(hypothesis, 'category') as Category
      : 'Other',
    era: value(hypothesis, 'era'),
    ruler: value(hypothesis, 'ruler'),
    denomination: value(hypothesis, 'denomination'),
    material: value(hypothesis, 'material') || 'Other',
    mint: value(hypothesis, 'mint'),
    dateRange: value(hypothesis, 'dateRange'),
    grade: value(hypothesis, 'grade'),
    rarityRating: value(hypothesis, 'rarityRating'),
    references: value(hypothesis, 'references') || value(hypothesis, 'coin_type'),
    weightGrams: value(hypothesis, 'weightGrams'),
    diameterMm: value(hypothesis, 'diameterMm'),
    currentValue: value(hypothesis, 'listedPrice'),
    listingStatus: value(hypothesis, 'listingStatus'),
    obverseInscription: value(hypothesis, 'obverseInscription'),
    obverseDescription: value(hypothesis, 'obverseDescription'),
    reverseInscription: value(hypothesis, 'reverseInscription'),
    reverseDescription: value(hypothesis, 'reverseDescription'),
    notes: value(hypothesis, 'notes'),
  })
}

function clearPreview() {
  if (previewURL.value) URL.revokeObjectURL(previewURL.value)
  previewURL.value = ''
  previewFile.value = null
}

async function loadPreview(imageURL: string) {
  clearPreview()
  if (!imageURL) return
  try {
    const response = await proxyImage(imageURL)
    const blob = response.data as Blob
    if (blob.size === 0) return
    const extension = blob.type.includes('png') ? '.png' : '.jpg'
    previewFile.value = new File([blob], `listing${extension}`, { type: blob.type || 'image/jpeg' })
    previewURL.value = URL.createObjectURL(blob)
  } catch {
    message.value = 'Proposal is ready, but the listing image could not be previewed.'
  }
}

async function analyze() {
  controller?.abort()
  controller = new AbortController()
  status.value = 'pending'
  message.value = ''
  analysis.value = null
  clearPreview()
  try {
    const response = await analyzeWishlistURL(sourceURL.value, controller.signal)
    analysis.value = response.data
    if (response.data.outcome === 'duplicate') {
      message.value = 'This listing is already on your wishlist.'
      status.value = 'idle'
      return
    }
    if (!response.data.hypothesis) throw new Error('The listing did not return a proposal.')
    fillDraft(response.data.hypothesis)
    status.value = 'idle'
    message.value = response.data.outcome === 'ready'
      ? 'Proposal ready for review.'
      : 'The listing needs more review before it can be added.'
    await loadPreview(response.data.imageUrl ?? '')
  } catch (error) {
    if (controller?.signal.aborted) {
      status.value = 'cancelled'
      message.value = 'Analysis cancelled.'
    } else {
      status.value = 'failed'
      message.value = getApiErrorMessage(error) || 'Unable to analyze this listing.'
    }
  } finally {
    controller = null
  }
}

function cancel() {
  controller?.abort()
}

function resetProposal() {
  analysis.value = null
  Object.assign(draft, emptyDraft())
  clearPreview()
  status.value = 'idle'
  message.value = ''
}

async function confirmCreate() {
  if (!analysis.value?.hypothesis || status.value === 'creating') return
  status.value = 'creating'
  message.value = ''
  const hypothesis = analysis.value.hypothesis
  const currentValue = Number(draft.currentValue)
  const payload: CoinMutationPayload = {
    name: draft.name,
    category: draft.category,
    era: draft.era,
    ruler: draft.ruler,
    denomination: draft.denomination,
    material: draft.material as Material,
    mint: draft.mint,
    dateRange: draft.dateRange,
    grade: draft.grade,
    rarityRating: draft.rarityRating,
    weightGrams: draft.weightGrams ? Number(draft.weightGrams) : null,
    diameterMm: draft.diameterMm ? Number(draft.diameterMm) : null,
    obverseInscription: draft.obverseInscription,
    obverseDescription: draft.obverseDescription,
    reverseInscription: draft.reverseInscription,
    reverseDescription: draft.reverseDescription,
    notes: draft.notes,
    referenceUrl: analysis.value.sourceUrl,
    referenceText: draft.references || value(hypothesis, 'dealerName'),
    listingStatus: draft.listingStatus,
    currentValue: draft.currentValue && Number.isFinite(currentValue) ? currentValue : null,
    isWishlist: true,
  }
  try {
    const created = await createCoin(payload)
    let imageWarning = false
    if (attachImage.value && previewFile.value) {
      try {
        await uploadImage(created.data.id, previewFile.value, 'obverse', true)
      } catch {
        imageWarning = true
      }
    }
    status.value = imageWarning ? 'image-warning' : 'created'
    message.value = imageWarning
      ? 'Wishlist coin added, but the optional image could not be attached.'
      : 'Wishlist coin added.'
    analysis.value = null
    clearPreview()
    emit('created')
  } catch (error) {
    status.value = 'failed'
    message.value = getApiErrorMessage(error) || 'Unable to add this wishlist coin.'
  }
}

onBeforeUnmount(() => {
  controller?.abort()
  clearPreview()
})

onMounted(loadOptions)
</script>

<style scoped>
.url-intake {
  margin-bottom: 1.5rem;
}

.url-intake-panel {
  margin-top: 0.75rem;
  padding: 1rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--bg-card);
}

.url-intake-panel.embedded {
  margin-top: 0;
  padding: 0;
  border: 0;
  background: transparent;
}

.url-row,
.review-actions,
.proposal-heading,
.image-choice {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.status-message,
.proposal-heading p,
.warnings {
  margin-top: 0.75rem;
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.proposal {
  margin-top: 1.5rem;
}

.proposal-heading {
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.proposal-heading p {
  margin-bottom: 0;
}

.proposal-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}

.proposal-wide {
  grid-column: 1 / -1;
}

.image-review {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-top: 1rem;
}

.image-review img {
  width: 96px;
  height: 96px;
  object-fit: contain;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-input);
}

.warnings {
  padding-left: 1.25rem;
}

.review-actions {
  margin-top: 1rem;
}

.url-intake-panel .btn,
.url-intake-panel .form-input {
  min-height: 44px;
}

@media (max-width: 640px) {
  .url-row,
  .image-review {
    align-items: stretch;
    flex-direction: column;
  }

  .proposal-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .proposal-wide {
    grid-column: auto;
  }
}
</style>
