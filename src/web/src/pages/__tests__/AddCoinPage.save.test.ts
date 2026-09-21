import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, type PropType } from 'vue'
import type { Coin } from '@/types'
import AddCoinPage from '../AddCoinPage.vue'
import CoinLookupCaptureWizard from '@/components/coin-lookup/CoinLookupCaptureWizard.vue'

const mocks = vi.hoisted(() => ({
  draft: vi.fn(), commit: vi.fn(), upload: vi.fn(), add: vi.fn(), alert: vi.fn(),
  push: vi.fn(), update: vi.fn(), extract: vi.fn(), normalize: vi.fn(), stopCamera: vi.fn(), isPwa: true,
}))
vi.mock('vue-router', () => ({ useRoute: () => ({ query: {} }), useRouter: () => ({ push: mocks.push }) }))
vi.mock('@/stores/coins', () => ({ useCoinsStore: () => ({ addCoin: mocks.add }) }))
vi.mock('@/composables/usePwa', () => ({ usePwa: () => ({ isPwa: mocks.isPwa }) }))
vi.mock('@/utils/galleryImage', () => ({ normalizeGalleryImage: mocks.normalize }))
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
  return mount(AddCoinPage, {
    global: { stubs: {
      CoinForm: CoinFormStub, RouterLink: true,
      InlineCameraCapturePanel: {
        name: 'InlineCameraCapturePanel', emits: ['captured', 'upload'],
        methods: { stopCamera: mocks.stopCamera },
        template: '<div class="camera-stub"></div>',
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

async function selectImage(wrapper: ReturnType<typeof render>, file: File) {
  const input = wrapper.get('input[type="file"]')
  Object.defineProperty(input.element, 'files', { configurable: true, value: [file] })
  await input.trigger('change')
  await flushPromises()
}

describe('Add Coin save workflow', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.isPwa = true
    mocks.normalize.mockImplementation(async (file: File) => file)
    mocks.draft.mockResolvedValue({ data: {
      draftId: 7, coin: { name: 'Denarius', category: 'Roman', material: 'Silver', era: 'Roman Imperial' },
      confidenceSummary: { overall: 'medium' }, unresolvedFields: [],
    } })
    mocks.commit.mockResolvedValue({ data: { coinId: 42 } })
    mocks.add.mockResolvedValue({ id: 43 })
    mocks.upload.mockResolvedValue({})
    mocks.alert.mockResolvedValue(undefined)
    vi.stubGlobal('URL', { createObjectURL: vi.fn((file: File) => `blob:${file.name}`), revokeObjectURL: vi.fn() })
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
    await wrapper.get('[aria-label="Add reverse image"]').trigger('click')
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

  it('opens the shared wizard in PWA mode and requires only the obverse', () => {
    const wrapper = render()
    const wizard = wrapper.getComponent(CoinLookupCaptureWizard)

    expect(wizard.props('purpose')).toBe('intake')
    expect(wrapper.get('ol[aria-label="Coin intake progress"]').text()).toContain('Card')
    expect(wrapper.text()).toContain('Step 1 of 3')
    expect(wrapper.text()).toContain('Add the obverse')
    expect(wrapper.text()).not.toContain('Generate Intake Draft')
    expect(wrapper.get('[aria-label="Add reverse image"]').attributes('disabled')).toBeDefined()
    expect(mocks.draft).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('preserves the desktop manual default and provides the same assisted wizard', async () => {
    mocks.isPwa = false
    const wrapper = render()
    expect(wrapper.findComponent(CoinLookupCaptureWizard).exists()).toBe(false)
    expect(wrapper.text()).toContain('Save manual')

    await click(wrapper, 'AI Assist Mode')
    expect(wrapper.getComponent(CoinLookupCaptureWizard).props('purpose')).toBe('intake')
    expect(wrapper.text()).toContain('Add the obverse')
    expect(mocks.draft).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('keeps an optional card separate when the reverse step is skipped', async () => {
    const wrapper = render()
    const card = new File(['card'], 'card.jpg', { type: 'image/jpeg' })
    wrapper.findComponent({ name: 'InlineCameraCapturePanel' }).vm.$emit('captured', photo)
    await flushPromises()
    await wrapper.get('[aria-label="Add reverse image"]').trigger('click')
    await wrapper.get('[aria-label="Add coin card"]').trigger('click')
    expect(wrapper.text()).toContain('Add a coin card')
    expect(wrapper.find('textarea').exists()).toBe(false)
    await selectImage(wrapper, card)
    await click(wrapper, 'Generate Intake Draft')

    expect(mocks.draft).toHaveBeenCalledWith([photo], card)
    expect(mocks.commit).not.toHaveBeenCalled()
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(mocks.upload).toHaveBeenCalledExactlyOnceWith(42, photo, 'obverse', true, true)
    wrapper.unmount()
  })

  it('normalizes gallery photos and preserves camera origin for each face', async () => {
    const wrapper = render()
    const heic = new File(['gallery'], 'obverse.heic', { type: 'image/heic' })
    mocks.normalize.mockResolvedValueOnce(photo)
    await selectImage(wrapper, heic)
    expect(mocks.normalize).toHaveBeenCalledWith(heic)
    await wrapper.get('[aria-label="Add reverse image"]').trigger('click')
    const reverse = new File(['reverse'], 'reverse.jpg', { type: 'image/jpeg' })
    wrapper.findComponent({ name: 'InlineCameraCapturePanel' }).vm.$emit('captured', reverse)
    await flushPromises()
    await click(wrapper, 'Generate Intake Draft')
    expect(mocks.draft).toHaveBeenCalledWith([photo, reverse], undefined)
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(mocks.upload.mock.calls).toEqual([
      [42, photo, 'obverse', true, false],
      [42, reverse, 'reverse', false, true],
    ])
    wrapper.unmount()
  })

  it('reports image preparation failure without creating a draft', async () => {
    const wrapper = render()
    mocks.normalize.mockRejectedValueOnce(new Error('The selected image could not be opened.'))
    await selectImage(wrapper, photo)

    expect(wrapper.get('[role="alert"]').text()).toContain('The selected image could not be opened.')
    expect(wrapper.getComponent(CoinLookupCaptureWizard).props('preparingImage')).toBe(false)
    expect(mocks.draft).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('blocks draft generation during preparation and ignores images finishing after unmount', async () => {
    const wrapper = render()
    let resolveImage!: (file: File) => void
    mocks.normalize.mockReturnValueOnce(new Promise<File>(resolve => { resolveImage = resolve }))
    const wizard = wrapper.getComponent(CoinLookupCaptureWizard)
    wizard.vm.$emit('selected', 'obverse', photo)
    await flushPromises()
    expect(wrapper.get('fieldset').attributes('disabled')).toBeDefined()
    wizard.vm.$emit('analyze')
    expect(mocks.draft).not.toHaveBeenCalled()

    wrapper.unmount()
    resolveImage(photo)
    await flushPromises()
    expect(URL.createObjectURL).not.toHaveBeenCalled()
  })

  it('releases image previews on replacement, removal, and unmount', async () => {
    const wrapper = render()
    const wizard = wrapper.getComponent(CoinLookupCaptureWizard)
    wizard.vm.$emit('captured', 'obverse', photo)
    await flushPromises()
    const replacement = new File(['new'], 'replacement.jpg', { type: 'image/jpeg' })
    wizard.vm.$emit('captured', 'obverse', replacement)
    await flushPromises()
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:obverse.jpg')
    await wrapper.get('[aria-label="Remove obverse image"]').trigger('click')
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:replacement.jpg')

    wizard.vm.$emit('captured', 'obverse', photo)
    await flushPromises()
    wrapper.unmount()
    expect(URL.revokeObjectURL).toHaveBeenCalledTimes(3)
  })

  it('stops the optional-face camera before generating a draft', async () => {
    const wrapper = render()
    await selectImage(wrapper, photo)
    await wrapper.get('[aria-label="Add reverse image"]').trigger('click')
    await click(wrapper, 'Generate Intake Draft')

    expect(mocks.stopCamera).toHaveBeenCalledOnce()
    expect(mocks.draft).toHaveBeenCalledWith([photo], undefined)
    wrapper.unmount()
  })
})
