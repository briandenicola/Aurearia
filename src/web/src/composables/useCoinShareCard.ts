import { readonly, ref } from 'vue'
import type { Coin } from '@/types'
import { getShareCardFilename, getShareImageUrls, renderCoinShareCard } from '@/utils/coinShareCard'
import type { CoinShareCardContext } from '@/utils/coinShareCard'
import { useDialog } from '@/composables/useDialog'
import { getAttributionCards } from '@/api/client'
import { fetchPrivateMediaBlob } from '@/utils/media'

const APP_NAME = 'Aurearia - Coin Collection'

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
  const { showAlert, showConfirm } = useDialog()

  async function shareCoinCard(coin: Coin, options: CoinShareCardOptions = {}): Promise<CoinShareResult> {
    sharing.value = true
    try {
      const { data: cards } = await getAttributionCards(coin.id)
      const includeOriginals = cards.length > 0 && await showConfirm(
        'Include the saved original attribution cards? They may contain dealer, price or handwritten notes.',
        { title: 'Include original attribution cards?', confirmLabel: 'Include originals', cancelLabel: 'Generated card only' },
      )
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
      const files = [file]
      if (includeOriginals) {
        for (const card of cards) {
          const original = await fetchPrivateMediaBlob(card.filePath)
          const extension = card.filePath.match(/\.(jpe?g|png|gif|webp)$/i)?.[1]
          if (!extension) throw new Error('Unsupported original attribution card format.')
          const name = `${filename.replace(/-share-card\.png$/, '')}-original-attribution-${card.id}.${extension.toLowerCase()}`
          files.push(new File([original], name, { type: original.type }))
        }
      }
      const contextSummary = options.context?.summary?.trim()
      const navigatorWithShare = navigator as unknown as ShareNavigator
      const shareData: ShareData = {
        files,
        title: coin.name || 'Coin share card',
        text: contextSummary ? `${contextSummary}\n\nShared from ${APP_NAME}` : `Shared from ${APP_NAME}`,
      }

      if (navigatorWithShare.share && navigatorWithShare.canShare?.(shareData)) {
        await navigatorWithShare.share(shareData)
        return { mode: 'shared' }
      }

      if (files.length > 1) {
        await showAlert('Your generated card and original attribution cards will download as separate files. Allow multiple downloads if your browser asks.', { title: 'Download share files' })
      }
      for (const attachment of files) downloadBlob(attachment, attachment.name)
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
