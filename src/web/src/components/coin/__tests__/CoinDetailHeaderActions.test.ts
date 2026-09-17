import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import CoinDetailHeaderActions from '../CoinDetailHeaderActions.vue'

const pushMock = vi.fn()
const pinned = ref(false)
const busy = ref(false)
const quickAccessError = ref('')
const pin = vi.fn()
const unpin = vi.fn()
const showToast = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
}))

vi.mock('@/composables/useQuickAccess', () => ({
  useQuickAccess: () => ({
    error: quickAccessError,
    pin,
    unpin,
    isPinned: () => pinned,
    isBusy: () => busy,
  }),
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ showToast }),
}))

const isPwa = ref(false)

vi.mock('@/composables/usePwa', () => ({
  usePwa: () => ({ isPwa: isPwa.value }),
}))

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

describe('CoinDetailHeaderActions', () => {
  beforeEach(() => {
    isPwa.value = false
    pinned.value = false
    busy.value = false
    quickAccessError.value = ''
    pin.mockReset()
    unpin.mockReset()
    showToast.mockReset()
    pin.mockResolvedValue(undefined)
    unpin.mockResolvedValue(undefined)
  })

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

  it('renders an accessible Quick Access toggle for eligible coins only', async () => {
    const wrapper = mount(CoinDetailHeaderActions, {
      props: { isWishlist: false, isSold: false, coinId: 42 },
    })
    const button = wrapper.find('button[aria-label="Pin coin to Quick Access"]')
    expect(button.attributes('aria-pressed')).toBe('false')

    await button.trigger('click')
    expect(pin).toHaveBeenCalledWith('coin', 42)

    await wrapper.setProps({ isSold: true })
    expect(wrapper.find('button[aria-label*="Quick Access"]').exists()).toBe(false)
  })

  it.each([true, false])('moves share, reminder and pin into the overflow menu in the PWA (isWishlist: %s)', async (isWishlist) => {
    isPwa.value = true
    const wrapper = mount(CoinDetailHeaderActions, {
      props: {
        isWishlist,
        isSold: false,
        coinId: 42,
        showReminderAction: true,
      },
      global: {
        stubs: { RouterLink: routerLinkStub },
      },
    })

    expect(wrapper.find('button[aria-label="Share"]').exists()).toBe(false)
    expect(wrapper.find('button[aria-label="Set Reminder"]').exists()).toBe(false)
    expect(wrapper.find('button[aria-label*="Quick Access"]').exists()).toBe(false)
    expect(wrapper.find('button[aria-label="Edit"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="Delete"]').exists()).toBe(true)

    await wrapper.find('button[aria-label="Open overflow actions"]').trigger('click')
    await wrapper.get('button[aria-label="Share"]').trigger('click')
    expect(wrapper.emitted('share')).toHaveLength(1)

    await wrapper.find('button[aria-label="Open overflow actions"]').trigger('click')
    await wrapper.get('button[aria-label="Set Reminder"]').trigger('click')
    expect(wrapper.emitted('reminder')).toHaveLength(1)

    await wrapper.find('button[aria-label="Open overflow actions"]').trigger('click')
    const pinItem = wrapper.get('button[aria-label="Pin coin to Quick Access"]')
    expect(pinItem.attributes('aria-pressed')).toBe('false')
    await pinItem.trigger('click')
    expect(pin).toHaveBeenCalledWith('coin', 42)
  })

  it('keeps share, reminder and pin on the action row outside the PWA', async () => {
    const wrapper = mount(CoinDetailHeaderActions, {
      props: {
        isWishlist: true,
        isSold: false,
        coinId: 42,
        showReminderAction: true,
      },
      global: {
        stubs: { RouterLink: routerLinkStub },
      },
    })

    expect(wrapper.find('button[aria-label="Share"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="Set Reminder"]').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="Pin coin to Quick Access"]').exists()).toBe(true)

    await wrapper.find('button[aria-label="Open overflow actions"]').trigger('click')
    expect(wrapper.findAll('button[aria-label="Share"]')).toHaveLength(1)
    expect(wrapper.findAll('button[aria-label="Set Reminder"]')).toHaveLength(1)
    expect(wrapper.findAll('button[aria-label="Pin coin to Quick Access"]')).toHaveLength(1)
  })

  it('hides the overflow pin entry for sold coins in the PWA and reflects reminder state', async () => {
    isPwa.value = true
    const wrapper = mount(CoinDetailHeaderActions, {
      props: {
        isWishlist: true,
        isSold: true,
        coinId: 42,
        showReminderAction: true,
        reminderActive: true,
      },
      global: {
        stubs: { RouterLink: routerLinkStub },
      },
    })

    await wrapper.find('button[aria-label="Open overflow actions"]').trigger('click')
    expect(wrapper.find('button[aria-label*="Quick Access"]').exists()).toBe(false)
    expect(wrapper.find('button[aria-label="Edit Reminder"]').exists()).toBe(true)
  })

  it('reflects pinned and pending state and reports failures without changing state', async () => {
    pinned.value = true
    busy.value = true
    const wrapper = mount(CoinDetailHeaderActions, {
      props: { isWishlist: false, isSold: false, coinId: 42 },
    })
    const button = wrapper.find('button[aria-label="Updating coin Quick Access pin"]')
    expect(button.attributes('aria-pressed')).toBe('true')
    expect(button.attributes('disabled')).toBeDefined()
    expect(button.attributes('aria-busy')).toBe('true')

    busy.value = false
    await wrapper.vm.$nextTick()
    quickAccessError.value = 'Server rejected pin'
    unpin.mockRejectedValue(new Error('failed'))
    await wrapper.get('button[aria-label="Unpin coin from Quick Access"]').trigger('click')
    expect(showToast).toHaveBeenCalledWith('Server rejected pin', 'error')
    expect(pinned.value).toBe(true)
  })
})
