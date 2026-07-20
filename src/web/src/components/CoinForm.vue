<template>
  <form class="mx-auto max-w-[900px]" @submit.prevent="$emit('submit')">
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
            <input v-model="form.mint" class="form-input" placeholder="e.g. Rome" />
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
        <div v-if="form.category === 'Roman'" class="form-group min-w-0">
          <label class="form-label">Imperial figure (optional)</label>
          <ImperialFigurePicker v-model="form.romanImperialFigureId!" />
          <p class="mt-1 text-sm text-text-muted">
            Matches this coin to a curated Roman emperor, empress, Caesar, or usurper for the
            <router-link to="/stats/emperors" class="underline">Emperor Tracker</router-link>. Leave blank if unsure — the
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
  </form>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { getStorageLocations } from '@/api/client'
import type { Coin, StorageLocation } from '@/types'
import AutocompleteInput from '@/components/AutocompleteInput.vue'
import ImperialFigurePicker from '@/components/ImperialFigurePicker.vue'
import { X, Camera } from 'lucide-vue-next'
import { usePwa } from '@/composables/usePwa'
import { useCoinOptions } from '@/composables/useCoinOptions'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'

const { isPwa } = usePwa()
const { categoryOptions, eraOptions, materialOptions, loadOptions } = useCoinOptions()

const props = defineProps<{
  form: Partial<Coin>
  submitLabel: string
  loading?: boolean
  coinId?: number
}>()

defineEmits<{ submit: [] }>()

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

const storageLocationIdModel = computed({
  get: () => props.form.storageLocationId == null ? '' : String(props.form.storageLocationId),
  set: (value: string) => {
    props.form.storageLocationId = value === '' ? null : Number(value)
  },
})

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
  } catch {
    storageLocations.value = []
    storageLocationError.value = 'Storage locations are unavailable'
  } finally {
    storageLocationsLoading.value = false
  }
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
})
</script>
