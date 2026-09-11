import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SettingsDataSection from '@/components/settings/SettingsDataSection.vue'
import type { MintLocation, StorageLocation } from '@/types'

const mockGetMintLocations = vi.fn()
const mockCreateMintLocation = vi.fn()
const mockUpdateMintLocation = vi.fn()
const mockDeleteMintLocation = vi.fn()
const mockGetStorageLocations = vi.fn()
const mockCreateStorageLocation = vi.fn()
const mockUpdateStorageLocation = vi.fn()
const mockDeleteStorageLocation = vi.fn()
const mockShowConfirm = vi.fn()

vi.mock('@/api/client', () => ({
  getTags: vi.fn().mockResolvedValue({ data: { tags: [] } }),
  createTag: vi.fn(),
  updateTag: vi.fn(),
  deleteTag: vi.fn(),
  getStorageLocations: () => mockGetStorageLocations(),
  createStorageLocation: (data: unknown) => mockCreateStorageLocation(data),
  updateStorageLocation: (id: number, data: unknown) => mockUpdateStorageLocation(id, data),
  deleteStorageLocation: (id: number) => mockDeleteStorageLocation(id),
  migrateLegacyReferences: vi.fn(),
  getMintLocations: () => mockGetMintLocations(),
  createMintLocation: (data: unknown) => mockCreateMintLocation(data),
  updateMintLocation: (id: number, data: unknown) => mockUpdateMintLocation(id, data),
  deleteMintLocation: (id: number) => mockDeleteMintLocation(id),
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({
    showConfirm: mockShowConfirm,
  }),
}))

const globalMint: MintLocation = {
  id: 1,
  userId: null,
  displayName: 'Rome',
  lat: 41.9,
  lng: 12.5,
  region: 'Italy',
  aliases: ['Roma'],
  createdAt: '2024-01-01T00:00:00Z',
  updatedAt: '2024-01-01T00:00:00Z',
}

const userMint: MintLocation = {
  id: 10,
  userId: 5,
  displayName: 'My Mint',
  lat: 37.97,
  lng: 23.73,
  region: 'Greece',
  aliases: ['Athens'],
  createdAt: '2024-01-01T00:00:00Z',
  updatedAt: '2024-01-01T00:00:00Z',
}

function mintResponse(locations: MintLocation[]) {
  return { data: { mintLocations: locations } }
}

function storageResponse(locations: StorageLocation[]) {
  return { data: { storageLocations: locations } }
}

const standardLocation: StorageLocation = {
  id: 21,
  name: 'Safe',
  type: 'standard',
  rows: null,
  columns: null,
  capacity: 0,
  occupied: 2,
}

const occupiedTray: StorageLocation = {
  id: 22,
  name: 'Cabinet A',
  type: 'tray',
  rows: 3,
  columns: 3,
  capacity: 9,
  occupied: 4,
}

describe('SettingsDataSection — Custom Mint Locations', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockShowConfirm.mockResolvedValue(false)
    mockGetStorageLocations.mockResolvedValue(storageResponse([]))
  })

  it('shows empty state when user has no custom locations', async () => {
    mockGetMintLocations.mockResolvedValue(mintResponse([globalMint]))
    const wrapper = mount(SettingsDataSection)
    await flushPromises()
    expect(wrapper.text()).toContain('No custom mint locations yet')
  })

  it('renders only user-scoped locations, not global', async () => {
    mockGetMintLocations.mockResolvedValue(mintResponse([globalMint, userMint]))
    const wrapper = mount(SettingsDataSection)
    await flushPromises()
    expect(wrapper.text()).toContain('My Mint')
    expect(wrapper.text()).not.toContain('Rome')
  })

  it('shows location count chip', async () => {
    mockGetMintLocations.mockResolvedValue(mintResponse([globalMint, userMint]))
    const wrapper = mount(SettingsDataSection)
    await flushPromises()
    expect(wrapper.text()).toContain('1 locations')
  })

  it('calls createMintLocation with correct payload on form submit', async () => {
    mockGetMintLocations.mockResolvedValue(mintResponse([]))
    mockCreateMintLocation.mockResolvedValue({ data: { ...userMint, id: 11 } })
    const wrapper = mount(SettingsDataSection)
    await flushPromises()

    const inputs = wrapper.findAll('input[type="text"]')
    const displayNameInput = inputs.find((i) => (i.element as HTMLInputElement).placeholder === 'e.g. Rome')
    expect(displayNameInput).toBeTruthy()
    await displayNameInput!.setValue('Athens Mint')

    const numberInputs = wrapper.findAll('input[type="number"]')
    await numberInputs.find((i) => (i.element as HTMLInputElement).placeholder === '41.9')?.setValue('37.97')
    await numberInputs.find((i) => (i.element as HTMLInputElement).placeholder === '12.5')?.setValue('23.73')

    const form = wrapper.find('form[aria-label], form')
    await form.trigger('submit')
    await flushPromises()

    expect(mockCreateMintLocation).toHaveBeenCalledWith(
      expect.objectContaining({
        displayName: 'Athens Mint',
        lat: 37.97,
        lng: 23.73,
      }),
    )
  })

  it('shows validation error for missing display name', async () => {
    mockGetMintLocations.mockResolvedValue(mintResponse([]))
    const wrapper = mount(SettingsDataSection)
    await flushPromises()

    const numberInputs = wrapper.findAll('input[type="number"]')
    await numberInputs.find((i) => (i.element as HTMLInputElement).placeholder === '41.9')?.setValue('37.97')
    await numberInputs.find((i) => (i.element as HTMLInputElement).placeholder === '12.5')?.setValue('23.73')

    const form = wrapper.find('form')
    await form.trigger('submit')
    await flushPromises()

    expect(wrapper.text()).toContain('Display name is required')
    expect(mockCreateMintLocation).not.toHaveBeenCalled()
  })

  it('populates form and calls updateMintLocation when editing', async () => {
    mockGetMintLocations.mockResolvedValue(mintResponse([userMint]))
    mockUpdateMintLocation.mockResolvedValue({ data: userMint })
    const wrapper = mount(SettingsDataSection)
    await flushPromises()

    const editBtn = wrapper.findAll('button').find((b) => b.text() === 'Edit')
    expect(editBtn).toBeTruthy()
    await editBtn!.trigger('click')

    expect(wrapper.text()).toContain('Edit Location')

    const form = wrapper.find('form')
    await form.trigger('submit')
    await flushPromises()

    expect(mockUpdateMintLocation).toHaveBeenCalledWith(
      userMint.id,
      expect.objectContaining({ displayName: 'My Mint' }),
    )
  })

  it('resets form on cancel while editing', async () => {
    mockGetMintLocations.mockResolvedValue(mintResponse([userMint]))
    const wrapper = mount(SettingsDataSection)
    await flushPromises()

    await wrapper.findAll('button').find((b) => b.text() === 'Edit')!.trigger('click')
    expect(wrapper.text()).toContain('Edit Location')

    await wrapper.findAll('button').find((b) => b.text() === 'Cancel')!.trigger('click')
    expect(wrapper.text()).toContain('Add Location')
  })

  it('calls deleteMintLocation after confirm', async () => {
    mockGetMintLocations.mockResolvedValue(mintResponse([userMint]))
    mockDeleteMintLocation.mockResolvedValue({})
    mockShowConfirm.mockResolvedValue(true)
    const wrapper = mount(SettingsDataSection)
    await flushPromises()

    const deleteBtn = wrapper.findAll('button').find((b) => b.text() === 'Delete')
    expect(deleteBtn).toBeTruthy()
    await deleteBtn!.trigger('click')
    await flushPromises()

    expect(mockDeleteMintLocation).toHaveBeenCalledWith(userMint.id)
  })

  it('does not call deleteMintLocation when user cancels confirm', async () => {
    mockGetMintLocations.mockResolvedValue(mintResponse([userMint]))
    mockShowConfirm.mockResolvedValue(false)
    const wrapper = mount(SettingsDataSection)
    await flushPromises()

    await wrapper.findAll('button').find((b) => b.text() === 'Delete')!.trigger('click')
    await flushPromises()

    expect(mockDeleteMintLocation).not.toHaveBeenCalled()
  })

  it('shows error when createMintLocation fails', async () => {
    mockGetMintLocations.mockResolvedValue(mintResponse([]))
    mockCreateMintLocation.mockRejectedValue({ response: { data: { error: 'duplicate name' } } })
    const wrapper = mount(SettingsDataSection)
    await flushPromises()

    const inputs = wrapper.findAll('input[type="text"]')
    await inputs.find((i) => (i.element as HTMLInputElement).placeholder === 'e.g. Rome')!.setValue('Test')
    await wrapper.findAll('input[type="number"]')[0]?.setValue('10')
    await wrapper.findAll('input[type="number"]')[1]?.setValue('10')

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.text()).toContain('duplicate name')
  })
})

describe('SettingsDataSection — Storage Locations', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetMintLocations.mockResolvedValue(mintResponse([]))
    mockGetStorageLocations.mockResolvedValue(storageResponse([standardLocation, occupiedTray]))
    mockShowConfirm.mockResolvedValue(false)
  })

  it('shows type-specific fields, capacity preview, and occupancy labels', async () => {
    const wrapper = mount(SettingsDataSection)
    await flushPromises()

    const tagsSection = wrapper.find('section[aria-labelledby="tags-heading"]')
    const storageSection = wrapper.find('section[aria-labelledby="storage-locations-heading"]')
    expect(wrapper.text()).toContain('Standard Location')
    expect(wrapper.text()).toContain('Coin Tray · 3×3 · 4 / 9')
    expect(tagsSection.find('select[aria-label="Storage location type"]').exists()).toBe(false)
    expect(tagsSection.find('input[aria-label="Tray rows"]').exists()).toBe(false)
    expect(storageSection.find('select[aria-label="Storage location type"]').exists()).toBe(true)
    expect(storageSection.find('input[aria-label="Tray rows"]').exists()).toBe(false)

    await storageSection.find('select[aria-label="Storage location type"]').setValue('tray')

    const dimensions = storageSection.findAll('input[type="number"]').filter((input) =>
      ['Tray rows', 'Tray columns'].includes(input.attributes('aria-label') ?? ''),
    )
    expect(dimensions).toHaveLength(2)
    await dimensions[0]!.setValue('4')
    await dimensions[1]!.setValue('5')
    expect(wrapper.text()).toContain('20 slots')
  })

  it('creates type-specific payloads and surfaces field validation errors', async () => {
    mockCreateStorageLocation.mockResolvedValue({})
    const wrapper = mount(SettingsDataSection)
    await flushPromises()

    const storageSection = wrapper.find('section[aria-labelledby="storage-locations-heading"]')
    await storageSection.find('select[aria-label="Storage location type"]').setValue('tray')
    await storageSection.find('input[placeholder="New storage location..."]').setValue('  Drawer  ')
    const dimensions = storageSection.findAll('input[type="number"]').filter((input) =>
      ['Tray rows', 'Tray columns'].includes(input.attributes('aria-label') ?? ''),
    )
    await dimensions[0]!.setValue('2')
    await dimensions[1]!.setValue('6')
    await storageSection.findAll('button').find((button) => button.text() === 'Create Location')!.trigger('click')
    await flushPromises()

    expect(mockCreateStorageLocation).toHaveBeenCalledWith({
      name: 'Drawer',
      type: 'tray',
      rows: 2,
      columns: 6,
    })

    mockCreateStorageLocation.mockRejectedValueOnce({
      response: { status: 400, data: { field: 'rows', message: 'Tray rows must be between 1 and 20' } },
    })
    await storageSection.find('input[placeholder="New storage location..."]').setValue('Invalid Tray')
    await storageSection.findAll('button').find((button) => button.text() === 'Create Location')!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Tray rows must be between 1 and 20')
  })

  it('keeps occupied dimensions disabled while allowing rename', async () => {
    mockUpdateStorageLocation.mockResolvedValue({})
    const wrapper = mount(SettingsDataSection)
    await flushPromises()

    const trayRow = wrapper.findAll('div.rounded-sm').find((row) =>
      row.text().includes('Cabinet A') && row.findAll('button').some((button) => button.text() === 'Edit'),
    )
    await trayRow!.findAll('button').find((button) => button.text() === 'Edit')!.trigger('click')

    const editingTrayRow = wrapper.findAll('div.rounded-sm').find((row) =>
      row.text().includes('Empty the tray before resizing.') && row.findAll('button').some((button) => button.text() === 'Save'),
    )
    const dimensions = editingTrayRow!.findAll('input[type="number"]')
    expect(dimensions).toHaveLength(2)
    expect(dimensions.every((input) => input.attributes('disabled') !== undefined)).toBe(true)
    expect(editingTrayRow!.text()).toContain('Empty the tray before resizing.')

    const nameInput = wrapper.findAll('input').find((input) => (input.element as HTMLInputElement).value === 'Cabinet A')
    await nameInput!.setValue('Renamed Cabinet')
    await editingTrayRow!.findAll('button').find((button) => button.text() === 'Save')!.trigger('click')
    await flushPromises()

    expect(mockUpdateStorageLocation).toHaveBeenCalledWith(occupiedTray.id, { name: 'Renamed Cabinet' })
  })

  it('shows referenced deletion counts and stable conflict messages', async () => {
    mockShowConfirm.mockResolvedValue(true)
    mockDeleteStorageLocation.mockRejectedValueOnce({
      response: {
        status: 409,
        data: { code: 'location_referenced', count: 2, message: '2 coins still reference this location' },
      },
    })
    const wrapper = mount(SettingsDataSection)
    await flushPromises()

    const standardRow = wrapper.findAll('div.rounded-sm').find((row) =>
      row.text().includes('Safe') && row.findAll('button').some((button) => button.text() === 'Delete'),
    )
    await standardRow!.findAll('button').find((button) => button.text() === 'Delete')!.trigger('click')
    await flushPromises()

    expect(mockDeleteStorageLocation).toHaveBeenCalledWith(standardLocation.id)
    expect(wrapper.text()).toContain('2 coins still reference this location')

    mockUpdateStorageLocation.mockRejectedValueOnce({
      response: { status: 409, data: { code: 'tray_occupied', message: 'Empty the tray before resizing' } },
    })
    const trayRow = wrapper.findAll('div.rounded-sm').find((row) =>
      row.text().includes('Cabinet A') && row.findAll('button').some((button) => button.text() === 'Edit'),
    )
    await trayRow!.findAll('button').find((button) => button.text() === 'Edit')!.trigger('click')
    const editingTrayRow = wrapper.findAll('div.rounded-sm').find((row) =>
      row.text().includes('Empty the tray before resizing.') && row.findAll('button').some((button) => button.text() === 'Save'),
    )
    await editingTrayRow!.findAll('button').find((button) => button.text() === 'Save')!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Empty the tray before resizing')
  })
})
