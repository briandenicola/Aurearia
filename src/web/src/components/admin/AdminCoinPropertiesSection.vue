<template>
  <section class="admin-section card">
    <div class="section-heading">
      <div>
        <p class="section-label">Collection metadata</p>
        <h2>Coin Properties</h2>
      </div>
      <button type="submit" form="coin-properties-form" class="btn btn-primary btn-sm" :disabled="saving">
        {{ saving ? 'Saving...' : 'Save Properties' }}
      </button>
    </div>

    <p class="section-description">
      Configure the category and era choices shown on Add Coin and Edit Coin. Enter one value per line.
    </p>

    <form id="coin-properties-form" class="properties-form" @submit.prevent="$emit('save')">
      <div class="property-card">
        <div class="property-card-header">
          <div>
            <label class="form-label" for="category-options">Category Options</label>
            <p class="form-hint">One category per line. Empty lines are ignored.</p>
          </div>
          <span class="chip-sm">{{ categoryPreview.length }} options</span>
        </div>
        <textarea
          id="category-options"
          v-model="localCategoryOptions"
          class="form-textarea property-textarea"
          rows="7"
          placeholder="Roman&#10;Greek&#10;Byzantine&#10;Modern&#10;Other"
        />
        <div class="option-preview" aria-label="Category option preview">
          <span v-for="option in categoryPreview" :key="option" class="chip-sm">{{ option }}</span>
        </div>
      </div>

      <div class="property-card">
        <div class="property-card-header">
          <div>
            <label class="form-label" for="era-options">Era Options</label>
            <p class="form-hint">One era per line. The coin form adds Unspecified automatically.</p>
          </div>
          <span class="chip-sm">{{ eraPreview.length }} options</span>
        </div>
        <textarea
          id="era-options"
          v-model="localEraOptions"
          class="form-textarea property-textarea"
          rows="7"
          placeholder="ancient&#10;medieval&#10;modern"
        />
        <div class="option-preview" aria-label="Era option preview">
          <span class="chip-sm">Unspecified</span>
          <span v-for="option in eraPreview" :key="option" class="chip-sm">{{ option }}</span>
        </div>
      </div>

      <p v-if="msg" class="msg" :class="{ error }">{{ msg }}</p>
    </form>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { parseOptionList } from '@/utils/options'
import { CATEGORIES, COIN_ERAS } from '@/types'

const props = defineProps<{
  categoryOptions: string
  eraOptions: string
  saving: boolean
  msg: string
  error: boolean
}>()

const emit = defineEmits<{
  save: []
  'update:categoryOptions': [value: string]
  'update:eraOptions': [value: string]
}>()

const localCategoryOptions = ref(props.categoryOptions)
const localEraOptions = ref(props.eraOptions)

watch(() => props.categoryOptions, (v) => { localCategoryOptions.value = v })
watch(() => props.eraOptions, (v) => { localEraOptions.value = v })

watch(localCategoryOptions, (v) => emit('update:categoryOptions', v))
watch(localEraOptions, (v) => emit('update:eraOptions', v))

const categoryPreview = computed(() => parseOptionList(localCategoryOptions.value, CATEGORIES))
const eraPreview = computed(() => parseOptionList(localEraOptions.value, COIN_ERAS))
</script>

<style scoped>
.section-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.75rem;
}

.section-heading h2 {
  margin: 0;
}

.section-description {
  font-size: 0.9rem;
  color: var(--text-secondary);
  margin: 0 0 1.5rem;
}

.properties-form {
  display: grid;
  gap: 1rem;
}

.property-card {
  background: var(--bg-card-hover);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1rem;
}

.property-card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.75rem;
}

.form-label {
  color: var(--text-heading);
}

.form-hint {
  display: block;
  margin: 0.25rem 0 0;
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.property-textarea {
  min-height: 11rem;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-family: inherit;
  font-size: 0.85rem;
  line-height: 1.5;
  resize: vertical;
}

.property-textarea:focus {
  border-color: var(--accent-gold);
  box-shadow: var(--shadow-glow);
  outline: none;
}

.option-preview {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  margin-top: 0.75rem;
}

.msg {
  font-size: 0.85rem;
  color: var(--accent-gold);
  margin: 0.5rem 0;
}

.msg.error {
  color: var(--cat-byzantine);
}

@media (max-width: 768px) {
  .section-heading,
  .property-card-header {
    flex-direction: column;
  }
}
</style>
