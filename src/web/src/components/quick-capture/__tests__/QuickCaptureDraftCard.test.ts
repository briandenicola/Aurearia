import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import QuickCaptureDraftCard from '../QuickCaptureDraftCard.vue'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'
import type { QuickCaptureDraft } from '@/types'

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

describe('QuickCaptureDraftCard', () => {
  it('renders the exact retained Numista identifier as a wrapping chip inside the resume link', () => {
    const number = '123456789012345678901234567890'
    const wrapper = mountCard(draft({
      workingTitle: 'Long referenced draft',
      selectedNumistaReference: {
        catalog: 'Numista',
        number,
        uri: `https://en.numista.com/catalogue/pieces${number}.html`,
      },
    }))

    const link = wrapper.get('a')
    expect(link.attributes('href')).toBe('/quick-capture/drafts/17')
    expect(link.text()).toContain('Long referenced draft')

    const chip = wrapper.findAll('.chip-sm').find(item => item.text() === `Numista #${number}`)
    expect(chip).toBeDefined()
    expect(chip!.classes()).toContain('max-w-full')
    expect(chip!.classes()).toContain('whitespace-normal')
    expect(chip!.classes().some(name => name === 'break-all' || name === 'break-words')).toBe(true)
    expect(chip!.classes()).not.toContain('truncate')
    expect(wrapper.findAll('a')).toHaveLength(1)
  })

  it('omits the Numista chip when the list item has no retained selection', () => {
    const wrapper = mountCard(draft())

    expect(wrapper.text()).not.toContain('Numista #')
    expect(wrapper.get('a').attributes('href')).toBe('/quick-capture/drafts/17')
  })

  it('keeps owner-safe preview and fallback content available from the linked card', () => {
    const withImage = mountCard(draft({
      images: [{
        id: 1,
        draftId: 17,
        filePath: 'quick-capture-draft-17/obverse.jpg',
        imageType: 'obverse',
        isPrimary: true,
        displayOrder: 0,
        createdAt: '2026-08-11T12:00:00Z',
      }],
    }))
    expect(withImage.findComponent(AuthenticatedImage).props('mediaPath')).toBe(
      'quick-capture-draft-17/obverse.jpg',
    )

    const withoutImage = mountCard(draft())
    expect(withoutImage.get('a').text()).toContain('No image')
    expect(withoutImage.text()).toContain('Incomplete Quick Capture draft')
  })
})

function mountCard(value: QuickCaptureDraft) {
  return mount(QuickCaptureDraftCard, {
    props: { draft: value },
    global: {
      stubs: {
        RouterLink: routerLinkStub,
        AuthenticatedImage: true,
      },
    },
  })
}

function draft(overrides: Partial<QuickCaptureDraft> = {}): QuickCaptureDraft {
  return {
    id: 17,
    userId: 7,
    workingTitle: 'Untitled draft',
    dateRange: '',
    era: '',
    acquisitionSource: '',
    purchasePrice: null,
    notes: '',
    source: 'find_coin_ai',
    ngcCertNumber: '',
    ngcLookupUrl: '',
    ngcGrade: '',
    labelText: '',
    aiConfidence: '',
    status: 'active',
    promotedCoinId: null,
    promotedAt: null,
    discardedAt: null,
    images: [],
    createdAt: '2026-08-11T12:00:00Z',
    updatedAt: '2026-08-11T12:00:00Z',
    ...overrides,
  }
}
