import type { Coin } from '@/types'
import type { MintLocation } from '@/types'

export interface MintReference {
  id: number
  displayName: string
  lat: number
  lng: number
  aliases: readonly string[]
  region: string
}

export interface MintGroup {
  mint: MintReference
  coins: Coin[]
  count: number
}

export interface UnmatchedMintGroup {
  normalizedName: string
  originalNames: string[]
  coins: Coin[]
}

export interface MintMapAggregation {
  matched: MintGroup[]
  unmatched: UnmatchedMintGroup[]
  unknown: Coin[]
}

export function normalizeMintName(value: string): string {
  return value
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/&/g, ' and ')
    .replace(/[^a-z0-9]+/g, ' ')
    .trim()
    .replace(/\s+/g, ' ')
}

function buildMintLookup(mintLocations: readonly MintLocation[]): Map<string, MintReference> {
  const lookup = new Map<string, MintReference>()
  for (const mint of mintLocations) {
    lookup.set(normalizeMintName(mint.displayName), mint)
    for (const alias of mint.aliases ?? []) {
      const normalizedAlias = normalizeMintName(alias)
      if (normalizedAlias) {
        lookup.set(normalizedAlias, mint)
      }
    }
  }
  return lookup
}

export function findMintReference(value: string, mintLocations: readonly MintLocation[]): MintReference | null {
  const normalized = normalizeMintName(value)
  if (!normalized) return null
  const mintLookup = buildMintLookup(mintLocations)
  return mintLookup.get(normalized) ?? null
}

export function groupCoinsByMint(coins: Coin[], mintLocations: readonly MintLocation[]): MintMapAggregation {
  const mintLookup = buildMintLookup(mintLocations)
  const matchedByMint = new Map<string, MintGroup>()
  const unmatchedByName = new Map<string, { originalNames: Set<string>; coins: Coin[] }>()
  const unknown: Coin[] = []

  for (const coin of coins) {
    const rawMint = coin.mint?.trim() ?? ''
    const normalizedName = normalizeMintName(rawMint)
    if (!normalizedName) {
      unknown.push(coin)
      continue
    }

    const reference = mintLookup.get(normalizedName) ?? null
    if (reference) {
      const existing = matchedByMint.get(String(reference.id))
      if (existing) {
        existing.coins.push(coin)
        existing.count = existing.coins.length
      } else {
        matchedByMint.set(String(reference.id), { mint: reference, coins: [coin], count: 1 })
      }
      continue
    }

    const existing = unmatchedByName.get(normalizedName)
    if (existing) {
      existing.originalNames.add(rawMint)
      existing.coins.push(coin)
    } else {
      unmatchedByName.set(normalizedName, { originalNames: new Set([rawMint]), coins: [coin] })
    }
  }

  const matched = [...matchedByMint.values()].sort((a, b) =>
    b.count - a.count || a.mint.displayName.localeCompare(b.mint.displayName),
  )
  const unmatched = [...unmatchedByName.entries()]
    .map(([normalizedName, group]) => ({
      normalizedName,
      originalNames: [...group.originalNames].sort((a, b) => a.localeCompare(b)),
      coins: group.coins,
    }))
    .sort((a, b) => b.coins.length - a.coins.length || a.normalizedName.localeCompare(b.normalizedName))

  return { matched, unmatched, unknown }
}
