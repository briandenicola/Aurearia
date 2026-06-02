<template>
  <Teleport to="body">
    <div v-if="isOpen" class="camera-modal-overlay" @click="handleBackdropClick">
      <div class="camera-modal" @click.stop>
        <div class="camera-modal-header">
          <h3>Capture Photo</h3>
          <button type="button" class="close-btn" @click="handleClose" aria-label="Close">
            <X :size="20" />
          </button>
        </div>
        
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
            <p>Camera starting...</p>
          </div>
          <div v-if="cameraError" class="camera-error-banner">{{ cameraError }}</div>
          
          <!-- Circular focus-guide overlay (only when camera active) -->
          <div v-if="cameraStream !== null" class="focus-overlay">
            <div class="focus-mask"></div>
            <div class="focus-ring"></div>
            <p class="focus-instruction">{{ instruction }}</p>
          </div>
        </div>
        
        <div class="camera-modal-actions">
          <button
            type="button"
            class="btn btn-secondary"
            @click="handleClose"
          >
            Cancel
          </button>
          <button
            type="button"
            class="btn btn-primary capture-btn"
            :disabled="!cameraReady"
            @click="handleCapture"
          >
            <Camera :size="20" />
            Capture
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, nextTick, computed } from 'vue'
import { Camera, X } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    isOpen: boolean
    instruction?: string
  }>(),
  {
    instruction: 'Focus one coin in the circle',
  }
)

const emit = defineEmits<{
  close: []
  captured: [file: File]
}>()

const cameraVideo = ref<HTMLVideoElement | null>(null)
const cameraStream = ref<MediaStream | null>(null)
const cameraError = ref('')
const videoReady = ref(false)

const cameraReady = computed(() => {
  return cameraStream.value !== null && videoReady.value && !cameraError.value
})

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
    
    // Wait for DOM to update before assigning srcObject
    await nextTick()
    
    if (cameraVideo.value) {
      cameraVideo.value.srcObject = stream
      await cameraVideo.value.play()
    }
  } catch (error) {
    const err = error as { name?: string }
    if (err.name === 'NotAllowedError') {
      cameraError.value = 'Camera permission was denied.'
    } else if (err.name === 'NotFoundError') {
      cameraError.value = 'No camera found on this device.'
    } else {
      cameraError.value = 'Camera is unavailable.'
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

/**
 * Compute the source rectangle from a raw video frame that corresponds
 * to the displayed portion when using object-fit: cover.
 * Returns { sx, sy, sw, sh } in native video coordinates.
 */
function computeCoverCropRect(
  videoWidth: number,
  videoHeight: number,
  displayWidth: number,
  displayHeight: number
): { sx: number; sy: number; sw: number; sh: number } {
  const videoAspect = videoWidth / (videoHeight || 1)
  const displayAspect = displayWidth / (displayHeight || 1)

  let sx: number, sy: number, sw: number, sh: number

  if (videoAspect > displayAspect) {
    // Video is wider than display → crop horizontally
    sh = videoHeight
    sw = sh * displayAspect
    sy = 0
    sx = (videoWidth - sw) / 2
  } else {
    // Video is taller than display → crop vertically
    sw = videoWidth
    sh = sw / displayAspect
    sx = 0
    sy = (videoHeight - sh) / 2
  }

  return { sx, sy, sw, sh }
}

async function handleCapture() {
  const video = cameraVideo.value
  if (!video || !cameraReady.value || video.videoWidth === 0 || video.videoHeight === 0) {
    cameraError.value = 'Camera is not ready yet. Try again in a moment.'
    return
  }

  // Get displayed video box dimensions (object-fit: cover uses these)
  const displayWidth = video.clientWidth ?? 0
  const displayHeight = video.clientHeight ?? 0
  
  if (displayWidth === 0 || displayHeight === 0) {
    cameraError.value = 'Could not determine video display size.'
    return
  }

  // Compute the cover-crop region that matches what the user sees on screen
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

  // Draw only the displayed region (cover-cropped) to the canvas
  context.drawImage(video, sx, sy, sw, sh, 0, 0, sw, sh)
  
  const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.92))
  if (!blob) {
    cameraError.value = 'Could not capture image from camera.'
    return
  }
  const file = new File([blob], `capture-${Date.now()}.jpg`, { type: 'image/jpeg' })
  
  // Stop camera and emit captured file
  stopCamera()
  emit('captured', file)
  emit('close')
}

function handleClose() {
  stopCamera()
  emit('close')
}

function handleBackdropClick() {
  handleClose()
}

// Watch for modal open state
watch(() => props.isOpen, (newVal) => {
  if (newVal) {
    startCamera()
  } else {
    stopCamera()
  }
})

// Start camera on mount if already open
onMounted(() => {
  if (props.isOpen) {
    startCamera()
  }
})

// Always stop camera on unmount
onUnmounted(() => {
  stopCamera()
})
</script>

<style scoped>
.camera-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.camera-modal {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  width: 100%;
  max-width: 600px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.camera-modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  border-bottom: 1px solid var(--border-subtle);
}

.camera-modal-header h3 {
  margin: 0;
  font-size: 1.2rem;
  color: var(--text-heading);
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.25rem;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color var(--transition-fast);
}

.close-btn:hover {
  color: var(--text-primary);
}

.camera-container {
  position: relative;
  width: 100%;
  aspect-ratio: 4 / 3;
  background: #000;
  overflow: hidden;
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

.camera-error-banner {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: rgba(231, 76, 60, 0.9);
  color: #fff;
  padding: 0.5rem 0.75rem;
  font-size: 0.8rem;
  text-align: center;
  z-index: 10;
}

/* Circular focus-guide overlay */
.focus-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
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
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, 0.55);
}

.focus-instruction {
  position: absolute;
  top: calc(env(safe-area-inset-top) + 20px);
  left: 50%;
  transform: translateX(-50%);
  color: #fff;
  font-size: 0.85rem;
  font-weight: 500;
  text-align: center;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.7);
  margin: 0;
  padding: 0 1rem;
}

.camera-modal-actions {
  display: flex;
  gap: 0.5rem;
  padding: 1rem;
  border-top: 1px solid var(--border-subtle);
  justify-content: flex-end;
}

.capture-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}
</style>
