import type {
  CoinCopilotSpecialistCapability,
  CoinCopilotSpecialistEvidence,
  CoinSuggestion,
} from '@/types'

export function isEligibleCopilotDealerListing(
  capability: CoinCopilotSpecialistCapability,
  item: CoinCopilotSpecialistEvidence,
): boolean {
  return capability === 'market_search' &&
    item.kind === 'dealer_listing' &&
    item.verificationState === 'verified' &&
    item.availability === 'available' &&
    item.title.trim().length > 0 &&
    item.sourceUrl.trim().length > 0
}

export function copilotDealerListingKey(toolCallId: string, sourceUrl: string): string {
  return `copilot:${toolCallId}:${sourceUrl}`
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
    imageUrl: '',
    sourceUrl: item.sourceUrl,
    sourceName: item.dealerName ?? '',
  }
}
