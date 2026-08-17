import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DeepAnalysisActivityTimeline from '../DeepAnalysisActivityTimeline.vue'
import type { DeepStreamEvent } from '@/types'

function ev(seq: number, type: string, payload: Record<string, unknown> = {}, ts = `2030-01-01T00:00:${String(seq).padStart(2, '0')}Z`): DeepStreamEvent {
  return { seq, jobId: 1, type, ts, payload }
}

describe('DeepAnalysisActivityTimeline', () => {
  it('shows a waiting message when no events have arrived yet', () => {
    const wrapper = mount(DeepAnalysisActivityTimeline, { props: { events: [] } })
    expect(wrapper.text()).toContain('Waiting for Deep Analysis to begin')
  })

  it('renders known lifecycle events in order with a Done state', () => {
    const wrapper = mount(DeepAnalysisActivityTimeline, {
      props: {
        events: [ev(1, 'job_accepted'), ev(2, 'router_selected', { selectedProviders: ['nomisma', 'numista'] })],
        terminalStatus: 'completed',
      },
    })
    const items = wrapper.findAll('ol[role="log"] > li')
    expect(items).toHaveLength(2)
    expect(items[0]?.text()).toContain('Job accepted')
    expect(items[0]?.text()).toContain('Done')
    expect(items[1]?.text()).toContain('Providers selected')
    expect(items[1]?.text()).toContain('nomisma, numista')
  })

  it('renders an unrecognized progress phase sensibly instead of dropping it', () => {
    const wrapper = mount(DeepAnalysisActivityTimeline, {
      props: {
        events: [
          ev(1, 'progress', { phase: 'vision_completed', message: 'Vision analysis produced 6 populated fields.' }),
        ],
      },
    })
    expect(wrapper.text()).toContain('Vision Completed')
    expect(wrapper.text()).toContain('Vision analysis produced 6 populated fields.')
  })

  it('renders a completely unknown event type generically instead of crashing or vanishing', () => {
    const wrapper = mount(DeepAnalysisActivityTimeline, {
      props: {
        events: [ev(1, 'a_future_event_type', { message: 'Something new happened.' })],
      },
    })
    expect(wrapper.text()).toContain('A Future Event Type')
    expect(wrapper.text()).toContain('Something new happened.')
  })

  it('groups per-provider events into a single fan-out step with individual rows', () => {
    const wrapper = mount(DeepAnalysisActivityTimeline, {
      props: {
        events: [
          ev(1, 'provider_started', { provider: 'ngc' }),
          ev(2, 'provider_started', { provider: 'numista' }),
          ev(3, 'provider_result', { provider: 'ngc', status: 'not_automated' }),
          ev(4, 'provider_result', { provider: 'numista', status: 'no_match' }),
        ],
        terminalStatus: 'partial',
      },
    })
    const groups = wrapper.findAll('ol[role="log"] > li')
    expect(groups).toHaveLength(1)
    const text = groups[0]?.text() ?? ''
    expect(text).toContain('Provider fan-out')
    expect(text).toContain('ngc')
    expect(text).toContain('Manual verification')
    expect(text).toContain('numista')
    expect(text).toContain('No match')
  })

  it('makes a step that yielded no result visually distinct from a successful one', () => {
    const wrapper = mount(DeepAnalysisActivityTimeline, {
      props: {
        events: [
          ev(1, 'provider_started', { provider: 'nomisma' }),
          ev(2, 'provider_result', { provider: 'nomisma', status: 'contributed' }),
          ev(3, 'provider_started', { provider: 'rpc' }),
          ev(4, 'provider_result', { provider: 'rpc', status: 'unavailable' }),
        ],
        terminalStatus: 'partial',
      },
    })
    const rows = wrapper.findAll('ol[role="log"] > li ul > li')
    const nomismaRow = rows.find((row) => row.text().includes('nomisma'))
    const rpcRow = rows.find((row) => row.text().includes('rpc'))
    expect(nomismaRow).toBeDefined()
    expect(rpcRow).toBeDefined()
    expect(nomismaRow?.text() ?? '').toContain('Contributed')
    expect(rpcRow?.text() ?? '').toContain('Unavailable')
    // The failed/no-result row must not share the successful row's "Done"
    // wording or its gold success styling class on the state icon.
    expect(rpcRow?.text() ?? '').not.toContain('Contributed')
    const nomismaIconClasses = nomismaRow?.find('svg').classes().join(' ') ?? ''
    const rpcIconClasses = rpcRow?.find('svg').classes().join(' ') ?? ''
    expect(nomismaIconClasses).not.toEqual(rpcIconClasses)
  })

  it('marks the most recent step as active while the job is still running', () => {
    const wrapper = mount(DeepAnalysisActivityTimeline, {
      props: {
        events: [ev(1, 'job_accepted'), ev(2, 'progress', { phase: 'image_evidence_ready', message: 'Vision pass complete.' })],
        terminalStatus: null,
        ended: false,
      },
    })
    const items = wrapper.findAll('ol[role="log"] > li')
    expect(items[0]?.text()).toContain('Done')
    expect(items[1]?.text()).toContain('In progress')
  })

  it('collapses automatically the first time the job reaches a terminal state', async () => {
    const wrapper = mount(DeepAnalysisActivityTimeline, {
      props: { events: [ev(1, 'job_accepted')], terminalStatus: null },
    })
    expect(wrapper.find('details').attributes('open')).toBeDefined()

    await wrapper.setProps({ terminalStatus: 'completed' })
    expect(wrapper.find('details').attributes('open')).toBeUndefined()
  })

  it('shows elapsed time between steps when timestamps are available', () => {
    const wrapper = mount(DeepAnalysisActivityTimeline, {
      props: {
        events: [
          ev(1, 'job_accepted', {}, '2030-01-01T00:00:00Z'),
          ev(2, 'router_selected', { selectedProviders: ['nomisma'] }, '2030-01-01T00:00:12Z'),
        ],
        terminalStatus: 'completed',
      },
    })
    expect(wrapper.text()).toContain('+12s')
  })
})
