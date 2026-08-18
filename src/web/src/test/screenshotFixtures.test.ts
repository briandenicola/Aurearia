import { describe, expect, it, vi } from 'vitest'
import type { APIRequestContext } from '@playwright/test'
import { ensureScreenshotFixtures, SCREENSHOT_FIXTURE_COINS, SCREENSHOT_PREFIX } from '../../e2e/screenshots/fixtures'

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

  it('every fixture name carries the distinctive [Screenshot] prefix and no price/valuation data', async () => {
    for (const fixture of SCREENSHOT_FIXTURE_COINS) {
      expect(fixture.name.startsWith(SCREENSHOT_PREFIX)).toBe(true)
    }
  })
})
