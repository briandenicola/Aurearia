import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import ZoomableSurface from '@/components/ZoomableSurface.vue'

function mountSurface() {
  const wrapper = mount(ZoomableSurface, {
    props: { ariaLabel: 'Zoomable value chart' },
    slots: { default: '<svg class="test-chart" />' },
  })
  const viewport = wrapper.find('.zoomable-viewport')
  const viewportElement = viewport.element as HTMLElement & {
    setPointerCapture: (pointerId: number) => void
  }
  viewportElement.setPointerCapture = vi.fn()
  viewportElement.getBoundingClientRect = () => ({
    x: 0,
    y: 0,
    left: 0,
    top: 0,
    right: 400,
    bottom: 300,
    width: 400,
    height: 300,
    toJSON: () => ({}),
  })

  return { wrapper, viewport }
}

function dispatchPointerEvent(
  element: Element,
  type: string,
  options: { pointerId: number; pointerType: string; clientX: number; clientY: number; buttons: number },
) {
  const event = new Event(type, { bubbles: true, cancelable: true })
  Object.defineProperties(event, {
    pointerId: { value: options.pointerId },
    pointerType: { value: options.pointerType },
    clientX: { value: options.clientX },
    clientY: { value: options.clientY },
    buttons: { value: options.buttons },
  })
  element.dispatchEvent(event)
}

describe('ZoomableSurface', () => {
  it('zooms in, zooms out, and resets from toolbar controls', async () => {
    const { wrapper } = mountSurface()

    expect(wrapper.find('.zoomable-status').text()).toBe('100%')

    await wrapper.find('[aria-label="Zoom in"]').trigger('click')
    expect(wrapper.find('.zoomable-status').text()).toBe('120%')
    expect(wrapper.find('.zoomable-content').attributes('style')).toContain('scale(1.2)')

    await wrapper.find('[aria-label="Zoom out"]').trigger('click')
    expect(wrapper.find('.zoomable-status').text()).toBe('100%')

    await wrapper.find('[aria-label="Zoom in"]').trigger('click')
    await wrapper.find('[aria-label="Reset chart zoom"]').trigger('click')

    expect(wrapper.find('.zoomable-status').text()).toBe('100%')
    expect(wrapper.find('.zoomable-content').attributes('style')).toContain('translate(0px, 0px) scale(1)')
  })

  it('zooms with the mouse wheel at the pointer location', async () => {
    const { wrapper, viewport } = mountSurface()

    viewport.element.dispatchEvent(new WheelEvent('wheel', {
      bubbles: true,
      cancelable: true,
      deltaY: -100,
      clientX: 200,
      clientY: 150,
    }))
    await nextTick()

    expect(wrapper.find('.zoomable-status').text()).toBe('120%')
    expect(wrapper.find('.zoomable-content').attributes('style')).toContain('scale(1.2)')
  })

  it('pans by dragging inside the viewport', async () => {
    const { wrapper, viewport } = mountSurface()

    dispatchPointerEvent(viewport.element, 'pointerdown', {
      pointerId: 1,
      pointerType: 'mouse',
      clientX: 10,
      clientY: 10,
      buttons: 1,
    })
    dispatchPointerEvent(viewport.element, 'pointermove', {
      pointerId: 1,
      pointerType: 'mouse',
      clientX: 34,
      clientY: 20,
      buttons: 1,
    })
    await nextTick()

    expect(wrapper.find('.zoomable-viewport').classes()).toContain('is-panning')
    expect(wrapper.find('.zoomable-content').attributes('style')).toContain('translate(24px, 10px)')
  })

  it('supports keyboard zoom, reset, and panning shortcuts', async () => {
    const { wrapper, viewport } = mountSurface()

    await viewport.trigger('keydown', { key: '=' })
    expect(wrapper.find('.zoomable-status').text()).toBe('120%')

    await viewport.trigger('keydown', { key: 'ArrowRight' })
    expect(wrapper.find('.zoomable-content').attributes('style')).toContain('translate(-64px, -30px)')

    await viewport.trigger('keydown', { key: '0' })
    expect(wrapper.find('.zoomable-status').text()).toBe('100%')
    expect(wrapper.find('.zoomable-content').attributes('style')).toContain('translate(0px, 0px) scale(1)')
  })
})
