import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CollectorProfileSection from '@/components/settings/CollectorProfileSection.vue'

const mockGet = vi.fn()
const mockPut = vi.fn()
const mockUpdateAdminSettings = vi.fn()

vi.mock('@/api/endpoints/collectorProfile', () => ({
  getCollectorProfile: (...args: unknown[]) => mockGet(...args),
  replaceCollectorProfile: (...args: unknown[]) => mockPut(...args),
}))
vi.mock('@/api/client', () => ({
  updateAppSettings: (...args: unknown[]) => mockUpdateAdminSettings(...args),
}))

const stored = {
  budgetMin: 100,
  budgetMax: 500,
  currency: 'USD',
  preferredPeriods: ['Flavian'],
  preferredCategories: ['Roman'],
  excludedCategories: ['Modern'],
  preferredDealers: ['Example Dealer'],
  collectingGoals: ['Add a documented denarius'],
  updatedAt: '2026-09-19T12:00:00Z',
  isDefault: false,
}

describe('CollectorProfileSection', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    mockGet.mockResolvedValue({ data: stored })
    mockPut.mockImplementation(async (input) => ({
      data: { ...input, updatedAt: '2026-09-20T12:00:00Z', isDefault: false },
    }))
  })

  it('loads, edits, saves, reloads, and clears through the profile API only', async () => {
    const wrapper = mount(CollectorProfileSection)
    await flushPromises()
    expect((wrapper.get('[data-test="currency"]').element as HTMLInputElement).value).toBe('USD')
    await wrapper.get('[data-test="currency"]').setValue('EUR')
    await wrapper.get('[data-test="periods"]').setValue('Severan, Antonine')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(mockPut).toHaveBeenLastCalledWith(expect.objectContaining({
      currency: 'EUR',
      preferredPeriods: ['Severan', 'Antonine'],
    }))
    expect(mockGet).toHaveBeenCalledTimes(1)
    expect(mockUpdateAdminSettings).not.toHaveBeenCalled()
    expect(localStorage.length).toBe(0)

    wrapper.unmount()
    mockGet.mockResolvedValueOnce({ data: { ...stored, currency: 'EUR', preferredPeriods: ['Severan', 'Antonine'] } })
    const reloaded = mount(CollectorProfileSection)
    await flushPromises()
    expect((reloaded.get('[data-test="currency"]').element as HTMLInputElement).value).toBe('EUR')
    expect(mockGet).toHaveBeenCalledTimes(2)

    await reloaded.get('[data-test="clear"]').trigger('click')
    await flushPromises()
    expect(mockPut).toHaveBeenLastCalledWith({
      budgetMin: null, budgetMax: null, currency: null,
      preferredPeriods: [], preferredCategories: [], excludedCategories: [],
      preferredDealers: [], collectingGoals: [],
    })
    reloaded.unmount()
  })

  it('shows sanitized server field validation', async () => {
    mockPut.mockRejectedValue({ response: { data: { error: 'Invalid collector profile', field: 'currency' } } })
    const wrapper = mount(CollectorProfileSection)
    await flushPromises()
    await wrapper.get('[data-test="currency"]').setValue('usd')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('currency')
    expect(wrapper.text()).not.toContain('[object Object]')
  })

  it('uses labelled keyboard controls, 44px targets, responsive layout, and theme tokens', async () => {
    const wrapper = mount(CollectorProfileSection)
    await flushPromises()
    expect(wrapper.get('form').attributes('class')).toContain('grid-cols-1')
    expect(wrapper.get('section').attributes('class')).toContain('bg-card')
    for (const control of wrapper.findAll('input, textarea, button')) {
      expect(control.attributes('class')).toContain('min-h-11')
    }
    for (const input of wrapper.findAll('input, textarea')) {
      expect(input.attributes('id')).toBeTruthy()
      expect(wrapper.find(`label[for="${input.attributes('id')}"]`).exists()).toBe(true)
    }
    await wrapper.get('[data-test="currency"]').trigger('keydown.enter')
    expect(wrapper.get('button[type="submit"]').exists()).toBe(true)
  })
})
