<template>
  <section class="card readiness-panel">
    <h2>Promote to Coin</h2>

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
      <p class="helper-text">Fill required fields to promote this draft into a normal coin record. Repeated promotion is safe.</p>

      <div class="field-grid">
        <label class="form-group full-width">
          <span>Name <span class="required">*</span></span>
          <input
            v-model="overrideName"
            type="text"
            maxlength="200"
            :placeholder="draft.workingTitle || 'Required for promotion'"
          >
          <span v-if="fieldErrors.name" class="field-error">{{ fieldErrors.name }}</span>
        </label>
        <label class="form-group">
          <span>Category</span>
          <select v-model="overrideCategory">
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
          <span>Material</span>
          <select v-model="overrideMaterial">
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
          <span>Era</span>
          <select v-model="overrideEra">
            <option value="">Use draft value</option>
            <option value="ancient">Ancient</option>
            <option value="medieval">Medieval</option>
            <option value="modern">Modern</option>
          </select>
          <span v-if="fieldErrors.era" class="field-error">{{ fieldErrors.era }}</span>
        </label>
        <label class="form-group">
          <span>Purchase price</span>
          <input v-model.number="overridePrice" type="number" min="0" step="0.01" :placeholder="draft.purchasePrice != null ? String(draft.purchasePrice) : ''">
        </label>
        <label class="form-group full-width">
          <span>Notes</span>
          <textarea v-model="overrideNotes" rows="3" :placeholder="draft.notes || ''"></textarea>
        </label>
      </div>

      <label class="confirm-row">
        <input v-model="confirmed" type="checkbox">
        <span>I confirm promotion. This creates a permanent coin record.</span>
      </label>

      <p v-if="promoteError" class="status-text status-warning">{{ promoteError }}</p>

      <button
        type="button"
        class="btn btn-primary"
        :disabled="!confirmed || promoting"
        @click="doPromote"
      >
        {{ promoting ? 'Promoting...' : 'Promote to Coin' }}
      </button>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { getApiErrorMessage, promoteQuickCaptureDraft } from '@/api/client'
import type { QuickCaptureDraft } from '@/types'

const props = defineProps<{ draft: QuickCaptureDraft }>()
const emit = defineEmits<{ promoted: [coinId: number] }>()

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
const confirmed = ref(false)
const promoting = ref(false)
const promoteError = ref('')
const fieldErrors = ref<Record<string, string>>({})
const successCoinId = ref<number | null>(null)

async function doPromote() {
  promoting.value = true
  promoteError.value = ''
  fieldErrors.value = {}
  try {
    const res = await promoteQuickCaptureDraft(props.draft.id, {
      confirm: true,
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
.confirm-row {
  display: flex;
  align-items: flex-start;
  gap: 0.6rem;
  margin: 1rem 0;
  cursor: pointer;
  font-size: 0.95rem;
}
.confirm-row input[type='checkbox'] {
  margin-top: 0.15rem;
  flex-shrink: 0;
}
.field-error {
  color: var(--color-warning, #d97706);
  font-size: 0.85rem;
}
.required {
  color: var(--color-warning, #d97706);
}
</style>
