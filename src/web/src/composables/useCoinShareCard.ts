import { readonly, ref } from 'vue'
import type { Coin } from '@/types'
import { getShareCardFilename, getShareImageUrls, renderCoinShareCard } from '@/utils/coinShareCard'
import type { CoinShareCardContext } from '@/utils/coinShareCard'
import { useDialog } from '@/composables/useDialog'

const APP_NAME = 'Ed-Mar Ancient Coins'

export type CoinShareResultMode = 'shared' | 'downloaded'

export interface CoinShareResult {
  mode: CoinShareResultMode
}

export interface CoinShareCardOptions {
  context?: CoinShareCardContext
}

interface ShareNavigator {
  canShare?: (data?: ShareData) => boolean
  share?: (data: ShareData) => Promise<void>
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.rel = 'noopener'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}

export function useCoinShareCard() {
  const sharing = ref(false)
  const { showAlert } = useDialog()

  async function shareCoinCard(coin: Coin, options: CoinShareCardOptions = {}): Promise<CoinShareResult> {
    sharing.value = true
    try {
      const imageUrls = getShareImageUrls(coin)
      const blob = await renderCoinShareCard({
        coin,
        imageUrl: imageUrls[0] ?? null,
        imageUrls,
        appName: APP_NAME,
        context: options.context,
      })
      const filename = getShareCardFilename(coin)
      const file = new File([blob], filename, { type: 'image/png' })
      const contextSummary = options.context?.summary?.trim()
      const navigatorWithShare = navigator as unknown as ShareNavigator
      const shareData: ShareData = {
        files: [file],
        title: coin.name || 'Coin share card',
        text: contextSummary ? `${contextSummary}\n\nShared from ${APP_NAME}` : `Shared from ${APP_NAME}`,
      }

      if (navigatorWithShare.share && navigatorWithShare.canShare?.(shareData)) {
        await navigatorWithShare.share(shareData)
        return { mode: 'shared' }
      }

      downloadBlob(blob, filename)
      return { mode: 'downloaded' }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to generate share card.'
      await showAlert(message, { title: 'Share Failed' })
      throw error
    } finally {
      sharing.value = false
    }
  }

  return {
    sharing: readonly(sharing),
    shareCoinCard,
  }
}
