import { beforeEach, afterEach, describe, it, expect, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import MuseumTrayWell from '../tray/MuseumTrayWell.vue'
import type { TrayCoin } from '@/utils/trayLayout'

describe('MuseumTrayWell', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response(new Blob(['image']), { status: 200 })))
    vi.stubGlobal('URL', {
      createObjectURL: vi.fn(() => 'blob:tray-image'),
      revokeObjectURL: vi.fn(),
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  const mockCoin: TrayCoin = {
    id: 1,
    name: 'Test Coin',
    diameterMm: 25,
    images: [
      {
        id: 1,
        coinId: 1,
        filePath: 'relative-image.jpg',
        imageType: 'obverse' as const,
        isPrimary: true,
        createdAt: '2026-01-01',
      },
    ],
  }

  const mockCoinAbsolutePath: TrayCoin = {
    id: 3,
    name: 'Coin With Absolute Path',
    diameterMm: 22,
    images: [
      {
        id: 3,
        coinId: 3,
        filePath: '/absolute/path/image.jpg',
        imageType: 'obverse' as const,
        isPrimary: true,
        createdAt: '2026-01-01',
      },
    ],
  }

  const mockCoinExternalUrl: TrayCoin = {
    id: 4,
    name: 'Coin With External URL',
    diameterMm: 24,
    images: [
      {
        id: 4,
        coinId: 4,
        filePath: 'https://example.com/coin.jpg',
        imageType: 'obverse' as const,
        isPrimary: true,
        createdAt: '2026-01-01',
      },
    ],
  }

  const mockCoinNoImage: TrayCoin = {
    id: 2,
    name: 'Coin Without Image',
    diameterMm: 20,
    images: [],
  }

  it('renders coin image through authenticated media blob when available', async () => {
    const wrapper = mount(MuseumTrayWell, {
      props: {
        coin: mockCoin,
        renderSizePx: 70,
      },
    })
    await flushPromises()

    const img = wrapper.find('img')
    expect(img.exists()).toBe(true)
    expect(img.attributes('src')).toBe('blob:tray-image')
    expect(img.attributes('alt')).toBe('Test Coin')
    expect(img.attributes('loading')).toBe('eager')
    expect(img.attributes('decoding')).toBe('async')
  })

  it('preserves absolute path for images', () => {
    const wrapper = mount(MuseumTrayWell, {
      props: {
        coin: mockCoinAbsolutePath,
        renderSizePx: 70,
      },
    })

    const img = wrapper.find('img')
    expect(img.exists()).toBe(true)
    expect(img.attributes('src')).toBe('/absolute/path/image.jpg')
  })

  it('preserves external URL for images', () => {
    const wrapper = mount(MuseumTrayWell, {
      props: {
        coin: mockCoinExternalUrl,
        renderSizePx: 70,
      },
    })

    const img = wrapper.find('img')
    expect(img.exists()).toBe(true)
    expect(img.attributes('src')).toBe('https://example.com/coin.jpg')
  })

  it('renders placeholder when no images', () => {
    const wrapper = mount(MuseumTrayWell, {
      props: {
        coin: mockCoinNoImage,
        renderSizePx: 70,
      },
    })

    const img = wrapper.find('img')
    expect(img.exists()).toBe(false)
    // Placeholder should be rendered (Coins icon from lucide)
    expect(wrapper.html()).toContain('well-placeholder')
  })

  it('emits coin-clicked on click', async () => {
    const wrapper = mount(MuseumTrayWell, {
      props: {
        coin: mockCoin,
        renderSizePx: 70,
      },
    })

    await wrapper.find('.tray-well').trigger('click')
    expect(wrapper.emitted('coin-clicked')).toBeTruthy()
    expect(wrapper.emitted('coin-clicked')?.[0]).toEqual([1])
  })

  it('emits coin-clicked on Enter key', async () => {
    const wrapper = mount(MuseumTrayWell, {
      props: {
        coin: mockCoin,
        renderSizePx: 70,
      },
    })

    await wrapper.find('.tray-well').trigger('keydown.enter')
    expect(wrapper.emitted('coin-clicked')).toBeTruthy()
    expect(wrapper.emitted('coin-clicked')?.[0]).toEqual([1])
  })

  it('does not emit on other keys', async () => {
    const wrapper = mount(MuseumTrayWell, {
      props: {
        coin: mockCoin,
        renderSizePx: 70,
      },
    })

    await wrapper.find('.tray-well').trigger('keydown.space')
    await wrapper.find('.tray-well').trigger('keydown.escape')
    expect(wrapper.emitted('coin-clicked')).toBeFalsy()
  })

  it('has accessible aria-label', () => {
    const wrapper = mount(MuseumTrayWell, {
      props: {
        coin: mockCoin,
        renderSizePx: 70,
      },
    })

    const well = wrapper.find('.tray-well')
    expect(well.attributes('aria-label')).toBe('Test Coin')
  })

  it('is keyboard focusable', () => {
    const wrapper = mount(MuseumTrayWell, {
      props: {
        coin: mockCoin,
        renderSizePx: 70,
      },
    })

    const well = wrapper.find('.tray-well')
    expect(well.attributes('tabindex')).toBe('0')
  })

  it('applies correct size from renderSizePx prop', () => {
    const wrapper = mount(MuseumTrayWell, {
      props: {
        coin: mockCoin,
        renderSizePx: 100,
      },
    })

    const well = wrapper.find('.tray-well')
    const style = well.attributes('style')
    expect(style).toContain('100px')
  })
})
