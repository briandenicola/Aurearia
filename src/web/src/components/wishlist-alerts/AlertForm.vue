<template>
  <form class="grid gap-3" @submit.prevent="submit">
    <p class="m-0 text-body text-text-muted">Search Alerts discover acquisition ideas. They do not check saved wishlist item availability. Cadence is metadata only in v1; use Run Now for manual, in-app review.</p>
    <label class="grid gap-1 text-base text-text-secondary">
      Name
      <input
        v-model.trim="draft.name"
        class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
        required
        maxlength="200"
      />
    </label>
    <div class="grid gap-3 md:grid-cols-2">
      <label class="grid gap-1 text-base text-text-secondary">
        Ruler or issuer
        <input v-model.trim="draft.criteria.rulerOrIssuer" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" maxlength="200" />
      </label>
      <label class="grid gap-1 text-base text-text-secondary">
        Coin type
        <input v-model.trim="draft.criteria.coinType" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" maxlength="200" />
      </label>
      <label class="grid gap-1 text-base text-text-secondary">
        Mint
        <input v-model.trim="draft.criteria.mint" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" maxlength="200" />
      </label>
      <label class="grid gap-1 text-base text-text-secondary">
        Material
        <input v-model.trim="draft.criteria.material" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" maxlength="100" />
      </label>
      <label class="grid gap-1 text-base text-text-secondary">
        Grade or condition
        <input v-model.trim="draft.criteria.gradeOrCondition" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" maxlength="200" />
      </label>
      <label class="grid gap-1 text-base text-text-secondary">
        Keywords
        <input v-model.trim="draft.criteria.keywords" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" maxlength="500" />
      </label>
      <label class="grid gap-1 text-base text-text-secondary">
        Date from
        <input v-model.number="draft.criteria.dateFrom" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" type="number" />
      </label>
      <label class="grid gap-1 text-base text-text-secondary">
        Date to
        <input v-model.number="draft.criteria.dateTo" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" type="number" />
      </label>
      <label class="grid gap-1 text-base text-text-secondary">
        Price min
        <input v-model.number="draft.criteria.priceMin" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" type="number" min="0" step="0.01" />
      </label>
      <label class="grid gap-1 text-base text-text-secondary">
        Price max
        <input v-model.number="draft.criteria.priceMax" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" type="number" min="0" step="0.01" />
      </label>
      <label class="grid gap-1 text-base text-text-secondary">
        Currency
        <input v-model.trim="draft.criteria.currency" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" maxlength="3" />
      </label>
      <label class="grid gap-1 text-base text-text-secondary">
        Cadence
        <select v-model="draft.cadence" class="form-select focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]">
          <option value="manual">Manual</option>
          <option value="daily">Daily metadata only</option>
          <option value="weekly">Weekly metadata only</option>
          <option value="monthly">Monthly metadata only</option>
        </select>
      </label>
    </div>
    <p class="m-0 text-body text-text-muted">Daily, weekly, and monthly values are saved for future scheduling; this screen does not enable push, email, or digest delivery.</p>
    <label class="grid gap-1 text-base text-text-secondary">
      Source domains
      <input
        v-model.trim="sourceFiltersText"
        class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
        placeholder="vcoins.com, ma-shops.com"
      />
    </label>
    <label class="grid gap-1 text-base text-text-secondary">
      Dealer preference
      <input v-model.trim="draft.criteria.dealerPreference" class="form-input focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" maxlength="500" />
    </label>
    <label class="grid gap-1 text-base text-text-secondary">
      Notes
      <textarea
        v-model.trim="draft.criteria.notes"
        class="form-input min-h-20 resize-y focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
        maxlength="5000"
      />
    </label>
    <label class="flex items-center gap-2 text-base text-text-secondary">
      <input
        v-model="draft.isActive"
        class="h-4 w-4 rounded-sm border border-border-subtle bg-input accent-[var(--accent-gold)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]"
        type="checkbox"
      />
      Active
    </label>
    <p v-if="error" class="m-0 text-body text-bronze">{{ error }}</p>
    <div class="flex flex-wrap gap-2">
      <button class="btn btn-primary focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" type="submit" :disabled="!!error || saving">{{ saving ? 'Saving...' : 'Save Search Alert' }}</button>
      <button class="btn btn-secondary focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" type="button" @click="$emit('cancel')">Cancel</button>
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
