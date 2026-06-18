import type { Coin, CoinImage, CoinReference, CoinSet, StorageLocation, Tag } from '@/types'

export const GOLDEN_COIN_FIXTURE_NAMES = [
  'roman-denarius-core',
  'greek-tetradrachm-valued',
  'byzantine-solidus-set-member',
  'wishlist-aureus-target',
  'sold-sestertius-archive',
  'private-provincial-bronze',
  'tagged-follis-storage',
  'image-heavy-drachm',
  'reference-rich-denarius',
] as const

export type GoldenCoinFixtureName = typeof GOLDEN_COIN_FIXTURE_NAMES[number]

export type GoldenCoinTrait =
  | 'roman'
  | 'greek'
  | 'byzantine'
  | 'wishlist'
  | 'sold'
  | 'private'
  | 'tagged'
  | 'set-member'
  | 'storage-location'
  | 'image-heavy'
  | 'legacy-custom-era'
  | 'valued'
  | 'reference-rich'
  | 'mint-alias'
  | 'mint-unmatched'
  | 'mint-unknown'

export interface GoldenCoinFixtureInfo {
  name: GoldenCoinFixtureName
  traits: readonly GoldenCoinTrait[]
}

export type CoinFixtureOverrides = Partial<Omit<Coin, 'images' | 'references' | 'tags' | 'sets' | 'storageLocation'>> & {
  images?: readonly CoinImage[]
  references?: readonly CoinReference[]
  tags?: readonly Tag[]
  sets?: readonly CoinSet[]
  storageLocation?: Coin['storageLocation']
}

interface CoinFixtureDefinition {
  coin: Coin
  traits: readonly GoldenCoinTrait[]
}

const fixtureUserId = 101
const createdAt = '2026-01-15T12:00:00Z'
const updatedAt = '2026-01-15T12:00:00Z'

const storageLocations = {
  trayA: {
    id: 201,
    userId: fixtureUserId,
    name: 'Cabinet Tray A',
    sortOrder: 1,
  },
  vaultBox: {
    id: 202,
    userId: fixtureUserId,
    name: 'Vault Box 2',
    sortOrder: 2,
  },
} satisfies Record<string, StorageLocation>

const tags = {
  photographed: {
    id: 301,
    userId: fixtureUserId,
    name: 'Photographed',
    color: '#c9a84c',
  },
  needsResearch: {
    id: 302,
    userId: fixtureUserId,
    name: 'Needs Research',
    color: '#4682b4',
  },
} satisfies Record<string, Tag>

const sets = {
  twelveCaesars: {
    id: 401,
    userId: fixtureUserId,
    name: 'Twelve Caesars',
    description: 'Representative imperial portrait set',
    color: '#c9a84c',
    icon: 'Crown',
    setType: 'defined',
    parentSetId: null,
    targetCompletionDate: '2026-12-31',
    createdAt,
    updatedAt,
  },
  byzantineGold: {
    id: 402,
    userId: fixtureUserId,
    name: 'Byzantine Gold',
    description: 'Gold issues for set-membership workflow tests',
    color: '#b08d57',
    icon: 'CircleDot',
    setType: 'open',
    parentSetId: null,
    targetCompletionDate: null,
    createdAt,
    updatedAt,
  },
} satisfies Record<string, CoinSet>

function buildImage(coinId: number, id: number, imageType: CoinImage['imageType'], isPrimary = false): CoinImage {
  return {
    id,
    coinId,
    filePath: `/uploads/test-fixtures/${coinId}-${imageType}-${id}.webp`,
    imageType,
    isPrimary,
    createdAt,
  }
}

function buildReference(coinId: number, id: number, catalog: string, number: string, volume = ''): CoinReference {
  return {
    id,
    coinId,
    catalog,
    volume,
    number,
    invoiceNumber: `INV-${coinId}-${id}`,
    uri: `https://example.test/catalog/${catalog.toLowerCase()}/${number}`,
    createdAt,
    updatedAt,
  }
}

function baseCoin(id: number, name: string): Coin {
  return {
    id,
    name,
    category: 'Roman',
    denomination: 'Denarius',
    ruler: 'Trajan',
    era: 'ancient',
    mint: 'Rome',
    material: 'Silver',
    weightGrams: 3.35,
    diameterMm: 18,
    grade: 'VF',
    obverseInscription: 'IMP TRAIANO AVG GER DAC P M TR P',
    reverseInscription: 'SPQR OPTIMO PRINCIPI',
    obverseDescription: 'Laureate bust right',
    reverseDescription: 'Victory standing left',
    rarityRating: 'Common',
    purchasePrice: 180,
    currentValue: 225,
    purchaseDate: '2024-03-15T00:00:00Z',
    purchaseLocation: 'Fixture Dealer',
    storageLocationId: null,
    storageLocation: null,
    notes: 'Deterministic frontend fixture coin.',
    aiAnalysis: '',
    obverseAnalysis: '',
    reverseAnalysis: '',
    referenceUrl: '',
    referenceText: '',
    isWishlist: false,
    isSold: false,
    soldPrice: null,
    soldDate: null,
    soldTo: '',
    isPrivate: false,
    listingStatus: 'unlisted',
    listingCheckedAt: null,
    listingCheckReason: '',
    userId: fixtureUserId,
    images: [buildImage(id, id * 10 + 1, 'obverse', true), buildImage(id, id * 10 + 2, 'reverse')],
    references: [],
    tags: [],
    sets: [],
    createdAt,
    updatedAt,
  }
}

const fixtureDefinitions = {
  'roman-denarius-core': {
    coin: {
      ...baseCoin(1001, 'Trajan Denarius Core'),
      referenceUrl: 'https://example.test/coins/roman-denarius-core',
      referenceText: 'RIC II Trajan 147',
    },
    traits: ['roman'],
  },
  'greek-tetradrachm-valued': {
    coin: {
      ...baseCoin(1002, 'Athens Owl Tetradrachm Valued'),
      category: 'Greek',
      denomination: 'Tetradrachm',
      ruler: 'Athens',
      mint: 'Athens',
      material: 'Silver',
      weightGrams: 17.18,
      diameterMm: 24,
      purchasePrice: 950,
      currentValue: 1250,
      purchaseDate: '2023-09-10T00:00:00Z',
      obverseInscription: '',
      reverseInscription: 'ΑΘΕ',
      obverseDescription: 'Helmeted head of Athena right',
      reverseDescription: 'Owl standing right, olive sprig and crescent',
    },
    traits: ['greek', 'valued'],
  },
  'byzantine-solidus-set-member': {
    coin: {
      ...baseCoin(1003, 'Justinian I Solidus Set Member'),
      category: 'Byzantine',
      denomination: 'Solidus',
      ruler: 'Justinian I',
      mint: 'Constantinople',
      material: 'Gold',
      weightGrams: 4.48,
      diameterMm: 21,
      purchasePrice: 700,
      currentValue: 875,
      sets: [{ ...sets.byzantineGold }],
    },
    traits: ['byzantine', 'set-member'],
  },
  'wishlist-aureus-target': {
    coin: {
      ...baseCoin(1004, 'Augustus Aureus Wishlist Target'),
      denomination: 'Aureus',
      ruler: 'Augustus',
      material: 'Gold',
      weightGrams: 7.9,
      diameterMm: 20,
      purchasePrice: null,
      currentValue: 8500,
      purchaseDate: null,
      purchaseLocation: '',
      isWishlist: true,
      notes: 'Wishlist target used for purchase workflow tests.',
    },
    traits: ['roman', 'wishlist', 'valued'],
  },
  'sold-sestertius-archive': {
    coin: {
      ...baseCoin(1005, 'Hadrian Sestertius Archive'),
      denomination: 'Sestertius',
      ruler: 'Hadrian',
      material: 'Bronze',
      weightGrams: 25.1,
      diameterMm: 32,
      purchasePrice: 240,
      currentValue: 300,
      isSold: true,
      soldPrice: 310,
      soldDate: '2025-04-20T00:00:00Z',
      soldTo: 'Archive Buyer',
    },
    traits: ['roman', 'sold'],
  },
  'private-provincial-bronze': {
    coin: {
      ...baseCoin(1006, 'Alexandria Provincial Bronze Private'),
      denomination: 'Drachm',
      ruler: 'Antoninus Pius',
      era: 'Roman Provincial Year 12',
      mint: 'Alexandria',
      material: 'Bronze',
      isPrivate: true,
      referenceText: 'Legacy handwritten tray note',
      notes: 'Private coin with a custom legacy era value.',
    },
    traits: ['roman', 'private', 'legacy-custom-era'],
  },
  'tagged-follis-storage': {
    coin: {
      ...baseCoin(1007, 'Diocletian Follis Tagged Storage'),
      denomination: 'Follis',
      ruler: 'Diocletian',
      material: 'Bronze',
      storageLocationId: storageLocations.trayA.id,
      storageLocation: { id: storageLocations.trayA.id, name: storageLocations.trayA.name },
      tags: [{ ...tags.photographed }, { ...tags.needsResearch }],
    },
    traits: ['roman', 'tagged', 'storage-location'],
  },
  'image-heavy-drachm': {
    coin: {
      ...baseCoin(1008, 'Syracuse Drachm Image Heavy'),
      category: 'Greek',
      denomination: 'Drachm',
      ruler: 'Syracuse',
      mint: 'Syracuse',
      images: [
        buildImage(1008, 10081, 'obverse', true),
        buildImage(1008, 10082, 'reverse'),
        buildImage(1008, 10083, 'detail'),
        buildImage(1008, 10084, 'other'),
      ],
    },
    traits: ['greek', 'image-heavy'],
  },
  'reference-rich-denarius': {
    coin: {
      ...baseCoin(1009, 'Vespasian Denarius Reference Rich'),
      ruler: 'Vespasian',
      referenceUrl: 'https://example.test/coins/reference-rich-denarius',
      referenceText: 'RIC II.1 Vespasian 772; BMCRE 161; RSC 554',
      references: [
        buildReference(1009, 1, 'RIC', '772', 'II.1'),
        buildReference(1009, 2, 'BMCRE', '161'),
        buildReference(1009, 3, 'RSC', '554'),
      ],
      sets: [{ ...sets.twelveCaesars }],
    },
    traits: ['roman', 'reference-rich', 'set-member'],
  },
} satisfies Record<GoldenCoinFixtureName, CoinFixtureDefinition>

function cloneCoin(coin: Coin): Coin {
  return {
    ...coin,
    storageLocation: coin.storageLocation ? { ...coin.storageLocation } : null,
    images: coin.images.map((image) => ({ ...image })),
    references: coin.references?.map((reference) => ({ ...reference })),
    tags: coin.tags?.map((tag) => ({ ...tag })),
    sets: coin.sets?.map((set) => ({ ...set })),
  }
}

function applyCoinOverrides(coin: Coin, overrides: CoinFixtureOverrides): Coin {
  return {
    ...coin,
    ...overrides,
    storageLocation: overrides.storageLocation === undefined
      ? coin.storageLocation
      : overrides.storageLocation
        ? { ...overrides.storageLocation }
        : null,
    images: overrides.images === undefined ? coin.images : overrides.images.map((image) => ({ ...image })),
    references: overrides.references === undefined
      ? coin.references
      : overrides.references.map((reference) => ({ ...reference })),
    tags: overrides.tags === undefined ? coin.tags : overrides.tags.map((tag) => ({ ...tag })),
    sets: overrides.sets === undefined ? coin.sets : overrides.sets.map((set) => ({ ...set })),
  }
}

export const goldenCoinFixtureCatalog: readonly GoldenCoinFixtureInfo[] = GOLDEN_COIN_FIXTURE_NAMES.map((name) => ({
  name,
  traits: fixtureDefinitions[name].traits,
}))

export function buildGoldenCoinFixture(name: GoldenCoinFixtureName, overrides: CoinFixtureOverrides = {}): Coin {
  return applyCoinOverrides(cloneCoin(fixtureDefinitions[name].coin), overrides)
}

export function buildGoldenCoinFixtures(overrides: Partial<Record<GoldenCoinFixtureName, CoinFixtureOverrides>> = {}): Coin[] {
  return GOLDEN_COIN_FIXTURE_NAMES.map((name) => buildGoldenCoinFixture(name, overrides[name]))
}

export function buildRomanDenariusCore(overrides?: CoinFixtureOverrides): Coin {
  return buildGoldenCoinFixture('roman-denarius-core', overrides)
}

export function buildGreekTetradrachmValued(overrides?: CoinFixtureOverrides): Coin {
  return buildGoldenCoinFixture('greek-tetradrachm-valued', overrides)
}

export function buildByzantineSolidusSetMember(overrides?: CoinFixtureOverrides): Coin {
  return buildGoldenCoinFixture('byzantine-solidus-set-member', overrides)
}

export function buildWishlistAureusTarget(overrides?: CoinFixtureOverrides): Coin {
  return buildGoldenCoinFixture('wishlist-aureus-target', overrides)
}

export function buildSoldSestertiusArchive(overrides?: CoinFixtureOverrides): Coin {
  return buildGoldenCoinFixture('sold-sestertius-archive', overrides)
}

export function buildPrivateProvincialBronze(overrides?: CoinFixtureOverrides): Coin {
  return buildGoldenCoinFixture('private-provincial-bronze', overrides)
}

export function buildTaggedFollisStorage(overrides?: CoinFixtureOverrides): Coin {
  return buildGoldenCoinFixture('tagged-follis-storage', overrides)
}

export function buildImageHeavyDrachm(overrides?: CoinFixtureOverrides): Coin {
  return buildGoldenCoinFixture('image-heavy-drachm', overrides)
}

export function buildReferenceRichDenarius(overrides?: CoinFixtureOverrides): Coin {
  return buildGoldenCoinFixture('reference-rich-denarius', overrides)
}

export function buildMintMapFixtureCoins(): Coin[] {
  return [
    buildRomanDenariusCore(),
    buildReferenceRichDenarius({ id: 1010, name: 'Roma Alias Denarius', mint: 'Roma' }),
    buildByzantineSolidusSetMember({ id: 1011, name: 'Byzantium Alias Solidus', mint: 'Byzantium' }),
    buildImageHeavyDrachm(),
    buildGreekTetradrachmValued({ id: 1012, name: 'Unmatched Camp Mint Bronze', mint: 'Traveling Camp' }),
    buildGreekTetradrachmValued({ id: 1013, name: 'Unknown Mint Fraction', mint: '' }),
  ]
}

export function buildTestStorageLocations(): StorageLocation[] {
  return Object.values(storageLocations).map((location) => ({ ...location }))
}

export function buildTestTags(): Tag[] {
  return Object.values(tags).map((tag) => ({ ...tag }))
}

export function buildTestCoinSets(): CoinSet[] {
  return Object.values(sets).map((set) => ({ ...set }))
}
