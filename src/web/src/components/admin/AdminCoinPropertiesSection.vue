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

    <section class="property-card custom-locations-card" aria-labelledby="custom-locations-heading">
      <div class="property-card-header">
        <div>
          <h3 id="custom-locations-heading">Custom Locations</h3>
          <p class="form-hint">Global mint coordinates used by the collection map. Aliases can be comma or line separated.</p>
        </div>
        <span class="chip-sm">{{ mintLocations.length }} locations</span>
      </div>

      <p v-if="mintLocationError" class="msg error">{{ mintLocationError }}</p>
      <p v-if="mintLocationsLoading" class="empty-state-text">Loading mint locations...</p>

      <div v-else class="mint-location-layout">
        <div class="mint-location-list" aria-label="Custom mint locations">
          <div v-if="!mintLocations.length" class="empty-location">
            No mint locations configured yet.
          </div>
          <div
            v-for="location in sortedMintLocations"
            :key="location.id"
            class="mint-location-row"
          >
            <div class="mint-location-main">
              <div class="mint-location-heading">
                <strong class="mint-location-name">{{ location.displayName }}</strong>
                <div class="location-actions">
                  <button type="button" class="btn btn-secondary btn-sm" :disabled="mintLocationSaving" @click="startEditMintLocation(location)">
                    Edit
                  </button>
                  <button
                    type="button"
                    class="btn btn-danger btn-sm"
                    :disabled="deletingMintLocationId === location.id"
                    @click="deleteMintLocation(location)"
                  >
                    {{ deletingMintLocationId === location.id ? 'Deleting...' : 'Delete' }}
                  </button>
                </div>
              </div>
              <span class="location-details">{{ location.region || 'No region' }} · {{ location.lat }}, {{ location.lng }}</span>
              <span v-if="location.aliases.length" class="location-aliases">{{ location.aliases.join(', ') }}</span>
            </div>
          </div>
        </div>

        <form class="mint-location-form" @submit.prevent="saveMintLocation">
          <h4>{{ editingMintLocation ? 'Edit Location' : 'Add Location' }}</h4>
          <div class="form-grid">
            <label>
              <span class="form-label">Display Name</span>
              <input v-model="mintLocationForm.displayName" class="form-input" type="text" maxlength="120" required />
            </label>
            <label>
              <span class="form-label">Region</span>
              <input v-model="mintLocationForm.region" class="form-input" type="text" maxlength="120" />
            </label>
            <label>
              <span class="form-label">Latitude</span>
              <input v-model="mintLocationForm.lat" class="form-input" type="number" min="-90" max="90" step="0.000001" required />
            </label>
            <label>
              <span class="form-label">Longitude</span>
              <input v-model="mintLocationForm.lng" class="form-input" type="number" min="-180" max="180" step="0.000001" required />
            </label>
          </div>
          <label>
            <span class="form-label">Aliases</span>
            <textarea
              v-model="mintLocationForm.aliases"
              class="form-textarea aliases-textarea"
              rows="4"
              placeholder="Roma, Rome mint"
            />
          </label>
          <div class="form-actions">
            <button type="submit" class="btn btn-primary btn-sm" :disabled="mintLocationSaving">
              {{ mintLocationSaving ? 'Saving...' : editingMintLocation ? 'Save Location' : 'Add Location' }}
            </button>
            <button v-if="editingMintLocation" type="button" class="btn btn-secondary btn-sm" :disabled="mintLocationSaving" @click="resetMintLocationForm">
              Cancel
            </button>
          </div>
        </form>
      </div>
    </section>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import {
  adminCreateMintLocation,
  adminDeleteMintLocation,
  adminUpdateMintLocation,
  getMintLocations,
  type MintLocationInput,
  type MintLocationsResponse,
} from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import { parseOptionList } from '@/utils/options'
import { CATEGORIES, COIN_ERAS } from '@/types'
import type { MintLocation } from '@/types'

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
const mintLocations = ref<MintLocation[]>([])
const mintLocationsLoading = ref(false)
const mintLocationSaving = ref(false)
const deletingMintLocationId = ref<number | null>(null)
const mintLocationError = ref('')
const editingMintLocation = ref<MintLocation | null>(null)
const mintLocationForm = reactive({
  displayName: '',
  region: '',
  lat: '',
  lng: '',
  aliases: '',
})
const { showConfirm } = useDialog()

watch(() => props.categoryOptions, (v) => { localCategoryOptions.value = v })
watch(() => props.eraOptions, (v) => { localEraOptions.value = v })

watch(localCategoryOptions, (v) => emit('update:categoryOptions', v))
watch(localEraOptions, (v) => emit('update:eraOptions', v))

const categoryPreview = computed(() => parseOptionList(localCategoryOptions.value, CATEGORIES))
const eraPreview = computed(() => parseOptionList(localEraOptions.value, COIN_ERAS))
const sortedMintLocations = computed(() =>
  [...mintLocations.value].sort((a, b) => a.displayName.localeCompare(b.displayName)),
)

function unwrapMintLocations(data: MintLocationsResponse): MintLocation[] {
  return Array.isArray(data) ? data : data.mintLocations ?? []
}

function apiErrorText(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error !== null && 'response' in error) {
    const axiosErr = error as { response?: { data?: { error?: string; message?: string } } }
    return axiosErr.response?.data?.message ?? axiosErr.response?.data?.error ?? fallback
  }
  return fallback
}

function parseAliases(value: string): string[] {
  return value
    .split(/[\n,]+/)
    .map((item) => item.trim())
    .filter((item, index, items) => item.length > 0 && items.indexOf(item) === index)
}

function buildMintLocationPayload(): MintLocationInput | null {
  const displayName = mintLocationForm.displayName.trim()
  const lat = Number(mintLocationForm.lat)
  const lng = Number(mintLocationForm.lng)
  if (!displayName) {
    mintLocationError.value = 'Display name is required.'
    return null
  }
  if (!Number.isFinite(lat) || lat < -90 || lat > 90) {
    mintLocationError.value = 'Latitude must be between -90 and 90.'
    return null
  }
  if (!Number.isFinite(lng) || lng < -180 || lng > 180) {
    mintLocationError.value = 'Longitude must be between -180 and 180.'
    return null
  }
  return {
    displayName,
    lat,
    lng,
    region: mintLocationForm.region.trim(),
    aliases: parseAliases(mintLocationForm.aliases),
  }
}

function resetMintLocationForm() {
  editingMintLocation.value = null
  mintLocationForm.displayName = ''
  mintLocationForm.region = ''
  mintLocationForm.lat = ''
  mintLocationForm.lng = ''
  mintLocationForm.aliases = ''
  mintLocationError.value = ''
}

async function loadMintLocations() {
  mintLocationsLoading.value = true
  mintLocationError.value = ''
  try {
    const res = await getMintLocations()
    mintLocations.value = unwrapMintLocations(res.data)
  } catch (error: unknown) {
    mintLocations.value = []
    mintLocationError.value = apiErrorText(error, 'Failed to load mint locations.')
  } finally {
    mintLocationsLoading.value = false
  }
}

function startEditMintLocation(location: MintLocation) {
  editingMintLocation.value = location
  mintLocationForm.displayName = location.displayName
  mintLocationForm.region = location.region ?? ''
  mintLocationForm.lat = String(location.lat)
  mintLocationForm.lng = String(location.lng)
  mintLocationForm.aliases = location.aliases.join('\n')
  mintLocationError.value = ''
}

async function saveMintLocation() {
  mintLocationError.value = ''
  const payload = buildMintLocationPayload()
  if (!payload) return
  mintLocationSaving.value = true
  try {
    if (editingMintLocation.value) {
      await adminUpdateMintLocation(editingMintLocation.value.id, payload)
    } else {
      await adminCreateMintLocation(payload)
    }
    resetMintLocationForm()
    await loadMintLocations()
  } catch (error: unknown) {
    mintLocationError.value = apiErrorText(error, 'Failed to save mint location.')
  } finally {
    mintLocationSaving.value = false
  }
}

async function deleteMintLocation(location: MintLocation) {
  mintLocationError.value = ''
  const confirmed = await showConfirm(`Delete mint location "${location.displayName}"? Coins with this mint text will become unmatched on the map until another location or alias matches them.`, {
    title: 'Delete Mint Location',
    variant: 'danger',
  })
  if (!confirmed) return
  deletingMintLocationId.value = location.id
  try {
    await adminDeleteMintLocation(location.id)
    if (editingMintLocation.value?.id === location.id) {
      resetMintLocationForm()
    }
    await loadMintLocations()
  } catch (error: unknown) {
    mintLocationError.value = apiErrorText(error, 'Failed to delete mint location.')
  } finally {
    deletingMintLocationId.value = null
  }
}

onMounted(() => {
  loadMintLocations()
})
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

.custom-locations-card {
  margin-top: 1rem;
}

.custom-locations-card h3,
.mint-location-form h4 {
  margin: 0;
}

.mint-location-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(260px, 0.9fr);
  gap: 1rem;
  align-items: start;
}

.mint-location-list,
.mint-location-form {
  min-width: 0;
}

.mint-location-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  max-height: min(34rem, 70vh);
  overflow-y: auto;
  padding-right: 0.25rem;
  scrollbar-gutter: stable;
}

.mint-location-row,
.empty-location {
  padding: 0.75rem;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  text-align: left;
}

.empty-location {
  display: flex;
  align-items: center;
}

.empty-location,
.empty-state-text {
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.mint-location-main {
  width: 100%;
  min-width: 0;
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 0.25rem;
  align-items: stretch;
  text-align: left;
}

.mint-location-heading {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: start;
  gap: 0.75rem;
  width: 100%;
  text-align: left;
}

.mint-location-name {
  min-width: 0;
  color: var(--text-primary);
  overflow-wrap: anywhere;
  text-align: left;
}

.mint-location-main span {
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.location-details,
.location-aliases {
  display: block;
  overflow-wrap: anywhere;
  text-align: left;
}

.location-actions,
.form-actions {
  display: flex;
  gap: 0.35rem;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.location-actions {
  flex-shrink: 0;
  justify-self: end;
}

.mint-location-form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  align-self: start;
  padding: 0.75rem;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.form-grid label,
.mint-location-form label {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.aliases-textarea {
  min-height: 6rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-family: inherit;
  font-size: 0.85rem;
  resize: vertical;
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

  .mint-location-layout,
  .form-grid {
    grid-template-columns: 1fr;
  }

  .form-actions {
    justify-content: flex-start;
  }

  .location-actions {
    justify-content: flex-end;
  }

  .mint-location-list {
    max-height: 28rem;
  }
}
</style>
