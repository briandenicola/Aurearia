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
      <p class="helper-text">Choose where this coin should land, fill any missing required fields, then promote it. Repeated promotion is safe.</p>

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

      <div class="field-grid">
        <label class="form-group full-width">
          <span class="section-label">Name <span class="required">*</span></span>
          <input
            v-model="overrideName"
            class="form-input"
            type="text"
            maxlength="200"
            :placeholder="draft.workingTitle || 'Required for promotion'"
          >
          <span v-if="fieldErrors.name" class="field-error">{{ fieldErrors.name }}</span>
        </label>
        <label class="form-group">
          <span class="section-label">Category</span>
          <select v-model="overrideCategory" class="form-select">
            <option value="">Other (default)</option>
            <option value="Roman">Roman</option>
            <option value="Greek">Greek</option>
            <option value="Byzantine">Byzantine</option>
            <option value="Medieval">Medieval</option>
            <option value="Modern">Modern</option>
            <option value="Other">Other</option>
          </select>
        </label>
        <label class="form-group">
          <span class="section-label">Material</span>
          <select v-model="overrideMaterial" class="form-select">
            <option value="">Other (default)</option>
            <option value="Gold">Gold</option>
            <option value="Silver">Silver</option>
            <option value="Bronze">Bronze</option>
            <option value="Copper">Copper</option>
            <option value="Electrum">Electrum</option>
            <option value="Other">Other</option>
          </select>
        </label>
        <label class="form-group">
          <span class="section-label">Era</span>
          <select v-model="overrideEra" class="form-select">
            <option value="">Use draft value</option>
            <option value="ancient">Ancient</option>
            <option value="medieval">Medieval</option>
            <option value="modern">Modern</option>
          </select>
          <span v-if="fieldErrors.era" class="field-error">{{ fieldErrors.era }}</span>
        </label>
        <label class="form-group">
          <span class="section-label">Purchase price</span>
          <input v-model.number="overridePrice" class="form-input" type="number" min="0" step="0.01" :placeholder="draft.purchasePrice != null ? String(draft.purchasePrice) : ''">
        </label>
        <label class="form-group full-width">
          <span class="section-label">Notes</span>
          <textarea v-model="overrideNotes" class="form-textarea" rows="3" :placeholder="draft.notes || ''"></textarea>
        </label>
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
          :disabled="!confirmed || promoting"
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
import type { QuickCaptureDraft } from '@/types'
import { Bookmark, Coins } from 'lucide-vue-next'

const props = defineProps<{ draft: QuickCaptureDraft }>()
const emit = defineEmits<{ promoted: [coinId: number] }>()

type PromotionTarget = 'collection' | 'wishlist'

const alreadyPromoted = computed(
  () => props.draft.status === 'promoted' && props.draft.promotedCoinId != null
)
const promotedCoinId = computed(() => props.draft.promotedCoinId)

const overrideName = ref('')
const overrideCategory = ref('')
const overrideMaterial = ref('')
const overrideEra = ref('')
const overridePrice = ref<number | null>(null)
const overrideNotes = ref('')
const target = ref<PromotionTarget>('collection')
const confirmed = ref(false)
const promoting = ref(false)
const promoteError = ref('')
const fieldErrors = ref<Record<string, string>>({})
const successCoinId = ref<number | null>(null)
const destinationLabel = computed(() => target.value === 'wishlist' ? 'Wishlist' : 'Collection')

async function doPromote() {
  promoting.value = true
  promoteError.value = ''
  fieldErrors.value = {}
  try {
    const res = await promoteQuickCaptureDraft(props.draft.id, {
      confirm: true,
      target: target.value,
      overrides: {
        name: overrideName.value || undefined,
        category: overrideCategory.value || undefined,
        material: overrideMaterial.value || undefined,
        era: overrideEra.value || undefined,
        purchasePrice: overridePrice.value ?? undefined,
        notes: overrideNotes.value || undefined,
      },
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

.field-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  margin-bottom: 0;
}

.full-width {
  grid-column: 1 / -1;
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
.field-error {
  color: var(--text-warning);
  font-size: 0.85rem;
}
.required {
  color: var(--text-warning);
}

.promotion-actions {
  display: flex;
  justify-content: flex-end;
}

@media (max-width: 600px) {
  .destination-options,
  .field-grid {
    grid-template-columns: 1fr;
  }

  .promotion-actions .btn {
    width: 100%;
    justify-content: center;
  }
}
</style>
