import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminCopilotQualitySection from '../AdminCopilotQualitySection.vue'

const mocks = vi.hoisted(() => ({
  getCopilotGradeSummary: vi.fn(),
  getCopilotGrades: vi.fn(),
  disputeCopilotGrade: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

function grade(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    runId: 'ccr_1',
    threadId: 'cct_1',
    quality: 0.43,
    answersDirectly: false,
    grounded: true,
    accurate: false,
    citesSources: 'no',
    noOfferMenu: true,
    noFalseInability: true,
    explanation: 'Calls the coin a reign-dated issue; it shows Caligula.',
    judgeModel: 'claude-sonnet-5-5',
    judgeInputTokens: 900,
    judgeOutputTokens: 120,
    disputed: false,
    gradedAt: '2026-10-21T05:00:00Z',
    question: 'Is this lot really a coin of Caligula?',
    answer: 'Not necessarily.',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.getCopilotGradeSummary.mockResolvedValue({
    data: {
      days: 30,
      graded: 4,
      disputed: 1,
      avgQuality: 0.75,
      trend: [{ day: '2026-10-20', avgQuality: 0.7, graded: 2 }, { day: '2026-10-21', avgQuality: 0.8, graded: 2 }],
      failRates: { answersDirectly: 0.25, grounded: 0, accurate: 0.5, noOfferMenu: 0, noFalseInability: 0 },
    },
  })
  mocks.getCopilotGrades.mockResolvedValue({ data: { grades: [grade()], total: 1, page: 1, limit: 20 } })
})

describe('AdminCopilotQualitySection', () => {
  it('shows the summary and loads undisputed grades worst first', async () => {
    const wrapper = mount(AdminCopilotQualitySection)
    await flushPromises()

    expect(mocks.getCopilotGradeSummary).toHaveBeenCalledWith(30)
    expect(mocks.getCopilotGrades).toHaveBeenCalledWith({ page: 1, limit: 20, sort: 'worst', disputed: false })
    expect(wrapper.text()).toContain('75%')
    expect(wrapper.text()).toContain('1 disputed and excluded')
    expect(wrapper.text()).toContain('Is this lot really a coin of Caligula?')
    expect(wrapper.text()).toContain('not direct, inaccurate, no sources')
    // Cites sources has no rate when no graded answer used external evidence.
    expect(wrapper.text()).toContain('n/a')
  })

  it('marks a grade wrong with a note and refreshes the summary', async () => {
    mocks.disputeCopilotGrade.mockResolvedValue({ data: grade({ disputed: true, disputeNote: 'Judge is right about RPC' }) })
    const wrapper = mount(AdminCopilotQualitySection)
    await flushPromises()

    await wrapper.find('button[aria-expanded]').trigger('click')
    expect(wrapper.text()).toContain('Calls the coin a reign-dated issue')
    await wrapper.find('textarea').setValue('Judge is right about RPC')
    const markWrong = wrapper.findAll('button').find(button => button.text() === 'Mark grade wrong')
    await markWrong!.trigger('click')
    await flushPromises()

    expect(mocks.disputeCopilotGrade).toHaveBeenCalledWith(1, true, 'Judge is right about RPC')
    expect(mocks.getCopilotGradeSummary).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('Marked wrong')
  })

  it('shows an empty state when nothing has been graded', async () => {
    mocks.getCopilotGradeSummary.mockResolvedValue({ data: { days: 30, graded: 0, disputed: 0, avgQuality: 0, trend: [], failRates: {} } })
    mocks.getCopilotGrades.mockResolvedValue({ data: { grades: [], total: 0, page: 1, limit: 20 } })
    const wrapper = mount(AdminCopilotQualitySection)
    await flushPromises()

    expect(wrapper.text()).toContain('No graded conversations in the last 30 days.')
    expect(wrapper.text()).toContain('No graded conversations to show.')
  })
})
