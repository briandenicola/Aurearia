<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-[1000] flex items-center justify-center bg-black/85 p-4 backdrop-blur-sm" @click.self="close" @keydown.esc="close">
      <div class="flex max-h-[90vh] w-full max-w-[900px] flex-col overflow-hidden rounded-md border border-border-accent bg-card shadow-glow max-md:max-h-screen max-md:max-w-full max-md:rounded-none" role="dialog" :aria-label="`${imageType} image viewer`">
        <div class="flex items-center justify-between border-b border-border-subtle bg-card px-5 py-4">
          <h2 class="m-0 font-display text-lg text-heading">{{ formatImageType(imageType) }}</h2>
          <button class="flex items-center justify-center rounded-sm p-1 text-text-muted transition-colors hover:bg-white/5 hover:text-text-primary" @click="close" title="Close (Esc)" aria-label="Close">
            <X :size="20" />
          </button>
        </div>

        <div class="flex flex-1 items-center justify-center overflow-auto bg-input p-6 max-md:p-4">
          <div class="relative flex max-h-full max-w-full items-center justify-center">
            <img
              v-if="processedImageUrl"
              :src="processedImageUrl"
              :alt="formatImageType(imageType)"
              class="max-h-full max-w-full rounded-sm object-contain shadow-[0_4px_12px_rgba(0,0,0,0.3)]"
              :class="{ 'opacity-30 blur-[2px]': processing }"
            />
            <AuthenticatedImage
              v-else
              :media-path="imagePath"
              :alt="formatImageType(imageType)"
              class="max-h-full max-w-full rounded-sm object-contain shadow-[0_4px_12px_rgba(0,0,0,0.3)]"
              :class="{ 'opacity-30 blur-[2px]': processing }"
            />

            <div v-if="processing" class="pointer-events-none absolute inset-0 flex flex-col items-center justify-center gap-4 text-center text-gold">
              <div class="h-10 w-10 animate-spin rounded-full border-[3px] border-border-subtle border-t-gold"></div>
              <p class="m-0 text-base">Removing background...</p>
              <p class="m-0 text-chip text-text-muted">This may take 30-60 seconds...</p>
            </div>
          </div>
        </div>

        <div class="flex justify-end gap-2 border-t border-border-subtle bg-card px-5 py-4 max-md:flex-wrap">
          <button
            v-if="!processedImageUrl"
            class="btn btn-primary btn-sm inline-flex items-center gap-1.5 max-md:min-w-0 max-md:flex-1"
            :disabled="processing"
            @click="handleRemoveBackground"
          >
            <Eraser :size="16" />
            Remove Background
          </button>

          <template v-else>
            <button
              class="btn btn-secondary btn-sm inline-flex items-center gap-1.5 max-md:min-w-0 max-md:flex-1"
              @click="resetToOriginal"
            >
              <RotateCcw :size="16" />
              Reset
            </button>
            <button
              class="btn btn-primary btn-sm inline-flex items-center gap-1.5 max-md:min-w-0 max-md:flex-1"
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
