import { describe, expect, it } from 'vitest'
import {
  buildDirectNumistaEvidence,
  buildDirectNumistaQuery,
  getNumistaStatusGuidance,
  isSelectionOutsideResults,
  retainNumistaSelection,
} from '../numistaLookup'
import { makeNumistaCandidate } from '@/test/numista-fixtures'
import type { Coin } from '@/types'

const coin: Pick<
  Coin,
  'name' | 'ruler' | 'denomination' | 'mint' | 'dateRange' | 'era' | 'material' | 'obverseInscription' | 'reverseInscription'
> = {
  name: 'Antoninus Pius denarius',
  ruler: 'Antoninus Pius',
  denomination: 'Denarius',
  mint: 'Rome',
  dateRange: '138–161 CE',
  era: 'Ancient',
  material: 'Silver',
  obverseInscription: 'ANTONINVS AVG PIVS',
  reverseInscription: 'TR POT COS III',
}

describe('numistaLookup helpers', () => {
  it('builds direct evidence from the coin date range without substituting era', () => {
    expect(buildDirectNumistaEvidence(coin)).toEqual({
      title: coin.name,
      issuer: coin.ruler,
      denomination: coin.denomination,
      mint: coin.mint,
      dateText: coin.dateRange,
      material: coin.material,
      obverseInscription: coin.obverseInscription,
      reverseInscription: coin.reverseInscription,
    })
    expect(buildDirectNumistaQuery(coin)).toBe(
      'Antoninus Pius denarius Antoninus Pius Denarius Rome 138–161 CE Silver ANTONINVS AVG PIVS TR POT COS III',
    )
  })

  it('omits nullable or partial date evidence without fabricating a replacement', () => {
    expect(buildDirectNumistaEvidence({ ...coin, dateRange: '' }).dateText).toBeUndefined()
    expect(buildDirectNumistaQuery({ ...coin, dateRange: '' })).not.toContain(coin.era)
    expect(buildDirectNumistaEvidence({ ...coin, dateRange: 'c. 138' }).dateText).toBe('c. 138')
  })

  it('omits empty values without rewriting collector source text', () => {
    expect(buildDirectNumistaQuery({
      ...coin,
      ruler: '',
      mint: '  Lugdunum?  ',
      obverseInscription: '',
    })).toBe('Antoninus Pius denarius Denarius Lugdunum? 138–161 CE Silver TR POT COS III')
  })

  it('bounds the visible query to the contract maximum', () => {
    const query = buildDirectNumistaQuery({
      ...coin,
      name: 'A'.repeat(300),
      ruler: 'B'.repeat(300),
      reverseInscription: 'C'.repeat(600),
    })
    expect(query).toHaveLength(500)
    expect(query.startsWith('A'.repeat(200))).toBe(true)
  })

  it('retains an explicit selection when retry results replace or omit it', () => {
    const selected = makeNumistaCandidate({ id: 10, title: 'Selected candidate' })
    const refreshed = makeNumistaCandidate({ id: 10, title: 'Refreshed candidate' })
    const replacement = retainNumistaSelection(selected, [refreshed])
    expect(replacement).toBe(refreshed)

    const omitted = retainNumistaSelection(selected, [makeNumistaCandidate({ id: 11 })])
    expect(omitted).toBe(selected)
    expect(isSelectionOutsideResults(omitted, [makeNumistaCandidate({ id: 11 })])).toBe(true)
  })

  it('maps unconfigured guidance without exposing settings to non-admin users', () => {
    expect(getNumistaStatusGuidance('unconfigured', false)).toEqual({
      state: 'unconfigured',
      label: 'Setup needed',
      title: 'Numista lookup is not configured',
      message: 'Numista lookup is not available on this instance. Ask an administrator for help.',
      canRetry: false,
      settingsHref: undefined,
    })
    expect(getNumistaStatusGuidance('unconfigured', true).settingsHref).toBe('/admin?tab=system')
  })
})
