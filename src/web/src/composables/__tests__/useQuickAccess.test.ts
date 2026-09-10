import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getQuickAccess, pinQuickAccess, unpinQuickAccess } from '@/api/client'
import { useQuickAccess } from '../useQuickAccess'
import type { QuickAccessItem } from '@/types'

vi.mock('@/api/client', () => ({
  getQuickAccess: vi.fn(),
  pinQuickAccess: vi.fn(),
  unpinQuickAccess: vi.fn(),
  getApiErrorMessage: (error: unknown) => error instanceof Error ? error.message : '',
}))

const coinItem: QuickAccessItem = {
  type: 'coin',
  id: 42,
  pinnedAt: '2026-09-10T12:30:00Z',
  coin: { name: 'Trajan Denarius', classification: 'wishlist', primaryImageUrl: '' },
}

const setItem: QuickAccessItem = {
  type: 'coin_set',
  id: 7,
  pinnedAt: '2026-09-09T12:30:00Z',
  coinSet: { name: 'Five Good Emperors', setType: 'goal', color: '#6b7280', icon: 'crown' },
}

describe('useQuickAccess', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useQuickAccess().clear()
  })

  it('preserves server order and exposes typed membership', async () => {
    vi.mocked(getQuickAccess).mockResolvedValue({ data: { items: [coinItem, setItem] } } as Awaited<ReturnType<typeof getQuickAccess>>)
    const state = useQuickAccess()

    await state.refresh()

    expect(state.items.value).toEqual([coinItem, setItem])
    expect(state.isPinned('coin', 42).value).toBe(true)
    expect(state.isPinned('auction_lot', 42).value).toBe(false)
  })

  it('retains prior items and exposes a recoverable refresh error', async () => {
    vi.mocked(getQuickAccess)
      .mockResolvedValueOnce({ data: { items: [coinItem] } } as Awaited<ReturnType<typeof getQuickAccess>>)
      .mockRejectedValueOnce(new Error('network unavailable'))
    const state = useQuickAccess()

    await state.refresh()
    await state.refresh()

    expect(state.items.value).toEqual([coinItem])
    expect(state.error.value).toBe('network unavailable')
  })

  it('deduplicates concurrent refreshes within the same authenticated generation', async () => {
    let resolve!: (value: Awaited<ReturnType<typeof getQuickAccess>>) => void
    vi.mocked(getQuickAccess).mockReturnValue(new Promise((done) => { resolve = done }))
    const state = useQuickAccess()

    const first = state.refresh()
    const second = state.refresh()

    expect(getQuickAccess).toHaveBeenCalledTimes(1)
    expect(second).toBe(first)

    resolve({ data: { items: [coinItem] } } as Awaited<ReturnType<typeof getQuickAccess>>)
    await Promise.all([first, second])
    expect(state.items.value).toEqual([coinItem])
  })

  it('prepends a 201 item and replaces a 200 item in place', async () => {
    vi.mocked(getQuickAccess).mockResolvedValue({ data: { items: [setItem] } } as Awaited<ReturnType<typeof getQuickAccess>>)
    vi.mocked(pinQuickAccess)
      .mockResolvedValueOnce({ status: 201, data: coinItem } as Awaited<ReturnType<typeof pinQuickAccess>>)
      .mockResolvedValueOnce({ status: 200, data: { ...coinItem, coin: { ...coinItem.coin, classification: 'owned' } } } as Awaited<ReturnType<typeof pinQuickAccess>>)
    const state = useQuickAccess()
    await state.refresh()

    await state.pin('coin', 42)
    expect(state.items.value.map((item) => item.id)).toEqual([42, 7])

    await state.pin('coin', 42)
    expect(state.items.value[0]).toMatchObject({ type: 'coin', coin: { classification: 'owned' } })
  })

  it('removes only after successful DELETE and preserves state on failure', async () => {
    vi.mocked(getQuickAccess).mockResolvedValue({ data: { items: [coinItem] } } as Awaited<ReturnType<typeof getQuickAccess>>)
    vi.mocked(unpinQuickAccess).mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce({ data: undefined } as Awaited<ReturnType<typeof unpinQuickAccess>>)
    const state = useQuickAccess()
    await state.refresh()

    await expect(state.unpin('coin', 42)).rejects.toThrow('offline')
    expect(state.items.value).toEqual([coinItem])

    await state.unpin('coin', 42)
    expect(state.items.value).toEqual([])
  })

  it('tracks per-item busy state while a mutation is pending', async () => {
    let resolve!: (value: Awaited<ReturnType<typeof pinQuickAccess>>) => void
    vi.mocked(pinQuickAccess).mockReturnValue(new Promise((done) => { resolve = done }))
    const state = useQuickAccess()

    const request = state.pin('coin', 42)
    expect(state.isBusy('coin', 42).value).toBe(true)
    expect(state.isBusy('coin_set', 42).value).toBe(false)
    resolve({ status: 201, data: coinItem } as Awaited<ReturnType<typeof pinQuickAccess>>)
    await request
    expect(state.isBusy('coin', 42).value).toBe(false)
  })

  it('clear invalidates a delayed response so prior-user data cannot return', async () => {
    let resolve!: (value: Awaited<ReturnType<typeof getQuickAccess>>) => void
    vi.mocked(getQuickAccess).mockReturnValue(new Promise((done) => { resolve = done }))
    const state = useQuickAccess()

    const request = state.refresh()
    state.clear()
    resolve({ data: { items: [coinItem] } } as Awaited<ReturnType<typeof getQuickAccess>>)
    await request

    expect(state.items.value).toEqual([])
    expect(state.loading.value).toBe(false)
  })

  it('clear invalidates delayed pin and unpin responses from the prior user', async () => {
    let resolvePin!: (value: Awaited<ReturnType<typeof pinQuickAccess>>) => void
    let resolveUnpin!: (value: Awaited<ReturnType<typeof unpinQuickAccess>>) => void
    vi.mocked(getQuickAccess).mockResolvedValue({ data: { items: [coinItem] } } as Awaited<ReturnType<typeof getQuickAccess>>)
    vi.mocked(pinQuickAccess).mockReturnValue(new Promise((done) => { resolvePin = done }))
    vi.mocked(unpinQuickAccess).mockReturnValue(new Promise((done) => { resolveUnpin = done }))
    const state = useQuickAccess()

    await state.refresh()
    const pinRequest = state.pin('coin_set', 7)
    const unpinRequest = state.unpin('coin', 42)
    state.clear()
    resolvePin({ status: 201, data: setItem } as Awaited<ReturnType<typeof pinQuickAccess>>)
    resolveUnpin({ data: undefined } as Awaited<ReturnType<typeof unpinQuickAccess>>)
    await Promise.all([pinRequest, unpinRequest])

    expect(state.items.value).toEqual([])
    expect(state.isBusy('coin_set', 7).value).toBe(false)
    expect(state.isBusy('coin', 42).value).toBe(false)
  })
})
