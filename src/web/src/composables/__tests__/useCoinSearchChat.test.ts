import { describe, expect, it } from 'vitest'
import type { CoinSuggestion } from '@/types'
import { buildWishlistCoinPayload, normalizeSuggestionEra } from '../useCoinSearchChat'

function makeSuggestion(overrides: Partial<CoinSuggestion> = {}): CoinSuggestion {
  return {
    name: 'Trajan Denarius',
    description: 'Silver denarius of Trajan',
    category: 'Roman',
    era: 'ancient',
    ruler: 'Trajan',
    material: 'Silver',
    denomination: 'Denarius',
    estPrice: '$125',
    imageUrl: '',
    sourceUrl: 'https://example.com/coin',
    sourceName: 'Example Dealer',
    ...overrides,
  }
}

describe('useCoinSearchChat wishlist payload', () => {
  it('normalizes AI era labels that the coin API would reject', () => {
    expect(normalizeSuggestionEra('Roman Imperial')).toBe('ancient')
    expect(normalizeSuggestionEra('Byzantine')).toBe('medieval')
    expect(normalizeSuggestionEra('Modern commemorative')).toBe('modern')
    expect(normalizeSuggestionEra('Unknown period')).toBe('')
  })

  it('builds a create-coin payload with only supported era values', () => {
    const payload = buildWishlistCoinPayload(makeSuggestion({
      era: 'Roman Imperial',
      category: 'Unclassified',
      material: 'Billon',
      estPrice: 'Estimate $1,250.50',
      candidateReferences: [
        { catalog: ' RIC ', number: ' 123 ', volume: ' II ', uri: ' https://example.com/ric ' },
        { catalog: 'RIC', number: '456' },
        { catalog: 'RPC', number: '   ' },
        { catalog: 'SNG', number: '789', volume: '' },
        { catalog: 'SEAR', number: '101' },
      ],
    }))

    expect(payload).toMatchObject({
      name: 'Trajan Denarius',
      category: 'Other',
      material: 'Other',
      era: 'ancient',
      isWishlist: true,
      currentValue: 1250.5,
      references: [
        {
          catalog: 'RIC',
          number: '123',
          volume: 'II',
          uri: 'https://example.com/ric',
        },
        {
          catalog: 'SEAR',
          number: '101',
          volume: '',
          uri: '',
        },
      ],
    })
  })

  it('truncates agent text fields to backend create-coin limits', () => {
    const payload = buildWishlistCoinPayload(makeSuggestion({
      name: 'A'.repeat(250),
      description: 'B'.repeat(6000),
      denomination: 'C'.repeat(250),
      ruler: 'D'.repeat(250),
      sourceUrl: `https://example.com/${'e'.repeat(2100)}`,
      sourceName: 'F'.repeat(2100),
    }))

    expect(payload.name).toHaveLength(200)
    expect(payload.notes).toHaveLength(5000)
    expect(payload.denomination).toHaveLength(200)
    expect(payload.ruler).toHaveLength(200)
    expect(payload.referenceUrl).toHaveLength(2000)
    expect(payload.referenceText).toHaveLength(2000)
  })
})
