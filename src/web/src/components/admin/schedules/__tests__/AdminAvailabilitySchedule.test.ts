import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminAvailabilitySchedule from '../AdminAvailabilitySchedule.vue'

const mocks = vi.hoisted(() => ({
  getAvailabilityCycles: vi.fn(),
  getAvailabilityCycleDetail: vi.fn(),
  getAvailabilityRuns: vi.fn(),
  getAvailabilityRunDetail: vi.fn(),
  triggerAvailabilityCheck: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

vi.mock('@/composables/useSafeExternalLink', () => ({
  sanitizeExternalUrl: (url: string | null | undefined) => url ?? null,
}))

function buildProps() {
  return {
    settings: {
      WishlistCheckEnabled: 'true',
      WishlistCheckStartTime: '08:00',
      WishlistCheckInterval: '120',
    },
    settingsSaving: false,
    settingsMsg: '',
    settingsError: false,
  }
}

function cycle(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 10,
    triggerType: 'admin',
    triggerUserId: 1,
    status: 'partial_failure',
    totalChildren: 3,
    queuedChildren: 0,
    runningChildren: 0,
    completedChildren: 2,
    failedChildren: 1,
    startedAt: '2026-08-01T12:00:00Z',
    completedAt: '2026-08-01T12:01:00Z',
    createdAt: '2026-08-01T12:00:00Z',
    ...overrides,
  }
}

function legacyRun(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 1,
    userId: 0,
    triggerType: 'manual',
    triggerUserId: null,
    status: 'completed',
    coinsChecked: 5,
    available: 4,
    unavailable: 1,
    unknown: 0,
    errors: 0,
    durationMs: 2200,
    startedAt: '2026-01-01T00:00:00Z',
    completedAt: '2026-01-01T00:00:02Z',
    createdAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

async function mountComponent() {
  const wrapper = mount(AdminAvailabilitySchedule, {
    props: buildProps(),
    global: {
      stubs: {
        SafeExternalLink: { template: '<a><slot /></a>' },
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('AdminAvailabilitySchedule', () => {
  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockReset())
    mocks.getAvailabilityCycles.mockResolvedValue({ data: { cycles: [], total: 0, page: 1, limit: 5 } })
    mocks.getAvailabilityCycleDetail.mockResolvedValue({ data: { ...cycle(), children: [] } })
    mocks.getAvailabilityRuns.mockResolvedValue({ data: { runs: [], total: 0 } })
    mocks.getAvailabilityRunDetail.mockResolvedValue({ data: {} })
    mocks.triggerAvailabilityCheck.mockResolvedValue({ data: { cycleId: 99, status: 'queued', message: 'Availability check queued' } })
  })

  it('renders parent cycle rows with counts', async () => {
    mocks.getAvailabilityCycles.mockResolvedValue({ data: { cycles: [cycle()], total: 1, page: 1, limit: 5 } })
    const wrapper = await mountComponent()

    expect(mocks.getAvailabilityCycles).toHaveBeenCalledWith(1, 5)
    expect(wrapper.text()).toContain('partial_failure')
    // totalChildren, completedChildren, failedChildren all render in the row.
    expect(wrapper.find('tbody tr').text()).toContain('3')
    expect(wrapper.find('tbody tr').text()).toContain('2')
    expect(wrapper.find('tbody tr').text()).toContain('1')
  })

  it('renders only legacy UserID=0 rows with a Legacy chip, distinct from cycle rows', async () => {
    mocks.getAvailabilityCycles.mockResolvedValue({ data: { cycles: [cycle({ id: 20 })], total: 1, page: 1, limit: 5 } })
    mocks.getAvailabilityRuns.mockResolvedValue({
      data: {
        runs: [legacyRun({ id: 1, userId: 0 }), legacyRun({ id: 2, userId: 7 })],
        total: 2,
      },
    })
    const wrapper = await mountComponent()

    // Only the UserID=0 row survives the legacy filter.
    const legacyChips = wrapper.findAll('.chip-sm').filter((chip) => chip.text() === 'Legacy')
    expect(legacyChips.length).toBeGreaterThan(0)
    expect(wrapper.text()).toContain('Legacy Runs')
    expect(wrapper.text()).not.toContain('No legacy availability runs recorded.')
  })

  it('expands a parent cycle and fetches its children from /admin/availability-cycles/{id}', async () => {
    mocks.getAvailabilityCycles.mockResolvedValue({ data: { cycles: [cycle({ id: 30 })], total: 1, page: 1, limit: 5 } })
    mocks.getAvailabilityCycleDetail.mockResolvedValue({
      data: {
        ...cycle({ id: 30 }),
        children: [
          {
            id: 101,
            userId: 5,
            userName: 'collector5',
            status: 'completed',
            coinsChecked: 3,
            available: 3,
            unavailable: 0,
            unknown: 0,
            errors: 0,
            startedAt: '2026-08-01T12:00:00Z',
            completedAt: '2026-08-01T12:00:30Z',
            cycleId: 30,
          },
        ],
      },
    })
    const wrapper = await mountComponent()

    const cycleRow = wrapper.findAll('tbody tr')[0]
    await cycleRow?.trigger('click')
    await flushPromises()

    expect(mocks.getAvailabilityCycleDetail).toHaveBeenCalledWith(30)
    expect(wrapper.text()).toContain('collector5')
  })

  it('surfaces a 409 duplicate-trigger response gracefully', async () => {
    mocks.triggerAvailabilityCheck.mockRejectedValue({ response: { status: 409 } })
    const wrapper = await mountComponent()

    const runNowButton = wrapper.findAll('button').find((button) => button.text().includes('Run Now'))
    expect(runNowButton).toBeTruthy()
    await runNowButton?.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('update:settingsMsg')?.at(-1)?.[0]).toContain('already in progress')
    expect(wrapper.emitted('update:settingsError')?.at(-1)?.[0]).toBe(true)
  })
})
