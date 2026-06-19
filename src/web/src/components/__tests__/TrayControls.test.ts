import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TrayControls from '../tray/TrayControls.vue'

describe('TrayControls', () => {
  it('renders tray navigation in Previous, label, Next order', () => {
    const wrapper = mount(TrayControls, {
      props: {
        drawerIndex: 1,
        totalDrawers: 5,
      },
    })

    const navigation = wrapper.find('.drawer-navigation')
    const items = navigation.element.children

    expect(items).toHaveLength(3)
    expect(items[0]?.textContent?.trim()).toBe('Prev')
    expect(items[1]?.textContent?.trim()).toBe('Tray 2 of 5')
    expect(items[2]?.textContent?.trim()).toBe('Next')
  })
})
