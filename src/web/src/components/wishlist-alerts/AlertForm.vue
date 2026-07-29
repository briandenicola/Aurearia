<template>
  <form class="grid gap-3" @submit.prevent="submit">
    <p class="m-0 text-body text-text-muted">Search Alerts discover acquisition ideas. They do not check saved wishlist item availability. Use Run Now for an immediate, in-app review, or set a cadence below for automatic runs (if enabled by your admin).</p>
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
        <select v-model="mintLocationIdModel" class="form-select focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent-gold)]" :disabled="mintLocationsLoading">
          <option value="">Unknown</option>
          <optgroup v-if="myMintLocations.length" label="My Mints">
            <option v-for="location in myMintLocations" :key="location.id" :value="String(location.id)">
              {{ location.displayName }}
            </option>
          </optgroup>
          <optgroup v-if="globalMintLocations.length" label="Mints">
            <option v-for="location in globalMintLocations" :key="location.id" :value="String(location.id)">
              {{ location.displayName }}
            </option>
          </optgroup>
          <option value="__create__">+ Create new mint…</option>
        </select>
        <p v-if="mintLocationError" class="mt-1 text-body text-text-secondary">{{ mintLocationError }}</p>
        <p v-else-if="draft.criteria.mint && draft.criteria.mintLocationId == null" class="mt-1 text-body text-text-secondary">
          Legacy mint text "{{ draft.criteria.mint }}" will be preserved as a free-text filter.
        </p>
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
          <option value="daily">Daily</option>
          <option value="weekly">Weekly</option>
          <option value="monthly">Monthly</option>
        </select>
      </label>
    </div>
    <p class="m-0 text-body text-text-muted">Daily, weekly, and monthly alerts run automatically once due, if your admin has enabled scheduled search alert checks. This does not enable push, email, or digest delivery — new candidates only appear in this in-app review queue.</p>
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
    <CreateMintModal
      :open="showCreateMintModal"
      :initial-name="pendingMintName"
      @close="onCreateMintClosed"
      @created="onMintCreated"
    />
  </form>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { getMintLocations, type MintLocationsResponse } from '@/api/client'
import type { MintLocation, WishlistSearchAlert, WishlistSearchAlertInput } from '@/types'
import CreateMintModal from '@/components/CreateMintModal.vue'

const props = defineProps<{ alert?: WishlistSearchAlert | null; saving?: boolean }>()
const emit = defineEmits<{ save: [value: WishlistSearchAlertInput]; cancel: [] }>()

function unwrapMintLocations(data: MintLocationsResponse): MintLocation[] {
  return Array.isArray(data) ? data : data.mintLocations ?? []
}

const blank = (): WishlistSearchAlertInput => ({
  name: '',
  criteria: {
    rulerOrIssuer: '',
    coinType: '',
    dateFrom: null,
    dateTo: null,
    mint: '',
    mintLocationId: null,
    material: '',
    gradeOrCondition: '',
    priceMin: null,
    priceMax: null,
    currency: 'USD',
    dealerPreference: '',
    sourceFilters: [],
    keywords: '',
    notes: '',
  },
  cadence: 'manual',
  isActive: true,
})

const draft = reactive<WishlistSearchAlertInput>(blank())
const sourceFiltersText = ref('')
const mintLocations = ref<MintLocation[]>([])
const mintLocationsLoading = ref(false)
const mintLocationError = ref('')
const showCreateMintModal = ref(false)
const pendingMintName = ref('')

const myMintLocations = computed(() => mintLocations.value.filter((m) => m.userId != null))
const globalMintLocations = computed(() => mintLocations.value.filter((m) => m.userId == null))

const mintLocationIdModel = computed({
  get: () => draft.criteria.mintLocationId == null ? '' : String(draft.criteria.mintLocationId),
  set: (value: string) => {
    if (value === '__create__') {
      pendingMintName.value = draft.criteria.mint.trim()
      showCreateMintModal.value = true
      return
    }
    draft.criteria.mintLocationId = value === '' ? null : Number(value)
    const selected = mintLocations.value.find((m) => String(m.id) === value)
    draft.criteria.mint = selected ? selected.displayName : draft.criteria.mint
  },
})

async function loadMintLocations() {
  mintLocationsLoading.value = true
  try {
    const res = await getMintLocations()
    mintLocations.value = unwrapMintLocations(res.data)
    mintLocationError.value = ''
  } catch {
    mintLocations.value = []
    mintLocationError.value = 'Mint locations are unavailable'
  } finally {
    mintLocationsLoading.value = false
  }
}

function onCreateMintClosed() {
  showCreateMintModal.value = false
}

function onMintCreated(mintLocation: MintLocation) {
  showCreateMintModal.value = false
  mintLocations.value.push(mintLocation)
  draft.criteria.mintLocationId = mintLocation.id
  draft.criteria.mint = mintLocation.displayName
}

watch(() => props.alert, (alert) => {
  Object.assign(draft, blank())
  if (!alert) {
    sourceFiltersText.value = ''
    return
  }
  draft.name = alert.name
  draft.criteria = {
    rulerOrIssuer: alert.rulerOrIssuer,
    coinType: alert.coinType,
    dateFrom: alert.dateFrom,
    dateTo: alert.dateTo,
    mint: alert.mint,
    mintLocationId: alert.mintLocationId,
    material: alert.material,
    gradeOrCondition: alert.gradeOrCondition,
    priceMin: alert.priceMin,
    priceMax: alert.priceMax,
    currency: alert.currency || 'USD',
    dealerPreference: alert.dealerPreference,
    sourceFilters: [...alert.sourceFilters],
    keywords: alert.keywords,
    notes: alert.notes,
  }
  draft.cadence = alert.cadence
  draft.isActive = alert.isActive
  sourceFiltersText.value = alert.sourceFilters.join(', ')
}, { immediate: true })

onMounted(() => {
  void loadMintLocations()
})

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
