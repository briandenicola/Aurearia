import { describe, expect, it } from 'vitest'
import {
  findMintReference,
  groupCoinsByMint,
  normalizeMintName,
} from '@/utils/mintMap'
import { buildMintMapFixtureCoins, buildRomanDenariusCore } from '@/test/fixtures/coins'

describe('mintMap utilities', () => {
  it('normalizes case, punctuation, diacritics, and whitespace', () => {
    expect(normalizeMintName('  RÓMA--Mint  ')).toBe('roma mint')
    expect(normalizeMintName('Lugdunum / Lyon')).toBe('lugdunum lyon')
  })

  it('matches canonical mints and aliases', () => {
    expect(findMintReference('Rome')?.id).toBe('rome')
    expect(findMintReference('Roma')?.id).toBe('rome')
    expect(findMintReference('Byzantium')?.id).toBe('constantinople')
    expect(findMintReference('not a known mint')).toBeNull()
  })

  it('groups aliases with their canonical mint and sorts by count', () => {
    const grouped = groupCoinsByMint(buildMintMapFixtureCoins())

    expect(grouped.matched[0]?.mint.id).toBe('rome')
    expect(grouped.matched[0]?.count).toBe(2)
    expect(grouped.matched.some((group) => group.mint.id === 'constantinople')).toBe(true)
    expect(grouped.unmatched).toHaveLength(1)
    expect(grouped.unmatched[0]?.originalNames).toEqual(['Traveling Camp'])
    expect(grouped.unknown.map((coin) => coin.name)).toEqual(['Unknown Mint Fraction'])
  })

  it('keeps duplicate unmatched names together', () => {
    const grouped = groupCoinsByMint([
      buildRomanDenariusCore({ id: 1, name: 'Camp One', mint: 'Traveling Camp' }),
      buildRomanDenariusCore({ id: 2, name: 'Camp Two', mint: 'traveling-camp' }),
    ])

    expect(grouped.unmatched).toHaveLength(1)
    expect(grouped.unmatched[0]?.coins).toHaveLength(2)
    expect(grouped.unknown).toHaveLength(0)
  })
})
