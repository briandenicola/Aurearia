import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { defineComponent, type PropType } from 'vue'
import type { Coin } from '@/types'
import AddCoinPage from '../AddCoinPage.vue'

const mocks = vi.hoisted(() => ({
  draft: vi.fn(), commit: vi.fn(), upload: vi.fn(), add: vi.fn(), alert: vi.fn(),
  push: vi.fn(), update: vi.fn(), extract: vi.fn(),
}))
vi.mock('vue-router', () => ({ useRoute: () => ({ query: {} }), useRouter: () => ({ push: mocks.push }) }))
vi.mock('@/stores/coins', () => ({ useCoinsStore: () => ({ addCoin: mocks.add }) }))
vi.mock('@/composables/usePwa', () => ({ usePwa: () => ({ isPwa: true }) }))
vi.mock('@/composables/useDialog', () => ({ useDialog: () => ({ showAlert: mocks.alert }) }))
vi.mock('@/api/client', () => ({
  createIntakeDraft: mocks.draft, commitIntakeDraft: mocks.commit, uploadImage: mocks.upload,
  updateCoin: mocks.update, extractText: mocks.extract,
}))
vi.mock('@/composables/useCoinOptions', async () => {
  const { ref } = await import('vue')
  return { useCoinOptions: () => ({
    categoryOptions: ref(['Roman', 'Other']), materialOptions: ref(['Silver', 'Other']),
    eraOptions: ref(['Ancient', 'Modern']), loadOptions: vi.fn(),
  }) }
})

const photo = new File(['photo'], 'obverse.jpg', { type: 'image/jpeg' })
const CoinFormStub = defineComponent({
  props: { form: Object as PropType<Partial<Coin>>, loading: Boolean }, emits: ['submit'],
  setup(_, { expose }) { expose({ obverseFile: photo }) },
  template: '<button @click="$emit(\'submit\')">Save manual</button>',
})
function render() {
  return shallowMount(AddCoinPage, {
    global: { stubs: {
      CoinForm: CoinFormStub, RouterLink: true,
      InlineCameraCapturePanel: {
        name: 'InlineCameraCapturePanel', emits: ['captured', 'upload'],
        methods: { stopCamera() {} },
        template: '<div><slot name="before-actions" /><slot name="footer" /></div>',
      },
    } },
  })
}
async function click(wrapper: ReturnType<typeof render>, text: string) {
  const button = wrapper.findAll('button').find(button => button.text().toLowerCase().includes(text.toLowerCase()))
  expect(button, `button ${text}`).toBeDefined()
  await button!.trigger('click')
  await flushPromises()
}
async function draft(wrapper: ReturnType<typeof render>) {
  wrapper.findComponent({ name: 'InlineCameraCapturePanel' }).vm.$emit('captured', photo)
  await flushPromises()
  await click(wrapper, 'Generate Intake Draft')
}

describe('Add Coin save workflow', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.draft.mockResolvedValue({ data: {
      draftId: 7, coin: { name: 'Denarius', category: 'Roman', material: 'Silver', era: 'Roman Imperial' },
      confidenceSummary: { overall: 'medium' }, unresolvedFields: [],
    } })
    mocks.commit.mockResolvedValue({ data: { coinId: 42 } })
    mocks.add.mockResolvedValue({ id: 43 })
    mocks.upload.mockResolvedValue({})
    mocks.alert.mockResolvedValue(undefined)
    vi.stubGlobal('URL', { createObjectURL: vi.fn(() => 'blob:photo'), revokeObjectURL: vi.fn() })
  })

  it('does not submit an unsupported invisible AI era', async () => {
    const wrapper = render()
    await draft(wrapper)
    expect(wrapper.text()).toContain('Roman Imperial')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    const payload = mocks.commit.mock.calls[0]?.[0]
    expect(payload.overrides.era).toBe('')
    expect(payload.overrides.notes).toContain('Roman Imperial')
    wrapper.unmount()
  })

  it('surfaces the server reason for a manual save failure', async () => {
    const wrapper = render()
    await click(wrapper, 'Use Manual Mode')
    mocks.add.mockRejectedValueOnce({ response: { data: { error: 'era is not supported' } } })
    await click(wrapper, 'Save manual')
    expect(mocks.alert).toHaveBeenCalledWith('era is not supported', { title: 'Error' })
    wrapper.unmount()
  })

  it('retries a failed intake image upload without committing the draft again', async () => {
    const wrapper = render()
    await draft(wrapper)
    mocks.upload.mockRejectedValueOnce(new Error('Upload interrupted'))
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(wrapper.text()).toContain('Coin saved')
    await click(wrapper, 'Retry remaining uploads')
    expect(mocks.commit).toHaveBeenCalledTimes(1)
    expect(mocks.upload).toHaveBeenCalledTimes(2)
    expect(mocks.push).toHaveBeenCalledWith('/coin/42')
    wrapper.unmount()
  })

  it('retries a manual image upload without creating a duplicate coin', async () => {
    const wrapper = render()
    await click(wrapper, 'Use Manual Mode')
    mocks.upload.mockRejectedValueOnce(new Error('Upload interrupted'))
    await click(wrapper, 'Save manual')
    expect(wrapper.text()).toContain('Coin saved')
    await click(wrapper, 'Retry remaining uploads')
    expect(mocks.add).toHaveBeenCalledTimes(1)
    expect(mocks.push).toHaveBeenCalledWith('/coin/43')
    wrapper.unmount()
  })

  it('does not repeat a successful obverse upload when the reverse fails', async () => {
    const wrapper = render()
    await draft(wrapper)
    const reverse = new File(['reverse'], 'reverse.jpg', { type: 'image/jpeg' })
    wrapper.findComponent({ name: 'InlineCameraCapturePanel' }).vm.$emit('captured', reverse)
    await flushPromises()
    mocks.upload.mockResolvedValueOnce({}).mockRejectedValueOnce(new Error('Reverse interrupted'))
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    await click(wrapper, 'Retry remaining uploads')
    expect(mocks.upload.mock.calls.map(call => call[2])).toEqual(['obverse', 'reverse', 'reverse'])
    expect(mocks.commit).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})
