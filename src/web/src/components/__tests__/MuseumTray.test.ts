import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import MuseumTray from '../tray/MuseumTray.vue'
import MuseumTrayWell from '../tray/MuseumTrayWell.vue'
import type { TrayCoin } from '@/utils/trayLayout'

describe('MuseumTray', () => {
  const mockCoins: TrayCoin[] = [
    {
      id: 1,
      name: 'Coin 1',
      diameterMm: 20,
      images: [],
    },
    {
      id: 2,
      name: 'Coin 2',
      diameterMm: 30,
      images: [],
    },
    {
      id: 3,
      name: 'Coin 3',
      diameterMm: 25,
      images: [],
    },
  ]

  it('renders all coin wells', () => {
    const wrapper = mount(MuseumTray, {
      props: {
        coins: mockCoins,
        feltTheme: 'red',
      },
    })

    const wells = wrapper.findAllComponents(MuseumTrayWell)
    expect(wells).toHaveLength(3)
  })

  it('applies red felt theme class', () => {
    const wrapper = mount(MuseumTray, {
      props: {
        coins: mockCoins,
        feltTheme: 'red',
      },
    })

    const tray = wrapper.find('.museum-tray')
    expect(tray.classes()).toContain('felt-red')
  })

  it('applies green felt theme class', () => {
    const wrapper = mount(MuseumTray, {
      props: {
        coins: mockCoins,
        feltTheme: 'green',
      },
    })

    const tray = wrapper.find('.museum-tray')
    expect(tray.classes()).toContain('felt-green')
  })

  it('applies navy felt theme class', () => {
    const wrapper = mount(MuseumTray, {
      props: {
        coins: mockCoins,
        feltTheme: 'navy',
      },
    })

    const tray = wrapper.find('.museum-tray')
    expect(tray.classes()).toContain('felt-navy')
  })

  it('emits coin-clicked when well is clicked', async () => {
    const wrapper = mount(MuseumTray, {
      props: {
        coins: mockCoins,
        feltTheme: 'red',
      },
    })

    const wells = wrapper.findAllComponents(MuseumTrayWell)
    await wells[0]?.vm.$emit('coin-clicked', 1)

    expect(wrapper.emitted('coin-clicked')).toBeTruthy()
    expect(wrapper.emitted('coin-clicked')?.[0]).toEqual([1])
  })

  it('handles empty coin array', () => {
    const wrapper = mount(MuseumTray, {
      props: {
        coins: [],
        feltTheme: 'red',
      },
    })

    const wells = wrapper.findAllComponents(MuseumTrayWell)
    expect(wells).toHaveLength(0)
  })

  it('calculates different render sizes for different diameters', () => {
    const wrapper = mount(MuseumTray, {
      props: {
        coins: mockCoins,
        feltTheme: 'red',
      },
    })

    const wells = wrapper.findAllComponents(MuseumTrayWell)
    const sizes = wells.map(well => well.props('renderSizePx'))
    
    // Should have different sizes for different diameter coins
    expect(new Set(sizes).size).toBeGreaterThan(1)
    
    // All sizes should be within bounds
    sizes.forEach(size => {
      expect(size).toBeGreaterThanOrEqual(40)
      expect(size).toBeLessThanOrEqual(120)
    })
  })

  it('has responsive grid layout class', () => {
    const wrapper = mount(MuseumTray, {
      props: {
        coins: mockCoins,
        feltTheme: 'red',
      },
    })

    const grid = wrapper.find('.tray-grid')
    expect(grid.exists()).toBe(true)
  })
})
