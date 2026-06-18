import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import MintMapLeaflet, { DEFAULT_MAP_CENTER, DEFAULT_MAP_ZOOM, OSM_ATTRIBUTION, OSM_TILE_URL } from '@/components/map/MintMapLeaflet.vue'
import { groupCoinsByMint } from '@/utils/mintMap'
import { buildMintMapFixtureCoins, buildTestMintLocations } from '@/test/fixtures/coins'

interface MarkerRecord {
  latLng: [number, number]
  options: Record<string, unknown>
  handlers: Record<string, () => void>
  element: HTMLElement
}

const mocks = vi.hoisted(() => {
  const markerRecords: MarkerRecord[] = []
  const markerAddTo = vi.fn()
  return {
    mapSetView: vi.fn(),
    mapFitBounds: vi.fn(),
    mapRemove: vi.fn(),
    tileAddTo: vi.fn(),
    layerAddTo: vi.fn(),
    layerClear: vi.fn(),
    boundsPad: vi.fn(() => 'padded-bounds'),
    markerAddTo,
    markerRecords,
    tileLayer: vi.fn(() => ({ addTo: vi.fn() })),
    divIcon: vi.fn((options: Record<string, unknown>) => ({ options })),
    marker: vi.fn((latLng: [number, number], options: Record<string, unknown>) => {
      const record: MarkerRecord = {
        latLng,
        options,
        handlers: {},
        element: document.createElement('button'),
      }
      markerRecords.push(record)
      return {
        on: vi.fn((event: string, handler: () => void) => {
          record.handlers[event] = handler
        }),
        addTo: markerAddTo,
        getElement: vi.fn(() => record.element),
      }
    }),
  }
})

vi.mock('leaflet', () => ({
  map: vi.fn(() => ({
    fitBounds: mocks.mapFitBounds,
    remove: mocks.mapRemove,
    setView: mocks.mapSetView,
  })),
  tileLayer: mocks.tileLayer,
  layerGroup: vi.fn(() => {
    const layer = {
      addTo: vi.fn(() => layer),
      clearLayers: mocks.layerClear,
    }
    return layer
  }),
  marker: mocks.marker,
  divIcon: mocks.divIcon,
  latLngBounds: vi.fn(() => ({
    pad: mocks.boundsPad,
  })),
}))

describe('MintMapLeaflet', () => {
  const groups = groupCoinsByMint(buildMintMapFixtureCoins(), buildTestMintLocations()).matched

  async function waitForMapMount() {
    await nextTick()
    await nextTick()
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mocks.markerRecords.length = 0
  })

  it('configures OpenStreetMap tiles with attribution and no collection data', async () => {
    mount(MintMapLeaflet, {
      props: { groups },
    })
    await waitForMapMount()

    expect(mocks.tileLayer).toHaveBeenCalledWith(OSM_TILE_URL, {
      attribution: OSM_ATTRIBUTION,
      maxZoom: 19,
    })
    expect(OSM_TILE_URL).toBe('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png')
    expect(OSM_TILE_URL).not.toMatch(/coin|user|jwt|mint|collection|[?&]/i)
    expect(OSM_ATTRIBUTION).toContain('OpenStreetMap')
  })

  it('creates markers from actual mint latitude and longitude with count badges', async () => {
    mount(MintMapLeaflet, {
      props: { groups, selectedMintId: 1 },
    })
    await waitForMapMount()

    const rome = groups.find((group) => group.mint.id === 1)
    expect(mocks.markerRecords[0]?.latLng).toEqual([rome?.mint.lat, rome?.mint.lng])
    expect(mocks.divIcon).toHaveBeenCalledWith(expect.objectContaining({
      className: expect.stringContaining('selected'),
      html: '<span class="mint-leaflet-marker-count">2</span>',
    }))
    expect(mocks.markerRecords[0]?.element.getAttribute('aria-label')).toBe('Rome: 2 coins')
    expect(mocks.mapFitBounds).toHaveBeenCalledWith('padded-bounds', { maxZoom: 8 })
  })

  it('uses the Mediterranean default view when there are no matched markers', async () => {
    mount(MintMapLeaflet, {
      props: { groups: [] },
    })
    await waitForMapMount()

    expect(mocks.marker).not.toHaveBeenCalled()
    expect(mocks.mapSetView).toHaveBeenCalledWith(DEFAULT_MAP_CENTER, DEFAULT_MAP_ZOOM)
  })

  it('emits selected mint from marker activation', async () => {
    const wrapper = mount(MintMapLeaflet, {
      props: { groups },
    })
    await waitForMapMount()

    mocks.markerRecords[0]?.handlers.click?.()

    expect(wrapper.emitted('select-mint')?.[0]?.[0]).toMatchObject({ mint: { id: 1 }, count: 2 })
  })
})
