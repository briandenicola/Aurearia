<template>
  <section class="card p-6">
    <div class="mb-3 flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
      <div>
        <p class="section-label">Collection metadata</p>
        <h2 class="m-0 text-xl font-medium text-heading">Coin Properties</h2>
      </div>
      <button
        type="submit"
        form="coin-properties-form"
        class="btn btn-primary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
        :disabled="saving"
      >
        {{ saving ? 'Saving...' : 'Save Properties' }}
      </button>
    </div>

    <p class="mb-6 text-base text-text-secondary">
      Configure the category and era choices shown on Add Coin and Edit Coin. Enter one value per line.
    </p>

    <form id="coin-properties-form" class="grid gap-4" @submit.prevent="$emit('save')">
      <div class="rounded-md border border-border-subtle bg-card-hover p-4">
        <div class="mb-3 flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
          <div>
            <label class="form-label text-heading" for="category-options">Category Options</label>
            <p class="mt-1 text-body text-text-secondary">One category per line. Empty lines are ignored.</p>
          </div>
          <span class="chip-sm">{{ categoryPreview.length }} options</span>
        </div>
        <textarea
          id="category-options"
          v-model="localCategoryOptions"
          class="form-textarea min-h-44 resize-y bg-input text-body leading-6"
          rows="7"
          placeholder="Roman&#10;Greek&#10;Byzantine&#10;Modern&#10;Other"
        />
        <div class="mt-3 flex flex-wrap gap-[0.35rem]" aria-label="Category option preview">
          <span v-for="option in categoryPreview" :key="option" class="chip-sm">{{ option }}</span>
        </div>
      </div>

      <div class="rounded-md border border-border-subtle bg-card-hover p-4">
        <div class="mb-3 flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
          <div>
            <label class="form-label text-heading" for="era-options">Era Options</label>
            <p class="mt-1 text-body text-text-secondary">One era per line. The coin form adds Unspecified automatically.</p>
          </div>
          <span class="chip-sm">{{ eraPreview.length }} options</span>
        </div>
        <textarea
          id="era-options"
          v-model="localEraOptions"
          class="form-textarea min-h-44 resize-y bg-input text-body leading-6"
          rows="7"
          placeholder="ancient&#10;medieval&#10;modern"
        />
        <div class="mt-3 flex flex-wrap gap-[0.35rem]" aria-label="Era option preview">
          <span class="chip-sm">Unspecified</span>
          <span v-for="option in eraPreview" :key="option" class="chip-sm">{{ option }}</span>
        </div>
      </div>

      <p v-if="msg" class="mt-2 text-body text-gold" :class="{ 'text-[var(--cat-byzantine)]': error }">{{ msg }}</p>
    </form>

    <section class="mt-4 rounded-md border border-border-subtle bg-card-hover p-4" aria-labelledby="custom-locations-heading">
      <div class="mb-3 flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h3 id="custom-locations-heading" class="m-0 text-lg font-medium text-heading">Custom Locations</h3>
          <p class="mt-1 text-body text-text-secondary">Global mint coordinates used by the collection map. Aliases can be comma or line separated.</p>
        </div>
        <span class="chip-sm">{{ mintLocations.length }} locations</span>
      </div>

      <p v-if="mintLocationError" class="mb-2 text-body text-[var(--cat-byzantine)]">{{ mintLocationError }}</p>
      <p v-if="mintLocationsLoading" class="text-body text-text-secondary">Loading mint locations...</p>

      <div v-else class="grid gap-4 md:grid-cols-[minmax(0,1.1fr)_minmax(260px,0.9fr)] md:items-start">
        <div class="flex max-h-[28rem] min-w-0 flex-col gap-2 overflow-y-auto pr-1 [scrollbar-gutter:stable] md:max-h-[min(34rem,70vh)]" aria-label="Custom mint locations">
          <div v-if="!mintLocations.length" class="rounded-sm border border-border-subtle bg-input p-3 text-left text-body text-text-secondary">
            No mint locations configured yet.
          </div>
          <div
            v-for="location in sortedMintLocations"
            :key="location.id"
            class="rounded-sm border border-border-subtle bg-input p-3 text-left"
          >
            <div class="flex min-w-0 flex-1 flex-col items-stretch gap-1 text-left">
              <div class="grid w-full grid-cols-[minmax(0,1fr)_auto] items-start gap-3 text-left">
                <strong class="min-w-0 text-text-primary [overflow-wrap:anywhere]">{{ location.displayName }}</strong>
                <div class="flex shrink-0 flex-wrap justify-end gap-[0.35rem]">
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
                    :disabled="mintLocationSaving"
                    @click="startEditMintLocation(location)"
                  >
                    Edit
                  </button>
                  <button
                    type="button"
                    class="btn btn-danger btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
                    :disabled="deletingMintLocationId === location.id"
                    @click="deleteMintLocation(location)"
                  >
                    {{ deletingMintLocationId === location.id ? 'Deleting...' : 'Delete' }}
                  </button>
                </div>
              </div>
              <span class="block text-body text-text-secondary [overflow-wrap:anywhere]">{{ location.region || 'No region' }} · {{ location.lat }}, {{ location.lng }}</span>
              <span v-if="location.aliases.length" class="block text-body text-text-secondary [overflow-wrap:anywhere]">{{ location.aliases.join(', ') }}</span>
            </div>
          </div>
        </div>

        <form class="flex min-w-0 flex-col gap-3 self-start rounded-sm border border-border-subtle bg-input p-3" @submit.prevent="saveMintLocation">
          <h4 class="m-0 text-lg font-medium text-heading">{{ editingMintLocation ? 'Edit Location' : 'Add Location' }}</h4>
          <div class="grid gap-3 md:grid-cols-2">
            <label class="flex flex-col gap-[0.35rem]">
              <span class="form-label">Display Name</span>
              <input v-model="mintLocationForm.displayName" class="form-input" type="text" maxlength="120" required />
            </label>
            <label class="flex flex-col gap-[0.35rem]">
              <span class="form-label">Region</span>
              <input v-model="mintLocationForm.region" class="form-input" type="text" maxlength="120" />
            </label>
            <label class="flex flex-col gap-[0.35rem]">
              <span class="form-label">Latitude</span>
              <input v-model="mintLocationForm.lat" class="form-input" type="number" min="-90" max="90" step="0.000001" required />
            </label>
            <label class="flex flex-col gap-[0.35rem]">
              <span class="form-label">Longitude</span>
              <input v-model="mintLocationForm.lng" class="form-input" type="number" min="-180" max="180" step="0.000001" required />
            </label>
          </div>
          <label class="flex flex-col gap-[0.35rem]">
            <span class="form-label">Aliases</span>
            <textarea
              v-model="mintLocationForm.aliases"
              class="form-textarea min-h-24 resize-y bg-card text-body"
              rows="4"
              placeholder="Roma, Rome mint"
            />
          </label>
          <div class="flex flex-wrap justify-start gap-[0.35rem] md:justify-end">
            <button type="submit" class="btn btn-primary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" :disabled="mintLocationSaving">
              {{ mintLocationSaving ? 'Saving...' : editingMintLocation ? 'Save Location' : 'Add Location' }}
            </button>
            <button
              v-if="editingMintLocation"
              type="button"
              class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
              :disabled="mintLocationSaving"
              @click="resetMintLocationForm"
            >
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

