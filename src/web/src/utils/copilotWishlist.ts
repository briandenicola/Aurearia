import type {
  CoinCopilotSpecialistCapability,
  CoinCopilotSpecialistEvidence,
  CoinSuggestion,
} from '@/types'

export function isEligibleCopilotDealerListing(
  capability: CoinCopilotSpecialistCapability,
  item: CoinCopilotSpecialistEvidence,
): boolean {
  // Listings read from search results are only partially verified; the wish
  // list re-checks availability later, so they stay saveable.
  return capability === 'market_search' &&
    item.kind === 'dealer_listing' &&
    (item.verificationState === 'verified' || item.verificationState === 'partial') &&
    (item.availability === 'available' || item.availability === 'unknown') &&
    item.title.trim().length > 0 &&
    item.sourceUrl.trim().length > 0
}

export function copilotDealerListingKey(toolCallId: string, sourceUrl: string): string {
  return `copilot:${toolCallId}:${sourceUrl}`
}

export function copilotDealerListingUncertainty(item: CoinCopilotSpecialistEvidence): string {
  return [
    item.verificationState === 'partial' ? 'This listing is only partially verified.' : '',
    item.availability === 'unknown' ? 'Availability is unknown.' : '',
  ].filter(Boolean).join(' ')
}

export function copilotDealerListingToSuggestion(
  item: CoinCopilotSpecialistEvidence,
): CoinSuggestion {
  const price = item.listedPrice == null
    ? ''
    : `${item.currency?.trim() ?? ''} ${item.listedPrice}`.trim()
  return {
    name: item.title,
    description: item.description ?? '',
    category: '',
    era: item.era ?? '',
    ruler: item.ruler ?? '',
    material: item.material ?? '',
    denomination: item.denomination ?? '',
    estPrice: price,
    imageUrl: item.imageUrl ?? '',
    sourceUrl: item.sourceUrl,
    sourceName: item.dealerName ?? '',
  }
}
