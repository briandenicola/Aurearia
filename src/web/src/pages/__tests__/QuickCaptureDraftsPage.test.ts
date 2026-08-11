import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import QuickCaptureDraftsPage from '../QuickCaptureDraftsPage.vue'
import { listQuickCaptureDrafts } from '@/api/client'
import type { QuickCaptureDraft } from '@/types'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const source = fs.readFileSync(path.resolve(__dirname, '../QuickCaptureDraftsPage.vue'), 'utf8')

vi.mock('@/api/client', () => ({
  getApiErrorMessage: vi.fn(() => ''),
  listQuickCaptureDrafts: vi.fn(),
}))

vi.mock('@/composables/usePwa', () => ({
  usePwa: () => ({ isPwa: false }),
}))

describe('QuickCaptureDraftsPage', () => {
  beforeEach(() => {
    vi.mocked(listQuickCaptureDrafts).mockReset()
  })

  it('loads only active drafts and renders list/empty/loading/error states', () => {
    expect(source).toContain('listQuickCaptureDrafts')
    expect(source).toContain("status: 'active'")
    expect(source).toContain('limit: 50')
    expect(source).toContain('QuickCaptureDraftCard')
    expect(source).toContain('Loading drafts...')
    expect(source).toContain('No active drafts yet.')
    expect(source).toContain('Unable to load quick capture drafts.')
    expect(source).toContain('<h1>Quick Capture</h1>')
    expect(source).not.toContain('Quick Capture Drafts')
    expect(source).not.toContain('New Draft')
    expect(source).toContain('CirclePlus')
    expect(source).toContain('aria-label="New capture"')
  })

  it('renders retained Numista chips from the owner-scoped list response and omits absent selections', async () => {
    vi.mocked(listQuickCaptureDrafts).mockResolvedValue({
      data: {
        drafts: [
          draft(17, {
            workingTitle: 'Referenced draft',
            selectedNumistaReference: {
              catalog: 'Numista',
              number: '12345',
              uri: 'https://en.numista.com/catalogue/pieces12345.html',
            },
          }),
          draft(18, { workingTitle: 'Unreferenced draft' }),
        ],
        total: 2,
        page: 1,
        limit: 50,
      },
    } as never)

    const wrapper = mount(QuickCaptureDraftsPage, {
      global: {
        stubs: {
          RouterLink: {
            props: ['to'],
            template: '<a :href="to"><slot /></a>',
          },
          AuthenticatedImage: true,
          CirclePlus: true,
          Plus: true,
        },
      },
    })
    await flushPromises()

    expect(listQuickCaptureDrafts).toHaveBeenCalledWith({ status: 'active', limit: 50 })
    const cards = wrapper.findAll('a[href^="/quick-capture/drafts/"]')
    expect(cards).toHaveLength(2)
    expect(cards[0]!.text()).toContain('Referenced draft')
    expect(cards[0]!.text()).toContain('Numista #12345')
    expect(cards[1]!.text()).toContain('Unreferenced draft')
    expect(cards[1]!.text()).not.toContain('Numista #')
  })
})

function draft(id: number, overrides: Partial<QuickCaptureDraft> = {}): QuickCaptureDraft {
  return {
    id,
    userId: 7,
    workingTitle: `Draft ${id}`,
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
