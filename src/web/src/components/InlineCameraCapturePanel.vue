<template>
  <div class="camera-first-card" :class="{ 'desktop-workspace': desktopWorkspace, immersive }">
    <div class="camera-container">
      <video
        ref="cameraVideo"
        class="camera-preview"
        v-show="cameraStream !== null"
        autoplay
        playsinline
        muted
        @loadedmetadata="onVideoMetadataLoaded"
      />
      <div v-if="!cameraStream" class="camera-placeholder">
        <template v-if="!immersive">
          <Camera :size="48" />
          <p>Start the camera when you're ready.</p>
          <button
            type="button"
            class="btn btn-secondary btn-sm camera-start-btn"
            @click="startCamera"
          >
            <Camera :size="16" />
            Start Camera
          </button>
        </template>
      </div>
      <div v-if="cameraError" class="camera-error-banner">{{ cameraError }}</div>

      <div v-if="cameraStream !== null && !immersive" class="focus-overlay">
        <div class="focus-mask"></div>
        <div class="focus-ring"></div>
        <p class="focus-instruction">{{ instruction }}</p>
      </div>
    </div>

    <slot name="before-actions"></slot>

    <div v-if="desktopWorkspace && !immersive" class="desktop-camera-actions">
      <button
        v-if="cameraStream === null"
        type="button"
        class="btn btn-primary"
        @click="startCamera"
      >
        <Camera :size="18" />
        Use Camera
      </button>
      <button
        v-else
        type="button"
        class="btn btn-primary"
        :disabled="!cameraReady"
        @click="captureFromCamera"
      >
        <Camera :size="18" />
        Take Photo
      </button>
      <button type="button" class="btn btn-secondary" @click="$emit('upload')">
        <Images :size="18" />
        Upload Image
      </button>
    </div>

    <div v-if="!immersive" class="camera-actions">
      <button
        type="button"
        class="shutter-btn"
        :disabled="!cameraReady"
        @click="captureFromCamera"
        aria-label="Capture photo"
      >
        <Camera :size="32" />
      </button>
      <button
        type="button"
        class="upload-icon-btn"
        @click="$emit('upload')"
        aria-label="Upload from library"
      >
        <Images :size="20" />
      </button>
    </div>

    <slot name="footer"></slot>

    <slot></slot>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref } from 'vue'
import { Camera, Images } from 'lucide-vue-next'
import type { CoinLookupImageRole } from '@/types'

const props = withDefaults(
  defineProps<{
    imageRole: CoinLookupImageRole
    filenamePrefix?: string
    instruction?: string
    desktopWorkspace?: boolean
    /**
     * Render as a bare viewfinder that fills its parent: no card chrome, no
     * placeholder copy, no built-in shutter or ring. PwaCaptureShell supplies
     * all of those and drives capture through the exposed methods below.
     */
    immersive?: boolean
  }>(),
  {
    filenamePrefix: 'capture',
    instruction: 'Focus one coin in the circle',
    desktopWorkspace: false,
    immersive: false,
  }
)

const emit = defineEmits<{
  captured: [file: File, role: CoinLookupImageRole]
  upload: []
}>()

const cameraVideo = ref<HTMLVideoElement | null>(null)
const cameraStream = ref<MediaStream | null>(null)
const cameraError = ref('')
const videoReady = ref(false)
const starting = ref(false)
const capturing = ref(false)
const cameraReady = computed(() => cameraStream.value !== null && videoReady.value && !capturing.value)
let generation = 0
let disposed = false

async function startCamera() {
  if (disposed || starting.value || cameraStream.value) return
  if (!navigator.mediaDevices?.getUserMedia) {
    cameraError.value = 'Camera access is unavailable on this device.'
    return
  }

  const requestGeneration = generation
  starting.value = true
  try {
    const stream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: { ideal: 'environment' } },
      audio: false,
    })
    if (requestGeneration !== generation) {
      for (const track of stream.getTracks()) track.stop()
      return
    }
    cameraStream.value = stream
    cameraError.value = ''
    videoReady.value = false

    await nextTick()

    if (requestGeneration !== generation) return
    if (cameraVideo.value) {
      cameraVideo.value.srcObject = stream
      await cameraVideo.value.play()
    }
  } catch (error) {
    if (requestGeneration !== generation) return
    stopCamera()
    const err = error as { name?: string }
    if (err.name === 'NotAllowedError') {
      cameraError.value = 'Camera permission was denied. You can still upload images.'
    } else if (err.name === 'NotFoundError') {
      cameraError.value = 'No camera found on this device.'
    } else {
      cameraError.value = 'Camera is unavailable. You can still upload images.'
    }
  } finally {
    if (requestGeneration === generation) starting.value = false
  }
}

function onVideoMetadataLoaded() {
  const video = cameraVideo.value
  if (cameraStream.value && video && video.videoWidth > 0 && video.videoHeight > 0) {
    videoReady.value = true
  }
}

function stopCamera() {
  generation += 1
  starting.value = false
  capturing.value = false
  for (const track of cameraStream.value?.getTracks() ?? []) {
    track.stop()
  }
  cameraStream.value = null
  if (cameraVideo.value) cameraVideo.value.srcObject = null
  videoReady.value = false
}

function computeCoverCropRect(
  videoWidth: number,
  videoHeight: number,
  displayWidth: number,
  displayHeight: number
): { sx: number; sy: number; sw: number; sh: number } {
  const videoAspect = videoWidth / (videoHeight || 1)
  const displayAspect = displayWidth / (displayHeight || 1)

  if (videoAspect > displayAspect) {
    const sh = videoHeight
    const sw = sh * displayAspect
    return { sx: (videoWidth - sw) / 2, sy: 0, sw, sh }
  }

  const sw = videoWidth
  const sh = sw / displayAspect
  return { sx: 0, sy: (videoHeight - sh) / 2, sw, sh }
}

async function captureFromCamera() {
  if (disposed || capturing.value) return
  const video = cameraVideo.value
  if (!video || !cameraReady.value || video.videoWidth === 0 || video.videoHeight === 0) {
    cameraError.value = 'Camera is not ready yet. Try again in a moment.'
    return
  }

  const displayWidth = video.clientWidth ?? 0
  const displayHeight = video.clientHeight ?? 0
  if (displayWidth === 0 || displayHeight === 0) {
    cameraError.value = 'Could not determine video display size.'
    return
  }

  const { sx, sy, sw, sh } = computeCoverCropRect(
    video.videoWidth,
    video.videoHeight,
    displayWidth,
    displayHeight
  )

  const canvas = document.createElement('canvas')
  canvas.width = sw
  canvas.height = sh
  const context = canvas.getContext('2d')
  if (!context) {
    cameraError.value = 'Could not prepare image capture. You can still upload images.'
    return
  }

  const captureGeneration = generation
  const role = props.imageRole
  const filename = `${props.filenamePrefix}-${Date.now()}.jpg`
  capturing.value = true
  try {
    context.drawImage(video, sx, sy, sw, sh, 0, 0, sw, sh)
    const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.92))
    if (captureGeneration !== generation) return
    if (!blob) {
      cameraError.value = 'Could not capture image from camera.'
      return
    }
    cameraError.value = ''
    emit('captured', new File([blob], filename, { type: 'image/jpeg' }), role)
  } catch {
    if (captureGeneration === generation) {
      cameraError.value = 'Could not capture image from camera. You can still upload images.'
    }
  } finally {
    if (captureGeneration === generation) capturing.value = false
  }
}

onBeforeUnmount(() => {
  disposed = true
  stopCamera()
})

defineExpose({
  startCamera,
  stopCamera,
  captureFromCamera,
  cameraReady,
  cameraActive: computed(() => cameraStream.value !== null),
})
</script>

<style scoped>
.camera-first-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.camera-container {
  position: relative;
  width: 100%;
  height: clamp(240px, 48vh, 420px);
  border-radius: var(--radius-sm);
  overflow: hidden;
  background: var(--bg-primary);
  border: 1px solid var(--border-subtle);
}

.camera-preview {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.camera-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  color: var(--text-muted);
}

.camera-placeholder p {
  margin: 0;
}

.camera-start-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.camera-error-banner {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: var(--error-bg);
  color: var(--text-primary);
  padding: 0.5rem 0.75rem;
  font-size: 0.8rem;
  text-align: center;
  z-index: 10;
}

.focus-overlay {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 5;
}

.focus-mask {
  position: absolute;
  inset: 0;
  background: radial-gradient(
    circle at 50% 52%,
    transparent 0%,
    transparent 36%,
    rgba(10, 12, 20, 0.2) 37%,
    rgba(10, 12, 20, 0.62) 100%
  );
}

.focus-ring {
  position: absolute;
  top: 52%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 74%;
  max-width: 360px;
  aspect-ratio: 1;
  border-radius: var(--radius-full);
  border: 2px solid var(--border-white-dim);
}

.focus-instruction {
  position: absolute;
  top: calc(env(safe-area-inset-top) + 20px);
  left: 50%;
  transform: translateX(-50%);
  color: var(--text-primary);
  font-size: 0.85rem;
  font-weight: 500;
  text-align: center;
  text-shadow: 0 2px 8px var(--overlay-dark);
  margin: 0;
}

.camera-actions {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.5rem;
  align-items: center;
}

.desktop-camera-actions {
  display: none;
}

.shutter-btn {
  grid-column: 2;
  justify-self: center;
  width: 3.5rem;
  height: 3.5rem;
  border-radius: var(--radius-full);
  background: linear-gradient(135deg, var(--accent-gold), var(--accent-bronze));
  border: 2px solid var(--border-white-dim);
  color: var(--bg-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-fast);
  box-shadow: var(--shadow-card);
}

@media (max-height: 700px) {
  .camera-first-card:not(.immersive) .camera-container {
    height: 40vh;
  }
}

.shutter-btn:hover:not(:disabled) {
  transform: scale(1.05);
  box-shadow: var(--shadow-glow);
}

.shutter-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.upload-icon-btn {
  grid-column: 3;
  justify-self: end;
  width: 2.5rem;
  height: 2.5rem;
  border-radius: var(--radius-full);
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.upload-icon-btn:hover {
  background: var(--bg-card-hover);
  border-color: var(--accent-gold);
  color: var(--accent-gold);
}

.camera-first-card.immersive {
  height: 100%;
  padding: 0;
  gap: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
}

.camera-first-card.immersive .camera-container {
  flex: 1;
  height: auto;
  min-height: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
}

@media (min-width: 769px) {
  .camera-first-card.desktop-workspace {
    padding: 0;
    border: 0;
    border-radius: 0;
    background: transparent;
  }

  .desktop-workspace .camera-container {
    height: min(48vh, 390px);
  }

  .desktop-workspace .camera-placeholder .camera-start-btn,
  .desktop-workspace .camera-actions {
    display: none;
  }

  .desktop-workspace .desktop-camera-actions {
    display: flex;
    justify-content: center;
    gap: 0.75rem;
  }
}
</style>
