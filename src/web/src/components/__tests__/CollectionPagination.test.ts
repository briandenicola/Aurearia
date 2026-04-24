import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import CollectionPagination from '../CollectionPagination.vue'

describe('CollectionPagination', () => {
  it('renders page info when total exceeds perPage in grid mode', () => {
    const wrapper = mount(CollectionPagination, {
      props: { page: 2, total: 30, perPage: 10, viewMode: 'grid' },
    })
    expect(wrapper.text()).toContain('Page 2 of 3')
  })

  it('hides when total is within one page', () => {
    const wrapper = mount(CollectionPagination, {
      props: { page: 1, total: 5, perPage: 10, viewMode: 'grid' },
    })
    expect(wrapper.find('.pagination').exists()).toBe(false)
  })

  it('hides when viewMode is not grid', () => {
    const wrapper = mount(CollectionPagination, {
      props: { page: 1, total: 30, perPage: 10, viewMode: 'list' },
    })
    expect(wrapper.find('.pagination').exists()).toBe(false)
  })

  it('disables previous button on first page', () => {
    const wrapper = mount(CollectionPagination, {
      props: { page: 1, total: 30, perPage: 10, viewMode: 'grid' },
    })
    const prevBtn = wrapper.findAll('button')[0]!
    expect(prevBtn.attributes('disabled')).toBeDefined()
  })

  it('disables next button on last page', () => {
    const wrapper = mount(CollectionPagination, {
      props: { page: 3, total: 30, perPage: 10, viewMode: 'grid' },
    })
    const nextBtn = wrapper.findAll('button')[1]!
    expect(nextBtn.attributes('disabled')).toBeDefined()
  })

  it('emits prev when previous button is clicked', async () => {
    const wrapper = mount(CollectionPagination, {
      props: { page: 2, total: 30, perPage: 10, viewMode: 'grid' },
    })
    await wrapper.findAll('button')[0]!.trigger('click')
    expect(wrapper.emitted('prev')).toHaveLength(1)
  })

  it('emits next when next button is clicked', async () => {
    const wrapper = mount(CollectionPagination, {
      props: { page: 1, total: 30, perPage: 10, viewMode: 'grid' },
    })
    await wrapper.findAll('button')[1]!.trigger('click')
    expect(wrapper.emitted('next')).toHaveLength(1)
  })
})
