import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PromotionReadinessPanel from '../PromotionReadinessPanel.vue'
import { promoteQuickCaptureDraft } from '@/api/client'
import type { QuickCaptureDraft, QuickCapturePromoteOverrides } from '@/types'

vi.mock('@/api/client', () => ({
  getApiErrorMessage: vi.fn(() => ''),
  promoteQuickCaptureDraft: vi.fn(),
}))

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const source = fs.readFileSync(path.resolve(__dirname, '../PromotionReadinessPanel.vue'), 'utf8')

function draft(overrides: Partial<QuickCaptureDraft> = {}): QuickCaptureDraft {
  return {
    id: 12,
    userId: 1,
    workingTitle: 'Edited denarius',
    dateRange: 'c. 100',
    era: 'ancient',
    acquisitionSource: 'Show table',
    purchasePrice: 42,
    notes: 'Draft notes',
    source: 'quick_capture',
    ngcCertNumber: '',
    ngcLookupUrl: '',
    ngcGrade: '',
    labelText: '',
    aiConfidence: '',
    status: 'active',
    promotedCoinId: null,
    promotedAt: null,
    discardedAt: null,
    images: [],
    createdAt: '2026-06-30T00:00:00Z',
    updatedAt: '2026-06-30T00:00:00Z',
    ...overrides,
  }
}

function mountPanel(promotionOverrides: QuickCapturePromoteOverrides = {}, draftOverrides: Partial<QuickCaptureDraft> = {}) {
  return mount(PromotionReadinessPanel, {
    props: { draft: draft(draftOverrides), promotionOverrides },
    global: { stubs: { RouterLink: true } },
  })
}

describe('PromotionReadinessPanel', () => {
  beforeEach(() => {
    vi.mocked(promoteQuickCaptureDraft).mockReset()
    vi.mocked(promoteQuickCaptureDraft).mockResolvedValue({
      data: { draftId: 12, status: 'promoted', coinId: 77, alreadyPromoted: false, target: 'collection' },
    })
  })

  it('provides explicit, retry-safe promotion controls and field guidance', () => {
    expect(source).toContain('promoteQuickCaptureDraft')
    expect(source).toContain('confirm')
    expect(source).toContain('fieldErrors.name')
    expect(source).toContain('alreadyPromoted')
    expect(source).toContain(':disabled="!confirmed || promoting || !hasRequiredName"')
    expect(source).toContain("emit('promoted'")
  })

  it('lets the collector promote to either collection or wishlist using the backend target contract', () => {
    expect(source).toContain("type PromotionTarget = 'collection' | 'wishlist'")
    expect(source).toContain('v-model="target"')
    expect(source).toContain('value="collection"')
    expect(source).toContain('value="wishlist"')
    expect(source).toContain('target: target.value')
    expect(source).toContain('fieldErrors.target')
    expect(source).toContain('Promote to ${destinationLabel}')
  })

  it('does not duplicate editable draft fields in the promotion panel', () => {
    const wrapper = mountPanel()

    expect(wrapper.find('input[type="text"]').exists()).toBe(false)
    expect(wrapper.find('input[type="number"]').exists()).toBe(false)
    expect(wrapper.find('select').exists()).toBe(false)
    expect(wrapper.find('textarea').exists()).toBe(false)
    expect(source).not.toContain('overrideName')
    expect(source).not.toContain('overrideCategory')
    expect(source).not.toContain('overrideMaterial')
    expect(source).not.toContain('overrideEra')
    expect(source).not.toContain('overrideNotes')
  })

  it('promotes to collection with current edited draft overrides from the page', async () => {
    const overrides: QuickCapturePromoteOverrides = {
      name: 'Current edited title',
      era: 'ancient',
      purchaseLocation: 'Current source',
      purchasePrice: 123,
      notes: 'Current notes',
    }
    const wrapper = mountPanel(overrides)

    await wrapper.find('input[type="checkbox"]').setValue(true)
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(promoteQuickCaptureDraft).toHaveBeenCalledWith(12, {
      confirm: true,
      target: 'collection',
      overrides,
    })
  })

  it('treats an explicitly cleared current title as missing instead of falling back to the saved draft title', async () => {
    const wrapper = mountPanel({ name: '', purchaseLocation: '', notes: '' })

    expect(wrapper.text()).toContain('Working title is required')
    expect(wrapper.text()).not.toContain('Required title is ready')

    await wrapper.find('input[type="checkbox"]').setValue(true)
    expect(wrapper.find('button.btn-primary').attributes('disabled')).toBeDefined()

    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(promoteQuickCaptureDraft).not.toHaveBeenCalled()
  })

  it('sends explicit null when saved price is 100 and current purchase price is cleared', async () => {
    const overrides: QuickCapturePromoteOverrides = {
      name: 'Current title',
      purchaseLocation: '',
      purchasePrice: null,
      notes: '',
    }
    const wrapper = mountPanel(overrides, { purchasePrice: 100 })

    await wrapper.find('input[type="checkbox"]').setValue(true)
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(promoteQuickCaptureDraft).toHaveBeenCalledWith(12, {
      confirm: true,
      target: 'collection',
      overrides,
    })
  })

  it('preserves wishlist target selection when promoting', async () => {
    vi.mocked(promoteQuickCaptureDraft).mockResolvedValue({
      data: { draftId: 12, status: 'promoted', coinId: 78, alreadyPromoted: false, target: 'wishlist' },
    })
    const wrapper = mountPanel({ name: 'Wishlist coin' })

    await wrapper.find('input[value="wishlist"]').setValue(true)
    await wrapper.find('input[type="checkbox"]').setValue(true)
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(promoteQuickCaptureDraft).toHaveBeenCalledWith(12, expect.objectContaining({
      confirm: true,
      target: 'wishlist',
      overrides: { name: 'Wishlist coin' },
    }))
  })
})
