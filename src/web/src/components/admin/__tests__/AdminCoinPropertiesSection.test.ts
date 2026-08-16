import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminCoinPropertiesSection from '../AdminCoinPropertiesSection.vue'
import type { MintLocation } from '@/types'

const mocks = vi.hoisted(() => ({
  getMintLocations: vi.fn(),
  adminCreateMintLocation: vi.fn(),
  adminUpdateMintLocation: vi.fn(),
  adminDeleteMintLocation: vi.fn(),
  searchNomismaMintCandidates: vi.fn(),
  linkNomismaMintLocation: vi.fn(),
  unlinkNomismaMintLocation: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

const mockShowConfirm = vi.fn()
vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({
    showConfirm: mockShowConfirm,
    showAlert: vi.fn(),
  }),
}))

vi.mock('@/composables/useSafeExternalLink', () => ({
  sanitizeExternalUrl: (url: string | null | undefined) => url ?? null,
}))

function buildGlobalMint(overrides: Partial<MintLocation> = {}): MintLocation {
  return {
    id: 1,
    userId: null,
    displayName: 'Rome',
    lat: 41.9,
    lng: 12.5,
    region: 'Italy',
    aliases: [],
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
    ...overrides,
  }
}

function mountSection() {
  return mount(AdminCoinPropertiesSection, {
    props: {
      categoryOptions: '',
      eraOptions: '',
      saving: false,
      msg: '',
      error: false,
    },
  })
}

async function submitNomismaSearch(wrapper: ReturnType<typeof mount>) {
  await wrapper.find('form.nomisma-search-form').trigger('submit')
}

describe('AdminCoinPropertiesSection - Nomisma authority linking', () => {
  beforeEach(() => {
    Object.values(mocks).forEach(mock => mock.mockReset())
    mockShowConfirm.mockReset()
    mockShowConfirm.mockResolvedValue(true)
    mocks.getMintLocations.mockResolvedValue({ data: [buildGlobalMint()] })
  })

  it('renders a Search Nomisma trigger for an unlinked global mint, shows candidates with no pre-selection, and confirms a link', async () => {
    mocks.searchNomismaMintCandidates.mockResolvedValue({
      data: { status: 'ok', candidates: [{ uri: 'http://nomisma.org/id/roma', label: 'Roma', score: 100, match: true }] },
    })
    mocks.linkNomismaMintLocation.mockResolvedValue({ data: buildGlobalMint({ nomismaUri: 'http://nomisma.org/id/roma', nomismaLabel: 'Roma' }) })
    mocks.getMintLocations
      .mockResolvedValueOnce({ data: [buildGlobalMint()] })
      .mockResolvedValueOnce({ data: [buildGlobalMint({ nomismaUri: 'http://nomisma.org/id/roma', nomismaLabel: 'Roma' })] })

    const wrapper = mount(AdminCoinPropertiesSection, {
      props: { categoryOptions: '', eraOptions: '', saving: false, msg: '', error: false },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Search Nomisma')
    await wrapper.findAll('button').find(b => /Search Nomisma|Change Nomisma Link/.test(b.text()))!.trigger('click')
    await flushPromises()

    // Candidate list should not pre-select anything: confirm must be an explicit click.
    await submitNomismaSearch(wrapper)
    await flushPromises()

    expect(mocks.searchNomismaMintCandidates).toHaveBeenCalledWith(1, 'Rome')
    expect(wrapper.text()).toContain('Roma')
    expect(wrapper.text()).toContain('Score 100.0')

    const confirmButtons = wrapper.findAll('button').filter(b => b.text() === 'Confirm')
    expect(confirmButtons).toHaveLength(1)
    await confirmButtons[0].trigger('click')
    await flushPromises()

    expect(mocks.linkNomismaMintLocation).toHaveBeenCalledWith(1, 'http://nomisma.org/id/roma', 'Roma')
    expect(wrapper.text()).toContain('Source: Nomisma.org · CC BY 4.0')
  })

  it('shows "no match" messaging without any candidate and without linking anything', async () => {
    mocks.searchNomismaMintCandidates.mockResolvedValue({ data: { status: 'no_match', candidates: [] } })
    const wrapper = mountSection()
    await flushPromises()

    await wrapper.findAll('button').find(b => /Search Nomisma|Change Nomisma Link/.test(b.text()))!.trigger('click')
    await submitNomismaSearch(wrapper)
    await flushPromises()

    expect(wrapper.text()).toContain('No Nomisma candidates found')
    expect(mocks.linkNomismaMintLocation).not.toHaveBeenCalled()
  })

  it('shows "unavailable" messaging on an upstream outage and never surfaces a raw error/5xx', async () => {
    mocks.searchNomismaMintCandidates.mockResolvedValue({ data: { status: 'unavailable', candidates: [] } })
    const wrapper = mountSection()
    await flushPromises()

    await wrapper.findAll('button').find(b => /Search Nomisma|Change Nomisma Link/.test(b.text()))!.trigger('click')
    await submitNomismaSearch(wrapper)
    await flushPromises()

    expect(wrapper.text()).toContain('Nomisma lookup is currently unavailable')
    expect(mocks.linkNomismaMintLocation).not.toHaveBeenCalled()
  })

  it('does not persist any link when the admin cancels the search without confirming', async () => {
    mocks.searchNomismaMintCandidates.mockResolvedValue({
      data: { status: 'ok', candidates: [{ uri: 'http://nomisma.org/id/roma', label: 'Roma', score: 100, match: true }] },
    })
    const wrapper = mountSection()
    await flushPromises()

    await wrapper.findAll('button').find(b => /Search Nomisma|Change Nomisma Link/.test(b.text()))!.trigger('click')
    await submitNomismaSearch(wrapper)
    await flushPromises()
    expect(wrapper.text()).toContain('Roma')

    const cancelButton = wrapper.findAll('button').find(b => b.text() === 'Cancel')
    await cancelButton?.trigger('click')
    await flushPromises()

    expect(mocks.linkNomismaMintLocation).not.toHaveBeenCalled()
    expect(wrapper.text()).not.toContain('Score 100.0')
  })

  it('allows replacing an existing link and shows an Unlink action, without altering name/coordinates/aliases', async () => {
    const linked = buildGlobalMint({ nomismaUri: 'http://nomisma.org/id/roma', nomismaLabel: 'Roma' })
    mocks.getMintLocations.mockResolvedValue({ data: [linked] })
    mocks.searchNomismaMintCandidates.mockResolvedValue({
      data: { status: 'ok', candidates: [{ uri: 'http://nomisma.org/id/roma-ii', label: 'Roma II', score: 90, match: false }] },
    })
    mocks.linkNomismaMintLocation.mockResolvedValue({ data: buildGlobalMint({ nomismaUri: 'http://nomisma.org/id/roma-ii', nomismaLabel: 'Roma II' }) })

    const wrapper = mountSection()
    await flushPromises()

    expect(wrapper.text()).toContain('Source: Nomisma.org · CC BY 4.0')
    expect(wrapper.text()).toContain('Change Nomisma Link')
    expect(wrapper.text()).toContain('Unlink')

    await wrapper.findAll('button').find(b => /Search Nomisma|Change Nomisma Link/.test(b.text()))!.trigger('click')
    await submitNomismaSearch(wrapper)
    await flushPromises()

    const confirmButtons = wrapper.findAll('button').filter(b => b.text() === 'Confirm')
    await confirmButtons[0].trigger('click')
    await flushPromises()

    expect(mocks.linkNomismaMintLocation).toHaveBeenCalledWith(1, 'http://nomisma.org/id/roma-ii', 'Roma II')
    // Name/coordinates/aliases fields are driven purely by the (unchanged) mint list response, not touched by the Nomisma flow.
    expect(wrapper.text()).toContain('Italy · 41.9, 12.5')
  })

  it('unlinks after an explicit confirmation and removes the attribution', async () => {
    const linked = buildGlobalMint({ nomismaUri: 'http://nomisma.org/id/roma', nomismaLabel: 'Roma' })
    mocks.getMintLocations
      .mockResolvedValueOnce({ data: [linked] })
      .mockResolvedValueOnce({ data: [buildGlobalMint()] })
    mocks.unlinkNomismaMintLocation.mockResolvedValue({ data: { message: 'Nomisma link removed' } })

    const wrapper = mountSection()
    await flushPromises()
    expect(wrapper.text()).toContain('Source: Nomisma.org · CC BY 4.0')

    const unlinkButton = wrapper.findAll('button').find(b => b.text() === 'Unlink')
    await unlinkButton?.trigger('click')
    await flushPromises()

    expect(mockShowConfirm).toHaveBeenCalled()
    expect(mocks.unlinkNomismaMintLocation).toHaveBeenCalledWith(1)
    expect(wrapper.text()).not.toContain('Source: Nomisma.org · CC BY 4.0')
  })

  it('does not unlink when the admin declines the confirmation dialog', async () => {
    mockShowConfirm.mockResolvedValue(false)
    const linked = buildGlobalMint({ nomismaUri: 'http://nomisma.org/id/roma', nomismaLabel: 'Roma' })
    mocks.getMintLocations.mockResolvedValue({ data: [linked] })

    const wrapper = mountSection()
    await flushPromises()

    const unlinkButton = wrapper.findAll('button').find(b => b.text() === 'Unlink')
    await unlinkButton?.trigger('click')
    await flushPromises()

    expect(mocks.unlinkNomismaMintLocation).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Source: Nomisma.org · CC BY 4.0')
  })

  it('does not render any Nomisma controls for a private mint location', async () => {
    mocks.getMintLocations.mockResolvedValue({
      data: [buildGlobalMint({ id: 2, userId: 7, displayName: 'My Private Mint' })],
    })
    const wrapper = mountSection()
    await flushPromises()

    expect(wrapper.text()).not.toContain('Search Nomisma')
    expect(wrapper.text()).not.toContain('Nomisma')
  })

  it('exposes the Nomisma candidate list as a keyboard/screen-reader accessible ARIA list with a live-region status', async () => {
    mocks.searchNomismaMintCandidates.mockResolvedValue({
      data: { status: 'ok', candidates: [{ uri: 'http://nomisma.org/id/roma', label: 'Roma', score: 100, match: true }] },
    })
    const wrapper = mountSection()
    await flushPromises()

    await wrapper.findAll('button').find(b => /Search Nomisma|Change Nomisma Link/.test(b.text()))!.trigger('click')
    await submitNomismaSearch(wrapper)
    await flushPromises()

    const list = wrapper.find('ul[aria-label="Nomisma candidates"]')
    expect(list.exists()).toBe(true)
    expect(list.attributes('role')).toBe('list')

    mocks.searchNomismaMintCandidates.mockResolvedValueOnce({ data: { status: 'unavailable', candidates: [] } })
    await submitNomismaSearch(wrapper)
    await flushPromises()
    const status = wrapper.find('[role="status"][aria-live="polite"]')
    expect(status.exists()).toBe(true)
  })
})


