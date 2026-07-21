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

describe('PWA update notification policy', () => {
  it('uses prompt-based service worker updates so the update banner can appear', () => {
    const config = readFileSync(VITE_CONFIG, 'utf-8')

    expect(config).toContain("registerType: 'prompt'")
    expect(config).toContain('skipWaiting: false')
    expect(config).toContain('clientsClaim: false')
    expect(config).not.toContain("registerType: 'autoUpdate'")
  })
})
