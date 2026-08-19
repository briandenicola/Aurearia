import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CoinShipmentSection from '../CoinShipmentSection.vue'
import { getCoinShipment, setCoinShipmentManualOverride, syncCoinShipment } from '@/api/client'
import type { Shipment, ShipmentStatus } from '@/types'

vi.mock('@/api/client', () => ({
  getCoinShipment: vi.fn(),
  upsertCoinShipment: vi.fn(),
  deleteCoinShipment: vi.fn(),
  setCoinShipmentManualOverride: vi.fn(),
  syncCoinShipment: vi.fn(),
}))

function makeShipment(overrides: Partial<Shipment> = {}): Shipment {
  return {
    id: 1,
    userId: 1,
    coinId: 42,
    carrier: 'parcel',
    manualCarrierName: '',
    trackingNumber: 'TRK123',
    currentStatus: 'in_transit',
    currentStatusSource: 'carrier_api',
    notes: '',
    manualOverrideEnabled: false,
    manualOverrideStatus: '',
    manualOverrideNote: '',
    manualOverrideUpdatedAt: null,
    lastSyncedAt: null,
    lastSyncError: '',
    estimatedDeliveryAt: null,
    deliveredAt: null,
    events: [],
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function findCheckButton(wrapper: ReturnType<typeof mount>) {
  return wrapper.findAll('button').find(b => b.text().includes('ParcelApp') || b.text().includes('Tracking Complete'))!
}

describe('CoinShipmentSection', () => {
  beforeEach(() => {
    vi.mocked(getCoinShipment).mockReset()
    vi.mocked(setCoinShipmentManualOverride).mockReset()
    vi.mocked(syncCoinShipment).mockReset()
  })

  it('keeps the manual ParcelApp check enabled for a non-delivered status', async () => {
    vi.mocked(getCoinShipment).mockResolvedValue({
      data: { shipment: makeShipment({ currentStatus: 'in_transit' }), trackingUrl: '' },
    } as Awaited<ReturnType<typeof getCoinShipment>>)

    const wrapper = mount(CoinShipmentSection, { props: { coinId: 42 } })
    await flushPromises()

    const button = findCheckButton(wrapper)
    expect(button.text()).toBe('Check ParcelApp Now')
    expect(button.attributes('disabled')).toBeUndefined()
  })

  it('disables the manual ParcelApp check with accessible completion text once delivered', async () => {
    vi.mocked(getCoinShipment).mockResolvedValue({
      data: { shipment: makeShipment({ currentStatus: 'delivered' }), trackingUrl: '' },
    } as Awaited<ReturnType<typeof getCoinShipment>>)

    const wrapper = mount(CoinShipmentSection, { props: { coinId: 42 } })
    await flushPromises()

    const button = findCheckButton(wrapper)
    expect(button.text()).toBe('Tracking Complete')
    expect(button.attributes('disabled')).toBeDefined()
    expect(button.attributes('title')).toMatch(/tracking is complete/i)
  })

  it('re-enables the manual ParcelApp check reactively once status moves away from delivered', async () => {
    vi.mocked(getCoinShipment).mockResolvedValue({
      data: { shipment: makeShipment({ currentStatus: 'delivered' }), trackingUrl: '' },
    } as Awaited<ReturnType<typeof getCoinShipment>>)

    const wrapper = mount(CoinShipmentSection, { props: { coinId: 42 } })
    await flushPromises()

    expect(findCheckButton(wrapper).attributes('disabled')).toBeDefined()

    vi.mocked(setCoinShipmentManualOverride).mockResolvedValue({
      data: { shipment: makeShipment({ currentStatus: 'exception' }), trackingUrl: '' },
    } as Awaited<ReturnType<typeof setCoinShipmentManualOverride>>)

    const statusSelect = wrapper.find('select')
    await statusSelect.setValue('exception' satisfies ShipmentStatus)
    await wrapper.find('.btn-secondary.btn-sm.w-fit').trigger('click')
    await flushPromises()

    const button = findCheckButton(wrapper)
    expect(button.text()).toBe('Check ParcelApp Now')
    expect(button.attributes('disabled')).toBeUndefined()
  })

  it('still disables the check for a non-parcel carrier regardless of status', async () => {
    vi.mocked(getCoinShipment).mockResolvedValue({
      data: { shipment: makeShipment({ currentStatus: 'in_transit', carrier: 'other', manualCarrierName: 'USPS' }), trackingUrl: '' },
    } as Awaited<ReturnType<typeof getCoinShipment>>)

    const wrapper = mount(CoinShipmentSection, { props: { coinId: 42 } })
    await flushPromises()

    const button = findCheckButton(wrapper)
    expect(button.text()).toBe('Check ParcelApp Now')
    expect(button.attributes('disabled')).toBeDefined()
  })

  it('clicking the check button while enabled still calls the sync endpoint', async () => {
    vi.mocked(getCoinShipment).mockResolvedValue({
      data: { shipment: makeShipment({ currentStatus: 'in_transit' }), trackingUrl: '' },
    } as Awaited<ReturnType<typeof getCoinShipment>>)
    vi.mocked(syncCoinShipment).mockResolvedValue({
      data: { shipment: makeShipment({ currentStatus: 'out_for_delivery' }), trackingUrl: '' },
    } as Awaited<ReturnType<typeof syncCoinShipment>>)

    const wrapper = mount(CoinShipmentSection, { props: { coinId: 42 } })
    await flushPromises()

    await findCheckButton(wrapper).trigger('click')
    await flushPromises()

    expect(syncCoinShipment).toHaveBeenCalledWith(42)
    expect(findCheckButton(wrapper).text()).toBe('Check ParcelApp Now')
  })
})
