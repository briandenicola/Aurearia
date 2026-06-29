<template>
  <Teleport to="body">
    <div class="lightbox-overlay" @click.self="close" @keydown.esc="close">
      <div class="lightbox-content" role="dialog" :aria-label="`${imageType} image viewer`">
        <div class="lightbox-header">
          <h2 class="lightbox-title">{{ formatImageType(imageType) }}</h2>
          <button class="lightbox-close" @click="close" title="Close (Esc)" aria-label="Close">
            <X :size="20" />
          </button>
        </div>

        <div class="lightbox-body">
          <div class="lightbox-image-container" :class="{ processing }">
            <img
              v-if="processedImageUrl"
              :src="processedImageUrl"
              :alt="formatImageType(imageType)"
              class="lightbox-image"
            />
            <AuthenticatedImage
              v-else
              :media-path="imagePath"
              :alt="formatImageType(imageType)"
              class="lightbox-image"
            />

            <div v-if="processing" class="lightbox-processing-overlay">
              <div class="spinner"></div>
              <p>Removing background...</p>
              <p class="processing-hint">This may take 30-60 seconds...</p>
            </div>
          </div>
        </div>

        <div class="lightbox-actions">
          <button
            v-if="!processedImageUrl"
            class="btn btn-primary btn-sm"
            :disabled="processing"
            @click="handleRemoveBackground"
          >
            <Eraser :size="16" />
            Remove Background
          </button>

          <template v-else>
            <button
              class="btn btn-secondary btn-sm"
              @click="resetToOriginal"
            >
              <RotateCcw :size="16" />
              Reset
            </button>
            <button
              class="btn btn-primary btn-sm"
              :disabled="saving"
              @click="saveProcessedImage"
            >
              <Save :size="16" />
              {{ saving ? 'Saving...' : 'Save' }}
            </button>
          </template>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { X, Eraser, RotateCcw, Save } from 'lucide-vue-next'
import { uploadImage, deleteImage } from '@/api/client'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'
import { removeCoinBackground } from '@/utils/backgroundRemoval'
import { fetchPrivateMediaBlob } from '@/utils/media'

const props = defineProps<{
  coinId: number
  imageId: number
  imagePath: string
  imageType: string
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const processing = ref(false)
const processedImageUrl = ref<string | null>(null)
const saving = ref(false)

function formatImageType(type: string) {
  if (!type) return 'Image'
  return type.charAt(0).toUpperCase() + type.slice(1)
}

async function handleRemoveBackground() {
  processing.value = true

  try {
    const srcBlob = await fetchPrivateMediaBlob(props.imagePath)
    const result = await removeCoinBackground(srcBlob)
    processedImageUrl.value = URL.createObjectURL(result)
  } catch (err) {
    console.error('Background removal failed:', err)
    alert('Background removal failed. Please try again.')
  } finally {
    processing.value = false
  }
}

function resetToOriginal() {
  if (processedImageUrl.value) {
    URL.revokeObjectURL(processedImageUrl.value)
  }
  processedImageUrl.value = null
}

async function saveProcessedImage() {
  if (!processedImageUrl.value) return

  saving.value = true

  try {
    const response = await fetch(processedImageUrl.value)
    const blob = await response.blob()
    const file = new File([blob], `${props.imageType}.png`, { type: 'image/png' })
    const isPrimary = props.imageType === 'obverse'
    await uploadImage(props.coinId, file, props.imageType, isPrimary)
    // Replace semantics: the upload appends a new record, so remove the
    // original image of this type to avoid duplicate obverse/reverse entries.
    await deleteImage(props.coinId, props.imageId)
    emit('saved')
    close()
  } catch (err) {
    console.error('Save failed:', err)
    alert('Failed to save image. Please try again.')
  } finally {
    saving.value = false
  }
}

function close() {
  emit('close')
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    close()
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
  document.body.style.overflow = 'hidden'
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
  document.body.style.overflow = ''
  if (processedImageUrl.value) {
    URL.revokeObjectURL(processedImageUrl.value)
  }
})
</script>

<style scoped>
.lightbox-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
  backdrop-filter: blur(4px);
}

.lightbox-content {
  width: 100%;
  max-width: 900px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-accent);
  box-shadow: var(--shadow-glow);
  overflow: hidden;
}

.lightbox-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-card);
}

.lightbox-title {
  margin: 0;
  font-family: 'Cinzel', serif;
  font-size: 1.2rem;
  color: var(--text-heading);
}

.lightbox-close {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: var(--radius-sm);
  transition: color var(--transition-fast), background var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
}

.lightbox-close:hover {
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.05);
}

.lightbox-body {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  overflow: auto;
  background: var(--bg-input);
}

.lightbox-image-container {
  position: relative;
  max-width: 100%;
  max-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.lightbox-image-container.processing .lightbox-image {
  opacity: 0.3;
  filter: blur(2px);
}

.lightbox-image {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: var(--radius-sm);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.lightbox-processing-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  color: var(--accent-gold);
  text-align: center;
  pointer-events: none;
}

.lightbox-processing-overlay p {
  margin: 0;
  font-size: 0.9rem;
}

.processing-hint {
  font-size: 0.8rem !important;
  color: var(--text-muted);
}

.spinner {
  border: 3px solid var(--border-subtle);
  border-top-color: var(--accent-gold);
  border-radius: 50%;
  width: 40px;
  height: 40px;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.lightbox-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  padding: 1rem 1.25rem;
  border-top: 1px solid var(--border-subtle);
  background: var(--bg-card);
}

.lightbox-actions .btn {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

@media (max-width: 768px) {
  .lightbox-content {
    max-width: 100%;
    max-height: 100vh;
    border-radius: 0;
  }

  .lightbox-body {
    padding: 1rem;
  }

  .lightbox-actions {
    flex-wrap: wrap;
  }

  .lightbox-actions .btn {
    flex: 1;
    min-width: 0;
  }
}
</style>
