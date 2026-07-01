<template>
  <section class="card readiness-panel">
    <div class="panel-heading">
      <div>
        <span class="section-label">Ready for cataloging</span>
        <h2>Promote Draft</h2>
      </div>
    </div>

    <!-- Already promoted -->
    <template v-if="alreadyPromoted">
      <p class="status-text">This draft was already promoted.</p>
      <RouterLink class="btn btn-primary" :to="`/coin/${promotedCoinId}`">View Coin</RouterLink>
    </template>

    <!-- Success -->
    <template v-else-if="successCoinId">
      <p class="status-text">Draft promoted successfully.</p>
      <RouterLink class="btn btn-primary" :to="`/coin/${successCoinId}`">View Coin</RouterLink>
    </template>

    <!-- Promotion form -->
    <template v-else>
      <p class="helper-text">Choose where this coin should land, review readiness, then promote it. Repeated promotion is safe.</p>

      <fieldset class="destination-options">
        <legend class="section-label">Promote to</legend>
        <label class="destination-option" :class="{ selected: target === 'collection' }">
          <input v-model="target" type="radio" value="collection">
          <Coins :size="20" />
          <span>
            <strong>Collection</strong>
            <small>Counts as an owned collection coin.</small>
          </span>
        </label>
        <label class="destination-option" :class="{ selected: target === 'wishlist' }">
          <input v-model="target" type="radio" value="wishlist">
          <Bookmark :size="20" />
          <span>
            <strong>Wishlist</strong>
            <small>Tracks as a wanted coin instead.</small>
          </span>
        </label>
        <span v-if="fieldErrors.target" class="field-error">{{ fieldErrors.target }}</span>
      </fieldset>

      <div class="readiness-summary" aria-live="polite">
        <div class="readiness-item" :class="{ ready: hasRequiredName }">
          <CheckCircle v-if="hasRequiredName" :size="18" />
          <AlertCircle v-else :size="18" />
          <span>
            <strong>{{ hasRequiredName ? 'Required title is ready' : 'Working title is required' }}</strong>
            <small>{{ hasRequiredName ? 'Promotion uses the current draft title above.' : 'Add a working title in the draft form before promoting.' }}</small>
          </span>
        </div>
        <div class="readiness-item ready">
          <ImageIcon :size="18" />
          <span>
            <strong>{{ imageCountLabel }}</strong>
            <small>Saved draft images will move with the promoted coin.</small>
          </span>
        </div>
        <p class="summary-note">Draft fields stay editable in the form above. This panel only chooses the destination and confirms the final action.</p>
        <span v-if="fieldErrors.name" class="field-error">{{ fieldErrors.name }}</span>
        <div v-if="readinessFieldErrors.length" class="field-error-list">
          <span v-for="[field, message] in readinessFieldErrors" :key="field">{{ message }}</span>
        </div>
      </div>

      <label class="confirm-row">
        <input v-model="confirmed" type="checkbox">
        <span>I confirm promotion to {{ destinationLabel }}. This creates a permanent coin record.</span>
      </label>

      <p v-if="promoteError" class="status-text status-warning">{{ promoteError }}</p>

      <div class="promotion-actions">
        <button
          type="button"
          class="btn btn-primary"
          :disabled="!confirmed || promoting || !hasRequiredName"
          @click="doPromote"
        >
          {{ promoting ? 'Promoting...' : `Promote to ${destinationLabel}` }}
        </button>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { getApiErrorMessage, promoteQuickCaptureDraft } from '@/api/client'
import type { QuickCaptureDraft, QuickCapturePromoteOverrides } from '@/types'
import { AlertCircle, Bookmark, CheckCircle, Coins, Image as ImageIcon } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{ draft: QuickCaptureDraft; promotionOverrides?: QuickCapturePromoteOverrides }>(),
  { promotionOverrides: () => ({}) }
)
const emit = defineEmits<{ promoted: [coinId: number] }>()

type PromotionTarget = 'collection' | 'wishlist'

const alreadyPromoted = computed(
  () => props.draft.status === 'promoted' && props.draft.promotedCoinId != null
)
const promotedCoinId = computed(() => props.draft.promotedCoinId)

const target = ref<PromotionTarget>('collection')
const confirmed = ref(false)
const promoting = ref(false)
const promoteError = ref('')
const fieldErrors = ref<Record<string, string>>({})
const successCoinId = ref<number | null>(null)
const destinationLabel = computed(() => target.value === 'wishlist' ? 'Wishlist' : 'Collection')
const hasRequiredName = computed(() => (props.promotionOverrides.name ?? props.draft.workingTitle ?? '').trim().length > 0)
const imageCount = computed(() => props.draft.images?.length ?? 0)
const imageCountLabel = computed(() => `${imageCount.value} saved ${imageCount.value === 1 ? 'image' : 'images'}`)
const readinessFieldErrors = computed(() => Object.entries(fieldErrors.value)
  .filter(([field]) => !['target', 'name', 'confirm'].includes(field)))

async function doPromote() {
  promoteError.value = ''
  fieldErrors.value = {}
  if (!hasRequiredName.value) {
    fieldErrors.value = { name: 'Name is required' }
    promoteError.value = 'Complete required fields before promotion.'
    return
  }
  promoting.value = true
  try {
    const res = await promoteQuickCaptureDraft(props.draft.id, {
      confirm: true,
      target: target.value,
      overrides: props.promotionOverrides,
    })
    if (res.data.alreadyPromoted) {
      // trigger idempotent path in parent
      emit('promoted', res.data.coinId)
    } else {
      successCoinId.value = res.data.coinId
      emit('promoted', res.data.coinId)
    }
  } catch (err: unknown) {
    const errData = (err as { response?: { data?: { error?: string; fields?: Record<string, string> } } })
      ?.response?.data
    if (errData?.fields) {
      fieldErrors.value = errData.fields
      promoteError.value = errData.error ?? 'Complete required fields before promotion.'
    } else {
      promoteError.value = getApiErrorMessage(err) || 'Promotion failed. Please try again.'
    }
  } finally {
    promoting.value = false
  }
}
</script>

<style scoped>
.readiness-panel {
  margin-top: 1.5rem;
}

.panel-heading {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}

.panel-heading h2 {
  margin: 0.25rem 0 0;
}

.destination-options {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.75rem;
  margin: 0;
  padding: 0;
  border: 0;
}

.destination-options legend {
  grid-column: 1 / -1;
  margin-bottom: 0.25rem;
}

.destination-option {
  display: flex;
  gap: 0.75rem;
  align-items: flex-start;
  padding: 0.75rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-input);
  color: var(--text-primary);
  cursor: pointer;
  transition: border-color var(--transition-fast), background var(--transition-fast), box-shadow var(--transition-fast);
}

.destination-option.selected {
  border-color: var(--accent-gold);
  background: var(--accent-gold-glow);
  box-shadow: var(--shadow-glow);
}

.destination-option input {
  margin-top: 0.2rem;
  accent-color: var(--accent-gold);
}

.destination-option svg {
  flex: 0 0 auto;
  color: var(--accent-gold);
}

.destination-option span {
  display: grid;
  gap: 0.25rem;
}

.destination-option strong {
  font-size: 0.9rem;
}

.destination-option small {
  color: var(--text-secondary);
  font-size: 0.8rem;
}

.readiness-summary {
  display: grid;
  gap: 0.75rem;
  padding: 0.75rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-input);
}

.readiness-item {
  display: flex;
  gap: 0.75rem;
  align-items: flex-start;
  color: var(--text-secondary);
}

.readiness-item.ready {
  color: var(--text-primary);
}

.readiness-item svg {
  flex: 0 0 auto;
  margin-top: 0.15rem;
  color: var(--text-warning);
}

.readiness-item.ready svg {
  color: var(--accent-gold);
}

.readiness-item span {
  display: grid;
  gap: 0.25rem;
}

.readiness-item strong {
  font-size: 0.9rem;
}

.readiness-item small,
.summary-note {
  color: var(--text-secondary);
  font-size: 0.8rem;
}

.summary-note {
  margin: 0;
}

.confirm-row {
  display: flex;
  align-items: flex-start;
  gap: 0.6rem;
  margin: 1rem 0;
  cursor: pointer;
  font-size: 0.9rem;
  color: var(--text-secondary);
}
.confirm-row input[type='checkbox'] {
  margin-top: 0.15rem;
  flex-shrink: 0;
  accent-color: var(--accent-gold);
}
.field-error,
.field-error-list {
  color: var(--text-warning);
  font-size: 0.85rem;
}

.field-error-list {
  display: grid;
}
.promotion-actions {
  display: flex;
  justify-content: flex-end;
}

@media (max-width: 600px) {
  .destination-options {
    grid-template-columns: 1fr;
  }

  .promotion-actions .btn {
    width: 100%;
    justify-content: center;
  }
}
</style>
