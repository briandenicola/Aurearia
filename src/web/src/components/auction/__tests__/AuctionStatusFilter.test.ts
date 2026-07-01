import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import AuctionStatusFilter from '../AuctionStatusFilter.vue'

describe('AuctionStatusFilter', () => {
  it('keeps status filters hidden behind the menu button and emits selected status', async () => {
    const wrapper = mount(AuctionStatusFilter, {
      props: {
        modelValue: 'bidding',
        counts: { bidding: 2, won: 1 },
      },
    })

    expect(wrapper.find('.status-menu').exists()).toBe(false)

    await wrapper.get('.menu-button').trigger('click')

    expect(wrapper.find('.status-menu').exists()).toBe(true)
    expect(wrapper.text()).toContain('All')
    expect(wrapper.text()).toContain('Watching')
    expect(wrapper.text()).toContain('Bidding')
    expect(wrapper.text()).toContain('Won')
    expect(wrapper.text()).toContain('Lost')
    expect(wrapper.text()).toContain('Passed')

    const wonButton = wrapper.findAll('.status-option').find(button => button.text().includes('Won'))
    if (!wonButton) throw new Error('Won status option not found')
    await wonButton.trigger('click')

    expect(wrapper.emitted('update:modelValue')).toEqual([['won']])
    expect(wrapper.find('.status-menu').exists()).toBe(false)
  })
})
