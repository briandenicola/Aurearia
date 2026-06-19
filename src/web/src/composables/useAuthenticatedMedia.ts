import { onBeforeUnmount, ref, watch, type Ref } from 'vue'
import { isPrivateUploadPath, privateMediaObjectUrl } from '@/utils/media'

export function useAuthenticatedMedia(mediaPath: Ref<string | null | undefined>) {
  const objectUrl = ref('')
  let activeObjectUrl: string | null = null
  let loadId = 0

  function revokeActiveObjectUrl() {
    if (activeObjectUrl) {
      URL.revokeObjectURL(activeObjectUrl)
      activeObjectUrl = null
    }
  }

  watch(mediaPath, async (path) => {
    const currentLoad = ++loadId
    revokeActiveObjectUrl()
    objectUrl.value = ''

    if (!path) return

    if (!isPrivateUploadPath(path)) {
      objectUrl.value = path
      return
    }

    try {
      const nextUrl = await privateMediaObjectUrl(path)
      if (currentLoad !== loadId) {
        if (nextUrl.startsWith('blob:')) URL.revokeObjectURL(nextUrl)
        return
      }
      activeObjectUrl = nextUrl.startsWith('blob:') ? nextUrl : null
      objectUrl.value = nextUrl
    } catch {
      objectUrl.value = ''
    }
  }, { immediate: true })

  onBeforeUnmount(() => {
    loadId++
    revokeActiveObjectUrl()
  })

  return { objectUrl }
}
