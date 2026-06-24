import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import CoinDetailHeaderActions from '../CoinDetailHeaderActions.vue'

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

describe('CoinDetailHeaderActions', () => {
  it('emits share and keeps existing actions available', async () => {
    const wrapper = mount(CoinDetailHeaderActions, {
      props: {
        isWishlist: false,
        isSold: false,
        coinId: 42,
      },
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          ArrowLeft: true,
          CircleDollarSign: true,
          Copy: true,
          Pencil: true,
          Share2: true,
          Trash2: true,
        },
      },
    })

    expect(wrapper.find('button[aria-label="Share"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="Sell"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/edit/42"]').exists()).toBe(true)
    expect(wrapper.find('a[aria-label="Edit"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="Duplicate"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="Delete"]').exists()).toBe(true)

    await wrapper.find('button[aria-label="Share"]').trigger('click')

    expect(wrapper.emitted('share')).toHaveLength(1)
  })

  it('disables the share button while a share card is being generated', () => {
    const wrapper = mount(CoinDetailHeaderActions, {
      props: {
        isWishlist: false,
        isSold: false,
        coinId: 42,
        sharing: true,
      },
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          ArrowLeft: true,
          CircleDollarSign: true,
          Copy: true,
          Pencil: true,
          Share2: true,
          Trash2: true,
        },
      },
    })

    const shareButton = wrapper.find('button[aria-label="Sharing..."]')
    expect(shareButton.attributes('disabled')).toBeDefined()
  })

  it('emits duplicate and disables while duplicate is pending', async () => {
    const activeWrapper = mount(CoinDetailHeaderActions, {
      props: {
        isWishlist: false,
        isSold: false,
        coinId: 42,
      },
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          ArrowLeft: true,
          CircleDollarSign: true,
          Copy: true,
          Pencil: true,
          Share2: true,
          Trash2: true,
        },
      },
    })

    await activeWrapper.find('button[aria-label="Duplicate"]').trigger('click')

    expect(activeWrapper.emitted('duplicate')).toHaveLength(1)

    const pendingWrapper = mount(CoinDetailHeaderActions, {
      props: {
        isWishlist: false,
        isSold: false,
        coinId: 42,
        duplicating: true,
      },
      global: {
        stubs: {
          RouterLink: routerLinkStub,
          ArrowLeft: true,
          CircleDollarSign: true,
          Copy: true,
          Pencil: true,
          Share2: true,
          Trash2: true,
        },
      },
    })

    const duplicateButton = pendingWrapper.find('button[aria-label="Duplicating..."]')
    expect(duplicateButton.attributes('disabled')).toBeDefined()
  })
})
