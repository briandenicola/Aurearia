import { onBeforeUnmount, ref, watch, type Ref } from 'vue'
import { proxyImage } from '@/api/client'

export function useProxiedImage(sourceUrl: Ref<string | null | undefined>) {
  const proxiedImageUrl = ref('')
  let activeObjectUrl = ''
  let loadVersion = 0

  function releaseObjectUrl() {
    if (!activeObjectUrl) return
    URL.revokeObjectURL(activeObjectUrl)
    activeObjectUrl = ''
  }

  async function load(url: string | null | undefined) {
    const requestVersion = ++loadVersion
    proxiedImageUrl.value = ''
    releaseObjectUrl()

    if (!url) return

    try {
      const res = await proxyImage(url)
      const blob = res.data
      if (!(blob instanceof Blob) || blob.size === 0) return

      const objectUrl = URL.createObjectURL(blob)
      if (requestVersion !== loadVersion) {
        URL.revokeObjectURL(objectUrl)
        return
      }
      activeObjectUrl = objectUrl
      proxiedImageUrl.value = objectUrl
    } catch (err) {
      console.warn('Failed to proxy image', err)
      proxiedImageUrl.value = ''
    }
  }

  watch(sourceUrl, (url) => {
    void load(url)
  }, { immediate: true })

  onBeforeUnmount(() => {
    loadVersion++
    releaseObjectUrl()
  })

  return { proxiedImageUrl }
}
