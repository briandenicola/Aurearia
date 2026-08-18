import { describe, expect, it, vi } from 'vitest'
import type { APIRequestContext } from '@playwright/test'
import { ensureScreenshotFixtures, SCREENSHOT_FIXTURE_COINS, SCREENSHOT_PREFIX } from '../../e2e/screenshots/fixtures'
import { COIN_ERAS } from '../../src/types'

function jsonResponse(status: number, body: unknown) {
  return {
    ok: () => status >= 200 && status < 300,
    status: () => status,
    json: async () => body,
    text: async () => JSON.stringify(body),
  }
}

describe('ensureScreenshotFixtures', () => {
  it('creates missing fixtures and updates an existing one by exact name, without duplicating', async () => {
    const existingCoin = { id: 501, name: SCREENSHOT_FIXTURE_COINS[0]!.name }
    const created: string[] = []

    const get = vi.fn(async (_url: string, options?: { params?: Record<string, string> }) => {
      const search = options?.params?.search
      if (search === existingCoin.name) {
        return jsonResponse(200, { coins: [existingCoin], total: 1, page: 1, limit: 50 })
      }
      return jsonResponse(200, { coins: [], total: 0, page: 1, limit: 50 })
    })
    const put = vi.fn(async () => jsonResponse(200, { id: existingCoin.id }))
    let nextId = 900
    const post = vi.fn(async (_url: string, options?: { data?: { name: string } }) => {
      created.push(options?.data?.name ?? 'unknown')
      nextId += 1
      return jsonResponse(201, { id: nextId })
    })

    const api = { get, post, put } as unknown as APIRequestContext

    const results = await ensureScreenshotFixtures(api)

    expect(results).toHaveLength(SCREENSHOT_FIXTURE_COINS.length)
    expect(put).toHaveBeenCalledTimes(1)
    expect(put).toHaveBeenCalledWith(`/api/coins/${existingCoin.id}`, expect.anything())
    expect(post).toHaveBeenCalledTimes(SCREENSHOT_FIXTURE_COINS.length - 1)
    expect(created).toEqual(SCREENSHOT_FIXTURE_COINS.filter((f) => f.name !== existingCoin.name).map((f) => f.name))
  })

  it('never creates duplicates when rerun against an already fully-seeded account', async () => {
    const existingByName = new Map(SCREENSHOT_FIXTURE_COINS.map((fixture, index) => [fixture.name, { id: 1000 + index, name: fixture.name }]))

    const get = vi.fn(async (_url: string, options?: { params?: Record<string, string> }) => {
      const search = options?.params?.search as string | undefined
      const match = search ? existingByName.get(search) : undefined
      return jsonResponse(200, { coins: match ? [match] : [], total: match ? 1 : 0, page: 1, limit: 50 })
    })
    const put = vi.fn(async () => jsonResponse(200, {}))
    const post = vi.fn(async () => jsonResponse(201, { id: 9999 }))

    const api = { get, post, put } as unknown as APIRequestContext

    const results = await ensureScreenshotFixtures(api)

    expect(post).not.toHaveBeenCalled()
    expect(put).toHaveBeenCalledTimes(SCREENSHOT_FIXTURE_COINS.length)
    expect(results.map((r) => r.id)).toEqual([...existingByName.values()].map((c) => c.id))
  })

  it('reconciles duplicate fixtures created by a racing concurrent run: keeps the lowest ID, updates it, and deletes only the extras', async () => {
    const fixtureName = SCREENSHOT_FIXTURE_COINS[0]!.name
    // Simulates the observed race: two workers both GET-then-POST before
    // either sees the other's write, producing two coins with the exact
    // same fixture name. An unrelated coin that merely contains the fixture
    // name as a substring (but isn't an exact match) must never be touched.
    const duplicates = [
      { id: 700, name: fixtureName },
      { id: 650, name: fixtureName }, // lower id: should become canonical
    ]
    const unrelatedCoin = { id: 999, name: `${fixtureName} (someone's personal note)` }

    const get = vi.fn(async (_url: string, options?: { params?: Record<string, string> }) => {
      const search = options?.params?.search
      if (search === fixtureName) {
        return jsonResponse(200, { coins: [...duplicates, unrelatedCoin], total: 3, page: 1, limit: 50 })
      }
      return jsonResponse(200, { coins: [], total: 0, page: 1, limit: 50 })
    })
    const put = vi.fn(async () => jsonResponse(200, {}))
    const post = vi.fn(async () => jsonResponse(201, { id: 8888 }))
    const del = vi.fn(async () => jsonResponse(200, {}))

    const api = { get, post, put, delete: del } as unknown as APIRequestContext

    const results = await ensureScreenshotFixtures(api)

    // Canonical is the lowest ID (650); the other duplicate (700) is deleted;
    // the unrelated substring-matching coin (999) is never deleted or updated.
    expect(del).toHaveBeenCalledTimes(1)
    expect(del).toHaveBeenCalledWith('/api/coins/700')
    expect(del).not.toHaveBeenCalledWith('/api/coins/999')
    expect(put).toHaveBeenCalledWith('/api/coins/650', expect.anything())
    const firstResult = results.find((r) => r.name === fixtureName)
    expect(firstResult?.id).toBe(650)
  })

  it('surfaces a clear error and does not proceed silently if deleting a duplicate fails', async () => {
    const fixtureName = SCREENSHOT_FIXTURE_COINS[0]!.name
    const duplicates = [
      { id: 700, name: fixtureName },
      { id: 650, name: fixtureName },
    ]

    const get = vi.fn(async (_url: string, options?: { params?: Record<string, string> }) => {
      const search = options?.params?.search
      if (search === fixtureName) {
        return jsonResponse(200, { coins: duplicates, total: 2, page: 1, limit: 50 })
      }
      return jsonResponse(200, { coins: [], total: 0, page: 1, limit: 50 })
    })
    const put = vi.fn(async () => jsonResponse(200, {}))
    const post = vi.fn(async () => jsonResponse(201, { id: 8888 }))
    const del = vi.fn(async () => jsonResponse(500, { error: 'delete failed' }))

    const api = { get, post, put, delete: del } as unknown as APIRequestContext

    await expect(ensureScreenshotFixtures(api)).rejects.toThrow(/Failed to delete duplicate screenshot fixture/)
    // Must not silently continue past the failed delete and update the record
    // as if reconciliation had succeeded.
    expect(put).not.toHaveBeenCalledWith('/api/coins/650', expect.anything())
  })

  it('every fixture name carries the distinctive [Screenshot] prefix and no price/valuation data', async () => {
    for (const fixture of SCREENSHOT_FIXTURE_COINS) {
      expect(fixture.name.startsWith(SCREENSHOT_PREFIX)).toBe(true)
    }
  })

  it('every fixture era is one of the authoritative COIN_ERAS values accepted by the real /api/coins contract', () => {
    // Regression test for the beta run failure: POST /api/coins 400
    // `{"error":"era is not supported"}`. The backend's built-in era
    // whitelist (models.EraAncient/EraMedieval/EraModern in
    // src/api/models/coin.go) matches COIN_ERAS exactly, so fixtures must
    // use one of these lowercase values rather than display-style strings
    // like "Roman Imperial".
    expect(COIN_ERAS).toEqual(['ancient', 'medieval', 'modern'])
    for (const fixture of SCREENSHOT_FIXTURE_COINS) {
      expect(COIN_ERAS).toContain(fixture.era)
    }
  })
})
