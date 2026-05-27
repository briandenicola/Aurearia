<template>
  <form class="coin-form" @submit.prevent="$emit('submit')">
    <div class="form-grid">
      <!-- Basic Info -->
      <fieldset class="form-section">
        <h2 class="form-section-title">Basic Information</h2>
        <div class="form-group">
          <label class="form-label">Name *</label>
          <AutocompleteInput v-model="form.name!" field="name" required placeholder="e.g. Augustus Denarius" />
        </div>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Category</label>
            <select v-model="form.category" class="form-select">
              <option v-for="c in CATEGORIES" :key="c" :value="c">{{ c }}</option>
            </select>
          </div>
          <div class="form-group">
            <label class="form-label">Material</label>
            <select v-model="form.material" class="form-select">
              <option v-for="m in MATERIALS" :key="m" :value="m">{{ m }}</option>
            </select>
          </div>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Denomination</label>
            <AutocompleteInput v-model="form.denomination!" field="denomination" placeholder="e.g. Denarius" />
          </div>
          <div class="form-group">
            <label class="form-label">Mint</label>
            <input v-model="form.mint" class="form-input" placeholder="e.g. Rome" />
          </div>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Ruler</label>
            <AutocompleteInput v-model="form.ruler!" field="ruler" placeholder="e.g. Augustus" />
          </div>
          <div class="form-group">
            <label class="form-label">Era</label>
            <AutocompleteInput v-model="form.era!" field="era" placeholder="e.g. 27 BC - 14 AD" />
          </div>
        </div>
      </fieldset>

      <!-- Physical Details -->
      <fieldset class="form-section">
        <h2 class="form-section-title">Physical Details</h2>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Weight (grams)</label>
            <input v-model.number="form.weightGrams" class="form-input" type="number" step="0.01" />
          </div>
          <div class="form-group">
            <label class="form-label">Diameter (mm)</label>
            <input v-model.number="form.diameterMm" class="form-input" type="number" step="0.1" />
          </div>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Grade</label>
            <input v-model="form.grade" class="form-input" placeholder="e.g. VF, EF, MS-65" />
          </div>
          <div class="form-group">
            <label class="form-label">Rarity Rating (RIC)</label>
            <input v-model="form.rarityRating" class="form-input" placeholder="e.g. RIC 207" />
          </div>
        </div>
      </fieldset>

      <!-- Inscriptions, Images & Descriptions -->
      <fieldset class="form-section full-width">
        <h2 class="form-section-title">Inscriptions & Descriptions</h2>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Obverse Inscription</label>
            <input v-model="form.obverseInscription" class="form-input" placeholder="Obverse legend text" />
          </div>
          <div class="form-group">
            <label class="form-label">Reverse Inscription</label>
            <input v-model="form.reverseInscription" class="form-input" placeholder="Reverse legend text" />
          </div>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Obverse Image</label>
            <div v-if="obversePreview || existingObverse" class="image-preview-box">
              <img :src="obversePreview || existingObverse!" alt="Obverse" class="image-preview" />
              <button type="button" class="image-remove-btn" @click="clearObverse" title="Remove"><X :size="12" /></button>
            </div>
            <div class="file-input-row">
              <input type="file" accept=".jpg,.jpeg,.png" class="form-input file-input" @change="onObverseFile" ref="obverseInput" />
              <label v-if="isPwa" class="btn btn-secondary btn-sm camera-btn">
                <Camera :size="14" /> Photo
                <input type="file" accept="image/*" capture="environment" hidden @change="onObverseFile" />
              </label>
            </div>
          </div>
          <div class="form-group">
            <label class="form-label">Reverse Image</label>
            <div v-if="reversePreview || existingReverse" class="image-preview-box">
              <img :src="reversePreview || existingReverse!" alt="Reverse" class="image-preview" />
              <button type="button" class="image-remove-btn" @click="clearReverse" title="Remove"><X :size="12" /></button>
            </div>
            <div class="file-input-row">
              <input type="file" accept=".jpg,.jpeg,.png" class="form-input file-input" @change="onReverseFile" ref="reverseInput" />
              <label v-if="isPwa" class="btn btn-secondary btn-sm camera-btn">
                <Camera :size="14" /> Photo
                <input type="file" accept="image/*" capture="environment" hidden @change="onReverseFile" />
              </label>
            </div>
          </div>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Obverse Description</label>
            <textarea v-model="form.obverseDescription" class="form-textarea" placeholder="Describe the obverse design" />
          </div>
          <div class="form-group">
            <label class="form-label">Reverse Description</label>
            <textarea v-model="form.reverseDescription" class="form-textarea" placeholder="Describe the reverse design" />
          </div>
        </div>
      </fieldset>

      <!-- Purchase Info -->
      <fieldset class="form-section">
        <h2 class="form-section-title">Purchase & Value</h2>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Purchase Price ($)</label>
            <input v-model.number="form.purchasePrice" class="form-input" type="number" step="0.01" />
          </div>
          <div class="form-group">
            <label class="form-label">Current Value ($)</label>
            <input v-model.number="form.currentValue" class="form-input" type="number" step="0.01" />
          </div>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Purchase Date</label>
            <input v-model="form.purchaseDate" class="form-input" type="date" />
          </div>
          <div class="form-group">
            <label class="form-label">Store</label>
            <AutocompleteInput v-model="form.purchaseLocation!" field="purchaseLocation" placeholder="e.g. Heritage Auctions" />
          </div>
        </div>
      </fieldset>

      <!-- Links & Notes -->
      <fieldset class="form-section">
        <h2 class="form-section-title">Reference & Notes</h2>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Reference URL</label>
            <input v-model="form.referenceUrl" class="form-input" type="url" placeholder="https://..." />
          </div>
          <div class="form-group">
            <label class="form-label">Reference Text</label>
            <input v-model="form.referenceText" class="form-input" placeholder="Link display text" />
          </div>
        </div>
        <div class="form-group">
          <label class="form-label">Store Card Image</label>
          <p class="form-hint">Upload a photo of the store card. Text will be extracted automatically and saved to Notes.</p>
          <div v-if="cardPreview" class="image-preview-box">
            <img :src="cardPreview" alt="Store card" class="image-preview" />
            <button type="button" class="image-remove-btn" @click="clearCard" title="Remove"><X :size="12" /></button>
          </div>
          <input type="file" accept=".jpg,.jpeg,.png" class="form-input file-input" @change="onCardFile" ref="cardInput" />
        </div>
        <div class="form-group">
          <label class="form-label">Notes</label>
          <textarea v-model="form.notes" class="form-textarea" rows="3" placeholder="Any additional notes..." />
        </div>
        <div class="form-group" style="display: flex; align-items: center; gap: 0.75rem;">
          <label class="form-label" style="margin: 0;">Private Coin</label>
          <label class="toggle">
            <input type="checkbox" v-model="form.isPrivate" />
            <span class="toggle-slider"></span>
          </label>
          <span style="font-size: 0.8rem; color: var(--text-secondary);">Hidden from followers</span>
        </div>
      </fieldset>
    </div>

    <div class="form-actions">
      <button type="button" class="btn btn-secondary" @click="$router.back()">Cancel</button>
      <button type="submit" class="btn btn-primary" :disabled="loading">
        {{ loading ? 'Saving...' : submitLabel }}
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { ref, computed, onBeforeUnmount } from 'vue'
import { CATEGORIES, MATERIALS } from '@/types'
import type { Coin } from '@/types'
import AutocompleteInput from '@/components/AutocompleteInput.vue'
import { X, Camera } from 'lucide-vue-next'
import { usePwa } from '@/composables/usePwa'

const { isPwa } = usePwa()

const props = defineProps<{
  form: Partial<Coin>
  submitLabel: string
  loading?: boolean
  coinId?: number
}>()

defineEmits<{ submit: [] }>()

const obverseFile = ref<File | null>(null)
const reverseFile = ref<File | null>(null)
const cardFile = ref<File | null>(null)
const obversePreview = ref<string | null>(null)
const reversePreview = ref<string | null>(null)
const cardPreview = ref<string | null>(null)
const obverseInput = ref<HTMLInputElement | null>(null)
const reverseInput = ref<HTMLInputElement | null>(null)
const cardInput = ref<HTMLInputElement | null>(null)
const removedObverseId = ref<number | null>(null)
const removedReverseId = ref<number | null>(null)

const existingObverse = computed(() => {
  if (removedObverseId.value) return null
  const img = props.form.images?.find((i) => i.imageType === 'obverse')
  return img ? `/uploads/${img.filePath}` : null
})

const existingReverse = computed(() => {
  if (removedReverseId.value) return null
  const img = props.form.images?.find((i) => i.imageType === 'reverse')
  return img ? `/uploads/${img.filePath}` : null
})

function onObverseFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (obversePreview.value) URL.revokeObjectURL(obversePreview.value)
  obverseFile.value = file
  obversePreview.value = URL.createObjectURL(file)
}

function onReverseFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (reversePreview.value) URL.revokeObjectURL(reversePreview.value)
  reverseFile.value = file
  reversePreview.value = URL.createObjectURL(file)
}

function clearObverse() {
  const existing = props.form.images?.find((i) => i.imageType === 'obverse')
  if (existing) removedObverseId.value = existing.id
  if (obversePreview.value) URL.revokeObjectURL(obversePreview.value)
  obverseFile.value = null
  obversePreview.value = null
  if (obverseInput.value) obverseInput.value.value = ''
}

function clearReverse() {
  const existing = props.form.images?.find((i) => i.imageType === 'reverse')
  if (existing) removedReverseId.value = existing.id
  if (reversePreview.value) URL.revokeObjectURL(reversePreview.value)
  reverseFile.value = null
  reversePreview.value = null
  if (reverseInput.value) reverseInput.value.value = ''
}

function onCardFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (cardPreview.value) URL.revokeObjectURL(cardPreview.value)
  cardFile.value = file
  cardPreview.value = URL.createObjectURL(file)
}

function clearCard() {
  if (cardPreview.value) URL.revokeObjectURL(cardPreview.value)
  cardFile.value = null
  cardPreview.value = null
  if (cardInput.value) cardInput.value.value = ''
}

onBeforeUnmount(() => {
  if (obversePreview.value) URL.revokeObjectURL(obversePreview.value)
  if (reversePreview.value) URL.revokeObjectURL(reversePreview.value)
  if (cardPreview.value) URL.revokeObjectURL(cardPreview.value)
})

// Expose pending images for parent to upload after save
defineExpose({
  obverseFile,
  reverseFile,
  cardFile,
  removedObverseId,
  removedReverseId,
})
</script>

<style scoped>
.coin-form {
  max-width: 900px;
  margin-left: auto;
  margin-right: auto;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
}

.form-section {
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1.25rem;
  margin: 0;
  background: var(--bg-card);
}

.form-section.full-width {
  grid-column: 1 / -1;
}

.form-section-title {
  font-family: 'Cinzel', serif;
  color: var(--accent-gold);
  font-size: 1.2rem;
  font-weight: 500;
  margin: 0 0 1rem;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.form-group {
  min-width: 0;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 2rem;
  padding-top: 1.5rem;
  border-top: 1px solid var(--border-subtle);
}

.file-input {
  font-size: 0.9rem;
  padding: 0.6rem 0.8rem;
}

.file-input-row {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.file-input-row .file-input {
  flex: 1;
  min-width: 0;
}

.camera-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  white-space: nowrap;
  cursor: pointer;
}

.image-preview-box {
  position: relative;
  display: inline-block;
  margin-bottom: 0.5rem;
}

.image-preview {
  width: 140px;
  height: 140px;
  object-fit: cover;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-subtle);
}

.image-remove-btn {
  position: absolute;
  top: -6px;
  right: -6px;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: none;
  background: #c0392b;
  color: white;
  font-size: 0.7rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
}

.form-hint {
  font-size: 0.85rem;
  color: var(--text-muted);
  margin-bottom: 0.5rem;
}

@media (max-width: 768px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
  .form-row {
    grid-template-columns: 1fr;
  }
}
</style>
