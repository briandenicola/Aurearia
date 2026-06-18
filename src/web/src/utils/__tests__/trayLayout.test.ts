import { describe, it, expect } from 'vitest'
import { normalizeDiameterMm, getCoinRenderSizePx, getDrawerCoins, getTotalDrawers } from '../trayLayout'
import type { TrayCoin } from '../trayLayout'

describe('trayLayout', () => {
  describe('normalizeDiameterMm', () => {
    it('returns diameterMm when positive', () => {
      expect(normalizeDiameterMm(25, 20)).toBe(25)
      expect(normalizeDiameterMm(35, 20)).toBe(35)
    })

    it('returns default when null', () => {
      expect(normalizeDiameterMm(null, 20)).toBe(20)
    })

    it('returns default when undefined', () => {
      expect(normalizeDiameterMm(undefined, 20)).toBe(20)
    })

    it('returns default when zero', () => {
      expect(normalizeDiameterMm(0, 20)).toBe(20)
    })

    it('returns default when negative', () => {
      expect(normalizeDiameterMm(-5, 20)).toBe(20)
    })
  })

  describe('getCoinRenderSizePx', () => {
    const options = {
      minCoinPx: 40,
      maxCoinPx: 120,
      defaultDiameterMm: 20,
    }

    it('scales small coins smaller than large coins', () => {
      const allDiameters = [8, 20, 35]
      const smallSize = getCoinRenderSizePx(8, allDiameters, options)
      const largeSize = getCoinRenderSizePx(35, allDiameters, options)
      expect(smallSize).toBeLessThan(largeSize)
    })

    it('clamps to minimum size', () => {
      const allDiameters = [8, 20, 35]
      const size = getCoinRenderSizePx(8, allDiameters, options)
      expect(size).toBeGreaterThanOrEqual(options.minCoinPx)
    })

    it('clamps to maximum size', () => {
      const allDiameters = [8, 20, 50]
      const size = getCoinRenderSizePx(50, allDiameters, options)
      expect(size).toBeLessThanOrEqual(options.maxCoinPx)
    })

    it('handles single coin', () => {
      const allDiameters = [25]
      const size = getCoinRenderSizePx(25, allDiameters, options)
      expect(size).toBeGreaterThanOrEqual(options.minCoinPx)
      expect(size).toBeLessThanOrEqual(options.maxCoinPx)
    })

    it('handles all coins same diameter', () => {
      const allDiameters = [20, 20, 20]
      const size = getCoinRenderSizePx(20, allDiameters, options)
      expect(size).toBeGreaterThanOrEqual(options.minCoinPx)
      expect(size).toBeLessThanOrEqual(options.maxCoinPx)
    })

    it('handles empty diameter array', () => {
      const size = getCoinRenderSizePx(20, [], options)
      expect(size).toBeGreaterThanOrEqual(options.minCoinPx)
      expect(size).toBeLessThanOrEqual(options.maxCoinPx)
    })

    it('proportionally scales intermediate sizes', () => {
      const allDiameters = [10, 20, 30]
      const smallSize = getCoinRenderSizePx(10, allDiameters, options)
      const mediumSize = getCoinRenderSizePx(20, allDiameters, options)
      const largeSize = getCoinRenderSizePx(30, allDiameters, options)
      
      expect(mediumSize).toBeGreaterThan(smallSize)
      expect(largeSize).toBeGreaterThan(mediumSize)
      expect(mediumSize - smallSize).toBeCloseTo(largeSize - mediumSize, 0)
    })
  })

  describe('getDrawerCoins', () => {
    const coins: TrayCoin[] = Array.from({ length: 100 }, (_, i) => ({
      id: i + 1,
      name: `Coin ${i + 1}`,
      diameterMm: 20,
      images: [],
    }))

    it('returns first 50 coins for drawer 0', () => {
      const drawerCoins = getDrawerCoins(coins, 0, 50)
      expect(drawerCoins).toHaveLength(50)
      expect(drawerCoins[0]?.id).toBe(1)
      expect(drawerCoins[49]?.id).toBe(50)
    })

    it('returns second 50 coins for drawer 1', () => {
      const drawerCoins = getDrawerCoins(coins, 1, 50)
      expect(drawerCoins).toHaveLength(50)
      expect(drawerCoins[0]?.id).toBe(51)
      expect(drawerCoins[49]?.id).toBe(100)
    })

    it('returns partial drawer for last drawer with remainder', () => {
      const coinsSmall: TrayCoin[] = Array.from({ length: 101 }, (_, i) => ({
        id: i + 1,
        name: `Coin ${i + 1}`,
        diameterMm: 20,
        images: [],
      }))
      const drawerCoins = getDrawerCoins(coinsSmall, 2, 50)
      expect(drawerCoins).toHaveLength(1)
      expect(drawerCoins[0]?.id).toBe(101)
    })

    it('returns empty array for out-of-bounds drawer index', () => {
      const drawerCoins = getDrawerCoins(coins, 10, 50)
      expect(drawerCoins).toHaveLength(0)
    })

    it('returns empty array for negative drawer index', () => {
      const drawerCoins = getDrawerCoins(coins, -1, 50)
      expect(drawerCoins).toHaveLength(0)
    })

    it('handles empty coin array', () => {
      const drawerCoins = getDrawerCoins([], 0, 50)
      expect(drawerCoins).toHaveLength(0)
    })
  })

  describe('getTotalDrawers', () => {
    it('returns 2 drawers for 100 coins with 50 per drawer', () => {
      expect(getTotalDrawers(100, 50)).toBe(2)
    })

    it('returns 3 drawers for 101 coins with 50 per drawer', () => {
      expect(getTotalDrawers(101, 50)).toBe(3)
    })

    it('returns 1 drawer for 25 coins with 50 per drawer', () => {
      expect(getTotalDrawers(25, 50)).toBe(1)
    })

    it('returns 0 drawers for 0 coins', () => {
      expect(getTotalDrawers(0, 50)).toBe(0)
    })

    it('handles single coin per drawer', () => {
      expect(getTotalDrawers(5, 1)).toBe(5)
    })
  })
})
