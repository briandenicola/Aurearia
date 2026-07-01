import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import WishlistAlertsPage from '../WishlistAlertsPage.vue'

const mocks = vi.hoisted(() => ({
  adjustWishlistSearchAlertCriteria: vi.fn(),
  convertWishlistSearchAlertCandidate: vi.fn(),
  createWishlistSearchAlert: vi.fn(),
  deleteWishlistSearchAlert: vi.fn(),
  dismissWishlistSearchAlertCandidate: vi.fn(),
  getApiErrorMessage: vi.fn(() => ''),
  getWishlistSearchAlertRun: vi.fn(),
  listWishlistSearchAlertCandidates: vi.fn(),
  listWishlistSearchAlertRuns: vi.fn(),
  listWishlistSearchAlerts: vi.fn(),
  restoreWishlistSearchAlertCandidate: vi.fn(),
  runWishlistSearchAlert: vi.fn(),
  updateWishlistSearchAlert: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

vi.mock('@/composables/usePwa', () => ({
  usePwa: () => ({ isPwa: false }),
}))

const alert = {
  id: 1,
  userId: 1,
  name: 'Domitian denarius',
  rulerOrIssuer: 'Domitian',
  coinType: 'Denarius',
  dateFrom: null,
  dateTo: null,
  mint: '',
  material: 'Silver',
  gradeOrCondition: '',
  priceMin: null,
  priceMax: 300,
  currency: 'USD',
  dealerPreference: '',
  sourceFilters: [],
  keywords: '',
  notes: '',
  cadence: 'manual',
  isActive: true,
  lastRunAt: null,
  createdAt: '',
  updatedAt: '',
}

function alertRun(status: string) {
  return {
    id: 7,
    alertId: 1,
    userId: 1,
    triggerType: 'manual',
    status,
    startedAt: '',
    completedAt: status === 'completed' ? '2026-07-01T18:20:00Z' : null,
    durationMs: 0,
    criteriaSnapshot: '{}',
    resultCount: status === 'completed' ? 2 : 0,
    newCount: status === 'completed' ? 2 : 0,
    duplicateCount: 0,
    dismissedCount: 0,
    partialWarnings: [],
    errorMessage: '',
    rateLimitStatus: 'ok',
    createdAt: '',
  }
}

describe('WishlistAlertsPage', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
    mocks.listWishlistSearchAlerts.mockResolvedValue({ data: { alerts: [alert], total: 1, page: 1, limit: 100 } })
    mocks.listWishlistSearchAlertRuns.mockResolvedValue({ data: { runs: [], total: 0, page: 1, limit: 20 } })
    mocks.listWishlistSearchAlertCandidates.mockResolvedValue({ data: { candidates: [], total: 0, page: 1, limit: 50 } })
    mocks.runWishlistSearchAlert.mockResolvedValue({
      data: {
        runId: 7,
        alertId: 1,
        status: 'queued',
        startedAt: '',
        completedAt: null,
        resultCount: 0,
        newCount: 0,
        duplicateCount: 0,
        dismissedCount: 0,
        partialWarnings: [],
        rateLimitStatus: 'ok',
      },
    })
    mocks.getWishlistSearchAlertRun
      .mockResolvedValueOnce({ data: alertRun('queued') })
      .mockResolvedValueOnce({ data: alertRun('completed') })
      .mockResolvedValue({ data: alertRun('completed') })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('queues Run Now and polls run detail until completion', async () => {
    const wrapper = mount(WishlistAlertsPage, {
      global: {
        stubs: {
          AlertCriteriaSummary: true,
          AlertForm: true,
          AlertRunHistory: true,
          CandidateReviewCard: true,
        },
      },
    })
    await flushPromises()

    await wrapper.find('.select-alert').trigger('click')
    await flushPromises()
    await wrapper.find('.selected-summary .btn-primary').trigger('click')
    await flushPromises()

    expect(mocks.runWishlistSearchAlert).toHaveBeenCalledWith(1, 20)
    expect(mocks.getWishlistSearchAlertRun).toHaveBeenCalledWith(1, 7)
    expect(wrapper.text()).toContain('Search alert run queued')

    await vi.advanceTimersByTimeAsync(3000)
    await flushPromises()

    expect(mocks.getWishlistSearchAlertRun).toHaveBeenCalledTimes(3)
    expect(wrapper.text()).toContain('Run completed with 2 candidates and 0 duplicates.')
    expect(mocks.listWishlistSearchAlertCandidates).toHaveBeenCalled()
  })
})
