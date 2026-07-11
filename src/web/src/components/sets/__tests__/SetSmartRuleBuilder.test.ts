import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SetSmartRuleBuilder from '@/components/sets/SetSmartRuleBuilder.vue'

const mockPreviewSmartSet = vi.fn()
const mockGetSuggestedCriteria = vi.fn()
const mockListCriteriaTemplates = vi.fn()
const mockSaveCriteriaTemplate = vi.fn()

vi.mock('@/api/client', () => ({
  previewSmartSet: (...args: unknown[]) => mockPreviewSmartSet(...args),
  getSuggestedCriteria: (...args: unknown[]) => mockGetSuggestedCriteria(...args),
  listCriteriaTemplates: (...args: unknown[]) => mockListCriteriaTemplates(...args),
  saveCriteriaTemplate: (...args: unknown[]) => mockSaveCriteriaTemplate(...args),
}))

function defaultMount() {
  return mount(SetSmartRuleBuilder, { attachTo: document.body })
}

describe('SetSmartRuleBuilder', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetSuggestedCriteria.mockResolvedValue({
      data: {
        suggestions: [
          {
            id: 'silver-coins',
            name: 'Silver Coins',
            description: 'All silver coins',
            criteria: { operator: 'and', rules: [{ field: 'material', op: 'eq', value: 'Silver' }] },
          },
          {
            id: 'roman-collection',
            name: 'Roman Collection',
            description: 'All Roman coins',
            criteria: { operator: 'and', rules: [{ field: 'category', op: 'eq', value: 'Roman' }] },
          },
        ],
      },
    })
    mockListCriteriaTemplates.mockResolvedValue({ data: { templates: [] } })
  })

  it('renders with a default rule row', async () => {
    const wrapper = defaultMount()
    await flushPromises()
    const rows = wrapper.findAll('.rule-row')
    expect(rows.length).toBeGreaterThanOrEqual(1)
  })

  it('emits update on mount with initial criteria', async () => {
    const wrapper = defaultMount()
    await flushPromises()
    expect(wrapper.emitted('update')).toBeTruthy()
    const emitted = wrapper.emitted('update') as unknown[][]
    const criteria = emitted[emitted.length - 1][0] as { operator: string; rules: unknown[] }
    expect(criteria.operator).toBe('and')
    expect(criteria.rules.length).toBe(1)
  })

  it('adds a new rule row when Add rule is clicked', async () => {
    const wrapper = defaultMount()
    await flushPromises()
    const before = wrapper.findAll('.rule-row').length
    await wrapper.find('button[data-testid="add-rule"], .rule-actions button:first-child').trigger('click')
    await wrapper.vm.$nextTick()
    expect(wrapper.findAll('.rule-row').length).toBe(before + 1)
  })

  it('removes a rule when remove button is clicked', async () => {
    const wrapper = defaultMount()
    await flushPromises()
    // Add a second rule first
    const addBtn = wrapper.findAll('.rule-actions button')[0]
    await addBtn.trigger('click')
    await wrapper.vm.$nextTick()
    expect(wrapper.findAll('.rule-row').length).toBe(2)

    // Remove the first rule
    const removeBtn = wrapper.find('.remove-btn')
    await removeBtn.trigger('click')
    await wrapper.vm.$nextTick()
    expect(wrapper.findAll('.rule-row').length).toBe(1)
  })

  it('switches operator between all/any', async () => {
    const wrapper = defaultMount()
    await flushPromises()
    const anyBtn = wrapper.find('.operator-toggle button:nth-child(3)')
    await anyBtn.trigger('click')
    await wrapper.vm.$nextTick()
    const emitted = wrapper.emitted('update') as unknown[][]
    const lastCriteria = emitted[emitted.length - 1][0] as { operator: string }
    expect(lastCriteria.operator).toBe('or')
  })

  it('shows suggestions after loading', async () => {
    const wrapper = defaultMount()
    await flushPromises()
    const chips = wrapper.findAll('.suggestion-chips .chip')
    expect(chips.length).toBe(2)
    expect(chips[0].text()).toBe('Silver Coins')
    expect(chips[1].text()).toBe('Roman Collection')
  })

  it('applies a suggestion when chip is clicked', async () => {
    const wrapper = defaultMount()
    await flushPromises()
    const firstChip = wrapper.find('.suggestion-chips .chip')
    await firstChip.trigger('click')
    await wrapper.vm.$nextTick()
    const emitted = wrapper.emitted('update') as unknown[][]
    const criteria = emitted[emitted.length - 1][0] as {
      operator: string
      rules: Array<{ field: string; op: string; value: unknown }>
    }
    expect(criteria.rules[0].field).toBe('material')
    expect(criteria.rules[0].value).toBe('Silver')
  })

  it('shows preview result after clicking Preview', async () => {
    mockPreviewSmartSet.mockResolvedValue({
      data: { coinCount: 5, totalValue: 250.0, coinIds: [1, 2, 3, 4, 5] },
    })
    const wrapper = defaultMount()
    await flushPromises()
    const previewBtn = wrapper.findAll('.rule-actions button')[1]
    await previewBtn.trigger('click')
    await flushPromises()
    expect(wrapper.find('.preview-result').exists()).toBe(true)
    expect(wrapper.find('.preview-count').text()).toBe('5')
  })

  it('shows preview error when preview fails', async () => {
    mockPreviewSmartSet.mockRejectedValue({
      response: { data: { error: 'criteria field "badfield" is not allowed' } },
    })
    const wrapper = defaultMount()
    await flushPromises()
    const previewBtn = wrapper.findAll('.rule-actions button')[1]
    await previewBtn.trigger('click')
    await flushPromises()
    expect(wrapper.find('.preview-error').exists()).toBe(true)
    expect(wrapper.find('.preview-error').text()).toContain('is not allowed')
  })

  it('shows save-as-template form when button is clicked', async () => {
    const wrapper = defaultMount()
    await flushPromises()
    const saveBtn = wrapper.find('.section-block .btn-ghost')
    await saveBtn.trigger('click')
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.save-form').exists()).toBe(true)
  })

  it('saves a template and adds it to saved list', async () => {
    const newTemplate = {
      id: 1,
      userId: 1,
      name: 'My Silver',
      description: '',
      criteria: { operator: 'and', rules: [{ field: 'material', op: 'eq', value: 'Silver' }] },
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    }
    mockSaveCriteriaTemplate.mockResolvedValue({ data: newTemplate })

    const wrapper = defaultMount()
    await flushPromises()

    const saveBtn = wrapper.find('.section-block .btn-ghost')
    await saveBtn.trigger('click')
    await wrapper.vm.$nextTick()

    const nameInput = wrapper.find('.save-form input')
    await nameInput.setValue('My Silver')

    const confirmBtn = wrapper.find('.save-form .btn-primary')
    await confirmBtn.trigger('click')
    await flushPromises()

    expect(mockSaveCriteriaTemplate).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'My Silver' })
    )
    // Save form should close
    expect(wrapper.find('.save-form').exists()).toBe(false)
    // Template should now appear in saved list
    expect(wrapper.find('.template-select').exists()).toBe(true)
  })
})
