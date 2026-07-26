import { mount, flushPromises } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CreateMintModal from '@/components/CreateMintModal.vue'

const mocks = vi.hoisted(() => ({
  geocodeMintName: vi.fn(),
  createMintLocation: vi.fn(),
  mapOn: vi.fn(),
  mapSetView: vi.fn(),
  mapRemove: vi.fn(),
  markerSetLatLng: vi.fn(),
  markerOn: vi.fn(),
  clickHandlers: [] as ((event: { latlng: { lat: number; lng: number } }) => void)[],
}))

vi.mock('@/api/client', () => ({
  geocodeMintName: mocks.geocodeMintName,
  createMintLocation: mocks.createMintLocation,
}))

vi.mock('leaflet', () => ({
  map: vi.fn(() => ({
    setView: mocks.mapSetView,
    remove: mocks.mapRemove,
    on: vi.fn((event: string, handler: (e: { latlng: { lat: number; lng: number } }) => void) => {
      if (event === 'click') mocks.clickHandlers.push(handler)
      mocks.mapOn(event, handler)
    }),
  })),
  tileLayer: vi.fn(() => ({ addTo: vi.fn() })),
  marker: vi.fn(() => ({
    addTo: vi.fn().mockReturnThis(),
    setLatLng: mocks.markerSetLatLng,
    on: mocks.markerOn,
    getLatLng: vi.fn(() => ({ lat: 1, lng: 2 })),
  })),
}))

async function waitForMapMount() {
  await nextTick()
  await nextTick()
}

function mountModal(props: Partial<{ open: boolean; initialName: string }> = {}) {
  return mount(CreateMintModal, {
    props: { open: true, ...props },
    global: { stubs: { Teleport: true } },
  })
}

describe('CreateMintModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.clickHandlers.length = 0
  })

  it('searches by name and places a pin at the top geocoding candidate', async () => {
    mocks.geocodeMintName.mockResolvedValue({
      data: { candidates: [{ displayName: 'Sirmium, Serbia', lat: 44.97, lng: 19.61 }] },
    })

    const wrapper = mountModal()
    await waitForMapMount()

    await wrapper.find('input').setValue('Sirmium')
    await wrapper.find('button.btn-secondary').trigger('click')
    await flushPromises()

    expect(mocks.geocodeMintName).toHaveBeenCalledWith('Sirmium')
    expect(wrapper.text()).toContain('Pin set at 44.9700, 19.6100')
  })

  it('shows a manual-placement hint when the geocoder finds nothing', async () => {
    mocks.geocodeMintName.mockResolvedValue({ data: { candidates: [] } })

    const wrapper = mountModal()
    await waitForMapMount()

    await wrapper.find('input').setValue('Nonexistentville')
    await wrapper.find('button.btn-secondary').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('No matches found for "Nonexistentville"')
    expect(wrapper.text()).toContain('Click the map to place a pin.')
  })

  it('lets the user place a pin manually by clicking the map', async () => {
    const wrapper = mountModal()
    await waitForMapMount()

    expect(mocks.clickHandlers.length).toBeGreaterThan(0)
    mocks.clickHandlers[0]!({ latlng: { lat: 10, lng: 20 } })
    await nextTick()

    expect(wrapper.text()).toContain('Pin set at 10.0000, 20.0000')
  })

  it('saves the new mint with the typed name, region, and pinned coordinates', async () => {
    mocks.createMintLocation.mockResolvedValue({
      data: { id: 99, displayName: 'Sirmium', lat: 44.97, lng: 19.61, region: 'Pannonia', aliases: [] },
    })

    const wrapper = mountModal()
    await waitForMapMount()

    await wrapper.find('input').setValue('Sirmium')
    mocks.clickHandlers[0]!({ latlng: { lat: 44.97, lng: 19.61 } })
    await nextTick()
    await wrapper.find('input[placeholder="e.g. Pannonia"]').setValue('Pannonia')

    const saveButton = wrapper.findAll('button').find((b) => b.text().includes('Save mint'))
    await saveButton!.trigger('click')
    await flushPromises()

    expect(mocks.createMintLocation).toHaveBeenCalledWith({
      displayName: 'Sirmium',
      lat: 44.97,
      lng: 19.61,
      region: 'Pannonia',
      aliases: [],
    })
    expect(wrapper.emitted('created')?.[0]?.[0]).toMatchObject({ id: 99, displayName: 'Sirmium' })
  })

  it('disables save until both a name and a pin are set', async () => {
    const wrapper = mountModal()
    await waitForMapMount()
    const findSaveButton = () => wrapper.findAll('button').find((b) => b.text().includes('Save mint'))

    expect(findSaveButton()?.attributes('disabled')).toBeDefined()

    await wrapper.find('input').setValue('Sirmium')
    expect(findSaveButton()?.attributes('disabled')).toBeDefined()

    mocks.clickHandlers[0]!({ latlng: { lat: 1, lng: 2 } })
    await nextTick()
    expect(findSaveButton()?.attributes('disabled')).toBeUndefined()
  })

  it('emits close when cancelled', async () => {
    const wrapper = mountModal()
    await waitForMapMount()

    const cancelButton = wrapper.findAll('button').find((b) => b.text().includes('Cancel'))
    await cancelButton!.trigger('click')

    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
