<template>
  <div class="image-processor">
    <!-- Step 1: Image Input -->
    <ImageInputPanel
      v-if="!sourceImage"
      :source-image="sourceImage"
      :url-loading="urlLoading"
      :input-error="inputError"
      @file-select="loadImageFromFile"
      @url-load="handleUrlLoad"
      @drop="loadImageFromFile"
    />

    <!-- Step 2: Processing & Preview -->
    <div v-if="sourceImage" class="processing-section">
      <div class="step-header">
        <button class="btn btn-secondary btn-sm" @click="reset">← Start Over</button>
        <div class="step-indicators">
          <span class="step" :class="{ active: step === 'preview', done: step !== 'preview' }">1. Original</span>
          <span class="step" :class="{ active: step === 'removing', done: step === 'crop' || step === 'done' }">2.
            Remove BG</span>
          <span class="step" :class="{ active: step === 'crop', done: step === 'done' }">3. Crop</span>
        </div>
      </div>

      <!-- Original preview + remove BG button -->
      <div v-if="step === 'preview'" class="preview-panel">
        <div class="image-preview">
          <img :src="sourceImage" alt="Original" />
        </div>
        <button class="btn btn-primary" @click="removeBackground">Remove Background</button>
      </div>

      <!-- Processing spinner -->
      <div v-if="step === 'removing'" class="preview-panel">
        <div class="image-preview processing">
          <img :src="sourceImage" alt="Processing" class="processing-img" />
          <div class="processing-overlay">
            <div class="spinner"></div>
            <p>Removing background...</p>
            <p class="processing-hint">First run downloads the ML model (~40MB)</p>
          </div>
        </div>
      </div>

      <!-- Crop step -->
      <div v-if="step === 'crop' || step === 'done'" class="crop-panel">
        <div class="crop-workspace">
          <canvas ref="cropCanvas" class="crop-canvas" @pointerdown="startCropDrag" @pointermove="onCropDrag"
            @pointerup="endCropDrag" />
        </div>
        <div class="crop-controls">
          <button class="btn btn-secondary btn-sm" @click="autoCrop">Auto Crop</button>
          <button class="btn btn-secondary btn-sm" @click="resetCrop">Reset Crop</button>
          <label class="padding-control">
            <span>Padding</span>
            <input v-model.number="cropPadding" type="range" min="0" max="50" />
            <span class="padding-value">{{ cropPadding }}px</span>
          </label>
        </div>

        <!-- Result preview -->
        <div class="result-row">
          <div class="result-preview">
            <h4>Result</h4>
            <canvas ref="resultCanvas" class="result-canvas" />
          </div>
          <div class="save-controls">
            <div class="save-tabs">
              <button class="save-tab" :class="{ active: saveTab === 'existing' }" @click="saveTab = 'existing'">Existing Coin</button>
              <button class="save-tab" :class="{ active: saveTab === 'new' }" @click="saveTab = 'new'">New Coin</button>
              <button class="save-tab" :class="{ active: saveTab === 'download' }" @click="saveTab = 'download'">Download</button>
            </div>

            <!-- Assign to existing coin -->
            <div v-if="saveTab === 'existing'" class="save-panel">
              <div class="coin-search">
                <input v-model="coinSearch" type="text" class="form-input" placeholder="Search coins..." @input="searchCoins" />
              </div>
              <div v-if="coinOptions.length" class="coin-list">
                <button v-for="c in coinOptions" :key="c.id" class="coin-option" :class="{ selected: selectedCoinId === c.id }" @click="selectedCoinId = c.id">
                  <span class="coin-option-name">{{ c.name }}</span>
                  <span class="coin-option-meta">{{ [c.ruler, c.era].filter(Boolean).join(' · ') }}</span>
                </button>
              </div>
              <p v-else-if="coinSearch && !coinsLoading" class="hint">No coins found</p>
              <p v-else-if="coinsLoading" class="hint">Searching...</p>
              <p v-else class="hint">Type to search your collection</p>
              <div v-if="selectedCoinId" class="type-row">
                <label class="radio-label">
                  <input v-model="saveImageType" type="radio" value="obverse" name="imgType" />
                  <span>Obverse</span>
                </label>
                <label class="radio-label">
                  <input v-model="saveImageType" type="radio" value="reverse" name="imgType" />
                  <span>Reverse</span>
                </label>
              </div>
              <button class="btn btn-primary" :disabled="!selectedCoinId || saving" @click="saveToExisting">
                {{ saving ? 'Saving...' : 'Upload to Coin' }}
              </button>
            </div>

            <!-- Create new coin -->
            <div v-if="saveTab === 'new'" class="save-panel">
              <label class="field-label">Coin Name</label>
              <input v-model="newCoinName" type="text" class="form-input" placeholder="e.g. Augustus Denarius" />
              <button class="btn btn-primary" :disabled="!newCoinName.trim() || saving" @click="saveToNewCoin">
                {{ saving ? 'Creating...' : 'Create Coin & Upload' }}
              </button>
              <p class="hint">Image will be saved as the obverse</p>
            </div>

            <!-- Download -->
            <div v-if="saveTab === 'download'" class="save-panel">
              <button class="btn btn-secondary" @click="downloadResult">💾 Download PNG</button>
            </div>

            <p v-if="saveMsg" class="msg" :class="{ error: saveError }">{{ saveMsg }}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { removeBackground as removeBg } from '@imgly/background-removal'
import { proxyImage, uploadImage, createCoin, getCoins } from '@/api/client'
import { getCoin } from '@/api/client'
import type { Coin } from '@/types'
import ImageInputPanel from '@/components/ImageInputPanel.vue'

const props = defineProps<{
  coinId?: number
}>()

const emit = defineEmits<{
  saved: [coinId: number]
}>()

// Input state
const sourceImage = ref<string | null>(null)
const urlLoading = ref(false)
const inputError = ref('')

// Processing state
type Step = 'preview' | 'removing' | 'crop' | 'done'
const step = ref<Step>('preview')
const processedBlob = ref<Blob | null>(null)
const processedImage = ref<HTMLImageElement | null>(null)

// Crop state
const cropCanvas = ref<HTMLCanvasElement | null>(null)
const resultCanvas = ref<HTMLCanvasElement | null>(null)
const cropPadding = ref(10)
const cropRect = ref({ x: 0, y: 0, w: 0, h: 0 })
const cropDragging = ref(false)
const cropDragType = ref<'move' | 'nw' | 'ne' | 'sw' | 'se' | null>(null)
const cropDragStart = ref({ x: 0, y: 0, rx: 0, ry: 0, rw: 0, rh: 0 })
let canvasScale = 1

// Save state
const saveTab = ref<'existing' | 'new' | 'download'>('existing')
const saveImageType = ref('obverse')
const saving = ref(false)
const saveMsg = ref('')
const saveError = ref(false)

// Existing coin selection
const coinSearch = ref('')
const coinOptions = ref<Coin[]>([])
const coinsLoading = ref(false)
const selectedCoinId = ref<number | null>(null)
let searchTimeout: ReturnType<typeof setTimeout> | null = null

// New coin
const newCoinName = ref('')

// Pre-select coin if coinId prop provided
onMounted(async () => {
  if (props.coinId) {
    try {
      const res = await getCoin(props.coinId)
      coinOptions.value = [res.data]
      selectedCoinId.value = res.data.id
    } catch { /* ignore */ }
  }
})

// --- Input Methods ---

function loadImageFromFile(file: File) {
  inputError.value = ''
  if (!file.type.startsWith('image/')) {
    inputError.value = 'Please select an image file'
    return
  }
  const reader = new FileReader()
  reader.onload = (e) => {
    sourceImage.value = e.target?.result as string
    step.value = 'preview'
  }
  reader.readAsDataURL(file)
}

async function handleUrlLoad(url: string) {
  if (!url) return
  inputError.value = ''
  urlLoading.value = true
  try {
    const res = await proxyImage(url)
    const blob = res.data as Blob
    const reader = new FileReader()
    reader.onload = (e) => {
      sourceImage.value = e.target?.result as string
      step.value = 'preview'
    }
    reader.readAsDataURL(blob)
  } catch {
    inputError.value = 'Failed to fetch image from URL'
  } finally {
    urlLoading.value = false
  }
}

// --- Background Removal ---

async function removeBackground() {
  if (!sourceImage.value) return
  step.value = 'removing'

  try {
    const response = await fetch(sourceImage.value)
    const srcBlob = await response.blob()
    const result = await removeBg(srcBlob, {
      output: { format: 'image/png', quality: 1 },
    })
    processedBlob.value = result
    const img = new Image()
    img.onload = () => {
      processedImage.value = img
      step.value = 'crop'
      nextTick(() => {
        autoCrop()
      })
    }
    img.src = URL.createObjectURL(result)
  } catch (err) {
    console.error('Background removal failed:', err)
    inputError.value = 'Background removal failed. Please try again.'
    step.value = 'preview'
  }
}

// --- Crop Logic ---

function autoCrop() {
  if (!processedImage.value || !cropCanvas.value) return

  // Draw to offscreen canvas to read pixels
  const img = processedImage.value
  const offscreen = document.createElement('canvas')
  offscreen.width = img.naturalWidth
  offscreen.height = img.naturalHeight
  const ctx = offscreen.getContext('2d')!
  ctx.drawImage(img, 0, 0)

  const data = ctx.getImageData(0, 0, offscreen.width, offscreen.height).data
  let minX = offscreen.width, minY = offscreen.height, maxX = 0, maxY = 0

  for (let y = 0; y < offscreen.height; y++) {
    for (let x = 0; x < offscreen.width; x++) {
      const alpha = data[(y * offscreen.width + x) * 4 + 3] ?? 0
      if (alpha > 10) {
        if (x < minX) minX = x
        if (x > maxX) maxX = x
        if (y < minY) minY = y
        if (y > maxY) maxY = y
      }
    }
  }

  if (maxX <= minX || maxY <= minY) {
    // No visible content found, use full image
    cropRect.value = { x: 0, y: 0, w: img.naturalWidth, h: img.naturalHeight }
  } else {
    const pad = cropPadding.value
    cropRect.value = {
      x: Math.max(0, minX - pad),
      y: Math.max(0, minY - pad),
      w: Math.min(img.naturalWidth - Math.max(0, minX - pad), maxX - minX + 1 + pad * 2),
      h: Math.min(img.naturalHeight - Math.max(0, minY - pad), maxY - minY + 1 + pad * 2),
    }
  }

  drawCropCanvas()
  drawResultCanvas()
}

function resetCrop() {
  if (!processedImage.value) return
  cropRect.value = {
    x: 0, y: 0,
    w: processedImage.value.naturalWidth,
    h: processedImage.value.naturalHeight,
  }
  drawCropCanvas()
  drawResultCanvas()
}

function drawCropCanvas() {
  const canvas = cropCanvas.value
  const img = processedImage.value
  if (!canvas || !img) return

  // Scale to fit container (max 500px wide)
  const maxW = Math.min(500, canvas.parentElement?.clientWidth || 500)
  canvasScale = maxW / img.naturalWidth
  const dispH = img.naturalHeight * canvasScale

  canvas.width = maxW
  canvas.height = dispH
  canvas.style.width = maxW + 'px'
  canvas.style.height = dispH + 'px'

  const ctx = canvas.getContext('2d')!

  // Checkerboard background for transparency
  drawCheckerboard(ctx, maxW, dispH)

  // Draw image
  ctx.drawImage(img, 0, 0, maxW, dispH)

  // Dim outside crop area
  const r = cropRect.value
  const sx = r.x * canvasScale
  const sy = r.y * canvasScale
  const sw = r.w * canvasScale
  const sh = r.h * canvasScale

  ctx.fillStyle = 'rgba(0, 0, 0, 0.5)'
  ctx.fillRect(0, 0, maxW, sy) // top
  ctx.fillRect(0, sy + sh, maxW, dispH - sy - sh) // bottom
  ctx.fillRect(0, sy, sx, sh) // left
  ctx.fillRect(sx + sw, sy, maxW - sx - sw, sh) // right

  // Crop border
  ctx.strokeStyle = '#c9a84c'
  ctx.lineWidth = 2
  ctx.strokeRect(sx, sy, sw, sh)

  // Corner handles
  const handleSize = 8
  ctx.fillStyle = '#c9a84c'
  for (const [hx, hy] of [[sx, sy], [sx + sw, sy], [sx, sy + sh], [sx + sw, sy + sh]] as const) {
    ctx.fillRect(hx! - handleSize / 2, hy! - handleSize / 2, handleSize, handleSize)
  }
}

function drawResultCanvas() {
  const canvas = resultCanvas.value
  const img = processedImage.value
  if (!canvas || !img) return

  const r = cropRect.value
  const w = Math.max(1, Math.round(r.w))
  const h = Math.max(1, Math.round(r.h))

  // Scale result preview to max 200px
  const maxDim = 200
  const scale = Math.min(maxDim / w, maxDim / h, 1)
  canvas.width = Math.round(w * scale)
  canvas.height = Math.round(h * scale)

  const ctx = canvas.getContext('2d')!
  drawCheckerboard(ctx, canvas.width, canvas.height)
  ctx.drawImage(img, r.x, r.y, r.w, r.h, 0, 0, canvas.width, canvas.height)
}

function drawCheckerboard(ctx: CanvasRenderingContext2D, w: number, h: number) {
  const size = 8
  for (let y = 0; y < h; y += size) {
    for (let x = 0; x < w; x += size) {
      ctx.fillStyle = (Math.floor(x / size) + Math.floor(y / size)) % 2 === 0 ? '#2a2a3e' : '#1e1e30'
      ctx.fillRect(x, y, size, size)
    }
  }
}

// --- Crop Drag ---

function getCanvasPos(e: PointerEvent) {
  const canvas = cropCanvas.value!
  const rect = canvas.getBoundingClientRect()
  return {
    x: (e.clientX - rect.left) / canvasScale,
    y: (e.clientY - rect.top) / canvasScale,
  }
}

function startCropDrag(e: PointerEvent) {
  const pos = getCanvasPos(e)
  const r = cropRect.value
  const handleThreshold = 12 / canvasScale

  // Check corners
  if (Math.abs(pos.x - r.x) < handleThreshold && Math.abs(pos.y - r.y) < handleThreshold) {
    cropDragType.value = 'nw'
  } else if (Math.abs(pos.x - (r.x + r.w)) < handleThreshold && Math.abs(pos.y - r.y) < handleThreshold) {
    cropDragType.value = 'ne'
  } else if (Math.abs(pos.x - r.x) < handleThreshold && Math.abs(pos.y - (r.y + r.h)) < handleThreshold) {
    cropDragType.value = 'sw'
  } else if (Math.abs(pos.x - (r.x + r.w)) < handleThreshold && Math.abs(pos.y - (r.y + r.h)) < handleThreshold) {
    cropDragType.value = 'se'
  } else if (pos.x >= r.x && pos.x <= r.x + r.w && pos.y >= r.y && pos.y <= r.y + r.h) {
    cropDragType.value = 'move'
  } else {
    return
  }

  cropDragging.value = true
  cropDragStart.value = { x: pos.x, y: pos.y, rx: r.x, ry: r.y, rw: r.w, rh: r.h }
  cropCanvas.value?.setPointerCapture(e.pointerId)
}

function onCropDrag(e: PointerEvent) {
  if (!cropDragging.value || !processedImage.value) return
  const pos = getCanvasPos(e)
  const s = cropDragStart.value
  const dx = pos.x - s.x
  const dy = pos.y - s.y
  const imgW = processedImage.value.naturalWidth
  const imgH = processedImage.value.naturalHeight

  const r = { ...cropRect.value }

  switch (cropDragType.value) {
    case 'move':
      r.x = Math.max(0, Math.min(imgW - s.rw, s.rx + dx))
      r.y = Math.max(0, Math.min(imgH - s.rh, s.ry + dy))
      break
    case 'nw':
      r.x = Math.max(0, Math.min(s.rx + s.rw - 20, s.rx + dx))
      r.y = Math.max(0, Math.min(s.ry + s.rh - 20, s.ry + dy))
      r.w = s.rw - (r.x - s.rx)
      r.h = s.rh - (r.y - s.ry)
      break
    case 'ne':
      r.w = Math.max(20, Math.min(imgW - s.rx, s.rw + dx))
      r.y = Math.max(0, Math.min(s.ry + s.rh - 20, s.ry + dy))
      r.h = s.rh - (r.y - s.ry)
      break
    case 'sw':
      r.x = Math.max(0, Math.min(s.rx + s.rw - 20, s.rx + dx))
      r.w = s.rw - (r.x - s.rx)
      r.h = Math.max(20, Math.min(imgH - s.ry, s.rh + dy))
      break
    case 'se':
      r.w = Math.max(20, Math.min(imgW - s.rx, s.rw + dx))
      r.h = Math.max(20, Math.min(imgH - s.ry, s.rh + dy))
      break
  }

  cropRect.value = r
  drawCropCanvas()
  drawResultCanvas()
}

function endCropDrag() {
  cropDragging.value = false
  cropDragType.value = null
}

// Redraw when padding changes
watch(cropPadding, () => {
  if (step.value === 'crop' || step.value === 'done') {
    autoCrop()
  }
})

// --- Coin Search ---

function searchCoins() {
  if (searchTimeout) clearTimeout(searchTimeout)
  selectedCoinId.value = null
  if (!coinSearch.value.trim()) {
    coinOptions.value = []
    return
  }
  searchTimeout = setTimeout(async () => {
    coinsLoading.value = true
    try {
      const res = await getCoins({ search: coinSearch.value.trim(), limit: 20 })
      coinOptions.value = res.data.coins || []
    } catch {
      coinOptions.value = []
    } finally {
      coinsLoading.value = false
    }
  }, 300)
}

// --- Save / Download ---

function getResultBlob(): Promise<Blob> {
  return new Promise((resolve) => {
    const img = processedImage.value!
    const r = cropRect.value
    const canvas = document.createElement('canvas')
    canvas.width = Math.round(r.w)
    canvas.height = Math.round(r.h)
    const ctx = canvas.getContext('2d')!
    ctx.drawImage(img, r.x, r.y, r.w, r.h, 0, 0, canvas.width, canvas.height)
    canvas.toBlob((blob) => resolve(blob!), 'image/png')
  })
}

async function saveToExisting() {
  if (!selectedCoinId.value) return
  saving.value = true
  saveMsg.value = ''
  saveError.value = false
  try {
    const blob = await getResultBlob()
    const file = new File([blob], `${saveImageType.value}.png`, { type: 'image/png' })
    const isPrimary = saveImageType.value === 'obverse'
    await uploadImage(selectedCoinId.value, file, saveImageType.value, isPrimary)
    const coin = coinOptions.value.find(c => c.id === selectedCoinId.value)
    saveMsg.value = `Saved as ${saveImageType.value} to "${coin?.name || 'coin'}"!`
    emit('saved', selectedCoinId.value)
  } catch {
    saveMsg.value = 'Failed to save image'
    saveError.value = true
  } finally {
    saving.value = false
  }
}

async function saveToNewCoin() {
  if (!newCoinName.value.trim()) return
  saving.value = true
  saveMsg.value = ''
  saveError.value = false
  try {
    const res = await createCoin({ name: newCoinName.value.trim() })
    const coin = res.data
    const blob = await getResultBlob()
    const file = new File([blob], 'obverse.png', { type: 'image/png' })
    await uploadImage(coin.id, file, 'obverse', true)
    saveMsg.value = `Created "${coin.name}" with obverse image!`
    emit('saved', coin.id)
  } catch {
    saveMsg.value = 'Failed to create coin'
    saveError.value = true
  } finally {
    saving.value = false
  }
}

async function downloadResult() {
  const blob = await getResultBlob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `coin-${saveImageType.value}-processed.png`
  a.click()
  URL.revokeObjectURL(url)
}

function reset() {
  if (processedImage.value) URL.revokeObjectURL(processedImage.value.src)
  sourceImage.value = null
  processedBlob.value = null
  processedImage.value = null
  step.value = 'preview'
  inputError.value = ''
  saveMsg.value = ''
}

onUnmounted(() => {
  if (processedImage.value) URL.revokeObjectURL(processedImage.value.src)
})
</script>

<style scoped>
.image-processor {
  max-width: 700px;
  margin: 0 auto;
}

/* Steps */
.step-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.25rem;
}

.step-indicators {
  display: flex;
  gap: 0.5rem;
}

.step {
  font-size: 0.75rem;
  padding: 0.3rem 0.6rem;
  border-radius: var(--radius-full);
  color: var(--text-muted);
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
}

.step.active {
  color: var(--accent-gold);
  border-color: var(--accent-gold);
  background: var(--accent-gold-glow);
}

.step.done {
  color: var(--text-secondary);
  border-color: var(--accent-gold-dim);
}

/* Preview */
.preview-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}

.image-preview {
  position: relative;
  max-width: 500px;
  width: 100%;
  border-radius: var(--radius-md);
  overflow: hidden;
  border: 1px solid var(--border-subtle);
}

.image-preview img {
  width: 100%;
  display: block;
}

.image-preview.processing {
  position: relative;
}

.processing-img {
  opacity: 0.3;
  filter: blur(2px);
}

.processing-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  color: var(--accent-gold);
}

.processing-overlay p {
  font-size: 0.9rem;
}

.processing-hint {
  font-size: 0.75rem !important;
  color: var(--text-muted) !important;
}

/* Crop */
.crop-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.crop-workspace {
  display: flex;
  justify-content: center;
}

.crop-canvas {
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
  cursor: crosshair;
  max-width: 100%;
  touch-action: none;
}

.crop-controls {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.padding-control {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin-left: auto;
}

.padding-control input[type="range"] {
  width: 100px;
  accent-color: var(--accent-gold);
}

.padding-value {
  min-width: 32px;
  text-align: right;
}

/* Result */
.result-row {
  display: flex;
  gap: 1.5rem;
  padding: 1rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}

.result-preview {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}

.result-preview h4,
.save-controls h4 {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin-bottom: 0.25rem;
}

.result-canvas {
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-subtle);
}

.save-controls {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  min-width: 0;
}

.save-tabs {
  display: flex;
  gap: 2px;
  background: var(--bg-primary);
  border-radius: var(--radius-sm);
  padding: 2px;
}

.save-tab {
  flex: 1;
  padding: 0.4rem 0.5rem;
  font-size: 0.75rem;
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
}

.save-tab:hover {
  color: var(--text-secondary);
}

.save-tab.active {
  background: var(--bg-card);
  color: var(--accent-gold);
}

.save-panel {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.coin-search .form-input {
  width: 100%;
  font-size: 0.85rem;
}

.coin-list {
  max-height: 140px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 2px;
}

.coin-option {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.1rem;
  padding: 0.4rem 0.6rem;
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  text-align: left;
  transition: all var(--transition-fast);
}

.coin-option:hover {
  background: var(--accent-gold-glow);
}

.coin-option.selected {
  background: var(--accent-gold-glow);
  outline: 1px solid var(--accent-gold-dim);
}

.coin-option-name {
  font-size: 0.8rem;
  color: var(--text-primary);
}

.coin-option-meta {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.type-row {
  display: flex;
  gap: 1rem;
}

.radio-label {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
  cursor: pointer;
}

.radio-label input[type="radio"] {
  accent-color: var(--accent-gold);
}

.field-label {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.hint {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.msg {
  font-size: 0.85rem;
  color: var(--accent-gold);
}

.msg.error {
  color: #e74c3c;
}

@media (max-width: 640px) {
  .result-row {
    flex-direction: column;
  }

  .step-header {
    flex-direction: column;
    gap: 0.75rem;
    align-items: stretch;
  }

  .step-indicators {
    justify-content: center;
  }
}
</style>
