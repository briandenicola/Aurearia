import { describe, expect, it } from 'vitest'
import {
  findMintReference,
  groupCoinsByMint,
  normalizeMintName,
} from '@/utils/mintMap'
import { buildMintMapFixtureCoins, buildRomanDenariusCore, buildTestMintLocations } from '@/test/fixtures/coins'

describe('mintMap utilities', () => {
  const mintLocations = buildTestMintLocations()

  it('normalizes case, punctuation, diacritics, and whitespace', () => {
    expect(normalizeMintName('  RÓMA--Mint  ')).toBe('roma mint')
    expect(normalizeMintName('Lugdunum / Lyon')).toBe('lugdunum lyon')
  })

  it('matches canonical mints and aliases', () => {
    expect(findMintReference('Rome', mintLocations)?.id).toBe(1)
    expect(findMintReference('Roma', mintLocations)?.id).toBe(1)
    expect(findMintReference('Byzantium', mintLocations)?.id).toBe(2)
    expect(findMintReference('not a known mint', mintLocations)).toBeNull()
  })

  it('groups aliases with their canonical mint and sorts by count', () => {
    const grouped = groupCoinsByMint(buildMintMapFixtureCoins(), mintLocations)

    expect(grouped.matched[0]?.mint.id).toBe(1)
    expect(grouped.matched[0]?.count).toBe(2)
    expect(grouped.matched.some((group) => group.mint.id === 2)).toBe(true)
    expect(grouped.unmatched).toHaveLength(1)
    expect(grouped.unmatched[0]?.originalNames).toEqual(['Traveling Camp'])
    expect(grouped.unknown.map((coin) => coin.name)).toEqual(['Unknown Mint Fraction'])
  })

  it('keeps duplicate unmatched names together', () => {
    const grouped = groupCoinsByMint([
      buildRomanDenariusCore({ id: 1, name: 'Camp One', mint: 'Traveling Camp' }),
      buildRomanDenariusCore({ id: 2, name: 'Camp Two', mint: 'traveling-camp' }),
    ], mintLocations)

    expect(grouped.unmatched).toHaveLength(1)
    expect(grouped.unmatched[0]?.coins).toHaveLength(2)
    expect(grouped.unknown).toHaveLength(0)
  })

  it('does not depend on the static seed list at runtime', () => {
    const customLocations = [{
      id: 99,
      displayName: 'Custom Camp',
      lat: 10,
      lng: 20,
      region: 'Field Notes',
      aliases: ['Traveling Camp'],
      createdAt: '2026-06-18T00:00:00Z',
      updatedAt: '2026-06-18T00:00:00Z',
    }]

    const grouped = groupCoinsByMint([
      buildRomanDenariusCore({ id: 1, name: 'Camp Coin', mint: 'Traveling Camp' }),
    ], customLocations)

    expect(grouped.matched[0]?.mint.id).toBe(99)
    expect(grouped.unmatched).toHaveLength(0)
  })
})
