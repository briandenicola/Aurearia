import { describe, it, expect } from 'vitest'
import { readFileSync } from 'fs'
import { join } from 'path'

const WEB_ROOT = join(__dirname, '..', '..')
const VITE_CONFIG = join(WEB_ROOT, 'vite.config.ts')

describe('PWA private media cache policy', () => {
  it('does not runtime CacheFirst cache uploaded media', () => {
    const config = readFileSync(VITE_CONFIG, 'utf-8')

    expect(config).not.toContain("cacheName: 'coin-images'")
    expect(config).not.toMatch(/urlPattern:\s*\/\\\/uploads\\\//)
    expect(config).not.toContain("cacheName: 'private-media-cache'")
    expect(config).not.toMatch(/api\\\/uploads/)
  })
})

describe('PWA write-request policy', () => {
  // Regression: routing non-GET /api/ requests through the service worker made
  // iOS WebKit drop multipart bodies, so coin image uploads reached the API
  // with an empty body ("multipart: NextPart: EOF") and the edit page reported
  // a failed save. `NetworkOnly` added nothing over letting the browser issue
  // the request itself, so no route may opt writes into the worker at all.
  it('does not route non-GET API requests through the service worker', () => {
    const config = readFileSync(VITE_CONFIG, 'utf-8')

    expect(config).not.toMatch(/method:\s*'(POST|PUT|DELETE|PATCH)'/)
  })
})

describe('PWA update notification policy', () => {
  it('uses prompt-based service worker updates so the update banner can appear', () => {
    const config = readFileSync(VITE_CONFIG, 'utf-8')

    expect(config).toContain("registerType: 'prompt'")
    expect(config).toContain('skipWaiting: false')
    expect(config).toContain('clientsClaim: false')
    expect(config).not.toContain("registerType: 'autoUpdate'")
  })
})
