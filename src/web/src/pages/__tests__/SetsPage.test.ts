import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SetsPage from '@/pages/SetsPage.vue'

const mockGetSets = vi.fn()
const mockCreateSetApi = vi.fn()
const mockCreateSetFromCsv = vi.fn()
const mockCreateSetBuilderRun = vi.fn()
const mockShowAlert = vi.fn()
const mockPush = vi.fn()

vi.mock('@/api/client', () => ({
  getApiErrorMessage: (error: unknown) => {
    const err = error as { response?: { data?: { error?: string } } }
    return err.response?.data?.error ?? ''
  },
  getSets: (...args: unknown[]) => mockGetSets(...args),
  createSet: (...args: unknown[]) => mockCreateSetApi(...args),
  createSetFromCsv: (...args: unknown[]) => mockCreateSetFromCsv(...args),
  createSetBuilderRun: (...args: unknown[]) => mockCreateSetBuilderRun(...args),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
}))

vi.mock('@/composables/usePwa', () => ({
  usePwa: () => ({ isPwa: false }),
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({
    showAlert: (...args: unknown[]) => mockShowAlert(...args),
    showConfirm: vi.fn(),
  }),
}))

describe('SetsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetSets.mockResolvedValue({ data: { sets: [] } })
  })

  it('submits an agentic set builder run instead of creating a set locally, then shows a confirmation dialog', async () => {
    mockCreateSetBuilderRun.mockResolvedValue({ data: { run: { id: 1, status: 'queued' } } })
    const wrapper = mount(SetsPage, { global: { stubs: { Teleport: true } } })
    await flushPromises()

    await wrapper.find('.btn-primary').trigger('click')
    await flushPromises()

    await wrapper.find('#setType').setValue('agentic')
    await wrapper.find('#agenticPrompt').setValue('All US silver quarters from 1940s to 1960s')
    await wrapper.find('#setName').setValue('Silver Quarters')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(mockCreateSetBuilderRun).toHaveBeenCalledWith({ prompt: 'All US silver quarters from 1940s to 1960s' })
    expect(mockCreateSetApi).not.toHaveBeenCalled()
    expect(mockCreateSetFromCsv).not.toHaveBeenCalled()
    expect(mockPush).not.toHaveBeenCalled()
    expect(mockShowAlert).toHaveBeenCalledWith(
      expect.stringContaining('proposal request'),
      expect.objectContaining({ title: 'Proposal Request Submitted' }),
    )
    expect(wrapper.find('.card').exists()).toBe(false)
  })

  it('shows a dialog with the server error when the agentic run request fails', async () => {
    mockCreateSetBuilderRun.mockRejectedValue({ response: { data: { error: 'Prompt is required' } } })
    const wrapper = mount(SetsPage, { global: { stubs: { Teleport: true } } })
    await flushPromises()

    await wrapper.find('.btn-primary').trigger('click')
    await flushPromises()

    await wrapper.find('#setType').setValue('agentic')
    await wrapper.find('#agenticPrompt').setValue('All US silver quarters')
    await wrapper.find('#setName').setValue('Silver Quarters')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(mockShowAlert).toHaveBeenCalledWith('Prompt is required', { title: 'Failed to Submit Proposal Request' })
  })
})
