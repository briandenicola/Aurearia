<template>
  <div class="container">
    <div class="form-wrapper">
      <div class="page-header">
        <h1>Quick Capture Draft</h1>
        <RouterLink class="btn btn-secondary" to="/quick-capture/drafts">All drafts</RouterLink>
      </div>

      <p v-if="loading" class="status-text">Loading draft...</p>
      <p v-else-if="loadError" class="status-text status-warning">{{ loadError }}</p>

      <template v-else-if="draft">
        <!-- Promoted state -->
        <section v-if="draft.status === 'promoted'" class="card">
          <p class="status-text">This draft was promoted to a coin.</p>
          <RouterLink v-if="draft.promotedCoinId" class="btn btn-primary" :to="`/coin/${draft.promotedCoinId}`">View Coin</RouterLink>
        </section>

        <!-- Discarded state -->
        <section v-else-if="draft.status === 'discarded'" class="card">
          <p class="status-text status-warning">This draft has been discarded.</p>
        </section>

        <!-- Active editing state -->
        <template v-else>
          <form class="card" @submit.prevent="saveDraft">
            <h2>Edit Draft</h2>
            <!-- Existing images -->
            <section v-if="draft.images.length" class="existing-images">
              <h3>Current images</h3>
              <div class="image-grid">
                <div v-for="img in draft.images" :key="img.id" class="image-item">
                  <AuthenticatedImage
                    :media-path="img.filePath"
                    :alt="img.imageType"
                    class="thumb"
                  />
                  <span class="chip-sm">{{ img.imageType }}</span>
                  <button
                    type="button"
                    class="btn-icon remove-btn"
                    :disabled="saving"
                    @click="toggleRemoveImage(img.id)"
                  >
                    <span v-if="removeImageIds.has(img.id)" class="status-text">Undo remove</span>
                    <span v-else class="status-text status-warning">Remove</span>
                  </button>
                </div>
              </div>
            </section>

            <!-- New images -->
            <QuickCaptureImageSlots
              v-model:obverse-image="newObverse"
              v-model:reverse-image="newReverse"
              v-model:detail-images="newDetails"
            />

            <div class="field-grid">
              <label class="form-group">
                <span>Working title</span>
                <input v-model="workingTitle" type="text" maxlength="200" placeholder="Unattributed denarius">
              </label>
              <label class="form-group">
                <span>Date range</span>
                <input v-model="dateRange" type="text" placeholder="c. 330–335">
              </label>
              <label class="form-group">
                <span>Era</span>
                <input v-model="era" type="text" placeholder="ancient">
              </label>
              <label class="form-group">
                <span>Acquisition source</span>
                <input v-model="acquisitionSource" type="text" placeholder="Show table">
              </label>
              <label class="form-group">
                <span>Purchase price</span>
                <input v-model.number="purchasePrice" type="number" min="0" step="0.01">
              </label>
              <label class="form-group full-width">
                <span>Notes</span>
                <textarea v-model="notes" rows="4" placeholder="Quick notes for later attribution"></textarea>
              </label>
            </div>

            <p v-if="saveError" class="status-text status-warning">{{ saveError }}</p>
            <p v-if="saveSuccess" class="status-text">Draft saved.</p>

            <div class="action-row">
              <button type="submit" class="btn btn-primary" :disabled="saving">
                {{ saving ? 'Saving...' : 'Save Changes' }}
              </button>
              <button
                v-if="!confirmingDiscard"
                type="button"
                class="btn btn-secondary"
                :disabled="saving"
                @click="confirmingDiscard = true"
              >
                Discard Draft
              </button>
              <template v-else>
                <span class="status-text status-warning">Discard this draft?</span>
                <button type="button" class="btn btn-secondary" :disabled="discarding" @click="doDiscard">
                  {{ discarding ? 'Discarding...' : 'Yes, discard' }}
                </button>
                <button type="button" class="btn btn-secondary" @click="confirmingDiscard = false">Cancel</button>
              </template>
            </div>
          </form>

          <!-- Promotion panel -->
          <PromotionReadinessPanel :draft="draft" @promoted="onPromoted" />
        </template>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import {
  getApiErrorMessage,
  getQuickCaptureDraft,
  updateQuickCaptureDraft,
  discardQuickCaptureDraft,
} from '@/api/client'
import type { QuickCaptureDraft } from '@/types'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'
import QuickCaptureImageSlots from '@/components/quick-capture/QuickCaptureImageSlots.vue'
import PromotionReadinessPanel from '@/components/quick-capture/PromotionReadinessPanel.vue'

const route = useRoute()
const router = useRouter()

const draft = ref<QuickCaptureDraft | null>(null)
const loading = ref(true)
const loadError = ref('')

// Edit form state
const workingTitle = ref('')
const dateRange = ref('')
const era = ref('')
const acquisitionSource = ref('')
const purchasePrice = ref<number | null>(null)
const notes = ref('')
const removeImageIds = ref<Set<number>>(new Set())
const newObverse = ref<File | null>(null)
const newReverse = ref<File | null>(null)
const newDetails = ref<File[]>([])

const saving = ref(false)
const saveError = ref('')
const saveSuccess = ref(false)

// Discard state
const confirmingDiscard = ref(false)
const discarding = ref(false)

function populateForm(d: QuickCaptureDraft) {
  workingTitle.value = d.workingTitle ?? ''
  dateRange.value = d.dateRange ?? ''
  era.value = d.era ?? ''
  acquisitionSource.value = d.acquisitionSource ?? ''
  purchasePrice.value = d.purchasePrice ?? null
  notes.value = d.notes ?? ''
  removeImageIds.value = new Set()
  newObverse.value = null
  newReverse.value = null
  newDetails.value = []
  saveError.value = ''
  saveSuccess.value = false
}

function toggleRemoveImage(id: number) {
  const s = removeImageIds.value
  if (s.has(id)) s.delete(id)
  else s.add(id)
}

onMounted(async () => {
  try {
    const res = await getQuickCaptureDraft(Number(route.params['id']))
    draft.value = res.data
    populateForm(res.data)
  } catch (err) {
    loadError.value = getApiErrorMessage(err) || 'Unable to load quick capture draft.'
  } finally {
    loading.value = false
  }
})

async function saveDraft() {
  saving.value = true
  saveError.value = ''
  saveSuccess.value = false
  try {
    const res = await updateQuickCaptureDraft(draft.value!.id, {
      workingTitle: workingTitle.value,
      dateRange: dateRange.value,
      era: era.value,
      acquisitionSource: acquisitionSource.value,
      purchasePrice: purchasePrice.value,
      notes: notes.value,
      removeImageIds: removeImageIds.value.size > 0 ? [...removeImageIds.value].join(',') : undefined,
      obverseImage: newObverse.value,
      reverseImage: newReverse.value,
      detailImages: newDetails.value,
    })
    draft.value = res.data
    populateForm(res.data)
    saveSuccess.value = true
  } catch (err) {
    saveError.value = getApiErrorMessage(err) || 'Failed to save draft. Please try again.'
  } finally {
    saving.value = false
  }
}

async function doDiscard() {
  discarding.value = true
  try {
    const res = await discardQuickCaptureDraft(draft.value!.id)
    draft.value = res.data
    confirmingDiscard.value = false
  } catch (err) {
    saveError.value = getApiErrorMessage(err) || 'Failed to discard draft.'
    confirmingDiscard.value = false
  } finally {
    discarding.value = false
  }
}

function onPromoted(coinId: number) {
  if (draft.value) {
    draft.value = { ...draft.value, status: 'promoted', promotedCoinId: coinId }
  }
  router.push(`/coin/${coinId}`)
}
</script>

<style scoped>
.existing-images {
  margin-bottom: 1rem;
}
.image-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  margin-top: 0.5rem;
}
.image-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.4rem;
}
.thumb {
  width: 80px;
  height: 80px;
  object-fit: cover;
  border-radius: 0.75rem;
  background: var(--color-surface-muted);
}
.action-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
  margin-top: 1.25rem;
}
.remove-btn {
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
}
</style>
