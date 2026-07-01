import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { AlertCandidate } from '@/types'
import CandidateReviewCard from '../CandidateReviewCard.vue'

function candidate(overrides: Partial<AlertCandidate> = {}): AlertCandidate {
  return {
    id: 7,
    userId: 1,
    alertId: 2,
    runId: 3,
    sourceUrl: 'https://www.vcoins.com/en/stores/example/123',
    canonicalSourceUrl: 'https://www.vcoins.com/en/stores/example/123',
    sourceName: 'VCoins',
    title: 'Pamphylia, Aspendos AR Stater',
    observedPrice: 680,
    observedCurrency: 'USD',
    reasonForMatch: 'Matches the alert criteria.',
    fields: {
      material: 'Silver',
      denomination: 'Stater',
      category: 'Greek',
      era: 'Ancient',
    },
    lastSeenAt: '2026-07-01T18:00:00Z',
    firstSeenAt: '2026-07-01T18:00:00Z',
    provenanceStatus: 'verified',
    lifecycleState: 'active',
    duplicateKey: 'duplicate-key',
    duplicateOfCandidateId: null,
    matchingWishlistCoinId: null,
    convertedCoinId: null,
    dismissalReason: '',
    provenance: [],
    createdAt: '',
    updatedAt: '',
    ...overrides,
  }
}

describe('CandidateReviewCard', () => {
  it('renders the conversion source as an external link while preserving the conversion payload URL', async () => {
    const wrapper = mount(CandidateReviewCard, {
      props: {
        candidate: candidate(),
      },
      global: {
        stubs: {
          RouterLink: true,
        },
      },
    })

    expect(wrapper.find('.convert-source input').exists()).toBe(false)

    const sourceLink = wrapper.find('.convert-source a')
    expect(sourceLink.exists()).toBe(true)
    expect(sourceLink.attributes('href')).toBe('https://www.vcoins.com/en/stores/example/123')
    expect(sourceLink.attributes('target')).toBe('_blank')

    await wrapper.find('.actions-row .btn-primary').trigger('click')
    const emitted = wrapper.emitted('convert')?.[0]
    expect(emitted?.[1]).toMatchObject({ referenceUrl: 'https://www.vcoins.com/en/stores/example/123' })
  })
})
