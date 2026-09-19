import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import CopilotRunProgress from '@/components/chat/CopilotRunProgress.vue'
import type {
  CoinCopilotRun,
  DeepAnalysisHandoffOutcome,
  DeepAnalysisHandoffResult,
} from '@/types'
import type { CoinCopilotToolProgress } from '@/composables/useCoinCopilot'

const run: CoinCopilotRun = {
  id: 'ccr_test',
  threadId: 'cct_test',
  status: 'completed',
  goal: 'Analyze the coin',
  checkpointVersion: 1,
  lastSeq: 1,
  attempt: 1,
  finalAnswer: null,
  failureCode: null,
  failureMessage: null,
  resumeDeadline: null,
  usage: { iterations: 1, toolCalls: 1, inputTokens: 10, outputTokens: 10 },
  createdAt: '2026-09-18T18:00:00Z',
  updatedAt: '2026-09-18T18:01:00Z',
}

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

function handoff(
  outcome: Exclude<DeepAnalysisHandoffOutcome, 'not_eligible' | 'target_unavailable'> = 'status',
): DeepAnalysisHandoffResult {
  return {
    schema_version: 1,
    operation: 'status',
    outcome,
    reason: null,
    target: { type: 'coin', id: 42, display_label: 'Hadrian denarius' },
    job: {
      id: 17,
      source: 'saved_coin',
      status: 'completed',
      reused: true,
      created_at: '2026-09-18T18:00:00Z',
      completed_at: '2026-09-18T18:01:00Z',
    },
    review_url: '/deep-analysis/17',
    fresh_analysis_available: false,
    result: {
      state: 'partial',
      narrative: 'Attribution is supported with one unresolved conflict.',
      partial_success: true,
      image_only: false,
      fields: [],
      disagreements: [{ field: 'date', summary: 'Two references disagree.' }],
      unresolved_questions: [],
      coverage: [
        { provider: 'numista', status: 'contributed' },
        { provider: 'ngc', status: 'not_automated' },
      ],
      attributions: [],
      limitations: ['Reverse legend is partly unreadable.'],
    },
    truncation: {
      truncated: true,
      original_bytes: 40000,
      persisted_bytes: 32768,
      digest: 'a'.repeat(64),
      omitted_fields: 1,
      omitted_evidence: 2,
      omitted_disagreements: 3,
      omitted_questions: 4,
    },
    limitations: ['Only public source evidence is shown.'],
  }
}

function tool(result?: DeepAnalysisHandoffResult): CoinCopilotToolProgress {
  return {
    toolCallId: 'call_01',
    toolName: 'deep_analysis_handoff',
    stepId: 'step_01',
    status: 'succeeded',
    deepAnalysisHandoffResult: result,
  }
}

function mountProgress(result?: DeepAnalysisHandoffResult) {
  return mount(CopilotRunProgress, {
    props: {
      run,
      plan: [],
      tools: [tool(result)],
      canCancel: false,
      cancelling: false,
      truncated: false,
      addingIdx: null,
      addedSet: new Set<string>(),
    },
    global: { stubs: { RouterLink: routerLinkStub } },
  })
}

describe('CopilotRunProgress Deep Analysis handoff', () => {
  it.each([
    ['accepted', 'Accepted'],
    ['reused_active', 'In progress'],
    ['reused_result', 'Result available'],
    ['status', 'Status'],
    ['retry_available', 'Retry available'],
    ['missing_images', 'Images required'],
    ['unavailable', 'Unavailable'],
    ['cancelled', 'Cancelled'],
  ] as const)('renders the %s outcome', (outcome, label) => {
    expect(mountProgress(handoff(outcome)).text()).toContain(label)
  })

  it.each([
    ['queued', 'Queued'],
    ['running', 'Running'],
    ['completed', 'Completed'],
    ['partial', 'Partial'],
    ['failed', 'Failed'],
    ['cancelled', 'Cancelled'],
  ] as const)('renders the %s job status', (status, label) => {
    const result = handoff()
    if ('schema_version' in result && result.job) result.job.status = status
    expect(mountProgress(result).text()).toContain(`Job status: ${label}`)
  })

  it.each([
    [{ outcome: 'not_eligible', reason: null }, 'Not eligible'],
    [{ outcome: 'target_unavailable', reason: null }, 'Target unavailable'],
  ] as const)('renders privacy-preserving outcomes without a review link', (result, label) => {
    const wrapper = mountProgress(result)
    expect(wrapper.text()).toContain(label)
    expect(wrapper.text()).not.toContain('Open Deep Analysis')
  })

  it('shows conflicts, coverage, limitations, and exact omission totals without apply controls', () => {
    const wrapper = mountProgress(handoff())

    expect(wrapper.text()).toContain('Two references disagree.')
    expect(wrapper.text()).toContain('Numista: Contributed')
    expect(wrapper.text()).toContain('Reverse legend is partly unreadable.')
    expect(wrapper.get('[data-testid="copilot-deep-analysis-omissions"]').text())
      .toContain('10 items omitted')
    expect(wrapper.text()).toContain('Open Deep Analysis')
    expect(wrapper.text()).not.toMatch(/\b(?:Apply|Edit proposal)\b/)
  })

  it('omits the card when no validated typed payload reaches the component', () => {
    const wrapper = mountProgress()
    expect(wrapper.find('[data-testid="copilot-deep-analysis-result"]').exists()).toBe(false)
  })

  it.each([
    'https://example.com/deep-analysis/17',
    '//example.com/deep-analysis/17',
    'https://user@example.com/deep-analysis/17',
    '/deep-analysis/0',
    '/deep-analysis/-17',
    '/deep-analysis/18',
  ])('rejects unsafe or mismatched review URL %s', (reviewUrl) => {
    const result = handoff()
    if ('schema_version' in result) result.review_url = reviewUrl
    expect(mountProgress(result).text()).not.toContain('Open Deep Analysis')
  })

  it('links only to the positive job-matching Deep Analysis route', () => {
    const link = mountProgress(handoff()).get('a')
    expect(link.attributes('href')).toBe('/deep-analysis/17')
    expect(link.text()).toBe('Open Deep Analysis')
  })
})
