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
          Share2: true,
        },
      },
    })

    expect(wrapper.text()).toContain('Share')
    expect(wrapper.text()).toContain('Sell')
    expect(wrapper.find('a[href="/edit/42"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Delete')

    await wrapper.findAll('button').find((button) => button.text().includes('Share'))!.trigger('click')

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
          Share2: true,
        },
      },
    })

    const shareButton = wrapper.findAll('button').find((button) => button.text().includes('Sharing...'))!
    expect(shareButton.attributes('disabled')).toBeDefined()
  })
})
