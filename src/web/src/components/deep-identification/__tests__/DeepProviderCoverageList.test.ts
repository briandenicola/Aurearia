import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DeepProviderCoverageList from '../DeepProviderCoverageList.vue'

describe('DeepProviderCoverageList', () => {
  it('renders exact provider statuses without collapsing unavailable states', () => {
    const wrapper = mount(DeepProviderCoverageList, {
      props: {
        coverage: [
          { provider: 'numista', status: 'no_match' },
          { provider: 'ngc', status: 'not_automated', linkOut: 'https://www.ngccoin.com/verify/' },
          { provider: 'rpc', status: 'unavailable' },
          { provider: 'ocre', status: 'timed_out' },
        ],
      },
    })

    expect(wrapper.text()).toContain('No match')
    expect(wrapper.text()).toContain('Not automated')
    expect(wrapper.text()).toContain('Unavailable')
    expect(wrapper.text()).toContain('Timed out')
    expect(wrapper.find('a[href="https://www.ngccoin.com/verify/"]').exists()).toBe(true)
  })
})
