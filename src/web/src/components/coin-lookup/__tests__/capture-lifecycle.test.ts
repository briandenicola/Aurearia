import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import InlineCameraCapturePanel from '@/components/InlineCameraCapturePanel.vue'
import CoinLookupCaptureWizard from '../CoinLookupCaptureWizard.vue'
import PwaCaptureShell from '../PwaCaptureShell.vue'

vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))
vi.mock('@/composables/usePwa', () => ({ usePwa: () => ({ isPwa: false }) }))
vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ showConfirm: vi.fn().mockResolvedValue(true) }),
}))

enableAutoUnmount(afterEach)

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (error: Error) => void
  const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

function mediaStream() {
  const stop = vi.fn()
  return { stream: { getTracks: () => [{ stop }] }, stop }
}

function renderCamera() {
  return mount(InlineCameraCapturePanel, {
    props: { imageRole: 'reverse', filenamePrefix: 'reverse', immersive: true },
  })
}

async function readyCamera(panel: ReturnType<typeof renderCamera>) {
  await panel.vm.startCamera()
  const video = panel.get('video')
  for (const property of ['videoWidth', 'videoHeight', 'clientWidth', 'clientHeight']) {
    Object.defineProperty(video.element, property, { configurable: true, value: 400 })
  }
  await video.trigger('loadedmetadata')
}

beforeEach(() => {
  vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue()
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({
    drawImage: vi.fn(),
  } as CanvasRenderingContext2D)
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('camera startup lifecycle', () => {
  for (const exit of ['stop', 'unmount'] as const) {
    for (const permissionTiming of ['before', 'after'] as const) {
      it(`stops permission granted ${permissionTiming} ${exit}`, async () => {
        const { stream, stop } = mediaStream()
        const permission = deferred<typeof stream>()
        vi.stubGlobal('navigator', { mediaDevices: { getUserMedia: () => permission.promise } })
        const panel = renderCamera()
        const starting = panel.vm.startCamera()
        if (permissionTiming === 'before') {
          permission.resolve(stream)
          await starting
        }
        if (exit === 'unmount') panel.unmount()
        else panel.vm.stopCamera()
        permission.resolve(stream)
        await starting
        expect(stop).toHaveBeenCalledOnce()
        expect(panel.vm.cameraActive).toBe(false)
      })
    }
  }

  it('coalesces repeated starts and permits a fresh start after stop', async () => {
    const old = mediaStream()
    const current = mediaStream()
    const permission = deferred<typeof old.stream>()
    const getUserMedia = vi.fn().mockReturnValueOnce(permission.promise).mockResolvedValue(current.stream)
    vi.stubGlobal('navigator', { mediaDevices: { getUserMedia } })
    const panel = renderCamera()
    const oldStart = panel.vm.startCamera()
    await panel.vm.startCamera()
    expect(getUserMedia).toHaveBeenCalledTimes(1)
    panel.vm.stopCamera()
    await panel.vm.startCamera()
    permission.resolve(old.stream)
    await oldStart
    expect(old.stop).toHaveBeenCalledOnce()
    expect(current.stop).not.toHaveBeenCalled()
    expect(panel.get('video').element.srcObject).toBe(current.stream)
    panel.unmount()
    expect(current.stop).toHaveBeenCalledOnce()
  })

  it('ignores a stale permission rejection without breaking the current stream', async () => {
    const current = mediaStream()
    const old = deferred<typeof current.stream>()
    const getUserMedia = vi.fn().mockReturnValueOnce(old.promise).mockResolvedValue(current.stream)
    vi.stubGlobal('navigator', { mediaDevices: { getUserMedia } })
    const panel = renderCamera()
    const oldStart = panel.vm.startCamera()
    panel.vm.stopCamera()
    await panel.vm.startCamera()
    old.reject(new Error('old request rejected'))
    await oldStart
    expect(panel.find('.camera-error-banner').exists()).toBe(false)
    expect(panel.vm.cameraActive).toBe(true)
    expect(current.stop).not.toHaveBeenCalled()
  })

  it('stops a stream when playback fails and allows retry', async () => {
    const first = mediaStream()
    const retry = mediaStream()
    vi.stubGlobal('navigator', {
      mediaDevices: { getUserMedia: vi.fn().mockResolvedValueOnce(first.stream).mockResolvedValue(retry.stream) },
    })
    vi.mocked(HTMLMediaElement.prototype.play).mockRejectedValueOnce(new Error('playback failed'))
    const panel = renderCamera()
    await panel.vm.startCamera()
    expect(first.stop).toHaveBeenCalledOnce()
    expect(panel.get('.camera-error-banner').text()).toContain('Camera is unavailable')
    expect(panel.get('video').element.srcObject).toBeNull()
    await panel.vm.startCamera()
    expect(panel.vm.cameraActive).toBe(true)
  })
})

describe('capture completion lifecycle', () => {
  for (const exit of ['stop', 'unmount'] as const) {
    it(`discards encoded frames completing after ${exit}`, async () => {
      const { stream } = mediaStream()
      vi.stubGlobal('navigator', { mediaDevices: { getUserMedia: vi.fn().mockResolvedValue(stream) } })
      let finish!: BlobCallback
      vi.spyOn(HTMLCanvasElement.prototype, 'toBlob').mockImplementation(callback => { finish = callback })
      const panel = renderCamera()
      await readyCamera(panel)
      const capture = panel.vm.captureFromCamera()
      if (exit === 'unmount') panel.unmount()
      else panel.vm.stopCamera()
      finish(new Blob(['frame']))
      await capture
      expect(panel.emitted('captured')).toBeUndefined()
    })
  }

  it('allows only one encoding at a time and freezes the filename and role', async () => {
    const { stream } = mediaStream()
    vi.stubGlobal('navigator', { mediaDevices: { getUserMedia: vi.fn().mockResolvedValue(stream) } })
    let finish!: BlobCallback
    const encode = vi.spyOn(HTMLCanvasElement.prototype, 'toBlob').mockImplementation(callback => { finish = callback })
    const panel = renderCamera()
    await readyCamera(panel)
    const capture = panel.vm.captureFromCamera()
    await panel.vm.captureFromCamera()
    expect(encode).toHaveBeenCalledOnce()
    await panel.setProps({ imageRole: 'notes', filenamePrefix: 'notes' })
    finish(new Blob(['reverse']))
    await capture
    const event = panel.emitted<[File, string]>('captured')?.[0]
    expect(event?.[0].name).toMatch(/^reverse-\d+\.jpg$/)
    expect(event?.[1]).toBe('reverse')
    expect(panel.vm.cameraReady).toBe(true)
  })

  for (const failure of ['null', 'throw'] as const) {
    it(`reports ${failure} encoding failures and permits retry`, async () => {
      const { stream } = mediaStream()
      vi.stubGlobal('navigator', { mediaDevices: { getUserMedia: vi.fn().mockResolvedValue(stream) } })
      vi.spyOn(HTMLCanvasElement.prototype, 'toBlob')
        .mockImplementationOnce(callback => {
          if (failure === 'throw') throw new Error('encoding failed')
          callback(null)
        })
        .mockImplementation(callback => callback(new Blob(['frame'])))
      const panel = renderCamera()
      await readyCamera(panel)
      await panel.vm.captureFromCamera()
      expect(panel.get('.camera-error-banner').text()).toContain('Could not capture image')
      await panel.vm.captureFromCamera()
      expect(panel.emitted('captured')).toHaveLength(1)
      expect(panel.find('.camera-error-banner').exists()).toBe(false)
    })
  }
})

describe('capture role integration', () => {
  for (const surface of ['pwa', 'desktop'] as const) {
    for (const purpose of ['intake', 'identify'] as const) {
      it(`keeps reverse evidence separate after navigating in ${surface} ${purpose}`, async () => {
        const { stream } = mediaStream()
        vi.stubGlobal('navigator', { mediaDevices: { getUserMedia: vi.fn().mockResolvedValue(stream) } })
        let finish!: BlobCallback
        vi.spyOn(HTMLCanvasElement.prototype, 'toBlob').mockImplementation(callback => { finish = callback })
        const props = {
          purpose, obverse: { file: new File(['front'], 'front.jpg'), preview: 'blob:front' },
          reverse: null, notesImage: null, notes: '', submitting: false, preparingImage: false, uploadError: '',
        }
        const wrapper = surface === 'pwa'
          ? mount(PwaCaptureShell, { props })
          : mount(CoinLookupCaptureWizard, { props })
        await wrapper.get('[aria-label="Add reverse image"]').trigger('click')
        const panel = wrapper.getComponent(InlineCameraCapturePanel)
        await readyCamera(panel)
        await wrapper.get('[aria-label="Capture photo"]').trigger('click')
        await wrapper.get(`[aria-label="${purpose === 'intake' ? 'Add coin card' : 'Add notes'}"]`).trigger('click')
        finish(new Blob(['reverse']))
        await flushPromises()
        expect(wrapper.emitted('captured')).toEqual([['reverse', expect.any(File)]])
      })
    }
  }
})
