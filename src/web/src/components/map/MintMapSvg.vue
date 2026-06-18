<template>
  <section class="mint-map-card card" aria-label="Ancient mint map">
    <div class="map-toolbar" aria-label="Map zoom controls">
      <button class="btn btn-sm btn-secondary" type="button" aria-label="Zoom in" @click="zoomIn">
        <Plus :size="16" />
      </button>
      <button class="btn btn-sm btn-secondary" type="button" aria-label="Zoom out" @click="zoomOut">
        <Minus :size="16" />
      </button>
      <button class="btn btn-sm btn-ghost" type="button" @click="resetView">Reset</button>
    </div>

    <svg
      class="mint-map-svg"
      :viewBox="viewBox"
      role="img"
      aria-label="Stylized map of the ancient Mediterranean world with mint pins"
      @wheel.prevent="handleWheel"
      @pointerdown="startPan"
      @pointermove="movePan"
      @pointerup="endPan"
      @pointercancel="endPan"
      @pointerleave="endPan"
    >
      <rect class="sea" x="-40" y="-40" width="1080" height="680" rx="12" />
      <path class="land" d="M0 120 C90 75 190 95 290 70 C420 40 530 90 650 70 C770 55 900 80 1000 45 L1000 0 L0 0 Z" />
      <path class="land" d="M0 390 C130 330 245 350 350 315 C455 280 555 320 680 300 C800 278 910 310 1000 270 L1000 600 L0 600 Z" />
      <path class="island" d="M440 330 C472 314 515 324 535 348 C505 365 463 360 440 330 Z" />
      <path class="island" d="M493 255 C525 240 565 250 585 280 C553 298 512 288 493 255 Z" />
      <path class="coastline" d="M90 210 C210 180 320 205 430 180 C565 150 690 185 830 165" />
      <path class="coastline" d="M320 420 C450 375 575 405 700 365 C790 338 895 350 970 325" />

      <MintPin
        v-for="group in groups"
        :key="group.mint.id"
        :group="group"
        :x="projected[group.mint.id]?.x ?? 0"
        :y="projected[group.mint.id]?.y ?? 0"
        :active="selectedMintId === group.mint.id"
        @select="$emit('select-mint', $event)"
      />
    </svg>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Minus, Plus } from 'lucide-vue-next'
import MintPin from '@/components/map/MintPin.vue'
import { projectLatLngToViewBox, type MintGroup } from '@/utils/mintMap'

const props = defineProps<{
  groups: MintGroup[]
  selectedMintId?: string | null
}>()

defineEmits<{
  'select-mint': [group: MintGroup]
}>()

const baseWidth = 1000
const baseHeight = 600
const zoom = ref(1)
const pan = ref({ x: 0, y: 0 })
const dragStart = ref<{ pointerId: number; x: number; y: number; panX: number; panY: number } | null>(null)

const projected = computed(() => Object.fromEntries(
  props.groups.map((group) => [group.mint.id, projectLatLngToViewBox(group.mint.lat, group.mint.lng)]),
))

const viewBox = computed(() => {
  const width = baseWidth / zoom.value
  const height = baseHeight / zoom.value
  const x = pan.value.x + (baseWidth - width) / 2
  const y = pan.value.y + (baseHeight - height) / 2
  return `${x} ${y} ${width} ${height}`
})

function setZoom(nextZoom: number) {
  zoom.value = Math.min(Math.max(nextZoom, 1), 3)
}

function zoomIn() {
  setZoom(zoom.value + 0.25)
}

function zoomOut() {
  setZoom(zoom.value - 0.25)
}

function resetView() {
  zoom.value = 1
  pan.value = { x: 0, y: 0 }
}

function handleWheel(event: globalThis.WheelEvent) {
  setZoom(zoom.value + (event.deltaY < 0 ? 0.15 : -0.15))
}

function startPan(event: PointerEvent) {
  const target = event.target as globalThis.Element | null
  if (target?.closest('.mint-pin')) return
  dragStart.value = {
    pointerId: event.pointerId,
    x: event.clientX,
    y: event.clientY,
    panX: pan.value.x,
    panY: pan.value.y,
  }
  ;(event.currentTarget as globalThis.SVGSVGElement).setPointerCapture(event.pointerId)
}

function movePan(event: PointerEvent) {
  if (!dragStart.value || dragStart.value.pointerId !== event.pointerId) return
  const scale = 1 / zoom.value
  pan.value = {
    x: dragStart.value.panX - (event.clientX - dragStart.value.x) * scale,
    y: dragStart.value.panY - (event.clientY - dragStart.value.y) * scale,
  }
}

function endPan(event: PointerEvent) {
  if (!dragStart.value || dragStart.value.pointerId !== event.pointerId) return
  const svg = event.currentTarget as globalThis.SVGSVGElement
  if (svg.hasPointerCapture(event.pointerId)) {
    svg.releasePointerCapture(event.pointerId)
  }
  dragStart.value = null
}
</script>

<style scoped>
.mint-map-card {
  position: relative;
  padding: 0;
  overflow: hidden;
  border-radius: var(--radius-md);
}

.map-toolbar {
  position: absolute;
  top: 0.75rem;
  right: 0.75rem;
  z-index: 2;
  display: flex;
  gap: 0.35rem;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.mint-map-svg {
  display: block;
  width: 100%;
  min-height: 420px;
  max-height: 70vh;
  touch-action: none;
  background: var(--bg-card);
}

.sea {
  fill: var(--bg-input);
}

.land {
  fill: var(--bg-card-hover);
  stroke: var(--border-subtle);
  stroke-width: 2;
}

.island {
  fill: var(--bg-card-hover);
  stroke: var(--border-subtle);
  stroke-width: 2;
}

.coastline {
  fill: none;
  stroke: var(--border-accent);
  stroke-width: 2;
  opacity: 0.5;
}

@media (max-width: 768px) {
  .mint-map-svg {
    min-height: 360px;
    max-height: 62vh;
  }

  .map-toolbar {
    left: 0.75rem;
    right: 0.75rem;
  }
}
</style>
