import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SettingsAccountSection from '@/components/settings/SettingsAccountSection.vue'
import type { OIDCLinkedIdentity, OIDCPublicProvider, User } from '@/types'

const mockGetOIDCIdentities = vi.fn()
const mockGetOIDCPublicProviders = vi.fn()
const mockStartOIDCLink = vi.fn()
const mockDeleteOIDCIdentity = vi.fn()
const mockShowConfirm = vi.fn()
const mockLocationAssign = vi.fn()

const authUser: User = {
  id: 1,
  username: 'collector',
  role: 'user',
  email: 'collector@example.com',
  avatarPath: '',
  isPublic: false,
  bio: '',
  zipCode: '',
}

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: authUser,
  }),
}))

vi.mock('@/api/client', () => ({
  getOIDCIdentities: () => mockGetOIDCIdentities(),
  getOIDCPublicProviders: () => mockGetOIDCPublicProviders(),
  startOIDCLink: (providerId: number, request: { redirectPath: string }) => mockStartOIDCLink(providerId, request),
  deleteOIDCIdentity: (identityId: number) => mockDeleteOIDCIdentity(identityId),
  getApiErrorMessage: (error: unknown) => {
    const maybeError = error as { response?: { data?: { error?: string; message?: string } }; message?: string }
    return maybeError.response?.data?.error ?? maybeError.response?.data?.message ?? maybeError.message ?? ''
  },
  changePassword: vi.fn(),
  uploadAvatar: vi.fn(),
  deleteAvatar: vi.fn(),
  updateProfile: vi.fn(),
  validateNumisBidsCredentials: vi.fn(),
  testPushover: vi.fn(),
  webauthnRegisterBegin: vi.fn(),
  webauthnRegisterFinish: vi.fn(),
  webauthnListCredentials: vi.fn(),
  webauthnDeleteCredential: vi.fn(),
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({
    showConfirm: mockShowConfirm,
  }),
}))

const linkedIdentity: OIDCLinkedIdentity = {
  id: 10,
  providerId: 1,
  providerDisplayName: 'Microsoft',
  issuer: 'https://login.microsoftonline.com/tenant/v2.0',
  subjectPreview: 'abc123...',
  email: 'collector@example.com',
  emailVerified: true,
  createdAt: '2026-06-23T15:59:00Z',
  lastLoginAt: '2026-06-24T14:10:00Z',
}

const providers: OIDCPublicProvider[] = [
  { id: 1, name: 'entra-work', displayName: 'Microsoft', providerType: 'entra' },
  { id: 2, name: 'pocket-id', displayName: 'Pocket ID', providerType: 'pocket_id' },
]

function mountSection() {
  return mount(SettingsAccountSection, {
    global: {
      stubs: {
        AuthenticatedImage: { template: '<img alt="Avatar" />' },
        LockKeyhole: true,
        LinkIcon: true,
        Teleport: true,
      },
    },
  })
}

function buttonByText(wrapper: ReturnType<typeof mount>, text: string) {
  const button = wrapper.findAll('button').find(candidate => candidate.text().includes(text))
  expect(button, `button containing ${text} should exist`).toBeTruthy()
  return button!
}

describe('SettingsAccountSection OIDC identities', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.defineProperty(window, 'PublicKeyCredential', {
      value: undefined,
      configurable: true,
    })
    Object.defineProperty(window, 'location', {
      value: { assign: mockLocationAssign },
      configurable: true,
    })
    mockGetOIDCIdentities.mockResolvedValue({ data: { identities: [linkedIdentity] } })
    mockGetOIDCPublicProviders.mockResolvedValue({ data: { providers } })
    mockStartOIDCLink.mockResolvedValue({
      data: {
        authorizationUrl: 'https://provider.example/authorize',
        expiresAt: '2026-06-24T16:00:00Z',
      },
    })
    mockDeleteOIDCIdentity.mockResolvedValue({ data: { message: 'OIDC identity unlinked' } })
    mockShowConfirm.mockResolvedValue(true)
  })

  it('lists linked identities with provider, issuer, subject, email, linked date, and last login', async () => {
    const wrapper = mountSection()
    await flushPromises()

    expect(wrapper.text()).toContain('Microsoft')
    expect(wrapper.text()).toContain('https://login.microsoftonline.com/tenant/v2.0')
    expect(wrapper.text()).toContain('abc123...')
    expect(wrapper.text()).toContain('collector@example.com')
    expect(wrapper.text()).toContain('Email verified')
    expect(wrapper.text()).toContain('Linked')
    expect(wrapper.text()).toContain('Last login')
    expect(wrapper.text()).toContain('Link Pocket ID')
    expect(wrapper.text()).not.toContain('Link Microsoft')
  })

  it('starts an OIDC link flow for an unlinked provider', async () => {
    const wrapper = mountSection()
    await flushPromises()

    await buttonByText(wrapper, 'Link Pocket ID').trigger('click')
    await flushPromises()

    expect(mockStartOIDCLink).toHaveBeenCalledWith(2, { redirectPath: '/settings?tab=account' })
    expect(mockLocationAssign).toHaveBeenCalledWith('https://provider.example/authorize')
  })

  it('unlinks an identity after confirmation and reloads linked identities', async () => {
    const wrapper = mountSection()
    await flushPromises()

    await buttonByText(wrapper, 'Unlink').trigger('click')
    await flushPromises()

    expect(mockShowConfirm).toHaveBeenCalled()
    expect(mockDeleteOIDCIdentity).toHaveBeenCalledWith(10)
    expect(wrapper.text()).toContain('Microsoft unlinked.')
  })

  it('distinguishes link conflicts from provider configuration failures', async () => {
    mockStartOIDCLink.mockRejectedValueOnce({
      response: { status: 409, data: { error: 'external identity already linked to another user' } },
    })
    const wrapper = mountSection()
    await flushPromises()

    await buttonByText(wrapper, 'Link Pocket ID').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('already linked to another user')

    mockStartOIDCLink.mockRejectedValueOnce({
      response: { status: 500, data: { error: 'provider discovery failed' } },
    })
    await buttonByText(wrapper, 'Link Pocket ID').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('not configured correctly')
  })

  it('distinguishes unlink safety conflicts from missing identities', async () => {
    mockDeleteOIDCIdentity.mockRejectedValueOnce({
      response: { status: 409, data: { error: 'unlink would leave account without usable sign-in method' } },
    })
    const wrapper = mountSection()
    await flushPromises()

    await buttonByText(wrapper, 'Unlink').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('no usable sign-in method')

    mockDeleteOIDCIdentity.mockRejectedValueOnce({
      response: { status: 404, data: { error: 'identity not found' } },
    })
    await buttonByText(wrapper, 'Unlink').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('not found for your account')
  })
})
