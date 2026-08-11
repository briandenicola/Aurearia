import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import CoinReferencesSection from '../CoinReferencesSection.vue'
import { createCoinReference, listCatalogs } from '@/api/client'
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
    uri,
    createdAt: '2026-06-19T00:00:00Z',
    updatedAt: '2026-06-19T00:00:00Z',
  }
}

describe('CoinReferencesSection', () => {
  beforeEach(() => {
    vi.mocked(listCatalogs).mockReset()
    vi.mocked(listCatalogs).mockResolvedValue([{
      id: 1,
      catalog: 'RIC',
      displayName: 'Roman Imperial Coinage',
      era: 'ancient',
      volumeRequired: false,
    }])
    vi.mocked(createCoinReference).mockReset()
  })

  it('renders only safe external reference links', () => {
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

  it('keeps manual entry and Search Numista as compact peer actions', async () => {
    const wrapper = mountSection()
    const addReference = buttonByText(wrapper, 'Add Reference')

    expect(addReference.classes()).toContain('btn-sm')

    await addReference.trigger('click')
    expect(wrapper.get('form').isVisible()).toBe(true)

    await wrapper.get('select').setValue('RIC')
    const inputs = wrapper.findAll('input')
    await inputs.find(input => input.attributes('placeholder') === 'Number')!.setValue('123')
    vi.mocked(createCoinReference).mockResolvedValue({
      data: {
        ...reference(9, 'https://example.com/ric/123'),
        number: '123',
      },
    } as never)
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(createCoinReference).toHaveBeenCalledWith(42, expect.objectContaining({
      catalog: 'RIC',
      number: '123',
    }))
    expect(wrapper.text()).toContain('RIC')
    expect(wrapper.text()).toContain('123')

    const searchNumista = buttonByText(wrapper, 'Search Numista')
    expect(searchNumista.classes()).toContain('btn-sm')
  })

  it('exposes an inline keyboard disclosure and moves focus into the editable lookup', async () => {
    const wrapper = mountSection(true)
    const disclosure = buttonByText(wrapper, 'Search Numista')

    expect(disclosure.attributes('aria-expanded')).toBe('false')
    expect(disclosure.attributes('aria-controls')).toBeTruthy()

    disclosure.element.focus()
    await disclosure.trigger('keydown', { key: 'Enter' })
    await flushPromises()

    expect(disclosure.attributes('aria-expanded')).toBe('true')
    expect(wrapper.get('[data-testid="numista-panel"]').isVisible()).toBe(true)
    expect(document.activeElement).toBe(wrapper.get('#numista-query').element)

    wrapper.unmount()
  })

  it('collapses after persistence and matching refresh, then returns focus to Search Numista', async () => {
    const existing = reference(1, 'https://example.com/ric/1')
    const wrapper = mountSection(true, [existing])
    const disclosure = buttonByText(wrapper, 'Search Numista')

    await disclosure.trigger('click')
    const persistReference = wrapper.get('[data-testid="confirm-numista"]')
    persistReference.element.focus()
    await persistReference.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('changed')).toHaveLength(1)
    expect(disclosure.attributes('aria-expanded')).toBe('true')
    expect(document.activeElement).toBe(persistReference.element)

    await wrapper.setProps({ references: [...wrapper.props('references')] })
    expect(disclosure.attributes('aria-expanded')).toBe('true')
    expect(document.activeElement).toBe(persistReference.element)

    await wrapper.setProps({
      references: [
        existing,
        {
          ...reference(2, 'https://en.numista.com/catalogue/pieces12345.html'),
          catalog: 'Numista',
          number: '12345',
        },
      ],
    })

    expect(disclosure.attributes('aria-expanded')).toBe('false')
    expect(wrapper.find('[data-testid="numista-panel"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('Numista')
    expect(wrapper.text()).toContain('12345')
    expect(document.activeElement).toBe(disclosure.element)
  })

  it('keeps the lookup expanded and leaves focus in place when persistence fails', async () => {
    const wrapper = mountSection(true)
    const disclosure = buttonByText(wrapper, 'Search Numista')

    await disclosure.trigger('click')
    const failedPersistence = wrapper.get('[data-testid="failed-numista"]')
    failedPersistence.element.focus()
    await failedPersistence.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('changed')).toBeUndefined()
    expect(disclosure.attributes('aria-expanded')).toBe('true')
    expect(wrapper.get('[data-testid="numista-panel"]').isVisible()).toBe(true)
    expect(document.activeElement).toBe(failedPersistence.element)
  })
})

function mountSection(attach = false, references: CoinReference[] = []) {
  return mount(CoinReferencesSection, {
    attachTo: attach ? document.body : undefined,
    props: {
      coinId: 42,
      references,
    },
    global: {
      stubs: {
        CoinNumistaPanel: {
          emits: ['referenceAdded'],
          template: `
            <div data-testid="numista-panel">
              <textarea id="numista-query"></textarea>
              <button data-testid="confirm-numista" @click="$emit('referenceAdded', { id: 12345 })">Add selected reference</button>
              <button data-testid="failed-numista">Retry failed reference</button>
            </div>
          `,
        },
      },
    },
  })
}

function buttonByText(wrapper: ReturnType<typeof mount>, text: string) {
  const button = wrapper.findAll('button').find(candidate => candidate.text().includes(text))
  expect(button, `button containing "${text}"`).toBeDefined()
  return button!
}
