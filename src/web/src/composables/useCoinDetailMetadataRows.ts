// T015: Coin detail metadata rows composable for #219
// This composable generates consistent metadata row schema from coin data

import { computed } from 'vue'
import type { Coin, CoinDetailMetadataRow } from '@/types'
import { formatCurrency } from '@/utils/format'
import { sanitizeExternalUrl } from '@/composables/useSafeExternalLink'
import { externalInvoiceUrl } from '@/utils/paperless'

export function useCoinDetailMetadataRows(
  coin: Coin | null,
  options: { paperlessConfigured?: boolean; paperlessError?: string } = {},
) {
  const rows = computed<CoinDetailMetadataRow[]>(() => {
    if (!coin) return []

    const result: CoinDetailMetadataRow[] = []

    // Purchase & Value
    if (coin.purchasePrice != null) {
      result.push({
        key: 'purchasePrice',
        label: 'Purchase Price',
        value: formatCurrency(coin.purchasePrice),
      })
    }

    if (coin.currentValue != null) {
      result.push({
        key: 'currentValue',
        label: 'Current Value',
        value: formatCurrency(coin.currentValue),
        valueClass: 'gold',
      })
    }

    if (coin.purchaseDate) {
      result.push({
        key: 'purchaseDate',
        label: 'Purchase Date',
        value: new Date(coin.purchaseDate).toLocaleDateString(),
      })
    }

    if (coin.vendorSku) {
      result.push({
        key: 'vendorSku',
        label: 'Vendor SKU',
        value: coin.vendorSku,
      })
    }

    const invoiceProcessing = Boolean(options.paperlessConfigured && coin.paperlessTaskId && !coin.paperlessDocumentId)
    if (options.paperlessConfigured && options.paperlessError && !coin.paperlessDocumentId) {
      result.push({
        key: 'vendorInvoice',
        label: 'Vendor Invoice',
        value: `Upload failed: ${options.paperlessError}`,
        valueClass: 'text-loss',
      })
    } else if (invoiceProcessing) {
      result.push({
        key: 'vendorInvoice',
        label: 'Vendor Invoice',
        value: coin.vendorInvoice ? `${coin.vendorInvoice} (processing in paperless…)` : 'Processing in paperless…',
      })
    } else if (coin.vendorInvoice || (coin.paperlessDocumentId && options.paperlessConfigured)) {
      const hasDocument = Boolean(coin.paperlessDocumentId && options.paperlessConfigured)
      const pastedUrl = hasDocument ? null : externalInvoiceUrl(coin.vendorInvoice)
      result.push({
        key: 'vendorInvoice',
        label: 'Vendor Invoice',
        value: coin.vendorInvoice || 'View invoice',
        url: pastedUrl,
        valueClass: pastedUrl ? 'gold' : undefined,
        activatable: hasDocument,
      })
    }

    result.push({
      key: 'storageLocation',
      label: 'Storage Location',
      value: coin.storageLocation?.name ?? '—',
    })

    // Physical attributes
    if (coin.denomination) {
      result.push({
        key: 'denomination',
        label: 'Denomination',
        value: coin.denomination,
      })
    }

    if (coin.era) {
      result.push({
        key: 'era',
        label: 'Era',
        value: coin.era,
      })
    }

    if (coin.mint) {
      result.push({
        key: 'mint',
        label: 'Mint',
        value: coin.mint,
      })
    }

    if (coin.material) {
      result.push({
        key: 'material',
        label: 'Material',
        value: coin.material,
      })
    }

    if (coin.weightGrams != null) {
      result.push({
        key: 'weight',
        label: 'Weight',
        value: `${coin.weightGrams}g`,
      })
    }

    if (coin.diameterMm != null) {
      result.push({
        key: 'diameter',
        label: 'Diameter',
        value: `${coin.diameterMm}mm`,
      })
    }

    if (coin.grade) {
      result.push({
        key: 'grade',
        label: 'Grade',
        value: coin.grade,
        valueClass: 'gold',
      })
    }

    // Purchase location (full-width last row, optional link)
    if (coin.purchaseLocation) {
      const safeUrl = sanitizeExternalUrl(coin.referenceUrl)
      result.push({
        key: 'purchaseLocation',
        label: '',
        value: coin.purchaseLocation,
        fullWidth: true,
        url: safeUrl,
      })
    }

    return result
  })

  return {
    rows,
  }
}
