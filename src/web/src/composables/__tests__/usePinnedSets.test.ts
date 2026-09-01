import { beforeEach, describe, expect, it, vi } from 'vitest'
import { usePinnedSets } from '../usePinnedSets'
import { getSets, updateSet } from '@/api/client'

vi.mock('@/api/client', () => ({
  getSets: vi.fn(),
  updateSet: vi.fn(),
}))

function buildSet(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    name: 'Twelve Caesars',
    color: '#c9a84c',
    setType: 'standard',
    coinCount: 3,
    totalValue: 300,
    pinned: false,
    pinnedAt: null,
    ...overrides,
  }
}

describe('usePinnedSets', () => {
  beforeEach(async () => {
    vi.clearAllMocks()
    // Reset the module-level singleton state between tests.
    const { clear } = usePinnedSets()
    clear()
  })

  it('filters to pinned sets only', async () => {
    vi.mocked(getSets).mockResolvedValue({
      data: {
        sets: [
          buildSet({ id: 1, name: 'Unpinned Set', pinned: false, pinnedAt: null }),
          buildSet({ id: 2, name: 'Pinned Set', pinned: true, pinnedAt: '2026-01-01T00:00:00Z' }),
        ],
      },
    } as Awaited<ReturnType<typeof getSets>>)

    const { pinnedSets, refresh } = usePinnedSets()
    await refresh()

    expect(pinnedSets.value).toHaveLength(1)
    expect(pinnedSets.value[0]?.id).toBe(2)
  })

  it('sorts pinned sets by pinnedAt ascending, then name', async () => {
    vi.mocked(getSets).mockResolvedValue({
      data: {
        sets: [
          buildSet({ id: 1, name: 'Zebra Set', pinned: true, pinnedAt: '2026-01-02T00:00:00Z' }),
          buildSet({ id: 2, name: 'Alpha Set', pinned: true, pinnedAt: '2026-01-01T00:00:00Z' }),
          buildSet({ id: 3, name: 'Beta Set', pinned: true, pinnedAt: '2026-01-01T00:00:00Z' }),
        ],
      },
    } as Awaited<ReturnType<typeof getSets>>)

    const { pinnedSets, refresh } = usePinnedSets()
    await refresh()

    expect(pinnedSets.value.map((s) => s.id)).toEqual([2, 3, 1])
  })

  it('setPinned calls updateSet with the pinned flag then refreshes', async () => {
    vi.mocked(updateSet).mockResolvedValue({ data: {} } as Awaited<ReturnType<typeof updateSet>>)
    vi.mocked(getSets).mockResolvedValue({
      data: { sets: [buildSet({ id: 5, name: 'Newly Pinned', pinned: true, pinnedAt: '2026-01-01T00:00:00Z' })] },
    } as Awaited<ReturnType<typeof getSets>>)

    const { pinnedSets, setPinned } = usePinnedSets()
    await setPinned(5, true)

    expect(updateSet).toHaveBeenCalledWith(5, { pinned: true })
    expect(getSets).toHaveBeenCalledTimes(1)
    expect(pinnedSets.value).toHaveLength(1)
    expect(pinnedSets.value[0]?.id).toBe(5)
  })

  it('setPinned rethrows so the caller can surface a server error toast', async () => {
    vi.mocked(updateSet).mockRejectedValue(new Error('you can pin up to 5 sets'))

    const { setPinned } = usePinnedSets()

    await expect(setPinned(9, true)).rejects.toThrow('you can pin up to 5 sets')
    expect(getSets).not.toHaveBeenCalled()
  })

  it('preserves current state and does not throw when refresh fails', async () => {
    vi.mocked(getSets).mockResolvedValueOnce({
      data: { sets: [buildSet({ id: 1, name: 'Stable Pin', pinned: true, pinnedAt: '2026-01-01T00:00:00Z' })] },
    } as Awaited<ReturnType<typeof getSets>>)

    const { pinnedSets, refresh } = usePinnedSets()
    await refresh()
    expect(pinnedSets.value).toHaveLength(1)

    vi.mocked(getSets).mockRejectedValueOnce(new Error('network error'))
    await expect(refresh()).resolves.toBeUndefined()

    expect(pinnedSets.value).toHaveLength(1)
    expect(pinnedSets.value[0]?.id).toBe(1)
  })

  it('clear() empties the pinned sets list', async () => {
    vi.mocked(getSets).mockResolvedValue({
      data: { sets: [buildSet({ id: 1, pinned: true, pinnedAt: '2026-01-01T00:00:00Z' })] },
    } as Awaited<ReturnType<typeof getSets>>)

    const { pinnedSets, refresh, clear } = usePinnedSets()
    await refresh()
    expect(pinnedSets.value).toHaveLength(1)

    clear()
    expect(pinnedSets.value).toHaveLength(0)
  })

  it('pinLimitReached becomes true at exactly 5 pinned sets', async () => {
    vi.mocked(getSets).mockResolvedValue({
      data: {
        sets: Array.from({ length: 5 }, (_, i) =>
          buildSet({ id: i + 1, name: `Set ${i + 1}`, pinned: true, pinnedAt: `2026-01-0${i + 1}T00:00:00Z` }),
        ),
      },
    } as Awaited<ReturnType<typeof getSets>>)

    const { pinLimitReached, refresh } = usePinnedSets()
    expect(pinLimitReached.value).toBe(false)
    await refresh()
    expect(pinLimitReached.value).toBe(true)
  })
})
