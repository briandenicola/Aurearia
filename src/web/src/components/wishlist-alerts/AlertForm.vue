<template>
  <form class="alert-form" @submit.prevent="submit">
    <p class="helper">Search Alerts discover acquisition ideas. They do not check saved wishlist item availability. Cadence is metadata only in v1; use Run Now for manual, in-app review.</p>
    <label>Name <input v-model.trim="draft.name" required maxlength="200" /></label>
    <div class="grid">
      <label>Ruler or issuer <input v-model.trim="draft.criteria.rulerOrIssuer" maxlength="200" /></label>
      <label>Coin type <input v-model.trim="draft.criteria.coinType" maxlength="200" /></label>
      <label>Mint <input v-model.trim="draft.criteria.mint" maxlength="200" /></label>
      <label>Material <input v-model.trim="draft.criteria.material" maxlength="100" /></label>
      <label>Grade or condition <input v-model.trim="draft.criteria.gradeOrCondition" maxlength="200" /></label>
      <label>Keywords <input v-model.trim="draft.criteria.keywords" maxlength="500" /></label>
      <label>Date from <input v-model.number="draft.criteria.dateFrom" type="number" /></label>
      <label>Date to <input v-model.number="draft.criteria.dateTo" type="number" /></label>
      <label>Price min <input v-model.number="draft.criteria.priceMin" type="number" min="0" step="0.01" /></label>
      <label>Price max <input v-model.number="draft.criteria.priceMax" type="number" min="0" step="0.01" /></label>
      <label>Currency <input v-model.trim="draft.criteria.currency" maxlength="3" /></label>
      <label>Cadence
        <select v-model="draft.cadence">
          <option value="manual">Manual</option>
          <option value="daily">Daily metadata only</option>
          <option value="weekly">Weekly metadata only</option>
          <option value="monthly">Monthly metadata only</option>
        </select>
      </label>
    </div>
    <p class="helper">Daily, weekly, and monthly values are saved for future scheduling; this screen does not enable push, email, or digest delivery.</p>
    <label>Source domains <input v-model.trim="sourceFiltersText" placeholder="vcoins.com, ma-shops.com" /></label>
    <label>Dealer preference <input v-model.trim="draft.criteria.dealerPreference" maxlength="500" /></label>
    <label>Notes <textarea v-model.trim="draft.criteria.notes" maxlength="5000" /></label>
    <label class="checkbox"><input v-model="draft.isActive" type="checkbox" /> Active</label>
    <p v-if="error" class="form-error">{{ error }}</p>
    <div class="actions">
      <button class="btn btn-primary" type="submit" :disabled="!!error || saving">{{ saving ? 'Saving...' : 'Save Search Alert' }}</button>
      <button class="btn btn-secondary" type="button" @click="$emit('cancel')">Cancel</button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import type { WishlistSearchAlert, WishlistSearchAlertInput } from '@/types'

const props = defineProps<{ alert?: WishlistSearchAlert | null; saving?: boolean }>()
const emit = defineEmits<{ save: [value: WishlistSearchAlertInput]; cancel: [] }>()

const blank = (): WishlistSearchAlertInput => ({
  name: '',
  criteria: { rulerOrIssuer: '', coinType: '', dateFrom: null, dateTo: null, mint: '', material: '', gradeOrCondition: '', priceMin: null, priceMax: null, currency: 'USD', dealerPreference: '', sourceFilters: [], keywords: '', notes: '' },
  cadence: 'manual',
  isActive: true,
})
const draft = reactive<WishlistSearchAlertInput>(blank())
const sourceFiltersText = ref('')

watch(() => props.alert, (alert) => {
  Object.assign(draft, blank())
  if (!alert) { sourceFiltersText.value = ''; return }
  draft.name = alert.name
  draft.criteria = { rulerOrIssuer: alert.rulerOrIssuer, coinType: alert.coinType, dateFrom: alert.dateFrom, dateTo: alert.dateTo, mint: alert.mint, material: alert.material, gradeOrCondition: alert.gradeOrCondition, priceMin: alert.priceMin, priceMax: alert.priceMax, currency: alert.currency || 'USD', dealerPreference: alert.dealerPreference, sourceFilters: [...alert.sourceFilters], keywords: alert.keywords, notes: alert.notes }
  draft.cadence = alert.cadence
  draft.isActive = alert.isActive
  sourceFiltersText.value = alert.sourceFilters.join(', ')
}, { immediate: true })

const error = computed(() => {
  const c = draft.criteria
  const hasCriteria = [c.rulerOrIssuer, c.coinType, c.mint, c.material, c.gradeOrCondition, c.dealerPreference, c.keywords, sourceFiltersText.value].some(Boolean) || c.dateFrom != null || c.dateTo != null || c.priceMin != null || c.priceMax != null
  if (!hasCriteria) return 'Add at least one search criterion.'
  if (c.priceMin != null && c.priceMax != null && c.priceMin > c.priceMax) return 'Price minimum must be less than or equal to maximum.'
  if (c.dateFrom != null && c.dateTo != null && c.dateFrom > c.dateTo) return 'Date from must be less than or equal to date to.'
  if (!/^[A-Za-z]{3}$/.test(c.currency)) return 'Currency must be a three-letter code.'
  return ''
})

function submit() {
  draft.criteria.sourceFilters = sourceFiltersText.value.split(',').map((s) => s.trim()).filter(Boolean)
  draft.criteria.currency = draft.criteria.currency.toUpperCase()
  emit('save', JSON.parse(JSON.stringify(draft)))
}
</script>

<style scoped>
.alert-form { display: grid; gap: .8rem; }
.helper { color: var(--text-muted); margin: 0; }
.grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: .75rem; }
label { display: grid; gap: .25rem; color: var(--text-secondary); font-size: 0.9rem; }
input, select, textarea { width: 100%; border: 1px solid var(--border-subtle); border-radius: var(--radius-sm); padding: .5rem; background: var(--bg-input); color: var(--text-primary); }
textarea { min-height: 80px; }
.checkbox { display: flex; align-items: center; gap: .5rem; }
.checkbox input { width: auto; }
.form-error { color: var(--danger); margin: 0; }
.actions { display: flex; gap: .5rem; }
</style>
