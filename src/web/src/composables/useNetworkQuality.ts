import { ref, onMounted, onUnmounted } from 'vue'

export type ImageSize = 'thumb' | 'medium' | 'full'

interface NetworkInformationLike extends EventTarget {
  effectiveType?: string
}

function imageSizeFromConnection(conn: NetworkInformationLike): ImageSize {
  const et = conn.effectiveType
  if (et === 'slow-2g' || et === '2g') return 'thumb'
  if (et === '3g') return 'medium'
  return 'full'
}

/**
 * Composable that tracks the current network quality and maps it to an image
 * size variant.  Falls back to 'full' when the Network Information API is
 * unavailable (e.g. Firefox, Safari, server-side rendering).
 *
 * Returned imageSize is reactive and updates automatically if the connection
 * type changes while the component is mounted.
 */
export function useNetworkQuality() {
  const imageSize = ref<ImageSize>('full')
  let conn: NetworkInformationLike | null = null

  function onConnectionChange() {
    if (conn) {
      imageSize.value = imageSizeFromConnection(conn)
    }
  }

  onMounted(() => {
    const nav = navigator as Navigator & {
      connection?: NetworkInformationLike
      mozConnection?: NetworkInformationLike
      webkitConnection?: NetworkInformationLike
    }
    conn = nav.connection ?? nav.mozConnection ?? nav.webkitConnection ?? null
    if (conn) {
      imageSize.value = imageSizeFromConnection(conn)
      conn.addEventListener('change', onConnectionChange)
    }
  })

  onUnmounted(() => {
    conn?.removeEventListener('change', onConnectionChange)
  })

  return { imageSize }
}
