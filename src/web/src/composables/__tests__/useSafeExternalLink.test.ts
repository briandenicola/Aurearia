import { describe, expect, it } from 'vitest'
import { sanitizeExternalUrl, useSafeExternalLink } from '@/composables/useSafeExternalLink'

describe('sanitizeExternalUrl', () => {
  it('allows http and https links', () => {
    expect(sanitizeExternalUrl('https://example.com/coin/1')).toBe('https://example.com/coin/1')
    expect(sanitizeExternalUrl('http://example.com/listings')).toBe('http://example.com/listings')
    expect(sanitizeExternalUrl(' HTTPS://EXAMPLE.com ')).toBe('https://example.com/')
  })

  it('rejects non-http protocols', () => {
    expect(sanitizeExternalUrl('javascript:alert(1)')).toBeNull()
    expect(sanitizeExternalUrl('data:text/html,<script>alert(1)</script>')).toBeNull()
    expect(sanitizeExternalUrl('ftp://example.com/file')).toBeNull()
    expect(sanitizeExternalUrl('mailto:test@example.com')).toBeNull()
  })

  it('rejects invalid, relative, and empty values', () => {
    expect(sanitizeExternalUrl('/relative/path')).toBeNull()
    expect(sanitizeExternalUrl('   ')).toBeNull()
    expect(sanitizeExternalUrl('')).toBeNull()
    expect(sanitizeExternalUrl(null)).toBeNull()
    expect(sanitizeExternalUrl(undefined)).toBeNull()
  })
})

describe('useSafeExternalLink', () => {
  it('returns the sanitized external url', () => {
    expect(useSafeExternalLink('https://numisbids.com/n.php?p=lot&sid=1&lot=2')).toBe(
      'https://numisbids.com/n.php?p=lot&sid=1&lot=2',
    )
    expect(useSafeExternalLink('javascript:alert(1)')).toBeNull()
  })
})
