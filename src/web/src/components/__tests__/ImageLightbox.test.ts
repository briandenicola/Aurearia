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

    const container = wrapper.find('.relative.flex.max-h-full.max-w-full.items-center.justify-center')
    expect(container.exists()).toBe(true)

    const image = container.find('authenticated-image-stub')
    expect(image.exists()).toBe(true)
    expect(image.classes()).toContain('opacity-30')
    expect(image.classes()).toContain('blur-[2px]')

    const overlay = container.find('.pointer-events-none.absolute.inset-0')
    expect(overlay.exists()).toBe(true)

    expect(mocks.fetchPrivateMediaBlob).toHaveBeenCalledWith('/media/test.jpg')
    expect(mocks.removeCoinBackground).toHaveBeenCalledTimes(1)

    const containerChildren = container.element.children
    const overlayIsChild = Array.from(containerChildren).some(
      (child) => child === overlay.element
    )
    expect(overlayIsChild).toBe(true)

    expect(overlay.text()).toContain('Removing background')
    expect(overlay.find('.animate-spin').exists()).toBe(true)
  })

  it('removes processing class when processing completes', async () => {
    const wrapper = mountLightbox()
    const imageSelector = '.relative.flex.max-h-full.max-w-full.items-center.justify-center authenticated-image-stub'
    const overlaySelector = '.pointer-events-none.absolute.inset-0'

    wrapper.vm.processing = true
    await wrapper.vm.$nextTick()
    expect(wrapper.find(imageSelector).classes()).toContain('opacity-30')
    expect(wrapper.find(imageSelector).classes()).toContain('blur-[2px]')
    expect(wrapper.find(overlaySelector).exists()).toBe(true)

    wrapper.vm.processing = false
    await wrapper.vm.$nextTick()
    expect(wrapper.find(imageSelector).classes()).not.toContain('opacity-30')
    expect(wrapper.find(overlaySelector).exists()).toBe(false)
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
