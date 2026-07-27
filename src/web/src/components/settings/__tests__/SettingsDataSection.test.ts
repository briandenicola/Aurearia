import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SettingsDataSection from '@/components/settings/SettingsDataSection.vue'
import type { MintLocation } from '@/types'

const mockGetMintLocations = vi.fn()
const mockCreateMintLocation = vi.fn()
const mockUpdateMintLocation = vi.fn()
const mockDeleteMintLocation = vi.fn()
const mockShowConfirm = vi.fn()

vi.mock('@/api/client', () => ({
  getTags: vi.fn().mockResolvedValue({ data: { tags: [] } }),
  createTag: vi.fn(),
  updateTag: vi.fn(),
  deleteTag: vi.fn(),
  getStorageLocations: vi.fn().mockResolvedValue({ data: { storageLocations: [] } }),
  createStorageLocation: vi.fn(),
  updateStorageLocation: vi.fn(),
  deleteStorageLocation: vi.fn(),
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

describe('SettingsDataSection — Custom Mint Locations', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockShowConfirm.mockResolvedValue(false)
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
