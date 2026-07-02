<template>
  <div class="camera-first-card">
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
      </div>
      <div v-if="cameraError" class="camera-error-banner">{{ cameraError }}</div>

      <div v-if="cameraStream !== null" class="focus-overlay">
        <div class="focus-mask"></div>
        <div class="focus-ring"></div>
        <p class="focus-instruction">{{ instruction }}</p>
      </div>
    </div>

    <slot name="before-actions"></slot>

    <div class="camera-actions">
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

const props = withDefaults(
  defineProps<{
    filenamePrefix?: string
    instruction?: string
  }>(),
  {
    filenamePrefix: 'capture',
    instruction: 'Focus one coin in the circle',
  }
)

const emit = defineEmits<{
  captured: [file: File]
  upload: []
}>()

const cameraVideo = ref<HTMLVideoElement | null>(null)
const cameraStream = ref<MediaStream | null>(null)
const cameraError = ref('')
const videoReady = ref(false)
const cameraReady = computed(() => cameraStream.value !== null && videoReady.value)

async function startCamera() {
  if (cameraStream.value) return
  if (!navigator.mediaDevices?.getUserMedia) {
    cameraError.value = 'Camera access is unavailable on this device.'
    return
  }

  try {
    const stream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: { ideal: 'environment' } },
      audio: false,
    })
    cameraStream.value = stream
    cameraError.value = ''
    videoReady.value = false

    await nextTick()

    if (cameraVideo.value) {
      cameraVideo.value.srcObject = stream
      await cameraVideo.value.play()
    }
  } catch (error) {
    const err = error as { name?: string }
    if (err.name === 'NotAllowedError') {
      cameraError.value = 'Camera permission was denied. You can still upload images.'
    } else if (err.name === 'NotFoundError') {
      cameraError.value = 'No camera found on this device.'
    } else {
      cameraError.value = 'Camera is unavailable. You can still upload images.'
    }
  }
}

function onVideoMetadataLoaded() {
  const video = cameraVideo.value
  if (video && video.videoWidth > 0 && video.videoHeight > 0) {
    videoReady.value = true
  }
}

function stopCamera() {
  if (!cameraStream.value) return
  for (const track of cameraStream.value.getTracks()) {
    track.stop()
  }
  cameraStream.value = null
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
  if (!context) return

  context.drawImage(video, sx, sy, sw, sh, 0, 0, sw, sh)

  const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.92))
  if (!blob) {
    cameraError.value = 'Could not capture image from camera.'
    return
  }

  emit('captured', new File([blob], `${props.filenamePrefix}-${Date.now()}.jpg`, { type: 'image/jpeg' }))
}

onBeforeUnmount(() => {
  stopCamera()
})

defineExpose({
  stopCamera,
})
</script>

<style scoped>
.camera-first-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.camera-container {
  position: relative;
  width: 100%;
  aspect-ratio: 4 / 3;
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

.shutter-btn {
  grid-column: 2;
  justify-self: center;
  width: 4rem;
  height: 4rem;
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
</style>
