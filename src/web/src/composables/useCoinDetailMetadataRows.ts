// T015: Coin detail metadata rows composable for #219
// This composable generates consistent metadata row schema from coin data

import { computed } from 'vue'
import type { Coin, CoinDetailMetadataRow } from '@/types'
import { formatCurrency } from '@/utils/format'
import { sanitizeExternalUrl } from '@/composables/useSafeExternalLink'

export function useCoinDetailMetadataRows(coin: Coin | null) {
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

    if (coin.rarityRating) {
      result.push({
        key: 'rarity',
        label: 'Rarity / RIC',
        value: coin.rarityRating,
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
