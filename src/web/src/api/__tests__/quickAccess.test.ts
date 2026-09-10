import { describe, expect, expectTypeOf, it, vi } from 'vitest'
import { getQuickAccess, pinQuickAccess, unpinQuickAccess } from '@/api/endpoints/quickAccess'
import type { QuickAccessItem } from '@/types'

const api = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
}))

vi.mock('@/api/http', () => ({ api }))

describe('Quick Access endpoints', () => {
  it('lists from the authenticated API path', async () => {
    api.get.mockResolvedValue({ data: { items: [] } })
    await getQuickAccess()
    expect(api.get).toHaveBeenCalledWith('/quick-access')
  })

  it.each([200, 201])('pins with PUT and no request body and preserves a %i response', async (status) => {
    const item: QuickAccessItem = {
      type: 'auction_lot',
      id: 19,
      pinnedAt: '2026-09-10T12:30:00Z',
      auctionLot: {
        title: 'Hadrian Aureus',
        status: 'bidding',
        auctionHouse: 'CNG',
        saleDate: null,
        auctionEndTime: '2026-10-01T19:00:00Z',
        imageUrl: '',
      },
    }
    api.put.mockResolvedValue({ status, data: item })
    const response = await pinQuickAccess('auction_lot', 19)

    expect(api.put).toHaveBeenCalledWith('/quick-access/auction_lot/19')
    expect(response).toMatchObject({ status, data: item })
    expectTypeOf(response.data).toEqualTypeOf<QuickAccessItem>()
  })

  it('unpins with DELETE and accepts a no-content response', async () => {
    api.delete.mockResolvedValue({ status: 204, data: undefined })
    await unpinQuickAccess('calendar_event', 11)
    expect(api.delete).toHaveBeenCalledWith('/quick-access/calendar_event/11')
  })
})
