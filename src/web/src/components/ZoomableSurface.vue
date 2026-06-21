<template>
  <div class="zoomable-surface">
    <div class="zoomable-toolbar" aria-label="Chart zoom controls">
      <span class="zoomable-status" aria-live="polite">{{ Math.round(scale * 100) }}%</span>
      <button
        type="button"
        class="btn btn-xs btn-ghost zoom-control"
        aria-label="Zoom out"
        @click="zoomOut"
      >
        <ZoomOut aria-hidden="true" :size="14" />
      </button>
      <button
        type="button"
        class="btn btn-xs btn-ghost zoom-control"
        aria-label="Reset chart zoom"
        @click="reset"
      >
        <RotateCcw aria-hidden="true" :size="14" />
      </button>
      <button
        type="button"
        class="btn btn-xs btn-ghost zoom-control"
        aria-label="Zoom in"
        @click="zoomIn"
      >
        <ZoomIn aria-hidden="true" :size="14" />
      </button>
    </div>

    <div
      ref="viewportRef"
      class="zoomable-viewport"
      :class="{ 'is-panning': isPanning }"
      role="region"
      tabindex="0"
      :aria-label="resolvedAriaLabel"
      @wheel="handleWheel"
      @pointerdown="handlePointerDown"
      @pointermove="handlePointerMove"
      @pointerup="handlePointerUp"
      @pointercancel="handlePointerCancel"
      @lostpointercapture="handlePointerCancel"
      @keydown="handleKeydown"
    >
      <div class="zoomable-content" :style="contentStyle">
        <slot />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { RotateCcw, ZoomIn, ZoomOut } from 'lucide-vue-next'

interface Point {
  x: number
  y: number
}

interface ChartWheelEvent extends Event {
  deltaY: number
  clientX: number
  clientY: number
}

const props = withDefaults(defineProps<{
  ariaLabel?: string
  minScale?: number
  maxScale?: number
  scaleStep?: number
  keyboardPanStep?: number
}>(), {
  minScale: 0.75,
  maxScale: 3,
  scaleStep: 0.2,
  keyboardPanStep: 24,
})

const viewportRef = ref<HTMLElement | null>(null)
const scale = ref(1)
const pan = ref<Point>({ x: 0, y: 0 })
const isPanning = ref(false)
const lastPointer = ref<Point | null>(null)
const pinchDistance = ref<number | null>(null)
const activePointers = new Map<number, Point>()
const resolvedAriaLabel = computed(() => props.ariaLabel ?? 'Zoomable chart')

const contentStyle = computed(() => ({
  transform: `translate(${pan.value.x}px, ${pan.value.y}px) scale(${scale.value})`,
}))

function clamp(value: number): number {
  return Math.min(props.maxScale, Math.max(props.minScale, value))
}

function viewportCenter(): Point {
  const rect = viewportRef.value?.getBoundingClientRect()
  if (!rect) return { x: 0, y: 0 }
  return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 }
}

function zoomAt(nextScale: number, clientPoint: Point) {
  const viewport = viewportRef.value
  if (!viewport) return
  const rect = viewport.getBoundingClientRect()
  const local = { x: clientPoint.x - rect.left, y: clientPoint.y - rect.top }
  const clamped = clamp(nextScale)
  const contentPoint = {
    x: (local.x - pan.value.x) / scale.value,
    y: (local.y - pan.value.y) / scale.value,
  }
  scale.value = clamped
  pan.value = {
    x: local.x - contentPoint.x * clamped,
    y: local.y - contentPoint.y * clamped,
  }
}

function zoomIn() {
  zoomAt(scale.value + props.scaleStep, viewportCenter())
}

function zoomOut() {
  zoomAt(scale.value - props.scaleStep, viewportCenter())
}

function reset() {
  scale.value = 1
  pan.value = { x: 0, y: 0 }
  isPanning.value = false
  lastPointer.value = null
  pinchDistance.value = null
  activePointers.clear()
}

function handleWheel(event: Event) {
  const wheelEvent = event as ChartWheelEvent
  wheelEvent.preventDefault()
  const direction = wheelEvent.deltaY < 0 ? 1 : -1
  zoomAt(scale.value + direction * props.scaleStep, { x: wheelEvent.clientX, y: wheelEvent.clientY })
}

function pointerPoint(event: PointerEvent): Point {
  return { x: event.clientX, y: event.clientY }
}

function handlePointerDown(event: PointerEvent) {
  activePointers.set(event.pointerId, pointerPoint(event))
  const target = event.currentTarget as HTMLElement | null
  target?.setPointerCapture?.(event.pointerId)
  if (activePointers.size === 1) {
    lastPointer.value = pointerPoint(event)
    isPanning.value = false
  } else if (activePointers.size === 2) {
    pinchDistance.value = distanceBetweenActivePointers()
  }
}

function handlePointerMove(event: PointerEvent) {
  if (!activePointers.has(event.pointerId)) return
  activePointers.set(event.pointerId, pointerPoint(event))

  if (activePointers.size >= 2) {
    const nextDistance = distanceBetweenActivePointers()
    const center = centerOfActivePointers()
    if (pinchDistance.value && nextDistance) {
      event.preventDefault()
      zoomAt(scale.value * (nextDistance / pinchDistance.value), center)
    }
    pinchDistance.value = nextDistance
    return
  }

  if (!lastPointer.value || (event.pointerType === 'mouse' && event.buttons !== 1)) return
  const current = pointerPoint(event)
  const dx = current.x - lastPointer.value.x
  const dy = current.y - lastPointer.value.y
  if (Math.abs(dx) + Math.abs(dy) > 1) {
    event.preventDefault()
    isPanning.value = true
    pan.value = { x: pan.value.x + dx, y: pan.value.y + dy }
  }
  lastPointer.value = current
}

function handlePointerUp(event: PointerEvent) {
  activePointers.delete(event.pointerId)
  lastPointer.value = null
  pinchDistance.value = activePointers.size >= 2 ? distanceBetweenActivePointers() : null
  window.setTimeout(() => {
    isPanning.value = false
  }, 0)
}

function handlePointerCancel(event: PointerEvent) {
  activePointers.delete(event.pointerId)
  if (!activePointers.size) {
    isPanning.value = false
    lastPointer.value = null
    pinchDistance.value = null
  }
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === '+' || event.key === '=') {
    event.preventDefault()
    zoomIn()
    return
  }
  if (event.key === '-') {
    event.preventDefault()
    zoomOut()
    return
  }
  if (event.key === '0' || event.key === 'Home') {
    event.preventDefault()
    reset()
    return
  }

  const step = props.keyboardPanStep
  const deltas: Record<string, Point> = {
    ArrowLeft: { x: step, y: 0 },
    ArrowRight: { x: -step, y: 0 },
    ArrowUp: { x: 0, y: step },
    ArrowDown: { x: 0, y: -step },
  }
  const delta = deltas[event.key]
  if (!delta) return
  event.preventDefault()
  pan.value = { x: pan.value.x + delta.x, y: pan.value.y + delta.y }
}

function activePointerPair(): Point[] {
  return [...activePointers.values()].slice(0, 2)
}

function distanceBetweenActivePointers(): number | null {
  const [first, second] = activePointerPair()
  if (!first || !second) return null
  return Math.hypot(second.x - first.x, second.y - first.y)
}

function centerOfActivePointers(): Point {
  const [first, second] = activePointerPair()
  if (!first || !second) return viewportCenter()
  return { x: (first.x + second.x) / 2, y: (first.y + second.y) / 2 }
}
</script>

<style scoped>
.zoomable-surface {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.zoomable-toolbar {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 0.35rem;
}

.zoomable-status {
  min-width: 2.5rem;
  color: var(--text-muted);
  font-size: 0.75rem;
  text-align: right;
}

.zoom-control {
  padding: 0.25rem 0.45rem;
}

.zoomable-viewport {
  position: relative;
  overflow: hidden;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-input);
  cursor: grab;
  touch-action: none;
}

.zoomable-viewport:focus-visible {
  outline: none;
  border-color: var(--accent-gold);
  box-shadow: 0 0 0 2px var(--accent-gold-glow);
}

.zoomable-viewport.is-panning {
  cursor: grabbing;
  user-select: none;
}

.zoomable-content {
  transform-origin: 0 0;
  will-change: transform;
  transition: transform var(--transition-fast);
}

.zoomable-viewport.is-panning .zoomable-content {
  transition: none;
}
</style>
