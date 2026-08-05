import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import CoinDetailHeaderActions from '../CoinDetailHeaderActions.vue'

const pushMock = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
}))

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

describe('CoinDetailHeaderActions', () => {
  it('routes back to wishlist gallery for wishlist items', async () => {
    pushMock.mockReset()
    const wrapper = mount(CoinDetailHeaderActions, {
      props: {
        isWishlist: true,
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

    await wrapper.find('button').trigger('click')
    expect(pushMock).toHaveBeenCalledWith('/wishlist')
  })

  it('routes back to collection gallery for non-wishlist items', async () => {
    pushMock.mockReset()
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

    await wrapper.find('button').trigger('click')
    expect(pushMock).toHaveBeenCalledWith('/')
  })

  it('keeps primary icons visible and moves sell/copy/details into overflow menu', async () => {
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
    expect(wrapper.find('button[aria-label="Edit"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="Delete"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="Open overflow actions"]').exists()).toBe(true)

    await wrapper.find('button[aria-label="Share"]').trigger('click')

    expect(wrapper.emitted('share')).toHaveLength(1)

    await wrapper.find('button[aria-label="Open overflow actions"]').trigger('click')
    expect(wrapper.find('button[aria-label="Sell Coin"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="Copy Coin"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Activity Journal')
    expect(wrapper.text()).toContain('Value Trend')
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

  it('emits duplicate from overflow and disables copy while pending', async () => {
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

    await activeWrapper.find('button[aria-label="Open overflow actions"]').trigger('click')
    await activeWrapper.find('button[aria-label="Copy Coin"]').trigger('click')

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

    await pendingWrapper.find('button[aria-label="Open overflow actions"]').trigger('click')
    const duplicateButton = pendingWrapper.find('button[aria-label="Copying coin..."]')
    expect(duplicateButton.attributes('disabled')).toBeDefined()
  })

  it('emits edit from the primary action row', async () => {
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
          Menu: true,
          Pencil: true,
          Share2: true,
          Trash2: true,
        },
      },
    })

    await wrapper.find('button[aria-label="Edit"]').trigger('click')
    expect(wrapper.emitted('edit')).toHaveLength(1)
  })
})
