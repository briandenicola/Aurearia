<template>
  <section class="mint-map-card card" aria-label="Ancient mint map">
    <div ref="mapElement" class="mint-map-leaflet" data-testid="mint-map-leaflet"></div>
  </section>
</template>

<script lang="ts">
export const OSM_TILE_URL = 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png'
export const OSM_ATTRIBUTION = '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
export const DEFAULT_MAP_CENTER: [number, number] = [38, 16]
export const DEFAULT_MAP_ZOOM = 5
</script>

<script setup lang="ts">
import 'leaflet/dist/leaflet.css'
import * as L from 'leaflet'
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { MintGroup } from '@/utils/mintMap'

const props = defineProps<{
  groups: MintGroup[]
  selectedMintId?: string | null
}>()

const emit = defineEmits<{
  'select-mint': [group: MintGroup]
}>()

const mapElement = ref<HTMLElement | null>(null)
let map: L.Map | null = null
let markerLayer: L.LayerGroup | null = null

function markerLabel(group: MintGroup): string {
  return `${group.mint.displayName}: ${group.count} ${group.count === 1 ? 'coin' : 'coins'}`
}

function renderMarkers() {
  if (!map || !markerLayer) return

  markerLayer.clearLayers()
  const markerLatLngs: L.LatLngExpression[] = []

  for (const group of props.groups) {
    const latLng: L.LatLngExpression = [group.mint.lat, group.mint.lng]
    const selected = props.selectedMintId === group.mint.id
    const marker = L.marker(latLng, {
      keyboard: true,
      title: markerLabel(group),
      alt: markerLabel(group),
      icon: L.divIcon({
        className: `mint-leaflet-marker${selected ? ' selected' : ''}`,
        html: `<span class="mint-leaflet-marker-count">${group.count}</span>`,
        iconSize: [34, 34],
        iconAnchor: [17, 17],
      }),
    })

    marker.on('click', () => emit('select-mint', group))
    marker.addTo(markerLayer)
    markerLatLngs.push(latLng)

    const element = marker.getElement()
    element?.setAttribute('role', 'button')
    element?.setAttribute('aria-label', markerLabel(group))
    element?.setAttribute('tabindex', '0')
    element?.addEventListener('keydown', (event) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault()
        emit('select-mint', group)
      }
    })
  }

  if (markerLatLngs.length > 0) {
    map.fitBounds(L.latLngBounds(markerLatLngs).pad(0.2), { maxZoom: 8 })
  } else {
    map.setView(DEFAULT_MAP_CENTER, DEFAULT_MAP_ZOOM)
  }
}

onMounted(async () => {
  await nextTick()
  if (!mapElement.value) return

  map = L.map(mapElement.value, {
    center: DEFAULT_MAP_CENTER,
    zoom: DEFAULT_MAP_ZOOM,
    scrollWheelZoom: true,
  })
  L.tileLayer(OSM_TILE_URL, {
    attribution: OSM_ATTRIBUTION,
    maxZoom: 19,
  }).addTo(map)
  markerLayer = L.layerGroup().addTo(map)
  renderMarkers()
})

watch(() => [props.groups, props.selectedMintId] as const, renderMarkers, { deep: true })

onBeforeUnmount(() => {
  map?.remove()
  map = null
  markerLayer = null
})
</script>

<style scoped>
.mint-map-card {
  min-height: 420px;
  padding: 0;
  overflow: hidden;
  border-radius: var(--radius-md);
}

.mint-map-leaflet {
  width: 100%;
  min-height: 420px;
  height: min(70vh, 640px);
  background: var(--bg-card);
}

:deep(.leaflet-container) {
  background: var(--bg-card);
  color: var(--text-primary);
  font-family: inherit;
}

:deep(.leaflet-control-attribution) {
  display: block;
}

:deep(.mint-leaflet-marker) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 2px solid var(--bg-card);
  border-radius: var(--radius-full);
  background: var(--accent-gold);
  box-shadow: 0 0 0 8px var(--accent-gold-glow);
  color: var(--bg-primary);
  cursor: pointer;
  transition: transform var(--transition-fast), box-shadow var(--transition-fast);
}

:deep(.mint-leaflet-marker:hover),
:deep(.mint-leaflet-marker:focus-visible),
:deep(.mint-leaflet-marker.selected) {
  box-shadow: 0 0 0 10px var(--accent-gold-dim);
  transform: scale(1.08);
}

:deep(.mint-leaflet-marker-count) {
  font-size: 0.75rem;
  font-weight: 700;
  line-height: 1;
}

@media (max-width: 768px) {
  .mint-map-card,
  .mint-map-leaflet {
    min-height: 360px;
  }

  .mint-map-leaflet {
    height: 62vh;
  }
}
</style>
