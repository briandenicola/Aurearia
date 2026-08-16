import { describe, expect, it } from 'vitest'
import type { CoinLookupResponse } from '@/types'
import { deriveAiObservations, normalizedEra } from '../coinLookupDraft'

describe('normalizedEra', () => {
  it('trims and passes through any non-empty era string, not just legacy defaults', () => {
    expect(normalizedEra('ancient')).toBe('ancient')
    expect(normalizedEra('  Roman Provincial Year 12  ')).toBe('Roman Provincial Year 12')
    expect(normalizedEra('Sassanian')).toBe('Sassanian')
  })

  describe('deriveAiObservations', () => {
    it('keeps concise analysis while excluding internal model transport payloads', () => {
      const lookup = (rawAnalysis: string) => ({
        extractedData: { rawAnalysis },
      } as CoinLookupResponse)

      expect(deriveAiObservations(lookup('AI saw a silver denarius.'), {}))
        .toContain('AI saw a silver denarius.')

      const transport = "[{'type': 'thinking', 'thinking': 'inspect'}, {'type': 'image', 'source': {'type': 'base64', 'data': 'AAAA'}}]"
      expect(deriveAiObservations(lookup(transport), {})).toBe('')
    })
  })

  it('returns undefined for blank or non-string values', () => {
    expect(normalizedEra('')).toBeUndefined()
    expect(normalizedEra('   ')).toBeUndefined()
    expect(normalizedEra(undefined)).toBeUndefined()
    expect(normalizedEra(42)).toBeUndefined()
  })
})
