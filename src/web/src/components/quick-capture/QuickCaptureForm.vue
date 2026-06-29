<template>
  <form class="quick-capture-form card" @submit.prevent="saveDraft">
    <QuickCaptureImageSlots
      v-model:obverse-image="obverseImage"
      v-model:reverse-image="reverseImage"
      v-model:detail-images="detailImages"
    />

    <div class="field-grid">
      <label class="form-group">
        <span class="section-label">Working title</span>
        <input v-model="workingTitle" class="form-input" type="text" maxlength="200" placeholder="Unattributed denarius">
      </label>
      <label class="form-group">
        <span class="section-label">Date range</span>
        <input v-model="dateRange" class="form-input" type="text" placeholder="c. 330-335">
      </label>
      <label class="form-group">
        <span class="section-label">Era</span>
        <input v-model="era" class="form-input" type="text" placeholder="ancient">
      </label>
      <label class="form-group">
        <span class="section-label">Acquisition source</span>
        <input v-model="acquisitionSource" class="form-input" type="text" placeholder="Show table">
      </label>
      <label class="form-group">
        <span class="section-label">Purchase price</span>
        <input v-model.number="purchasePrice" class="form-input" type="number" min="0" step="0.01">
      </label>
      <label class="form-group full-width">
        <span class="section-label">Notes</span>
        <textarea v-model="notes" class="form-textarea" rows="4" placeholder="Quick notes for later attribution"></textarea>
      </label>
    </div>

    <p class="helper-text">Save with a title, note, or at least one image. Drafts stay out of your collection until promoted later.</p>
    <p v-if="error" class="status-text status-warning">{{ error }}</p>
    <p v-if="savedMessage" class="status-text">{{ savedMessage }}</p>
    <button type="submit" class="btn btn-primary" :disabled="saving || !canSave">
      {{ saving ? 'Saving...' : 'Save Quick Capture Draft' }}
    </button>
  </form>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { createQuickCaptureDraft, getApiErrorMessage } from '@/api/client'
import type { QuickCaptureDraft } from '@/types'
import QuickCaptureImageSlots from './QuickCaptureImageSlots.vue'

const emit = defineEmits<{
  saved: [draft: QuickCaptureDraft]
}>()

const workingTitle = ref('')
const dateRange = ref('')
const era = ref('')
const acquisitionSource = ref('')
const purchasePrice = ref<number | null>(null)
const notes = ref('')
const obverseImage = ref<File | null>(null)
const reverseImage = ref<File | null>(null)
const detailImages = ref<File[]>([])
const saving = ref(false)
const error = ref('')
const savedMessage = ref('')

const canSave = computed(() =>
  workingTitle.value.trim() !== '' ||
  notes.value.trim() !== '' ||
  Boolean(obverseImage.value || reverseImage.value || detailImages.value.length),
)

async function saveDraft() {
  if (!canSave.value || saving.value) return
  saving.value = true
  error.value = ''
  savedMessage.value = ''
  try {
    const response = await createQuickCaptureDraft({
      workingTitle: workingTitle.value,
      dateRange: dateRange.value,
      era: era.value,
      acquisitionSource: acquisitionSource.value,
      purchasePrice: purchasePrice.value,
      notes: notes.value,
      obverseImage: obverseImage.value,
      reverseImage: reverseImage.value,
      detailImages: detailImages.value,
    })
    savedMessage.value = 'Draft saved as active and incomplete.'
    emit('saved', response.data)
  } catch (err) {
    error.value = getApiErrorMessage(err) || 'Unable to save quick capture draft.'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.quick-capture-form {
  display: grid;
  gap: 1rem;
}

.field-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.full-width {
  grid-column: 1 / -1;
}

.form-input,
.form-textarea {
  width: 100%;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-input);
  color: var(--text-primary);
  font-family: inherit;
  font-size: 0.9rem;
  padding: 0.6rem 0.75rem;
}

.form-input:focus,
.form-textarea:focus {
  outline: none;
  border-color: var(--accent-gold);
  box-shadow: 0 0 0 2px var(--accent-gold-glow);
}

.form-textarea {
  min-height: 7rem;
  resize: vertical;
  line-height: 1.5;
}

.helper-text {
  color: var(--text-secondary);
  margin: 0;
}

@media (max-width: 600px) {
  .field-grid {
    grid-template-columns: 1fr;
  }
}
</style>
