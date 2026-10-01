import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import DesktopPageContext from '@/components/DesktopPageContext.vue'

let isPwa = false

vi.mock('@/composables/usePwa', () => ({
  usePwa: () => ({ isPwa }),
}))

describe('DesktopPageContext', () => {
  beforeEach(() => {
    isPwa = false
    document.body.innerHTML = `
      <div id="desktop-page-title"></div>
      <div id="desktop-page-actions"></div>
    `
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('teleports the page title and actions into the desktop shell', () => {
    const wrapper = mount(DesktopPageContext, {
      props: { title: 'Wishlist' },
      slots: {
        actions: '<button aria-label="Add to wishlist">Add</button>',
      },
    })

    expect(document.querySelector('#desktop-page-title')?.textContent).toBe('Wishlist')
    expect(document.querySelector('#desktop-page-actions button')?.getAttribute('aria-label')).toBe('Add to wishlist')

    wrapper.unmount()

    expect(document.querySelector('#desktop-page-title')?.textContent).toBe('')
    expect(document.querySelector('#desktop-page-actions')?.textContent).toBe('')
  })

  it('leaves the desktop shell empty in PWA mode', () => {
    isPwa = true

    mount(DesktopPageContext, {
      props: { title: 'Wishlist' },
      slots: {
        actions: '<button>Add</button>',
      },
    })

    expect(document.querySelector('#desktop-page-title')?.textContent).toBe('')
    expect(document.querySelector('#desktop-page-actions')?.textContent).toBe('')
  })
})
