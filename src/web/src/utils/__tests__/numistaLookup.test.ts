import { describe, expect, it } from 'vitest'
import {
  buildDirectNumistaEvidence,
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
  it('keeps rich direct evidence available for the server-owned proposal and scorer', () => {
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
  })

  it('omits nullable or partial date evidence without fabricating a replacement', () => {
    expect(buildDirectNumistaEvidence({ ...coin, dateRange: '' }).dateText).toBeUndefined()
    expect(buildDirectNumistaEvidence({ ...coin, dateRange: 'c. 138' }).dateText).toBe('c. 138')
  })

  it('omits empty evidence values without rewriting collector source text', () => {
    expect(buildDirectNumistaEvidence({
      ...coin,
      ruler: '',
      mint: '  Lugdunum?  ',
      obverseInscription: '',
    })).toEqual({
      title: coin.name,
      issuer: undefined,
      denomination: coin.denomination,
      mint: 'Lugdunum?',
      dateText: coin.dateRange,
      material: coin.material,
      obverseInscription: undefined,
      reverseInscription: coin.reverseInscription,
    })
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
