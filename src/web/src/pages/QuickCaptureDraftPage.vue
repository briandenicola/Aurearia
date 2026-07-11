<template>
  <div class="container">
    <div class="form-wrapper grid gap-4">
      <div class="page-header">
        <h1>Draft</h1>
        <div class="pwa-actions">
          <RouterLink class="pwa-icon-btn" to="/quick-capture/drafts" title="All drafts" aria-label="All drafts">
            <List :size="22" />
          </RouterLink>
        </div>
      </div>

      <p v-if="loading" class="m-0 text-chip text-text-secondary">Loading draft...</p>
      <p v-else-if="loadError" class="m-0 text-chip text-warning">{{ loadError }}</p>

      <template v-else-if="draft">
        <!-- Promoted state -->
        <section v-if="draft.status === 'promoted'" class="card grid gap-4">
          <p class="m-0 text-chip text-text-secondary">This draft was promoted to a coin.</p>
          <RouterLink v-if="draft.promotedCoinId" class="btn btn-primary w-fit" :to="`/coin/${draft.promotedCoinId}`">View Coin</RouterLink>
        </section>

        <!-- Discarded state -->
        <section v-else-if="draft.status === 'discarded'" class="card grid gap-4">
          <p class="m-0 text-chip text-warning">This draft has been discarded.</p>
        </section>

        <!-- Active editing state -->
        <template v-else>
          <form class="card grid gap-5" @submit.prevent="saveDraft">
            <h2 class="m-0">Edit Draft</h2>
            <section
              v-if="draft.source === 'find_coin_ai' || draft.ngcCertNumber || draft.labelText"
              class="rounded-sm border border-border-subtle bg-input p-4"
            >
              <h3 class="m-0">Find Coin / AI Capture</h3>
              <div class="mt-3 grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(160px,1fr))]">
                <div v-if="draft.source === 'find_coin_ai'" class="flex flex-col gap-1">
                  <span class="section-label">Source</span>
                  <strong>Quick AI Draft</strong>
                </div>
                <div v-if="draft.aiConfidence" class="flex flex-col gap-1">
                  <span class="section-label">Confidence</span>
                  <strong>{{ draft.aiConfidence }}</strong>
                </div>
                <div v-if="draft.ngcCertNumber" class="flex flex-col gap-1">
                  <span class="section-label">NGC Coin Number</span>
                  <strong>{{ draft.ngcCertNumber }}</strong>
                </div>
                <div v-if="draft.ngcGrade" class="flex flex-col gap-1">
                  <span class="section-label">NGC Grade</span>
                  <strong>{{ draft.ngcGrade }}</strong>
                </div>
              </div>
              <p v-if="draft.labelText" class="mt-3 mb-0 text-body text-text-secondary">{{ draft.labelText }}</p>
            </section>
            <!-- Existing images -->
            <section v-if="draft.images.length" class="grid gap-3">
              <h3 class="m-0">Current images</h3>
              <div class="flex flex-wrap gap-4">
                <div v-for="img in draft.images" :key="img.id" class="flex flex-col items-center gap-1.5">
                  <AuthenticatedImage
                    :media-path="img.filePath"
                    :alt="img.imageType"
                    class="h-20 w-20 rounded-sm bg-input object-cover"
                  />
                  <span class="chip-sm">{{ img.imageType }}</span>
                  <button
                    type="button"
                    class="cursor-pointer border-0 bg-transparent p-0 disabled:cursor-not-allowed disabled:opacity-50"
                    :disabled="saving"
                    @click="toggleRemoveImage(img.id)"
                  >
                    <span v-if="removeImageIds.has(img.id)" class="text-chip text-text-secondary">Undo remove</span>
                    <span v-else class="text-chip text-warning">Remove</span>
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

            <div class="grid gap-4 max-[600px]:grid-cols-1 [grid-template-columns:repeat(auto-fit,minmax(220px,1fr))]">
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
              <label class="form-group col-[1/-1]">
                <span class="section-label">Notes</span>
                <textarea v-model="notes" class="form-textarea min-h-28 leading-6" rows="4" placeholder="Quick notes for later attribution"></textarea>
              </label>
            </div>

            <p v-if="saveError" class="m-0 text-chip text-warning">{{ saveError }}</p>
            <p v-if="saveSuccess" class="m-0 text-chip text-text-secondary">Draft saved.</p>

            <div class="flex flex-wrap items-center gap-3 max-[600px]:[&>.btn]:w-full max-[600px]:[&>.btn]:justify-center">
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
                <span class="text-chip text-warning">Discard this draft?</span>
                <button type="button" class="btn btn-secondary" :disabled="discarding" @click="doDiscard">
                  {{ discarding ? 'Discarding...' : 'Yes, discard' }}
                </button>
                <button type="button" class="btn btn-secondary" @click="confirmingDiscard = false">Cancel</button>
              </template>
            </div>
          </form>

          <!-- Promotion panel -->
          <PromotionReadinessPanel :draft="draft" :promotion-overrides="promotionOverrides" @promoted="onPromoted" />
        </template>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import {
  getApiErrorMessage,
  getQuickCaptureDraft,
  updateQuickCaptureDraft,
  discardQuickCaptureDraft,
} from '@/api/client'
import type { QuickCaptureDraft, QuickCapturePromoteOverrides } from '@/types'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'
import QuickCaptureImageSlots from '@/components/quick-capture/QuickCaptureImageSlots.vue'
import PromotionReadinessPanel from '@/components/quick-capture/PromotionReadinessPanel.vue'
import { List } from 'lucide-vue-next'

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

const promotionOverrides = computed<QuickCapturePromoteOverrides>(() => ({
  name: workingTitle.value.trim(),
  era: era.value.trim(),
  purchaseLocation: acquisitionSource.value.trim(),
  purchasePrice: currentPurchasePrice(),
  notes: notes.value.trim(),
}))

function currentPurchasePrice(): number | null {
  return typeof purchasePrice.value === 'number' ? purchasePrice.value : null
}

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
      purchasePrice: currentPurchasePrice(),
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
