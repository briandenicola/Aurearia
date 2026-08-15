import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DeepAnalysisProgressTimeline from '../DeepAnalysisProgressTimeline.vue'
import type { DeepStreamEvent } from '@/types'

function ev(seq: number, type: string, payload: Record<string, unknown> = {}): DeepStreamEvent {
  return { seq, jobId: 1, type, ts: '2030-01-01T00:00:00Z', payload }
}

describe('DeepAnalysisProgressTimeline', () => {
  it('renders events in the order received', () => {
    const wrapper = mount(DeepAnalysisProgressTimeline, {
      props: {
        events: [ev(1, 'job_accepted'), ev(2, 'router_selected', { selectedProviders: ['nomisma', 'numista'] })],
        connected: true,
        streaming: true,
      },
    })
    const items = wrapper.findAll('li')
    expect(items).toHaveLength(2)
    expect(items[0].text()).toContain('Job accepted')
    expect(items[1].text()).toContain('Providers selected')
    expect(items[1].text()).toContain('nomisma, numista')
  })

  it('shows a connected/live indicator distinct from reconnecting', () => {
    const connectedWrapper = mount(DeepAnalysisProgressTimeline, {
      props: { events: [], connected: true, streaming: true },
    })
    expect(connectedWrapper.text()).toContain('Live')

    const reconnectingWrapper = mount(DeepAnalysisProgressTimeline, {
      props: { events: [], connected: false, streaming: false },
    })
    expect(reconnectingWrapper.text()).toContain('Reconnecting')
  })

  it('surfaces the stream_truncated notice when set', () => {
    const wrapper = mount(DeepAnalysisProgressTimeline, {
      props: { events: [], connected: true, streaming: true, truncated: true },
    })
    expect(wrapper.text()).toContain('no longer available')
  })

  it('renders an accessible, enabled cancel button while the job is active', async () => {
    const wrapper = mount(DeepAnalysisProgressTimeline, {
      props: { events: [], connected: true, streaming: true },
    })
    const button = wrapper.find('button[aria-label="Cancel Deep Analysis"]')
    expect(button.exists()).toBe(true)
    expect(button.attributes('disabled')).toBeUndefined()

    await button.trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })

  it('hides the cancel button once the job reaches a terminal state', () => {
    const wrapper = mount(DeepAnalysisProgressTimeline, {
      props: { events: [], connected: false, streaming: false, terminalStatus: 'completed' },
    })
    expect(wrapper.find('button[aria-label="Cancel Deep Analysis"]').exists()).toBe(false)
  })

  it('disables the cancel button while a cancel request is in flight', () => {
    const wrapper = mount(DeepAnalysisProgressTimeline, {
      props: { events: [], connected: true, streaming: true, cancelling: true },
    })
    const button = wrapper.find('button[aria-label="Cancel Deep Analysis"]')
    expect(button.attributes('disabled')).toBeDefined()
    expect(button.text()).toContain('Cancelling')
  })
})
