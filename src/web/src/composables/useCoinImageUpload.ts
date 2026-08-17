import { type MaybeRefOrGetter, ref, toValue } from 'vue'
import { proxyImage, uploadImage } from '@/api/client'

/**
 * Image upload seam extracted from `CoinActionsPanel.vue` (F6 god-component
 * cleanup). Owns the three ways a coin image can arrive - file picker,
 * pasted URL, and in-app camera capture - plus the shared status/error
 * message the three paths report through. Kept as one composable rather
 * than three because all three write the same `uploadStatus`/`uploadError`
 * pair and are only ever used together on the same panel.
 */
export function useCoinImageUpload(
  coinId: MaybeRefOrGetter<number>,
  imageCount: MaybeRefOrGetter<number>,
  options: { onUploaded?: () => void } = {},
) {
  const uploadType = ref('obverse')
  const uploadStatus = ref('')
  const uploadError = ref(false)
  const imageUrl = ref('')
  const urlLoading = ref(false)

  function isFirstImage(): boolean {
    return toValue(imageCount) === 0
  }

  async function handleImageUpload(e: Event) {
    const file = (e.target as HTMLInputElement).files?.[0]
    if (!file) return

    uploadStatus.value = 'Uploading...'
    uploadError.value = false

    try {
      await uploadImage(toValue(coinId), file, uploadType.value, isFirstImage())
      uploadStatus.value = 'Upload complete!'
      options.onUploaded?.()
    } catch {
      uploadStatus.value = 'Upload failed'
      uploadError.value = true
    }
  }

  async function handleCameraCaptured(file: File) {
    uploadStatus.value = 'Uploading...'
    uploadError.value = false

    try {
      // Pass circleClip=true for obverse/reverse, false for other types
      const shouldCircleClip = uploadType.value === 'obverse' || uploadType.value === 'reverse'
      await uploadImage(toValue(coinId), file, uploadType.value, isFirstImage(), shouldCircleClip)
      uploadStatus.value = 'Upload complete!'
      options.onUploaded?.()
    } catch {
      uploadStatus.value = 'Upload failed'
      uploadError.value = true
    }
  }

  async function handleUrlUpload() {
    if (!imageUrl.value) return

    urlLoading.value = true
    uploadStatus.value = 'Fetching image...'
    uploadError.value = false

    try {
      const imgRes = await proxyImage(imageUrl.value)
      const blob = imgRes.data as Blob
      if (blob.size === 0) {
        uploadStatus.value = 'No image data received from URL'
        uploadError.value = true
        return
      }
      const ext = blob.type.includes('png') ? '.png' : '.jpg'
      const file = new File([blob], `${uploadType.value}${ext}`, { type: blob.type || 'image/jpeg' })
      await uploadImage(toValue(coinId), file, uploadType.value, isFirstImage())
      uploadStatus.value = 'Image saved from URL!'
      imageUrl.value = ''
      options.onUploaded?.()
    } catch {
      uploadStatus.value = 'Failed to fetch image from URL'
      uploadError.value = true
    } finally {
      urlLoading.value = false
    }
  }

  return {
    uploadType,
    uploadStatus,
    uploadError,
    imageUrl,
    urlLoading,
    handleImageUpload,
    handleCameraCaptured,
    handleUrlUpload,
  }
}

export type CoinImageUpload = ReturnType<typeof useCoinImageUpload>
