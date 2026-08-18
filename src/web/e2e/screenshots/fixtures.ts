import type { APIRequestContext } from '@playwright/test'
import { COIN_ERAS } from '../../src/types'

/**
 * Distinctive prefix for every screenshot fixture coin. Used both to name the
 * records and to filter the UI down to only these coins before capturing.
 */
export const SCREENSHOT_PREFIX = '[Screenshot]'

interface ScreenshotFixtureCoin {
  name: string
  category: string
  denomination: string
  ruler: string
  era: string
  mint: string
  material: string
  weightGrams: number
  diameterMm: number
  grade: string
  obverseInscription: string
  reverseInscription: string
  obverseDescription: string
  reverseDescription: string
  rarityRating: string
  isWishlist: boolean
  notes: string
}

// Sanitized, plausible public-domain numismatic data only — no personal
// prices, purchase notes, or dealer URLs. Covers the three representative
// records requested: an ancient collection coin, a modern/slabbed coin, and
// a wishlist target.
export const SCREENSHOT_FIXTURE_COINS: ScreenshotFixtureCoin[] = [
  {
    name: `${SCREENSHOT_PREFIX} Trajan Denarius (Ancient Collection)`,
    category: 'Roman',
    denomination: 'Denarius',
    ruler: 'Trajan',
    era: COIN_ERAS[0], // 'ancient' — the only server-accepted built-in era value for this coin (see models.EraAncient in src/api/models/coin.go)
    mint: 'Rome',
    material: 'Silver',
    weightGrams: 3.4,
    diameterMm: 18,
    grade: 'VF',
    obverseInscription: 'IMP CAES NER TRAIANO OPTIMO AVG GER DAC',
    reverseInscription: 'PM TR P COS VI PP SPQR',
    obverseDescription: 'Laureate bust of Trajan facing right',
    reverseDescription: 'Victory standing left, holding wreath and palm branch',
    rarityRating: 'Common',
    isWishlist: false,
    notes: 'Representative ancient collection coin used for the repeatable Aurearia screenshot tour. Public numismatic reference data only.',
  },
  {
    name: `${SCREENSHOT_PREFIX} 2021 American Silver Eagle (NGC MS70)`,
    category: 'Modern',
    denomination: 'Dollar',
    ruler: 'United States',
    era: COIN_ERAS[2], // 'modern' — built-in era value (models.EraModern)
    mint: 'West Point',
    material: 'Silver',
    weightGrams: 31.1,
    diameterMm: 40.6,
    grade: 'NGC MS70',
    obverseInscription: 'LIBERTY IN GOD WE TRUST',
    reverseInscription: 'UNITED STATES OF AMERICA E PLURIBUS UNUM',
    obverseDescription: 'Walking Liberty design',
    reverseDescription: 'Heraldic eagle with shield',
    rarityRating: 'Common',
    isWishlist: false,
    notes: 'Representative modern slabbed coin used for the repeatable Aurearia screenshot tour. Public numismatic reference data only.',
  },
  {
    name: `${SCREENSHOT_PREFIX} Augustus Aureus (Wishlist Target)`,
    category: 'Roman',
    denomination: 'Aureus',
    ruler: 'Augustus',
    era: COIN_ERAS[0], // 'ancient' — built-in era value (models.EraAncient)
    mint: 'Lugdunum',
    material: 'Gold',
    weightGrams: 7.9,
    diameterMm: 20,
    grade: 'AU',
    obverseInscription: 'CAESAR AVGVSTVS DIVI F PATER PATRIAE',
    reverseInscription: 'AVGVSTI F COS DESIG PRINC IVVENT',
    obverseDescription: 'Laureate head of Augustus facing right',
    reverseDescription: 'Gaius and Lucius Caesar standing, shields and spears between them',
    rarityRating: 'Scarce',
    isWishlist: true,
    notes: 'Representative wishlist target used for the repeatable Aurearia screenshot tour. Public numismatic reference data only.',
  },
]

function toMutationPayload(fixture: ScreenshotFixtureCoin) {
  return {
    name: fixture.name,
    category: fixture.category,
    denomination: fixture.denomination,
    ruler: fixture.ruler,
    era: fixture.era,
    mint: fixture.mint,
    material: fixture.material,
    weightGrams: fixture.weightGrams,
    diameterMm: fixture.diameterMm,
    grade: fixture.grade,
    obverseInscription: fixture.obverseInscription,
    reverseInscription: fixture.reverseInscription,
    obverseDescription: fixture.obverseDescription,
    reverseDescription: fixture.reverseDescription,
    rarityRating: fixture.rarityRating,
    isWishlist: fixture.isWishlist,
    isSold: false,
    isPrivate: false,
    notes: fixture.notes,
    // Deliberately no price/valuation/purchase data — sanitized public fixtures only.
    purchasePrice: null,
    currentValue: null,
    purchaseDate: null,
    purchaseLocation: '',
    vendorSku: '',
    vendorInvoice: '',
    referenceUrl: '',
    referenceText: '',
    storageLocationId: null,
  }
}

export interface EnsuredScreenshotCoin {
  id: number
  name: string
  isWishlist: boolean
}

/**
 * Finds every coin whose name is an exact match, ordered lowest-ID-first.
 * A prior run under concurrent workers (before `workers: 1` was enforced in
 * playwright.config.ts) could have raced a GET-then-POST check and created
 * more than one coin with the exact same `[Screenshot]`-prefixed name. This
 * returns all of them so the caller can reconcile down to one canonical
 * record instead of only ever seeing the first.
 */
async function findCoinsByExactName(api: APIRequestContext, name: string): Promise<Array<{ id: number }>> {
  const res = await api.get('/api/coins', { params: { search: name, limit: '50' } })
  if (!res.ok()) {
    throw new Error(`Failed to search for existing screenshot fixture "${name}": ${res.status()} ${await res.text()}`)
  }
  const body = (await res.json()) as { coins: Array<{ id: number; name: string }> }
  return body.coins
    .filter((coin) => coin.name === name)
    .map((coin) => ({ id: coin.id }))
    .sort((a, b) => a.id - b.id)
}

/**
 * Removes extra same-name screenshot fixture duplicates, keeping only
 * `canonicalId`. Only ever called with IDs that were just matched by exact
 * name against a known `SCREENSHOT_FIXTURE_COINS` entry (which is always
 * `[Screenshot]`-prefixed), so this never touches unrelated or non-prefixed
 * records. Surfaces any delete failure immediately rather than silently
 * leaving duplicate cards in the gallery.
 */
async function reconcileDuplicates(api: APIRequestContext, name: string, canonicalId: number, duplicateIds: number[]): Promise<void> {
  for (const duplicateId of duplicateIds) {
    const res = await api.delete(`/api/coins/${duplicateId}`)
    if (!res.ok()) {
      throw new Error(
        `Failed to delete duplicate screenshot fixture "${name}" (id ${duplicateId}, keeping canonical id ${canonicalId}): ` +
          `${res.status()} ${await res.text()}`,
      )
    }
  }
}

/**
 * Creates or updates the `[Screenshot]`-prefixed fixture coins used by the
 * beta screenshot tour, using the real `/api/coins` contract (same payload
 * shape as `createCoin`/`updateCoin` in `src/api/client.ts`).
 *
 * Idempotent: matches existing coins by exact name and updates them in place
 * instead of creating duplicates on rerun. Never deletes or touches any other
 * coin in the account.
 */
/**
 * Creates or updates the `[Screenshot]`-prefixed fixture coins used by the
 * beta screenshot tour, using the real `/api/coins` contract (same payload
 * shape as `createCoin`/`updateCoin` in `src/api/client.ts`).
 *
 * Idempotent: matches existing coins by exact name and updates them in
 * place instead of creating duplicates on rerun. If more than one coin
 * already shares the exact fixture name (e.g. from a prior run that raced
 * fixture seeding across parallel workers), the lowest-ID record is kept as
 * canonical and updated, and the extra duplicates are deleted via the real
 * authenticated DELETE contract. Never deletes or touches any coin that
 * isn't an exact name match for one of `SCREENSHOT_FIXTURE_COINS` (all of
 * which carry the `[Screenshot]` prefix).
 */
export async function ensureScreenshotFixtures(api: APIRequestContext): Promise<EnsuredScreenshotCoin[]> {
  const results: EnsuredScreenshotCoin[] = []
  for (const fixture of SCREENSHOT_FIXTURE_COINS) {
    const payload = toMutationPayload(fixture)
    const matches = await findCoinsByExactName(api, fixture.name)

    if (matches.length > 0) {
      const [canonical, ...duplicates] = matches
      if (duplicates.length > 0) {
        await reconcileDuplicates(
          api,
          fixture.name,
          canonical!.id,
          duplicates.map((d) => d.id),
        )
      }
      const res = await api.put(`/api/coins/${canonical!.id}`, { data: payload })
      if (!res.ok()) {
        throw new Error(`Failed to update screenshot fixture "${fixture.name}": ${res.status()} ${await res.text()}`)
      }
      results.push({ id: canonical!.id, name: fixture.name, isWishlist: fixture.isWishlist })
      continue
    }

    const res = await api.post('/api/coins', { data: payload })
    if (!res.ok()) {
      throw new Error(`Failed to create screenshot fixture "${fixture.name}": ${res.status()} ${await res.text()}`)
    }
    const created = (await res.json()) as { id: number }
    results.push({ id: created.id, name: fixture.name, isWishlist: fixture.isWishlist })
  }
  return results
}
