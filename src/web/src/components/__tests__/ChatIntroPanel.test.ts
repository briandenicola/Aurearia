import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ChatIntroPanel from '@/components/chat/ChatIntroPanel.vue'

describe('ChatIntroPanel', () => {
  it('offers prompts supported by the Coin Copilot tool surface', async () => {
    const wrapper = mount(ChatIntroPanel)
    const buttons = wrapper.findAll('button')

    expect(wrapper.text()).toContain('sources configured by your administrator')
    expect(wrapper.text()).not.toContain('NumisBids')

    await buttons[0]!.trigger('click')
    await buttons[1]!.trigger('click')
    await buttons[3]!.trigger('click')
    await buttons[4]!.trigger('click')

    expect(wrapper.emitted('send')).toEqual([
      ['Find current dealer listings and upcoming auctions online for Julius Caesar denarii'],
      ['Find current dealer listings for Byzantine gold solidi under $1,000'],
      ['What are the clearest gaps in my coin collection?'],
      ['What is the completed-sale price trend for Athenian owl tetradrachms?'],
    ])
    expect(wrapper.text()).not.toContain('coin shows')
  })
})
