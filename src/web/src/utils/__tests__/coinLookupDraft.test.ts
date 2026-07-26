import { describe, expect, it } from 'vitest'
import { normalizedEra } from '../coinLookupDraft'

describe('normalizedEra', () => {
  it('trims and passes through any non-empty era string, not just legacy defaults', () => {
    expect(normalizedEra('ancient')).toBe('ancient')
    expect(normalizedEra('  Roman Provincial Year 12  ')).toBe('Roman Provincial Year 12')
    expect(normalizedEra('Sassanian')).toBe('Sassanian')
  })

  it('returns undefined for blank or non-string values', () => {
    expect(normalizedEra('')).toBeUndefined()
    expect(normalizedEra('   ')).toBeUndefined()
    expect(normalizedEra(undefined)).toBeUndefined()
    expect(normalizedEra(42)).toBeUndefined()
  })
})
