import { describe, expect, it } from 'vitest'
import {
  GOLDEN_COIN_FIXTURE_NAMES,
  buildGoldenCoinFixture,
  buildGoldenCoinFixtures,
  goldenCoinFixtureCatalog,
  type GoldenCoinTrait,
} from './coins'

describe('golden coin fixtures', () => {
  it('catalog covers every required F013 workflow trait', () => {
    expect(goldenCoinFixtureCatalog).toHaveLength(GOLDEN_COIN_FIXTURE_NAMES.length)

    const requiredTraits: GoldenCoinTrait[] = [
      'roman',
      'greek',
      'byzantine',
      'wishlist',
      'sold',
      'private',
      'tagged',
      'set-member',
      'storage-location',
      'image-heavy',
      'legacy-custom-era',
    ]
    const seenTraits = new Set(goldenCoinFixtureCatalog.flatMap((fixture) => fixture.traits))

    for (const trait of requiredTraits) {
      expect(seenTraits.has(trait), `missing golden fixture trait ${trait}`).toBe(true)
    }
  })

  it('builders return independent fixture clones with expected associations', () => {
    const first = buildGoldenCoinFixture('tagged-follis-storage')
    const second = buildGoldenCoinFixture('tagged-follis-storage')

    if (!first.tags?.[0] || !second.tags?.[0]) {
      throw new Error('tagged fixture must include tags')
    }

    first.tags[0].name = 'Mutated'
    first.storageLocation = { id: 999, name: 'Mutated' }

    expect(second.tags[0].name).toBe('Photographed')
    expect(second.storageLocation?.name).toBe('Cabinet Tray A')

    const fixtures = buildGoldenCoinFixtures()
    expect(fixtures).toHaveLength(GOLDEN_COIN_FIXTURE_NAMES.length)
    expect(buildGoldenCoinFixture('image-heavy-drachm').images).toHaveLength(4)
    expect(buildGoldenCoinFixture('reference-rich-denarius').references).toHaveLength(3)
    expect(buildGoldenCoinFixture('byzantine-solidus-set-member').sets).toHaveLength(1)
  })
})
