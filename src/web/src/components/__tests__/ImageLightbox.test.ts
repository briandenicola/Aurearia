import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ImageLightbox from '@/components/ImageLightbox.vue'

const mocks = vi.hoisted(() => ({
  fetchPrivateMediaBlob: vi.fn(),
  removeCoinBackground: vi.fn(),
  uploadImage: vi.fn(),
  deleteImage: vi.fn(),
}))

vi.mock('@/utils/media', () => ({
  fetchPrivateMediaBlob: mocks.fetchPrivateMediaBlob,
}))

vi.mock('@/utils/backgroundRemoval', () => ({
  removeCoinBackground: mocks.removeCoinBackground,
}))

vi.mock('@/api/client', () => ({
  uploadImage: mocks.uploadImage,
  deleteImage: mocks.deleteImage,
}))

describe('ImageLightbox', () => {
  beforeEach(() => {
    mocks.fetchPrivateMediaBlob.mockReset()
    mocks.removeCoinBackground.mockReset()
    mocks.uploadImage.mockReset()
    mocks.deleteImage.mockReset()
  })

  it('renders processing overlay over the image during background removal', async () => {
    mocks.fetchPrivateMediaBlob.mockResolvedValue(new Blob(['source'], { type: 'image/jpeg' }))
    mocks.removeCoinBackground.mockReturnValue(new Promise(() => {}))

    const wrapper = mountLightbox()
    await wrapper.find('button.btn-primary').trigger('click')

    await wrapper.vm.$nextTick()

    const container = wrapper.find('.lightbox-image-container')
    expect(container.exists()).toBe(true)
    expect(container.classes()).toContain('processing')

    const overlay = wrapper.find('.lightbox-processing-overlay')
    expect(overlay.exists()).toBe(true)

    expect(mocks.fetchPrivateMediaBlob).toHaveBeenCalledWith('/media/test.jpg')
    expect(mocks.removeCoinBackground).toHaveBeenCalledTimes(1)

    const containerChildren = container.element.children
    const overlayIsChild = Array.from(containerChildren).some(
      (child) => child.classList.contains('lightbox-processing-overlay')
    )
    expect(overlayIsChild).toBe(true)

    expect(overlay.classes()).toContain('lightbox-processing-overlay')
    expect(overlay.text()).toContain('Removing background')
    expect(overlay.find('.spinner').exists()).toBe(true)
  })

  it('removes processing class when processing completes', async () => {
    const wrapper = mountLightbox()

    wrapper.vm.processing = true
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.lightbox-image-container').classes()).toContain('processing')

    wrapper.vm.processing = false
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.lightbox-image-container').classes()).not.toContain('processing')
    expect(wrapper.find('.lightbox-processing-overlay').exists()).toBe(false)
  })
})

function mountLightbox() {
  return mount(ImageLightbox, {
    props: {
      coinId: 1,
      imageId: 100,
      imagePath: '/media/test.jpg',
      imageType: 'obverse',
    },
    global: {
      stubs: {
        Teleport: true,
        AuthenticatedImage: true,
      },
    },
  })
}
