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
  it('renders a locked surface when interactive is false', () => {
    const wrapper = mount(ZoomableSurface, {
      props: { interactive: false, ariaLabel: 'Locked chart' },
      slots: { default: '<svg class="test-chart" />' },
    })

    expect(wrapper.find('.zoomable-toolbar').exists()).toBe(false)
    expect(wrapper.find('.zoomable-viewport').classes()).toContain('is-static')
    expect(wrapper.find('.zoomable-viewport').attributes('tabindex')).toBe('-1')
  })

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

describe('ZoomableSurface document-viewer options', () => {
  function mountDocumentSurface() {
    const wrapper = mount(ZoomableSurface, {
      props: { ariaLabel: 'Invoice document', bare: true, wheelPans: true, boundPan: true, minScale: 1, maxScale: 4 },
      slots: { default: '<div class="doc" />' },
    })
    const viewport = wrapper.find('.zoomable-viewport')
    const viewportEl = viewport.element as HTMLElement
    Object.defineProperty(viewportEl, 'clientWidth', { value: 400 })
    Object.defineProperty(viewportEl, 'clientHeight', { value: 300 })
    viewportEl.getBoundingClientRect = () => ({ x: 0, y: 0, left: 0, top: 0, right: 400, bottom: 300, width: 400, height: 300, toJSON: () => ({}) })
    const contentEl = wrapper.find('.zoomable-content').element as HTMLElement
    Object.defineProperty(contentEl, 'offsetWidth', { value: 400 })
    Object.defineProperty(contentEl, 'offsetHeight', { value: 1200 })
    return { wrapper, viewport, contentEl }
  }

  function wheel(el: Element, init: { deltaX?: number; deltaY: number; ctrlKey?: boolean }) {
    const event = new Event('wheel', { bubbles: true, cancelable: true })
    Object.defineProperties(event, {
      deltaX: { value: init.deltaX ?? 0 },
      deltaY: { value: init.deltaY },
      ctrlKey: { value: init.ctrlKey ?? false },
      clientX: { value: 200 },
      clientY: { value: 150 },
    })
    el.dispatchEvent(event)
  }

  it('hides the chart zoom toolbar in the bare variant', () => {
    const { wrapper } = mountDocumentSurface()
    expect(wrapper.find('.zoomable-toolbar').exists()).toBe(false)
    expect(wrapper.find('.zoomable-surface').classes()).toContain('is-bare')
  })

  it('scrolls with the wheel instead of zooming, and stops at the document edges', async () => {
    const { viewport, contentEl } = mountDocumentSurface()
    wheel(viewport.element, { deltaY: 300 })
    await nextTick()
    expect(contentEl.style.transform).toBe('translate(0px, -300px) scale(1)')

    wheel(viewport.element, { deltaY: 5000 })
    await nextTick()
    // 1200px of content in a 300px viewport can scroll at most 900px.
    expect(contentEl.style.transform).toBe('translate(0px, -900px) scale(1)')

    wheel(viewport.element, { deltaY: -9999 })
    await nextTick()
    expect(contentEl.style.transform).toBe('translate(0px, 0px) scale(1)')
  })

  it('still zooms on trackpad pinch (ctrl + wheel)', async () => {
    const { viewport, contentEl } = mountDocumentSurface()
    wheel(viewport.element, { deltaY: -100, ctrlKey: true })
    await nextTick()
    expect(contentEl.style.transform).toContain('scale(1.2)')
  })

  it('keeps the default chart behaviour unchanged (wheel zooms, toolbar shown)', async () => {
    const wrapper = mount(ZoomableSurface, { props: { ariaLabel: 'Chart' }, slots: { default: '<svg />' } })
    expect(wrapper.find('.zoomable-toolbar').exists()).toBe(true)
    const viewportEl = wrapper.find('.zoomable-viewport').element as HTMLElement
    viewportEl.getBoundingClientRect = () => ({ x: 0, y: 0, left: 0, top: 0, right: 400, bottom: 300, width: 400, height: 300, toJSON: () => ({}) })
    wheel(viewportEl, { deltaY: -100 })
    await nextTick()
    expect((wrapper.find('.zoomable-content').element as HTMLElement).style.transform).toContain('scale(1.2)')
  })
})
