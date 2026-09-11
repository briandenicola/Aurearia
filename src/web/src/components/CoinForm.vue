<template>
  <form class="mx-auto max-w-[900px]" @submit.prevent="handleSubmit">
    <div class="grid gap-6 md:grid-cols-2">
      <!-- Basic Info -->
      <fieldset class="m-0 rounded-md border border-border-subtle bg-card p-5">
        <h2 class="mb-4 font-display text-lg font-medium text-gold">Basic Information</h2>
        <div class="form-group min-w-0">
          <label class="form-label">Name *</label>
          <AutocompleteInput v-model="form.name!" field="name" required placeholder="e.g. Augustus Denarius" />
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <div class="form-group min-w-0">
            <label class="form-label">Category</label>
            <select v-model="form.category" class="form-select">
              <option v-for="c in categoryOptions" :key="c" :value="c">{{ c }}</option>
            </select>
          </div>
          <div class="form-group min-w-0">
            <label class="form-label">Material</label>
            <select v-model="form.material" class="form-select">
              <option v-for="m in materialOptions" :key="m" :value="m">{{ m }}</option>
            </select>
          </div>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <div class="form-group min-w-0">
            <label class="form-label">Denomination</label>
            <AutocompleteInput v-model="form.denomination!" field="denomination" placeholder="e.g. Denarius" />
          </div>
          <div class="form-group min-w-0">
            <label class="form-label">Mint</label>
            <select v-model="mintLocationIdModel" class="form-select" :disabled="mintLocationsLoading">
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
            <p v-if="mintLocationError" class="mt-2 text-body text-text-secondary">{{ mintLocationError }}</p>
            <p v-else-if="form.mint && !form.mintLocationId" class="mt-2 text-body text-text-secondary">
              Unlinked legacy mint: "{{ form.mint }}" — pick a location above or create one to link it.
            </p>
          </div>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <div class="form-group min-w-0">
            <label class="form-label">Ruler</label>
            <AutocompleteInput v-model="form.ruler!" field="ruler" placeholder="e.g. Augustus" />
          </div>
          <div class="form-group min-w-0">
            <label class="form-label">Era</label>
            <select v-model="form.era" class="form-select">
              <option value="">Unspecified</option>
              <option v-for="era in displayedEraOptions" :key="era" :value="era">{{ era }}</option>
            </select>
          </div>
        </div>
        <div class="form-group min-w-0">
          <label class="form-label">Date or Date Range</label>
          <input
            v-model="form.dateRange"
            class="form-input"
            type="text"
            maxlength="200"
            placeholder="e.g. 138–161 CE or c. 330–335"
          />
        </div>
        <div v-if="form.category === 'Roman'" class="form-group min-w-0">
          <label class="form-label">Imperial figure (optional)</label>
          <ImperialFigurePicker v-model="form.romanImperialFigureId!" />
          <p class="mt-1 text-sm text-text-muted">
            Matches this coin to a curated Roman emperor, empress, Caesar, or usurper for the
            <router-link to="/sets/emperors" class="underline">Emperor Tracker</router-link>. Leave blank if unsure — the
            free-text Ruler field above is unaffected.
          </p>
        </div>
        <div class="form-group min-w-0">
          <label class="form-label">Storage Location</label>
          <select v-model="storageLocationIdModel" class="form-select" :disabled="storageLocationsLoading">
            <option value="">None</option>
            <option
              v-for="location in storageLocations"
              :key="location.id"
              :value="String(location.id)"
            >
              {{ location.name }}
            </option>
          </select>
          <p v-if="storageLocationError" class="mt-2 text-body text-text-secondary">{{ storageLocationError }}</p>
          <div v-if="selectedStorageLocation?.type === 'tray'" class="mt-3">
            <p class="text-body text-text-secondary">
              Choose an exact slot. {{ selectedStorageLocation.occupied }} / {{ selectedStorageLocation.capacity }} occupied.
            </p>
            <p v-if="occupancyLoading" class="text-body text-text-muted">Loading tray slots...</p>
            <div
              v-else-if="trayOccupancy"
              class="slot-picker"
              role="grid"
              :aria-label="`${selectedStorageLocation.name} slots`"
              :style="{ gridTemplateColumns: `repeat(${trayOccupancy.columns}, minmax(44px, 1fr))` }"
            >
              <button
                v-for="slot in trayOccupancy.capacity"
                :key="slot"
                type="button"
                role="gridcell"
                class="slot-choice"
                :class="{ selected: form.storageSlot === slot, occupied: isOccupiedSlot(slot) }"
                :disabled="isOccupiedSlot(slot)"
                :aria-label="slotLabel(slot)"
                :aria-selected="form.storageSlot === slot"
                @click="form.storageSlot = slot"
              >
                <span>{{ slotCoordinates(slot) }}</span>
                <small>{{ isOccupiedSlot(slot) ? 'Occupied' : form.storageSlot === slot ? 'Selected' : 'Available' }}</small>
              </button>
            </div>
            <p v-if="occupancyError" class="mt-2 text-body text-text-secondary">
              {{ occupancyError }}
              <button type="button" class="btn btn-xs btn-secondary" @click="loadOccupancy">Retry</button>
            </p>
            <p v-if="slotValidationError" class="mt-2 text-body text-text-secondary" role="alert">{{ slotValidationError }}</p>
          </div>
        </div>
      </fieldset>

      <!-- Physical Details -->
      <fieldset class="m-0 rounded-md border border-border-subtle bg-card p-5">
        <h2 class="mb-4 font-display text-lg font-medium text-gold">Physical Details</h2>
        <div class="grid gap-3 md:grid-cols-2">
          <div class="form-group min-w-0">
            <label class="form-label">Weight (grams)</label>
            <input v-model.number="form.weightGrams" class="form-input" type="number" step="0.01" />
          </div>
          <div class="form-group min-w-0">
            <label class="form-label">Diameter (mm)</label>
            <input v-model.number="form.diameterMm" class="form-input" type="number" step="0.1" />
          </div>
        </div>
        <div class="form-group min-w-0">
          <label class="form-label">Grade</label>
          <input v-model="form.grade" class="form-input" placeholder="e.g. VF, EF, MS-65" />
        </div>
      </fieldset>

      <!-- Inscriptions, Images & Descriptions -->
      <fieldset class="m-0 rounded-md border border-border-subtle bg-card p-5 md:col-span-2">
        <h2 class="mb-4 font-display text-lg font-medium text-gold">Inscriptions & Descriptions</h2>
        <div class="grid gap-3 md:grid-cols-2">
          <div class="form-group min-w-0">
            <label class="form-label">Obverse Inscription</label>
            <input v-model="form.obverseInscription" class="form-input" placeholder="Obverse legend text" />
          </div>
          <div class="form-group min-w-0">
            <label class="form-label">Reverse Inscription</label>
            <input v-model="form.reverseInscription" class="form-input" placeholder="Reverse legend text" />
          </div>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <div class="form-group min-w-0">
            <label class="form-label">Obverse Image</label>
            <div v-if="obversePreview || existingObverse" class="relative mb-2 inline-block">
              <img v-if="obversePreview" :src="obversePreview" alt="Obverse" class="h-[140px] w-[140px] rounded-sm border border-border-subtle object-cover" />
              <AuthenticatedImage v-else :media-path="existingObverse" alt="Obverse" class="h-[140px] w-[140px] rounded-sm border border-border-subtle object-cover" />
              <button type="button" class="absolute -top-1.5 -right-1.5 flex h-[22px] w-[22px] items-center justify-center rounded-full border-0 bg-red-700 p-0 text-white" @click="clearObverse" title="Remove" aria-label="Remove obverse image"><X :size="12" /></button>
            </div>
            <div class="flex items-center gap-2">
              <input type="file" accept=".jpg,.jpeg,.png" class="form-input min-w-0 flex-1 text-base" aria-label="Upload obverse image" @change="onObverseFile" ref="obverseInput" />
              <label v-if="isPwa" class="btn btn-secondary btn-sm inline-flex cursor-pointer items-center gap-1.5 whitespace-nowrap">
                <Camera :size="14" /> Photo
                <input type="file" accept="image/*" capture="environment" hidden @change="onObverseFile" />
              </label>
            </div>
          </div>
          <div class="form-group min-w-0">
            <label class="form-label">Reverse Image</label>
            <div v-if="reversePreview || existingReverse" class="relative mb-2 inline-block">
              <img v-if="reversePreview" :src="reversePreview" alt="Reverse" class="h-[140px] w-[140px] rounded-sm border border-border-subtle object-cover" />
              <AuthenticatedImage v-else :media-path="existingReverse" alt="Reverse" class="h-[140px] w-[140px] rounded-sm border border-border-subtle object-cover" />
              <button type="button" class="absolute -top-1.5 -right-1.5 flex h-[22px] w-[22px] items-center justify-center rounded-full border-0 bg-red-700 p-0 text-white" @click="clearReverse" title="Remove" aria-label="Remove reverse image"><X :size="12" /></button>
            </div>
            <div class="flex items-center gap-2">
              <input type="file" accept=".jpg,.jpeg,.png" class="form-input min-w-0 flex-1 text-base" aria-label="Upload reverse image" @change="onReverseFile" ref="reverseInput" />
              <label v-if="isPwa" class="btn btn-secondary btn-sm inline-flex cursor-pointer items-center gap-1.5 whitespace-nowrap">
                <Camera :size="14" /> Photo
                <input type="file" accept="image/*" capture="environment" hidden @change="onReverseFile" />
              </label>
            </div>
          </div>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <div class="form-group min-w-0">
            <label class="form-label">Obverse Description</label>
            <textarea v-model="form.obverseDescription" class="form-textarea" placeholder="Describe the obverse design" />
          </div>
          <div class="form-group min-w-0">
            <label class="form-label">Reverse Description</label>
            <textarea v-model="form.reverseDescription" class="form-textarea" placeholder="Describe the reverse design" />
          </div>
        </div>
      </fieldset>

      <!-- Purchase Info -->
      <fieldset class="m-0 rounded-md border border-border-subtle bg-card p-5">
        <h2 class="mb-4 font-display text-lg font-medium text-gold">Purchase & Value</h2>
        <div class="grid gap-3 md:grid-cols-2">
          <div class="form-group min-w-0">
            <label class="form-label">Purchase Price ($)</label>
            <input v-model.number="form.purchasePrice" class="form-input" type="number" step="0.01" />
          </div>
          <div class="form-group min-w-0">
            <label class="form-label">Current Value ($)</label>
            <input v-model.number="form.currentValue" class="form-input" type="number" step="0.01" />
          </div>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <div class="form-group min-w-0">
            <label class="form-label">Purchase Date</label>
            <input v-model="form.purchaseDate" class="form-input" type="date" />
          </div>
          <div class="form-group min-w-0">
            <label class="form-label">Store</label>
            <AutocompleteInput v-model="form.purchaseLocation!" field="purchaseLocation" placeholder="e.g. Heritage Auctions" />
          </div>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <div class="form-group min-w-0">
            <label class="form-label">Vendor SKU</label>
            <input v-model="form.vendorSku" class="form-input" placeholder="e.g. CNG-12345" />
          </div>
          <div class="form-group min-w-0">
            <label class="form-label">Vendor Invoice</label>
            <input v-model="form.vendorInvoice" class="form-input" placeholder="e.g. INV-2026-0421" />
          </div>
        </div>
      </fieldset>

      <!-- Links & Notes -->
      <fieldset class="m-0 rounded-md border border-border-subtle bg-card p-5">
        <h2 class="mb-4 font-display text-lg font-medium text-gold">Reference & Notes</h2>
        <div class="grid gap-3 md:grid-cols-2">
          <div class="form-group min-w-0">
            <label class="form-label">Reference URL</label>
            <input v-model="form.referenceUrl" class="form-input" type="url" placeholder="https://..." />
          </div>
          <div class="form-group min-w-0">
            <label class="form-label">Reference Text</label>
            <input v-model="form.referenceText" class="form-input" placeholder="Link display text" />
          </div>
        </div>
        <div class="form-group min-w-0">
          <label class="form-label">Store Card Image</label>
          <p class="mb-2 text-body text-text-muted">Upload a photo of the store card. Text will be extracted automatically and saved to Notes.</p>
          <div v-if="cardPreview" class="relative mb-2 inline-block">
            <img :src="cardPreview" alt="Store card" class="h-[140px] w-[140px] rounded-sm border border-border-subtle object-cover" />
            <button type="button" class="absolute -top-1.5 -right-1.5 flex h-[22px] w-[22px] items-center justify-center rounded-full border-0 bg-red-700 p-0 text-white" @click="clearCard" title="Remove" aria-label="Remove store card image"><X :size="12" /></button>
          </div>
          <input type="file" accept=".jpg,.jpeg,.png" class="form-input text-base" aria-label="Upload store card image" @change="onCardFile" ref="cardInput" />
        </div>
        <div class="form-group min-w-0">
          <label class="form-label">Notes</label>
          <textarea v-model="form.notes" class="form-textarea" rows="3" placeholder="Any additional notes..." />
        </div>
        <div class="form-group flex items-center gap-3">
          <label class="form-label mb-0">Private Coin</label>
          <label class="relative inline-flex cursor-pointer items-center">
            <input v-model="form.isPrivate" type="checkbox" class="peer sr-only" />
            <span class="relative h-6 w-11 rounded-full border border-border-subtle bg-input transition-colors peer-checked:border-border-accent peer-checked:bg-gold-dim after:absolute after:top-[1px] after:left-[1px] after:h-5 after:w-5 after:rounded-full after:bg-text-primary after:content-[''] after:transition-transform peer-checked:after:translate-x-5"></span>
          </label>
          <span class="text-chip text-text-secondary">Hidden from followers</span>
        </div>
      </fieldset>
    </div>

    <div class="mt-8 flex justify-end gap-3 border-t border-border-subtle pt-6">
      <button type="button" class="btn btn-secondary" @click="$router.back()">Cancel</button>
      <button type="submit" class="btn btn-primary" :disabled="loading">
        {{ loading ? 'Saving...' : submitLabel }}
      </button>
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
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { getStorageLocations, getStorageLocationOccupancy, getMintLocations, type MintLocationsResponse } from '@/api/client'
import type { Coin, StorageLocation, MintLocation, TrayOccupancy } from '@/types'
import AutocompleteInput from '@/components/AutocompleteInput.vue'
import ImperialFigurePicker from '@/components/ImperialFigurePicker.vue'
import CreateMintModal from '@/components/CreateMintModal.vue'
import { X, Camera } from 'lucide-vue-next'
import { usePwa } from '@/composables/usePwa'
import { useCoinOptions } from '@/composables/useCoinOptions'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'

function unwrapMintLocations(data: MintLocationsResponse): MintLocation[] {
  return Array.isArray(data) ? data : data.mintLocations ?? []
}

const { isPwa } = usePwa()
const { categoryOptions, eraOptions, materialOptions, loadOptions } = useCoinOptions()

const props = defineProps<{
  form: Partial<Coin>
  submitLabel: string
  loading?: boolean
  coinId?: number
}>()

const emit = defineEmits<{ submit: [] }>()

const obverseFile = ref<File | null>(null)
const reverseFile = ref<File | null>(null)
const cardFile = ref<File | null>(null)
const obversePreview = ref<string | null>(null)
const reversePreview = ref<string | null>(null)
const cardPreview = ref<string | null>(null)
const obverseInput = ref<HTMLInputElement | null>(null)
const reverseInput = ref<HTMLInputElement | null>(null)
const cardInput = ref<HTMLInputElement | null>(null)
const removedObverseId = ref<number | null>(null)
const removedReverseId = ref<number | null>(null)
const storageLocations = ref<StorageLocation[]>([])
const storageLocationsLoading = ref(false)
const storageLocationError = ref('')
const trayOccupancy = ref<TrayOccupancy | null>(null)
const occupancyLoading = ref(false)
const occupancyError = ref('')
const slotValidationError = ref('')
const selectedStorageLocation = computed(() =>
  storageLocations.value.find((location) => location.id === props.form.storageLocationId) ?? null
)

const storageLocationIdModel = computed({
  get: () => props.form.storageLocationId == null ? '' : String(props.form.storageLocationId),
  set: (value: string) => {
    const next = value === '' ? null : Number(value)
    if (props.form.storageLocationId !== next) {
      props.form.storageSlot = null
      trayOccupancy.value = null
      slotValidationError.value = ''
    }
    props.form.storageLocationId = next
  },
})

async function loadOccupancy() {
  const location = selectedStorageLocation.value
  if (!location || location.type !== 'tray') {
    trayOccupancy.value = null
    return
  }
  occupancyLoading.value = true
  occupancyError.value = ''
  try {
    const response = await getStorageLocationOccupancy(location.id, props.coinId)
    trayOccupancy.value = response.data
  } catch {
    trayOccupancy.value = null
    occupancyError.value = 'Tray occupancy changed or could not be loaded.'
  } finally {
    occupancyLoading.value = false
  }
}

function isOccupiedSlot(slot: number): boolean {
  return trayOccupancy.value?.occupiedSlots.includes(slot) === true &&
    trayOccupancy.value.currentCoinSlot !== slot
}

function slotCoordinates(slot: number): string {
  const columns = trayOccupancy.value?.columns ?? 1
  return `${Math.floor((slot - 1) / columns) + 1},${((slot - 1) % columns) + 1}`
}

function slotLabel(slot: number): string {
  const state = isOccupiedSlot(slot) ? 'occupied' : props.form.storageSlot === slot ? 'selected' : 'available'
  const [row, column] = slotCoordinates(slot).split(',')
  return `Row ${row}, column ${column}, ${state}`
}

function handleSubmit() {
  if (selectedStorageLocation.value?.type === 'tray' && props.form.storageSlot == null) {
    slotValidationError.value = 'Choose an exact tray slot before saving.'
    return
  }
  slotValidationError.value = ''
  emit('submit')
}

watch(() => props.form.storageLocationId, () => {
  void loadOccupancy()
})

const mintLocations = ref<MintLocation[]>([])
const mintLocationsLoading = ref(false)
const mintLocationError = ref('')
const showCreateMintModal = ref(false)
const pendingMintName = ref('')

const myMintLocations = computed(() => mintLocations.value.filter((m) => m.userId != null))
const globalMintLocations = computed(() => mintLocations.value.filter((m) => m.userId == null))

const mintLocationIdModel = computed({
  get: () => props.form.mintLocationId == null ? '' : String(props.form.mintLocationId),
  set: (value: string) => {
    if (value === '__create__') {
      pendingMintName.value = ''
      showCreateMintModal.value = true
      return
    }
    props.form.mintLocationId = value === '' ? null : Number(value)
    const selected = mintLocations.value.find((m) => String(m.id) === value)
    props.form.mint = selected ? selected.displayName : ''
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
  props.form.mintLocationId = mintLocation.id
  props.form.mint = mintLocation.displayName
}

const displayedEraOptions = computed(() => {
  const currentEra = typeof props.form.era === 'string' ? props.form.era.trim() : ''
  if (currentEra && !eraOptions.value.includes(currentEra)) {
    return [currentEra, ...eraOptions.value]
  }
  return eraOptions.value
})

watch(() => props.form.category, (category) => {
  if (category !== 'Roman') {
    props.form.romanImperialFigureId = null
  }
})

onMounted(async () => {
  // Load coin property options from settings
  loadOptions()
  
  // Load storage locations
  storageLocationsLoading.value = true
  try {
    const res = await getStorageLocations()
    storageLocations.value = res.data?.storageLocations ?? []
    await loadOccupancy()
  } catch {
    storageLocations.value = []
    storageLocationError.value = 'Storage locations are unavailable'
  } finally {
    storageLocationsLoading.value = false
  }

  loadMintLocations()
})

const existingObverse = computed(() => {
  if (removedObverseId.value) return null
  const img = props.form.images?.find((i) => i.imageType === 'obverse')
  return img ? img.filePath : null
})

const existingReverse = computed(() => {
  if (removedReverseId.value) return null
  const img = props.form.images?.find((i) => i.imageType === 'reverse')
  return img ? img.filePath : null
})

function onObverseFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (obversePreview.value) URL.revokeObjectURL(obversePreview.value)
  obverseFile.value = file
  obversePreview.value = URL.createObjectURL(file)
}

function onReverseFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (reversePreview.value) URL.revokeObjectURL(reversePreview.value)
  reverseFile.value = file
  reversePreview.value = URL.createObjectURL(file)
}

function clearObverse() {
  const existing = props.form.images?.find((i) => i.imageType === 'obverse')
  if (existing) removedObverseId.value = existing.id
  if (obversePreview.value) URL.revokeObjectURL(obversePreview.value)
  obverseFile.value = null
  obversePreview.value = null
  if (obverseInput.value) obverseInput.value.value = ''
}

function clearReverse() {
  const existing = props.form.images?.find((i) => i.imageType === 'reverse')
  if (existing) removedReverseId.value = existing.id
  if (reversePreview.value) URL.revokeObjectURL(reversePreview.value)
  reverseFile.value = null
  reversePreview.value = null
  if (reverseInput.value) reverseInput.value.value = ''
}

function onCardFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (cardPreview.value) URL.revokeObjectURL(cardPreview.value)
  cardFile.value = file
  cardPreview.value = URL.createObjectURL(file)
}

function clearCard() {
  if (cardPreview.value) URL.revokeObjectURL(cardPreview.value)
  cardFile.value = null
  cardPreview.value = null
  if (cardInput.value) cardInput.value.value = ''
}

onBeforeUnmount(() => {
  if (obversePreview.value) URL.revokeObjectURL(obversePreview.value)
  if (reversePreview.value) URL.revokeObjectURL(reversePreview.value)
  if (cardPreview.value) URL.revokeObjectURL(cardPreview.value)
})

// Expose pending images for parent to upload after save
defineExpose({
  obverseFile,
  reverseFile,
  cardFile,
  removedObverseId,
  removedReverseId,
  refreshStorageOccupancy: loadOccupancy,
})
</script>

<style scoped>
.slot-picker {
  display: grid;
  gap: 0.35rem;
  max-width: 100%;
  overflow-x: auto;
  padding: 0.35rem;
}

.slot-choice {
  min-width: 44px;
  min-height: 44px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-input);
  color: var(--text-primary);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.slot-choice small {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.slot-choice.selected {
  border-color: var(--accent-gold);
  background: var(--accent-gold-dim);
}

.slot-choice.occupied {
  opacity: 0.55;
}

.slot-choice:focus-visible {
  outline: 2px solid var(--accent-gold);
  outline-offset: 2px;
}
</style>
