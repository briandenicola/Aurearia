import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import CoinReferencesSection from '../CoinReferencesSection.vue'
import { listCatalogs } from '@/api/client'
import type { CoinReference } from '@/types'

vi.mock('@/api/client', () => ({
  createCoinReference: vi.fn(),
  deleteCoinReference: vi.fn(),
  updateCoinReference: vi.fn(),
  listCatalogs: vi.fn(),
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({
    showAlert: vi.fn(),
    showConfirm: vi.fn(),
  }),
}))

function reference(id: number, uri: string): CoinReference {
  return {
    id,
    coinId: 42,
    catalog: 'RIC',
    volume: '',
    number: String(id),
    invoiceNumber: '',
    uri,
    createdAt: '2026-06-19T00:00:00Z',
    updatedAt: '2026-06-19T00:00:00Z',
  }
}

describe('CoinReferencesSection', () => {
  it('renders only safe external reference links', () => {
    vi.mocked(listCatalogs).mockResolvedValue([])

    const wrapper = mount(CoinReferencesSection, {
      props: {
        coinId: 42,
        references: [
          reference(1, 'javascript:alert(1)'),
          reference(2, 'data:text/html,<p>x</p>'),
          reference(3, '/relative/reference'),
          reference(4, 'http://example.com/reference'),
          reference(5, 'https://example.com/reference'),
        ],
      },
    })

    const links = wrapper.findAll('a.btn-ghost')
    expect(links.map(link => link.attributes('href'))).toEqual([
      'http://example.com/reference',
      'https://example.com/reference',
    ])
    expect(wrapper.html()).not.toContain('javascript:alert')
    expect(wrapper.html()).not.toContain('data:text/html')
    expect(wrapper.html()).not.toContain('/relative/reference')
  })
})
