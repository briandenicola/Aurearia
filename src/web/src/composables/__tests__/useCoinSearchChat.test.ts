import { describe, expect, it, vi } from 'vitest'
import type { CoinSuggestion } from '@/types'
import { buildWishlistCoinPayload, normalizeSuggestionEra, resolveCategoryAndEra } from '../useCoinSearchChat'

const mockMatchCategoryEra = vi.fn()
vi.mock('@/api/client', () => ({
  matchCategoryEra: (type: 'category' | 'era', value: string) => mockMatchCategoryEra(type, value),
}))

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

  it('does not require or send catalog references for wishlist suggestions', () => {
    const payload = buildWishlistCoinPayload(makeSuggestion({
      candidateReferences: [
        { catalog: '', number: '123' },
        { catalog: ' RIC ', number: '456', volume: '' },
        { catalog: 'SEAR', number: '' },
      ],
    }))

    expect(payload).toMatchObject({
      name: 'Trajan Denarius',
      isWishlist: true,
    })
    expect(payload.references).toBeUndefined()
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

describe('resolveCategoryAndEra', () => {
  const categoryOptions = ['Roman', 'Greek', 'Byzantine', 'Modern', 'Other', 'Celtic']
  const eraOptions = ['ancient', 'medieval', 'modern']

  it('resolves an exact match without calling the backend or asking the user', async () => {
    const requestConfirmation = vi.fn()
    const result = await resolveCategoryAndEra(
      makeSuggestion({ category: 'roman', era: 'ancient' }),
      categoryOptions,
      eraOptions,
      requestConfirmation,
    )
    expect(result).toEqual({ category: 'Roman', era: 'ancient' })
    expect(mockMatchCategoryEra).not.toHaveBeenCalled()
    expect(requestConfirmation).not.toHaveBeenCalled()
  })

  it('short-circuits era via the existing keyword heuristic when it lands on a known era', async () => {
    const requestConfirmation = vi.fn()
    mockMatchCategoryEra.mockResolvedValue({ data: { matched: false, match: '' } })
    const result = await resolveCategoryAndEra(
      makeSuggestion({ category: 'Roman', era: 'Byzantine' }),
      categoryOptions,
      eraOptions,
      requestConfirmation,
    )
    expect(result).toEqual({ category: 'Roman', era: 'medieval' })
    expect(mockMatchCategoryEra).not.toHaveBeenCalledWith('era', expect.anything())
  })

  it('falls back to the backend fuzzy matcher for a near-miss category', async () => {
    mockMatchCategoryEra.mockResolvedValue({ data: { matched: true, match: 'Roman' } })
    const requestConfirmation = vi.fn()
    const result = await resolveCategoryAndEra(
      makeSuggestion({ category: 'Roman Republic', era: 'ancient' }),
      categoryOptions,
      eraOptions,
      requestConfirmation,
    )
    expect(result).toEqual({ category: 'Roman', era: 'ancient' })
    expect(mockMatchCategoryEra).toHaveBeenCalledWith('category', 'Roman Republic')
    expect(requestConfirmation).not.toHaveBeenCalled()
  })

  it('asks the user to confirm when nothing matches confidently', async () => {
    mockMatchCategoryEra.mockResolvedValue({ data: { matched: false, match: '' } })
    const requestConfirmation = vi.fn().mockResolvedValue('Celtic')
    const result = await resolveCategoryAndEra(
      makeSuggestion({ category: 'Gaulish', era: 'ancient' }),
      categoryOptions,
      eraOptions,
      requestConfirmation,
    )
    expect(result).toEqual({ category: 'Celtic', era: 'ancient' })
    expect(requestConfirmation).toHaveBeenCalledWith({
      fieldLabel: 'Category',
      suggestedValue: 'Gaulish',
      options: categoryOptions,
    })
  })

  it('returns null (never creates the coin) if the user cancels confirmation', async () => {
    mockMatchCategoryEra.mockResolvedValue({ data: { matched: false, match: '' } })
    const requestConfirmation = vi.fn().mockResolvedValue(null)
    const result = await resolveCategoryAndEra(
      makeSuggestion({ category: 'Gaulish', era: 'ancient' }),
      categoryOptions,
      eraOptions,
      requestConfirmation,
    )
    expect(result).toBeNull()
  })
})
