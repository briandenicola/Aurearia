import { ancientMints } from '@/data/ancientMints'
import type { Coin } from '@/types'

export interface MintReference {
  id: string
  displayName: string
  lat: number
  lng: number
  aliases: readonly string[]
  region?: string
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

export interface ProjectedPoint {
  x: number
  y: number
}

const VIEWBOX_WIDTH = 1000
const VIEWBOX_HEIGHT = 600
const MIN_LAT = 28
const MAX_LAT = 52
const MIN_LNG = -12
const MAX_LNG = 45

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
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

const mintLookup = new Map<string, MintReference>()

for (const mint of ancientMints) {
  mintLookup.set(normalizeMintName(mint.displayName), mint)
  for (const alias of mint.aliases) {
    mintLookup.set(normalizeMintName(alias), mint)
  }
}

export function findMintReference(value: string): MintReference | null {
  const normalized = normalizeMintName(value)
  if (!normalized) return null
  return mintLookup.get(normalized) ?? null
}

export function groupCoinsByMint(coins: Coin[]): MintMapAggregation {
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

    const reference = findMintReference(rawMint)
    if (reference) {
      const existing = matchedByMint.get(reference.id)
      if (existing) {
        existing.coins.push(coin)
        existing.count = existing.coins.length
      } else {
        matchedByMint.set(reference.id, { mint: reference, coins: [coin], count: 1 })
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

export function projectLatLngToViewBox(lat: number, lng: number): ProjectedPoint {
  const normalizedLng = (clamp(lng, MIN_LNG, MAX_LNG) - MIN_LNG) / (MAX_LNG - MIN_LNG)
  const normalizedLat = (MAX_LAT - clamp(lat, MIN_LAT, MAX_LAT)) / (MAX_LAT - MIN_LAT)

  return {
    x: Number((normalizedLng * VIEWBOX_WIDTH).toFixed(2)),
    y: Number((normalizedLat * VIEWBOX_HEIGHT).toFixed(2)),
  }
}
