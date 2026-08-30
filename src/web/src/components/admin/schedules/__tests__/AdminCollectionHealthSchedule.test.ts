import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminCollectionHealthSchedule from '../AdminCollectionHealthSchedule.vue'

const mocks = vi.hoisted(() => ({
  getCollectionHealthSnapshotRuns: vi.fn(),
  getCollectionHealthSnapshotStatus: vi.fn(),
  triggerCollectionHealthSnapshots: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

function buildProps() {
  return {
    settings: {
      CollectionHealthSnapshotsEnabled: 'true',
      CollectionHealthSnapshotsStartTime: '02:00',
    },
    settingsSaving: false,
    settingsMsg: '',
    settingsError: false,
  }
}

function run(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 1,
    triggerType: 'scheduled',
    status: 'success',
    usersEligible: 5,
    usersSnapshotted: 5,
    usersFailed: 0,
    durationMs: 51,
    startedAt: '2026-08-29T02:00:00Z',
    completedAt: '2026-08-29T02:00:01Z',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.getCollectionHealthSnapshotStatus.mockResolvedValue({ data: { enabled: true, nextRunIn: 3.6e12 } })
  mocks.getCollectionHealthSnapshotRuns.mockResolvedValue({ data: { runs: [run(), run({ id: 2 })] } })
})

describe('AdminCollectionHealthSchedule', () => {
  it('renders the run history', async () => {
    const wrapper = mount(AdminCollectionHealthSchedule, { props: buildProps() })
    await flushPromises()

    expect(wrapper.text()).toContain('Collection Health Snapshot Run History')
    expect(wrapper.findAll('tbody tr')).toHaveLength(2)
  })

  // The seven-column table ran off the card on a narrow PWA viewport, putting Duration out
  // of reach entirely — the card clipped it with nothing to scroll (see also
  // WishlistAvailabilityHistoryPage's identical regression).
  it('keeps the run-history table inside a horizontally scrollable wrapper', async () => {
    const wrapper = mount(AdminCollectionHealthSchedule, { props: buildProps() })
    await flushPromises()

    const table = wrapper.find('table')
    expect(table.exists()).toBe(true)
    expect(table.element.parentElement?.className).toContain('overflow-x-auto')
  })
})
