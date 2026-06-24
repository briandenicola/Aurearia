export interface TrayCoinImage {
  id?: number
  coinId?: number
  filePath: string
  imageType?: string | null
  isPrimary?: boolean | null
  createdAt?: string
}

export interface TrayCoin {
  id: number
  name: string
  diameterMm: number | null
  images: readonly TrayCoinImage[]
}

export interface TrayLayoutOptions {
  minCoinPx: number
  maxCoinPx: number
  defaultDiameterMm: number
}

export function selectTrayCoinImage(images: readonly TrayCoinImage[]): TrayCoinImage | null {
  return images.find(image => image.imageType?.toLowerCase() === 'obverse')
    ?? images.find(image => image.imageType?.toLowerCase() === 'reverse')
    ?? images.find(image => image.isPrimary)
    ?? images[0]
    ?? null
}

/**
 * Normalizes diameter to a valid positive number, using default if missing or invalid
 */
export function normalizeDiameterMm(
  diameterMm: number | null | undefined,
  defaultDiameterMm: number
): number {
  if (diameterMm == null || diameterMm <= 0) {
    return defaultDiameterMm
  }
  return diameterMm
}

export function hasKnownDiameterMm(diameterMm: number | null | undefined): boolean {
  return diameterMm != null && diameterMm > 0
}

export function selectTrayCoinImagePath(images: readonly TrayCoinImage[]): string | null {
  return selectTrayCoinImage(images)?.filePath ?? null
}

/**
 * Calculates render size in pixels for a coin based on its diameter relative to all coins
 * Scales proportionally within min/max bounds
 */
export function getCoinRenderSizePx(
  diameterMm: number,
  allDiameters: number[],
  options: TrayLayoutOptions
): number {
  const { minCoinPx, maxCoinPx } = options

  // Handle edge cases: empty array or single value
  if (allDiameters.length === 0) {
    return minCoinPx + (maxCoinPx - minCoinPx) / 2
  }

  const maxDiameter = Math.max(...allDiameters)

  if (maxDiameter <= 0) {
    return minCoinPx + (maxCoinPx - minCoinPx) / 2
  }

  const size = (diameterMm / maxDiameter) * maxCoinPx

  // Clamp to bounds
  return Math.max(minCoinPx, Math.min(maxCoinPx, size))
}

/**
 * Returns the coins for a specific drawer (page of results)
 */
export function getDrawerCoins(
  coins: TrayCoin[],
  drawerIndex: number,
  coinsPerDrawer: number
): TrayCoin[] {
  if (drawerIndex < 0) return []
  const start = drawerIndex * coinsPerDrawer
  const end = start + coinsPerDrawer
  return coins.slice(start, end)
}

/**
 * Calculates total number of drawers needed for pagination
 */
export function getTotalDrawers(totalCoins: number, coinsPerDrawer: number): number {
  if (totalCoins === 0) return 0
  return Math.ceil(totalCoins / coinsPerDrawer)
}
