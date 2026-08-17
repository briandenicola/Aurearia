import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DeepAnalysisProgressTimeline from '../DeepAnalysisProgressTimeline.vue'
import type { DeepStreamEvent } from '@/types'

function ev(seq: number, type: string, payload: Record<string, unknown> = {}): DeepStreamEvent {
  return { seq, jobId: 1, type, ts: '2030-01-01T00:00:00Z', payload }
}

describe('DeepAnalysisProgressTimeline', () => {
  it('renders events in the order received via the activity timeline', () => {
    const wrapper = mount(DeepAnalysisProgressTimeline, {
      props: {
        events: [ev(1, 'job_accepted'), ev(2, 'router_selected', { selectedProviders: ['nomisma', 'numista'] })],
        connected: true,
        streaming: true,
      },
    })
    const items = wrapper.findAll('ol[role="log"] > li')
    expect(items).toHaveLength(2)
    expect(items[0]?.text()).toContain('Job accepted')
    expect(items[1]?.text()).toContain('Providers selected')
    expect(items[1]?.text()).toContain('nomisma, numista')
  })

  it('shows a connected/live indicator distinct from reconnecting and disconnected', () => {
    const connectedWrapper = mount(DeepAnalysisProgressTimeline, {
      props: { events: [], connected: true, streaming: true },
    })
    expect(connectedWrapper.text()).toContain('Live')

    const connectingWrapper = mount(DeepAnalysisProgressTimeline, {
      props: { events: [], connected: false, streaming: true },
    })
    expect(connectingWrapper.text()).toContain('Connecting')

    const reconnectingWrapper = mount(DeepAnalysisProgressTimeline, {
      props: { events: [], connected: false, streaming: false, reconnecting: true },
    })
    expect(reconnectingWrapper.text()).toContain('Reconnecting')
  })

  it('never displays "Reconnecting…" when no reconnect is actually scheduled (T085/B6)', () => {
    // connected=false, streaming=false, reconnecting unset/false: nothing is
    // happening to recover the stream, so the badge must say so honestly
    // instead of claiming a reconnect that will never occur.
    const wrapper = mount(DeepAnalysisProgressTimeline, {
      props: { events: [], connected: false, streaming: false },
    })
    expect(wrapper.text()).not.toContain('Reconnecting')
    expect(wrapper.text()).toContain('Disconnected')
  })

  it('offers a Retry control only when genuinely disconnected, and it recovers the stream', async () => {
    const wrapper = mount(DeepAnalysisProgressTimeline, {
      props: { events: [], connected: false, streaming: false },
    })
    const retryButton = wrapper.find('button[aria-label="Retry Deep Analysis connection"]')
    expect(retryButton.exists()).toBe(true)

    await retryButton.trigger('click')
    expect(wrapper.emitted('retry')).toHaveLength(1)

    // Once the parent reconnects the underlying stream and it comes back
    // live, the Retry control disappears again.
    await wrapper.setProps({ connected: true, streaming: true })
    expect(wrapper.find('button[aria-label="Retry Deep Analysis connection"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('Live')
  })

  it('does not offer Retry while connecting, reconnecting, or terminal', () => {
    expect(
      mount(DeepAnalysisProgressTimeline, { props: { events: [], connected: false, streaming: true } })
        .find('button[aria-label="Retry Deep Analysis connection"]').exists(),
    ).toBe(false)
    expect(
      mount(DeepAnalysisProgressTimeline, { props: { events: [], connected: false, streaming: false, reconnecting: true } })
        .find('button[aria-label="Retry Deep Analysis connection"]').exists(),
    ).toBe(false)
    expect(
      mount(DeepAnalysisProgressTimeline, { props: { events: [], connected: false, streaming: false, terminalStatus: 'failed' } })
        .find('button[aria-label="Retry Deep Analysis connection"]').exists(),
    ).toBe(false)
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
