<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-[300] flex items-center justify-center bg-[rgba(0,0,0,0.6)] p-4" @click.self="handleClose">
      <div class="flex max-h-[90vh] w-full max-w-[520px] flex-col overflow-hidden rounded-md border border-border-subtle bg-card shadow-[0_12px_40px_rgba(0,0,0,0.5)]">
        <div class="flex items-center justify-between gap-4 border-b border-border-subtle px-5 py-4">
          <h3 class="m-0 text-lg font-medium text-heading">Create new mint</h3>
          <button class="inline-flex h-[30px] w-[30px] items-center justify-center p-0 text-text-muted hover:text-text-primary" @click="handleClose">×</button>
        </div>

        <div class="flex flex-1 flex-col gap-3 overflow-y-auto p-5">
          <div class="form-group min-w-0">
            <label class="form-label">Mint name</label>
            <div class="flex gap-2">
              <input
                v-model="name"
                class="form-input flex-1"
                placeholder="e.g. Sirmium"
                @keydown.enter.prevent="search"
              />
              <button type="button" class="btn btn-secondary btn-sm" :disabled="searching || !name.trim()" @click="search">
                {{ searching ? 'Searching…' : 'Search' }}
              </button>
            </div>
          </div>

          <p v-if="searched && candidates.length === 0" class="text-body text-text-secondary">
            No matches found for "{{ lastSearchedName }}" — click the map below to place a pin manually.
          </p>

          <div v-if="candidates.length > 1" class="flex flex-wrap gap-[0.35rem]">
            <button
              v-for="(candidate, idx) in candidates"
              :key="idx"
              type="button"
              class="chip chip-sm"
              :class="{ active: selectedCandidateIdx === idx }"
              @click="selectCandidate(idx)"
            >
              {{ candidate.displayName }}
            </button>
          </div>

          <div ref="mapElement" class="mint-pin-picker" data-testid="create-mint-map"></div>

          <p class="text-body text-text-secondary">
            {{ pin ? `Pin set at ${pin.lat.toFixed(4)}, ${pin.lng.toFixed(4)} — drag to adjust.` : 'Click the map to place a pin.' }}
          </p>

          <div class="form-group min-w-0">
            <label class="form-label">Region (optional)</label>
            <input v-model="region" class="form-input" placeholder="e.g. Pannonia" />
          </div>

          <p v-if="error" class="text-body text-[var(--color-negative)]">{{ error }}</p>
        </div>

        <div class="flex justify-end gap-2 border-t border-border-subtle px-5 py-4">
          <button type="button" class="btn btn-secondary btn-sm" @click="handleClose">Cancel</button>
          <button
            type="button"
            class="btn btn-primary btn-sm"
            :disabled="!canSave || saving"
            @click="save"
          >
            {{ saving ? 'Saving…' : 'Save mint' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import 'leaflet/dist/leaflet.css'
import * as L from 'leaflet'
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { createMintLocation, geocodeMintName } from '@/api/client'
import type { GeocodeCandidate, MintLocation } from '@/types'
import { DEFAULT_MAP_CENTER, DEFAULT_MAP_ZOOM, OSM_ATTRIBUTION, OSM_TILE_URL } from '@/components/map/MintMapLeaflet.vue'

const props = defineProps<{
  open: boolean
  initialName?: string
}>()

const emit = defineEmits<{
  close: []
  created: [mintLocation: MintLocation]
}>()

const name = ref('')
const region = ref('')
const searching = ref(false)
const searched = ref(false)
const lastSearchedName = ref('')
const candidates = ref<GeocodeCandidate[]>([])
const selectedCandidateIdx = ref<number | null>(null)
const pin = ref<{ lat: number; lng: number } | null>(null)
const saving = ref(false)
const error = ref('')

const mapElement = ref<HTMLElement | null>(null)
let map: L.Map | null = null
let marker: L.Marker | null = null

const canSave = computed(() => name.value.trim() !== '' && pin.value !== null)

function resetState() {
  name.value = props.initialName ?? ''
  region.value = ''
  searching.value = false
  searched.value = false
  lastSearchedName.value = ''
  candidates.value = []
  selectedCandidateIdx.value = null
  pin.value = null
  saving.value = false
  error.value = ''
}

function setPin(lat: number, lng: number) {
  pin.value = { lat, lng }
  if (map) {
    if (marker) {
      marker.setLatLng([lat, lng])
    } else if (mapElement.value) {
      marker = L.marker([lat, lng], { draggable: true }).addTo(map)
      marker.on('dragend', () => {
        const pos = marker!.getLatLng()
        pin.value = { lat: pos.lat, lng: pos.lng }
      })
    }
    map.setView([lat, lng], 6)
  }
}

function selectCandidate(idx: number) {
  selectedCandidateIdx.value = idx
  const candidate = candidates.value[idx]
  if (candidate) setPin(candidate.lat, candidate.lng)
}

async function search() {
  const query = name.value.trim()
  if (!query) return
  searching.value = true
  error.value = ''
  try {
    const res = await geocodeMintName(query)
    candidates.value = res.data.candidates ?? []
    searched.value = true
    lastSearchedName.value = query
    selectedCandidateIdx.value = null
    if (candidates.value.length > 0) {
      selectCandidate(0)
    }
  } catch {
    candidates.value = []
    searched.value = true
    lastSearchedName.value = query
  } finally {
    searching.value = false
  }
}

async function save() {
  if (!canSave.value || !pin.value) return
  saving.value = true
  error.value = ''
  try {
    const res = await createMintLocation({
      displayName: name.value.trim(),
      lat: pin.value.lat,
      lng: pin.value.lng,
      region: region.value.trim(),
      aliases: [],
    })
    emit('created', res.data)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    error.value = err.response?.data?.error || 'Failed to save mint'
  } finally {
    saving.value = false
  }
}

function handleClose() {
  emit('close')
}

function initMap() {
  if (!mapElement.value || map) return
  map = L.map(mapElement.value, {
    center: DEFAULT_MAP_CENTER,
    zoom: DEFAULT_MAP_ZOOM,
    scrollWheelZoom: true,
  })
  L.tileLayer(OSM_TILE_URL, { attribution: OSM_ATTRIBUTION, maxZoom: 19 }).addTo(map)
  map.on('click', (event: L.LeafletMouseEvent) => {
    setPin(event.latlng.lat, event.latlng.lng)
  })
}

function destroyMap() {
  marker = null
  map?.remove()
  map = null
}

watch(() => props.open, async (isOpen) => {
  if (isOpen) {
    resetState()
    await nextTick()
    initMap()
  } else {
    destroyMap()
  }
}, { immediate: true })

onBeforeUnmount(destroyMap)
</script>

<style scoped>
.mint-pin-picker {
  width: 100%;
  height: 260px;
  border-radius: var(--radius-sm);
  overflow: hidden;
  background: var(--bg-card);
}

:deep(.leaflet-container) {
  background: var(--bg-card);
  color: var(--text-primary);
  font-family: inherit;
}

:deep(.leaflet-container img) {
  max-width: none;
}
</style>
