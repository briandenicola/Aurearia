<template>
  <div class="smart-builder">
    <div class="builder-header">
      <span class="section-label">Smart rules</span>
      <div class="operator-toggle">
        <span class="operator-label">Match</span>
        <button
          type="button"
          :class="['btn btn-xs', operator === 'and' ? 'btn-primary' : 'btn-ghost']"
          @click="operator = 'and'"
        >All</button>
        <button
          type="button"
          :class="['btn btn-xs', operator === 'or' ? 'btn-primary' : 'btn-ghost']"
          @click="operator = 'or'"
        >Any</button>
        <span class="operator-label">of these rules</span>
      </div>
    </div>

    <div v-for="(rule, idx) in rules" :key="idx" class="rule-row">
      <select v-model="rule.field" class="rule-input" @change="onFieldChange(rule)">
        <optgroup label="Coin Properties">
          <option value="material">Material</option>
          <option value="category">Category</option>
          <option value="denomination">Denomination</option>
          <option value="ruler">Ruler / Issuer</option>
          <option value="era">Era</option>
          <option value="mint">Mint</option>
          <option value="grade">Grade</option>
        </optgroup>
        <optgroup label="Value & Dates">
          <option value="currentValue">Current value ($)</option>
          <option value="purchasePrice">Purchase price ($)</option>
          <option value="purchaseDate">Purchase date</option>
          <option value="createdAt">Date added</option>
        </optgroup>
        <optgroup label="Status">
          <option value="isWishlist">Wishlist</option>
          <option value="isSold">Sold</option>
          <option value="isPrivate">Private</option>
        </optgroup>
      </select>

      <select v-model="rule.op" class="rule-input" @change="onOpChange(rule)">
        <option v-for="op in opsForField(rule.field)" :key="op.value" :value="op.value">
          {{ op.label }}
        </option>
      </select>

      <template v-if="rule.op !== 'isNull' && rule.op !== 'isNotNull'">
        <input
          v-if="isBoolField(rule.field)"
          type="hidden"
          :value="true"
        />
        <select
          v-else-if="rule.field === 'material'"
          v-model="rule.value"
          class="rule-input"
          @change="emitCriteria"
        >
          <option value="Silver">Silver</option>
          <option value="Gold">Gold</option>
          <option value="Bronze">Bronze</option>
          <option value="Copper">Copper</option>
          <option value="Electrum">Electrum</option>
          <option value="Other">Other</option>
        </select>
        <select
          v-else-if="rule.field === 'category'"
          v-model="rule.value"
          class="rule-input"
          @change="emitCriteria"
        >
          <option value="Roman">Roman</option>
          <option value="Greek">Greek</option>
          <option value="Byzantine">Byzantine</option>
          <option value="Modern">Modern</option>
          <option value="Other">Other</option>
        </select>
        <input
          v-else-if="isNumericField(rule.field)"
          v-model="rule.value"
          type="number"
          class="rule-input rule-input--value"
          :placeholder="rule.op === 'between' ? 'min,max' : 'Amount'"
          @input="emitCriteria"
        />
        <input
          v-else-if="isDateField(rule.field)"
          v-model="rule.value"
          type="date"
          class="rule-input rule-input--value"
          @change="emitCriteria"
        />
        <input
          v-else
          v-model="rule.value"
          class="rule-input rule-input--value"
          placeholder="Value"
          @input="emitCriteria"
        />
      </template>

      <span v-if="validationErrors[idx]" class="rule-error">{{ validationErrors[idx] }}</span>

      <button
        v-if="rules.length > 1"
        type="button"
        class="btn btn-ghost btn-xs remove-btn"
        title="Remove rule"
        @click="removeRule(idx)"
      >✕</button>
    </div>

    <div class="rule-actions">
      <button type="button" class="btn btn-ghost btn-sm" @click="addRule">+ Add rule</button>
      <button type="button" class="btn btn-secondary btn-sm" @click="preview" :disabled="previewLoading">
        {{ previewLoading ? 'Checking…' : 'Preview' }}
      </button>
    </div>

    <p v-if="previewResult !== null" class="preview-result">
      <span class="preview-count">{{ previewResult.coinCount }}</span> matching coin{{ previewResult.coinCount !== 1 ? 's' : '' }},
      <span class="preview-value">${{ previewResult.totalValue.toFixed(2) }}</span> total value
    </p>
    <p v-if="previewError" class="preview-error">{{ previewError }}</p>

    <!-- Suggestions -->
    <div v-if="suggestions.length" class="section-block">
      <span class="section-label">Suggested starters</span>
      <div class="suggestion-chips">
        <button
          v-for="s in suggestions"
          :key="s.id"
          type="button"
          class="chip chip-sm"
          :title="s.description"
          @click="applySuggestion(s)"
        >{{ s.name }}</button>
      </div>
    </div>

    <!-- Saved templates -->
    <div v-if="savedTemplates.length || canSave" class="section-block">
      <span class="section-label">Saved templates</span>
      <div class="template-row">
        <select
          v-if="savedTemplates.length"
          class="rule-input template-select"
          @change="onTemplateSelect"
        >
          <option value="">Apply a saved template…</option>
          <option v-for="t in savedTemplates" :key="t.id" :value="t.id">{{ t.name }}</option>
        </select>
        <button
          v-if="canSave"
          type="button"
          class="btn btn-ghost btn-sm"
          @click="showSaveForm = !showSaveForm"
        >Save as template</button>
      </div>
      <div v-if="showSaveForm" class="save-form">
        <input
          v-model="saveTemplateName"
          class="rule-input rule-input--value"
          placeholder="Template name"
          maxlength="80"
        />
        <button type="button" class="btn btn-primary btn-sm" :disabled="!saveTemplateName.trim() || saving" @click="saveTemplate">
          {{ saving ? 'Saving…' : 'Save' }}
        </button>
        <button type="button" class="btn btn-ghost btn-sm" @click="showSaveForm = false">Cancel</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import {
  previewSmartSet,
  getSuggestedCriteria,
  listCriteriaTemplates,
  saveCriteriaTemplate,
} from '@/api/client'
import type {
  SmartCriteriaGroup,
  SmartCriteriaRule,
  SmartCriteriaRuleOp,
  SmartSetPreview,
  SmartCriteriaTemplate,
  SuggestedSmartCriteria,
} from '@/types'

const emit = defineEmits<{
  update: [criteria: SmartCriteriaGroup]
}>()

// ---- rule state ----
const operator = ref<'and' | 'or'>('and')

interface EditableRule {
  field: string
  op: SmartCriteriaRuleOp
  value: unknown
}

const rules = reactive<EditableRule[]>([{ field: 'material', op: 'eq', value: 'Silver' }])

// ---- preview state ----
const previewResult = ref<SmartSetPreview | null>(null)
const previewError = ref<string | null>(null)
const previewLoading = ref(false)

// ---- validation ----
const validationErrors = ref<string[]>([])

// ---- template / suggestion state ----
const suggestions = ref<SuggestedSmartCriteria[]>([])
const savedTemplates = ref<SmartCriteriaTemplate[]>([])
const showSaveForm = ref(false)
const saveTemplateName = ref('')
const saving = ref(false)

// ---- lifecycle ----
onMounted(async () => {
  try {
    const [suggestRes, tmplRes] = await Promise.all([getSuggestedCriteria(), listCriteriaTemplates()])
    suggestions.value = suggestRes.data.suggestions
    savedTemplates.value = tmplRes.data.templates
  } catch {
    // non-fatal
  }
  emitCriteria()
})

watch([operator, rules], emitCriteria, { deep: true })

// ---- field helpers ----
const BOOL_FIELDS = new Set(['isWishlist', 'isSold', 'isPrivate'])
const NUMERIC_FIELDS = new Set(['currentValue', 'purchasePrice'])
const DATE_FIELDS = new Set(['purchaseDate', 'createdAt'])

function isBoolField(field: string) { return BOOL_FIELDS.has(field) }
function isNumericField(field: string) { return NUMERIC_FIELDS.has(field) }
function isDateField(field: string) { return DATE_FIELDS.has(field) }

interface OpOption { value: SmartCriteriaRuleOp; label: string }

function opsForField(field: string): OpOption[] {
  if (BOOL_FIELDS.has(field)) {
    return [
      { value: 'eq', label: 'Is true' },
      { value: 'neq', label: 'Is false' },
    ]
  }
  if (NUMERIC_FIELDS.has(field)) {
    return [
      { value: 'eq', label: 'Equals' },
      { value: 'neq', label: 'Not equals' },
      { value: 'gte', label: 'At least' },
      { value: 'lte', label: 'At most' },
      { value: 'between', label: 'Between (min,max)' },
      { value: 'isNull', label: 'Has no value' },
      { value: 'isNotNull', label: 'Has a value' },
    ]
  }
  if (DATE_FIELDS.has(field)) {
    return [
      { value: 'eq', label: 'On' },
      { value: 'gte', label: 'On or after' },
      { value: 'lte', label: 'On or before' },
      { value: 'between', label: 'Between (start,end)' },
      { value: 'isNull', label: 'Not set' },
      { value: 'isNotNull', label: 'Is set' },
    ]
  }
  return [
    { value: 'eq', label: 'Equals' },
    { value: 'neq', label: 'Not equals' },
    { value: 'contains', label: 'Contains' },
    { value: 'startsWith', label: 'Starts with' },
    { value: 'in', label: 'One of (comma-separated)' },
    { value: 'isNull', label: 'Is empty' },
    { value: 'isNotNull', label: 'Is not empty' },
  ]
}

function onFieldChange(rule: EditableRule) {
  // Reset op to first valid one for the new field
  const ops = opsForField(rule.field)
  if (!ops.find(o => o.value === rule.op)) {
    rule.op = ops[0].value
  }
  // Reset value for bool fields
  if (BOOL_FIELDS.has(rule.field)) {
    rule.value = true
  } else {
    rule.value = ''
  }
  emitCriteria()
}

function onOpChange(rule: EditableRule) {
  if (rule.op === 'isNull' || rule.op === 'isNotNull') {
    rule.value = undefined
  }
  emitCriteria()
}

// ---- rule management ----
function addRule() {
  rules.push({ field: 'material', op: 'eq', value: 'Silver' })
}

function removeRule(idx: number) {
  rules.splice(idx, 1)
}

// ---- criteria building ----
function buildCriteria(): SmartCriteriaGroup {
  return {
    operator: operator.value,
    rules: rules.map(r => {
      const ruleObj: SmartCriteriaRule = { field: r.field, op: r.op }
      if (r.op !== 'isNull' && r.op !== 'isNotNull') {
        ruleObj.value = BOOL_FIELDS.has(r.field) ? (r.op === 'neq' ? false : true) : r.value
      }
      return ruleObj
    }),
  }
}

function validateRules(): boolean {
  const errors: string[] = rules.map((r) => {
    if (r.op === 'isNull' || r.op === 'isNotNull') return ''
    if (BOOL_FIELDS.has(r.field)) return ''
    const val = r.value
    if (val === '' || val === null || val === undefined) return 'Value is required'
    return ''
  })
  validationErrors.value = errors
  return errors.every(e => e === '')
}

function emitCriteria() {
  validateRules()
  emit('update', buildCriteria())
}

// ---- preview ----
async function preview() {
  if (!validateRules()) return
  previewLoading.value = true
  previewError.value = null
  try {
    const res = await previewSmartSet(buildCriteria())
    previewResult.value = res.data
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    previewError.value = err?.response?.data?.error ?? 'Preview failed'
    previewResult.value = null
  } finally {
    previewLoading.value = false
  }
}

// ---- suggestions ----
function applySuggestion(s: SuggestedSmartCriteria) {
  applyCriteriaGroup(s.criteria)
}

function applyCriteriaGroup(criteria: SmartCriteriaGroup) {
  operator.value = criteria.operator
  rules.splice(0, rules.length)
  for (const r of criteria.rules) {
    if ('field' in r) {
      rules.push({ field: r.field, op: r.op, value: r.value ?? '' })
    }
  }
  previewResult.value = null
}

// ---- templates ----
function onTemplateSelect(e: Event) {
  const id = Number((e.target as HTMLSelectElement).value)
  if (!id) return
  const tmpl = savedTemplates.value.find(t => t.id === id)
  if (tmpl) applyCriteriaGroup(tmpl.criteria)
  ;(e.target as HTMLSelectElement).value = ''
}

const canSave = ref(true)

async function saveTemplate() {
  if (!saveTemplateName.value.trim()) return
  saving.value = true
  try {
    const tmpl = await saveCriteriaTemplate({
      name: saveTemplateName.value.trim(),
      criteria: buildCriteria(),
    })
    savedTemplates.value.push(tmpl.data)
    showSaveForm.value = false
    saveTemplateName.value = ''
  } catch {
    // ignore — user can retry
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.smart-builder {
  margin-top: 1rem;
}

.builder-header {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.operator-toggle {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  margin-left: auto;
}

.operator-label {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.rule-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.4rem;
  margin-bottom: 0.4rem;
}

.rule-input {
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-input);
  color: var(--text-primary);
  padding: 0.4rem 0.55rem;
  font-size: 0.85rem;
  min-width: 0;
}

.rule-input--value {
  flex: 1 1 120px;
}

.rule-error {
  color: #e74c3c;
  font-size: 0.75rem;
  white-space: nowrap;
}

.remove-btn {
  opacity: 0.6;
  padding: 0.2rem 0.45rem;
}
.remove-btn:hover {
  opacity: 1;
}

.rule-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.25rem;
}

.preview-result {
  color: var(--text-secondary);
  font-size: 0.85rem;
  margin-top: 0.5rem;
}

.preview-count {
  font-weight: 600;
  color: var(--accent-gold);
}

.preview-value {
  font-weight: 600;
  color: var(--accent-gold);
}

.preview-error {
  color: #e74c3c;
  font-size: 0.8rem;
  margin-top: 0.4rem;
}

.section-block {
  margin-top: 1rem;
  border-top: 1px solid var(--border-subtle);
  padding-top: 0.75rem;
}

.suggestion-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  margin-top: 0.4rem;
}

.template-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 0.4rem;
  align-items: center;
}

.template-select {
  flex: 1 1 180px;
}

.save-form {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.5rem;
  flex-wrap: wrap;
  align-items: center;
}
</style>

