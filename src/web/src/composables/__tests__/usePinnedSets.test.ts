import { beforeEach, expect, it, vi } from 'vitest'
import { usePinnedSets } from '../usePinnedSets'
import { useQuickAccess } from '../useQuickAccess'
import { getQuickAccess, pinQuickAccess, unpinQuickAccess, getSets } from '@/api/client'
import type { QuickAccessCoinSetItem, QuickAccessItem } from '@/types'

vi.mock('@/api/client', () => ({
  getSets: vi.fn(), getQuickAccess: vi.fn(), pinQuickAccess: vi.fn(), unpinQuickAccess: vi.fn(),
  getApiErrorMessage: (error: unknown) => error instanceof Error ? error.message : '',
}))

function item(id = 1, name = 'Twelve Caesars', pinnedAt = '2026-01-01T00:00:00Z'): QuickAccessCoinSetItem {
  return { type: 'coin_set', id, pinnedAt, coinSet: { name, color: '', icon: '', setType: 'standard' } }
}
function response(items: QuickAccessItem[]) {
  return { data: { items } } as Awaited<ReturnType<typeof getQuickAccess>>
}

beforeEach(() => { vi.resetAllMocks(); useQuickAccess().clear() })

it('projects only set pins from the authoritative source without a second list request', async () => {
  vi.mocked(getQuickAccess).mockResolvedValue(response([
    { type: 'coin', id: 8, pinnedAt: '2026-01-01T00:00:00Z', coin: { name: 'Coin', classification: 'owned', primaryImageUrl: '' } },
    item(),
  ]))
  await usePinnedSets().refresh()
  expect(usePinnedSets().pinnedSets.value.map(set => set.id)).toEqual([1])
  expect(getSets).not.toHaveBeenCalled()
  expect(getQuickAccess).toHaveBeenCalledTimes(1)
})

it('sorts oldest pinned first with the name tie-break without altering Quick Access order', () => {
  useQuickAccess().items.value = [item(1, 'Zebra', '2026-01-02T00:00:00Z'), item(3, 'Beta'), item(2, 'Alpha')]
  expect(usePinnedSets().pinnedSets.value.map(set => set.id)).toEqual([2, 3, 1])
  expect(useQuickAccess().items.value.map(pin => pin.id)).toEqual([1, 3, 2])
})

it('reflects pin and unpin immediately in both surfaces without another cache', async () => {
  vi.mocked(pinQuickAccess).mockResolvedValue({ status: 201, data: item(5, 'New pin') } as Awaited<ReturnType<typeof pinQuickAccess>>)
  const state = usePinnedSets()
  await state.setPinned(5, true)
  expect(pinQuickAccess).toHaveBeenCalledWith('coin_set', 5)
  expect(state.pinnedSets.value.map(set => set.id)).toEqual([5])
  expect(useQuickAccess().isPinned('coin_set', 5).value).toBe(true)
  await state.setPinned(5, false)
  expect(unpinQuickAccess).toHaveBeenCalledWith('coin_set', 5)
  expect(state.pinnedSets.value).toEqual([])
  expect(useQuickAccess().items.value).toEqual([])
  expect(getQuickAccess).not.toHaveBeenCalled()
})

it('rethrows pin failures and exposes recoverable refresh errors while retaining current data', async () => {
  vi.mocked(pinQuickAccess).mockRejectedValue(new Error('you can pin up to 5 sets'))
  const state = usePinnedSets()
  await expect(state.setPinned(9, true)).rejects.toThrow('you can pin up to 5 sets')
  useQuickAccess().items.value = [item()]
  vi.mocked(getQuickAccess).mockRejectedValue(new Error('network error'))
  await expect(state.refresh()).resolves.toBeUndefined()
  expect(state.pinnedSets.value).toHaveLength(1)
  expect(state.error.value).toBe('network error')
})

it('clear invalidates the pending A refresh after B has loaded', async () => {
  let resolve!: (value: Awaited<ReturnType<typeof getQuickAccess>>) => void
  vi.mocked(getQuickAccess).mockReturnValueOnce(new Promise(done => { resolve = done }))
    .mockResolvedValueOnce(response([item(2, 'Account B')]))
  const state = usePinnedSets()
  const old = state.refresh()
  state.clear()
  expect(state.pinnedSets.value).toEqual([])
  await state.refresh()
  resolve(response([item(1, 'Account A')]))
  await old
  expect(state.pinnedSets.value.map(set => set.name)).toEqual(['Account B'])
})

it('old account mutations cannot add or remove next-account pins', async () => {
  let resolvePin!: (value: Awaited<ReturnType<typeof pinQuickAccess>>) => void
  let resolveUnpin!: () => void
  vi.mocked(pinQuickAccess).mockReturnValueOnce(new Promise(done => { resolvePin = done }))
  vi.mocked(unpinQuickAccess).mockImplementationOnce(() => new Promise(done => {
    resolveUnpin = () => done({ data: undefined } as Awaited<ReturnType<typeof unpinQuickAccess>>)
  }))
  const state = usePinnedSets()
  const pin = state.setPinned(1, true)
  const unpin = state.setPinned(2, false)
  state.clear()
  useQuickAccess().items.value = [item(2, 'Account B')]
  resolvePin({ status: 201, data: item(1, 'Account A') } as Awaited<ReturnType<typeof pinQuickAccess>>)
  resolveUnpin()
  await Promise.all([pin, unpin])
  expect(state.pinnedSets.value.map(set => set.name)).toEqual(['Account B'])
})

it('enforces the five-set limit without counting other target types', () => {
  const state = usePinnedSets()
  useQuickAccess().items.value = Array.from({ length: 4 }, (_, index) => item(index))
  expect(state.pinLimitReached.value).toBe(false)
  useQuickAccess().items.value.push(item(5))
  expect(state.pinLimitReached.value).toBe(true)
  state.clear()
  expect(state.pinLimitReached.value).toBe(false)
  expect(state.pinnedSets.value).toEqual([])
})
