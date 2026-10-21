import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminCopilotGradingSchedule from '../AdminCopilotGradingSchedule.vue'

const mocks = vi.hoisted(() => ({
  getCopilotGradingStatus: vi.fn(),
  runCopilotGrading: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

function buildProps() {
  return {
    settings: {
      CopilotGradingEnabled: 'true',
      CopilotGradingStartTime: '05:00',
      CopilotGradingDailyCap: '50',
      CopilotGradingJudgeModel: 'claude-sonnet-5-5',
    },
    settingsSaving: false,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.getCopilotGradingStatus.mockResolvedValue({
    data: {
      name: 'copilot-grading',
      enabled: true,
      isRunning: false,
      nextRunIn: 3.6e12,
      lastCycle: { triggerType: 'scheduled', startedAt: '2026-10-21T05:00:00Z', graded: 12, failed: 1 },
    },
  })
})

describe('AdminCopilotGradingSchedule', () => {
  it('shows status and the last cycle', async () => {
    const wrapper = mount(AdminCopilotGradingSchedule, { props: buildProps() })
    await flushPromises()

    expect(wrapper.text()).toContain('Copilot Answer Grading')
    expect(wrapper.text()).toContain('next run in 1h 0m')
    expect(wrapper.text()).toContain('12 graded, 1 failed')
  })

  it('reports an overlapping manual run', async () => {
    mocks.runCopilotGrading.mockRejectedValue({ response: { status: 409 } })
    const wrapper = mount(AdminCopilotGradingSchedule, { props: buildProps() })
    await flushPromises()

    const gradeNow = wrapper.findAll('button').find(button => button.text() === 'Grade Now')
    await gradeNow!.trigger('click')
    await flushPromises()

    expect(mocks.runCopilotGrading).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain('Grading is already running.')
  })
})
