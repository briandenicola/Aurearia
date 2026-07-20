import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import HelpSection from '../HelpSection.vue'

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
}))

describe('HelpSection', () => {
  it('documents auction provider capability differences', () => {
    const wrapper = mount(HelpSection)

    expect(wrapper.text()).toContain('CNG Auctions supports richer hosted-auction sync')
    expect(wrapper.text()).toContain('NumisBids supports watchlist/import tracking only today')
    expect(wrapper.text()).toContain('manually update won/lost')
  })

  it('documents current app help for stats, capture, notifications, and external tools', () => {
    const wrapper = mount(HelpSection)
    const text = wrapper.text()

    expect(text).toContain('Stats Views')
    expect(text).toContain('Value snapshots, allocation, acquisition-year performance')
    expect(text).toContain('Quick Capture, Coin Lookup, and image upload flows start the camera only after you tap')
    expect(text).toContain('The notification badge tracks unread social, auction, wishlist, set milestone')
    expect(text).toContain('Header Name: X-API-Key')
  })
})
