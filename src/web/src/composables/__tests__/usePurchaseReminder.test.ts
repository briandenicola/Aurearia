import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

// Independent QA coverage for Feature 355 -- usePurchaseReminder composable.
// Owned by Brutus (Tester/QA).
//
// Frozen contract (spec.md FR-001..FR-005, FR-010, FR-013, D3, T027):
//   - fetchReminder: calls GET /coins/{id}/reminder; null if 404.
//   - saveReminder: calls POST /coins/{id}/reminder with { remindDate, timezone }.
//   - cancelReminder: calls DELETE /coins/{id}/reminder; clears reminder ref.
//   - Auto-detects browser timezone via Intl.DateTimeFormat().resolvedOptions().timeZone (D3).
//   - Loading / saving flags are correctly toggled around async calls.

const mocks = vi.hoisted(() => ({
  getPurchaseReminder: vi.fn(),
  createOrUpdatePurchaseReminder: vi.fn(),
  deletePurchaseReminder: vi.fn(),
  listPurchaseReminders: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

import { usePurchaseReminder } from '@/composables/usePurchaseReminder'

describe('usePurchaseReminder (Feature 355)', () => {
  beforeEach(() => {
    Object.values(mocks).forEach((m) => m.mockReset())
  })

  it('fetchReminder calls getPurchaseReminder with coinId and populates reminder ref', async () => {
    const expectedReminder = {
      id: 1,
      coinId: 42,
      remindDate: '2026-09-15',
      timezone: 'America/Chicago',
      status: 'pending' as const,
      createdAt: '2026-09-01T00:00:00Z',
      updatedAt: '2026-09-01T00:00:00Z',
    }
    mocks.getPurchaseReminder.mockResolvedValue({ data: expectedReminder })

    const coinId = ref(42)
    const { reminder, fetchReminder } = usePurchaseReminder(coinId)
    await fetchReminder()

    expect(mocks.getPurchaseReminder).toHaveBeenCalledWith(42)
    expect(reminder.value).toEqual(expectedReminder)
  })

  it('fetchReminder sets reminder to null on 404 response', async () => {
    mocks.getPurchaseReminder.mockRejectedValue({
      response: { status: 404 },
    })

    const coinId = ref(42)
    const { reminder, fetchReminder } = usePurchaseReminder(coinId)
    await fetchReminder()

    expect(reminder.value).toBeNull()
  })

  it('saveReminder sends { remindDate, timezone } via createOrUpdatePurchaseReminder', async () => {
    const saved = {
      id: 1,
      coinId: 42,
      remindDate: '2026-10-01',
      timezone: 'America/Chicago',
      status: 'pending' as const,
      createdAt: '2026-09-01T00:00:00Z',
      updatedAt: '2026-09-01T00:00:00Z',
    }
    mocks.createOrUpdatePurchaseReminder.mockResolvedValue({ data: saved })

    const coinId = ref(42)
    const { reminder, saveReminder } = usePurchaseReminder(coinId)
    await saveReminder('2026-10-01')

    expect(mocks.createOrUpdatePurchaseReminder).toHaveBeenCalledWith(42, {
      remindDate: '2026-10-01',
      timezone: expect.any(String),
    })
    expect(reminder.value).toEqual(saved)
  })

  it('saveReminder sends a non-empty IANA timezone string (D3 auto-detect)', async () => {
    mocks.createOrUpdatePurchaseReminder.mockResolvedValue({
      data: { id: 1, coinId: 42, status: 'pending' as const, createdAt: '', updatedAt: '', remindDate: '', timezone: '' },
    })

    const coinId = ref(42)
    const { saveReminder } = usePurchaseReminder(coinId)
    await saveReminder('2026-10-01')

    const [, payload] = mocks.createOrUpdatePurchaseReminder.mock.calls[0] ?? []
    expect(typeof payload?.timezone).toBe('string')
    expect((payload?.timezone ?? '').length).toBeGreaterThan(0)
  })

  it('cancelReminder calls deletePurchaseReminder with coinId and clears reminder ref', async () => {
    mocks.getPurchaseReminder.mockResolvedValue({
      data: { id: 1, coinId: 42, remindDate: '2026-10-01', timezone: 'UTC', status: 'pending' as const, createdAt: '', updatedAt: '' },
    })
    mocks.deletePurchaseReminder.mockResolvedValue({})

    const coinId = ref(42)
    const { reminder, fetchReminder, cancelReminder } = usePurchaseReminder(coinId)
    await fetchReminder()
    expect(reminder.value).not.toBeNull()

    await cancelReminder()

    expect(mocks.deletePurchaseReminder).toHaveBeenCalledWith(42)
    // FR-010: cancelled reminder cleared from active view
    expect(reminder.value).toBeNull()
  })

  it('loading flag is true during fetchReminder and false after', async () => {
    let resolveFetch!: (v: unknown) => void
    mocks.getPurchaseReminder.mockReturnValue(new Promise((r) => { resolveFetch = r }))

    const coinId = ref(42)
    const { loading, fetchReminder } = usePurchaseReminder(coinId)
    expect(loading.value).toBe(false)

    const promise = fetchReminder()
    expect(loading.value).toBe(true)

    resolveFetch({ data: null })
    await promise
    expect(loading.value).toBe(false)
  })

  it('saving flag is true during saveReminder and false after', async () => {
    let resolveSave!: (v: unknown) => void
    mocks.createOrUpdatePurchaseReminder.mockReturnValue(new Promise((r) => { resolveSave = r }))

    const coinId = ref(42)
    const { saving, saveReminder } = usePurchaseReminder(coinId)
    expect(saving.value).toBe(false)

    const promise = saveReminder('2026-10-01')
    expect(saving.value).toBe(true)

    resolveSave({ data: { id: 1, coinId: 42, status: 'pending' as const, createdAt: '', updatedAt: '', remindDate: '', timezone: '' } })
    await promise
    expect(saving.value).toBe(false)
  })

  it('composable uses the updated coinId when coinId ref changes', async () => {
    mocks.getPurchaseReminder.mockResolvedValue({ data: null })

    const coinId = ref(42)
    const { fetchReminder } = usePurchaseReminder(coinId)
    await fetchReminder()
    expect(mocks.getPurchaseReminder).toHaveBeenCalledWith(42)

    coinId.value = 99
    await fetchReminder()
    expect(mocks.getPurchaseReminder).toHaveBeenCalledWith(99)
  })

  it('returns a non-null error string on non-404 fetch failure', async () => {
    mocks.getPurchaseReminder.mockRejectedValue(new Error('Network error'))

    const coinId = ref(42)
    const { error, fetchReminder } = usePurchaseReminder(coinId)
    await fetchReminder()

    expect(error.value.length).toBeGreaterThan(0)
  })
})
