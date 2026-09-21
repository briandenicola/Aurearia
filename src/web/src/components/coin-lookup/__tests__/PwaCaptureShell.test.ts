import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import PwaCaptureShell from '../PwaCaptureShell.vue'
import { useImmersiveShell } from '@/composables/useImmersiveShell'

const mocks = vi.hoisted(() => ({ push: vi.fn(), confirm: vi.fn(), alert: vi.fn() }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push }) }))
vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ showConfirm: mocks.confirm, showAlert: mocks.alert }),
}))

const cameraStub = {
  name: 'InlineCameraCapturePanel',
  emits: ['captured', 'upload'],
  props: { immersive: Boolean, instruction: String, filenamePrefix: String },
  data() { return { cameraActive: false } },
  methods: {
    startCamera(this: { cameraActive: boolean }) { this.cameraActive = true },
    captureFromCamera: vi.fn(),
    stopCamera: vi.fn(),
  },
  template: '<div class="camera-stub"></div>',
}

const photo = { file: new File(['o'], 'obverse.jpg', { type: 'image/jpeg' }), preview: 'blob:obverse' }

function render(props: Record<string, unknown> = {}) {
  return mount(PwaCaptureShell, {
    props: {
      obverse: null, reverse: null, notesImage: null, notes: '',
      submitting: false, preparingImage: false, uploadError: '',
      ...props,
    },
    global: { stubs: { InlineCameraCapturePanel: cameraStub } },
  })
}

describe('PwaCaptureShell', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.confirm.mockResolvedValue(true)
  })

  it('renders the mockup chrome and locks later steps until the obverse exists', () => {
    const wrapper = render({ purpose: 'intake' })

    expect(wrapper.get('ol[aria-label="Coin intake progress"]').text()).toBe('1 Obverse2 Reverse3 Details')
    expect(wrapper.get('[aria-label="Add reverse image"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[aria-label="Add coin card"]').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('Obverse · required')
    expect(wrapper.text()).toContain('Fill the ring · soft, even light')
    // The primary action only appears once there is something to analyze.
    expect(wrapper.text()).not.toContain('Generate Intake Draft')
    wrapper.unmount()
  })

  it('claims immersive mode while mounted so the app chrome steps aside', () => {
    const { immersive } = useImmersiveShell()
    expect(immersive.value).toBe(false)

    const wrapper = render()
    expect(immersive.value).toBe(true)

    wrapper.unmount()
    expect(immersive.value).toBe(false)
  })

  it('drives the camera from the shutter and retakes once a photo is on the stage', async () => {
    const wrapper = render()
    const shutter = wrapper.get('.shutter')

    expect(shutter.attributes('aria-label')).toBe('Start camera')
    await shutter.trigger('click')
    await flushPromises()
    expect(shutter.attributes('aria-label')).toBe('Capture photo')

    await wrapper.setProps({ obverse: photo })
    expect(wrapper.get('.shutter').attributes('aria-label')).toBe('Retake photo')
    expect(wrapper.text()).toContain('Obverse added')

    await wrapper.get('.shutter').trigger('click')
    await flushPromises()
    expect(wrapper.emitted('remove')).toEqual([['obverse']])
    wrapper.unmount()
  })

  it('offers Manual on intake and Deep Analysis on identify', async () => {
    const intake = render({ purpose: 'intake' })
    await intake.get('[aria-label="Use manual entry"]').trigger('click')
    expect(intake.emitted('manual')).toHaveLength(1)
    expect(intake.find('[aria-label="Deep Analysis"]').exists()).toBe(false)
    intake.unmount()

    const identify = render({ purpose: 'identify', deepAnalysisEnabled: true, obverse: photo })
    expect(identify.find('[aria-label="Use manual entry"]').exists()).toBe(false)
    await identify.get('[aria-label="Deep Analysis"]').trigger('click')
    // Deep Analysis needs a reverse, so the shell moves there and explains why.
    expect(identify.emitted('deepAnalyze')).toBeUndefined()
    expect(identify.get('[role="alert"]').text()).toContain('Add a reverse image')
    identify.unmount()
  })

  it('keeps the identification notes field on the details step only', async () => {
    const wrapper = render({ purpose: 'identify', obverse: photo, notes: 'hi' })
    expect(wrapper.find('textarea').exists()).toBe(false)

    await wrapper.get('[aria-label="Add notes"]').trigger('click')
    const textarea = wrapper.get('textarea')
    expect((textarea.element as HTMLTextAreaElement).value).toBe('hi')
    await textarea.setValue('22mm, 3.4g')
    expect(wrapper.emitted('update:notes')).toEqual([['22mm, 3.4g']])

    const intake = render({ purpose: 'intake', obverse: photo })
    await intake.get('[aria-label="Add coin card"]').trigger('click')
    expect(intake.find('textarea').exists()).toBe(false)
    wrapper.unmount()
    intake.unmount()
  })

  it('confirms before a tab switch or close would discard captured photos', async () => {
    const wrapper = render({ purpose: 'intake', obverse: photo })

    mocks.confirm.mockResolvedValueOnce(false)
    await wrapper.get('[role="tab"][aria-selected="false"]').trigger('click')
    await flushPromises()
    expect(mocks.push).not.toHaveBeenCalled()

    await wrapper.get('[role="tab"][aria-selected="false"]').trigger('click')
    await flushPromises()
    expect(mocks.push).toHaveBeenCalledWith('/lookup')

    await wrapper.get('[aria-label="Quick Capture"]').trigger('click')
    await flushPromises()
    expect(mocks.push).toHaveBeenLastCalledWith('/quick-capture')
    wrapper.unmount()
  })

  it('leaves without a prompt when nothing has been captured', async () => {
    const wrapper = render()
    await wrapper.get('[aria-label="Close capture"]').trigger('click')
    await flushPromises()
    expect(mocks.confirm).not.toHaveBeenCalled()
    expect(mocks.push).toHaveBeenCalledWith('/')
    wrapper.unmount()
  })
})
