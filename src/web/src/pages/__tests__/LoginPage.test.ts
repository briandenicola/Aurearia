import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LoginPage from '@/pages/LoginPage.vue'

const mockPush = vi.fn()
const mockDoLogin = vi.fn()
const mockDoWebAuthnLogin = vi.fn()
const mockWebAuthnCheck = vi.fn()
const mockGetOIDCPublicProviders = vi.fn()
const mockStartOIDCLogin = vi.fn()
const mockLocationAssign = vi.fn()
const mockRoute = { query: {} as Record<string, string | string[] | undefined> }

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
  useRoute: () => mockRoute,
  RouterLink: { template: '<a><slot /></a>' },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    doLogin: mockDoLogin,
    doWebAuthnLogin: mockDoWebAuthnLogin,
  }),
}))

vi.mock('@/api/client', () => ({
  getApiErrorMessage: (error: unknown) => {
    if (typeof error === 'object' && error !== null && 'response' in error) {
      const response = (error as { response?: { data?: { error?: string; message?: string } } }).response
      return response?.data?.error ?? response?.data?.message ?? ''
    }
    return error instanceof Error ? error.message : ''
  },
  getOIDCPublicProviders: () => mockGetOIDCPublicProviders(),
  startOIDCLogin: (providerId: number, request: { redirectPath: string; callbackPath?: string }) => mockStartOIDCLogin(providerId, request),
  webauthnCheck: (username: string) => mockWebAuthnCheck(username),
}))

function mountLogin() {
  return mount(LoginPage, {
    global: {
      stubs: {
        RouterLink: { template: '<a><slot /></a>' },
        LockKeyhole: true,
      },
    },
  })
}

describe('LoginPage', () => {
  beforeEach(() => {
    vi.useRealTimers()
    vi.clearAllMocks()
    mockRoute.query = {}
    vi.stubGlobal('localStorage', {
      getItem: vi.fn(() => null),
      setItem: vi.fn(),
      removeItem: vi.fn(),
    })
    Object.defineProperty(window, 'location', {
      value: { assign: mockLocationAssign },
      configurable: true,
    })
    Object.defineProperty(window, 'PublicKeyCredential', {
      value: undefined,
      configurable: true,
    })
    mockWebAuthnCheck.mockResolvedValue({ data: { available: false } })
    mockGetOIDCPublicProviders.mockResolvedValue({ data: { providers: [] } })
    mockStartOIDCLogin.mockResolvedValue({ data: { authorizationUrl: 'https://provider.example/authorize', expiresAt: '2026-06-24T14:00:00Z' } })
  })

  it('uses a generic message for failed password login', async () => {
    mockDoLogin.mockRejectedValue(new Error('invalid'))
    const wrapper = mountLogin()

    await wrapper.find('input[autocomplete="username"]').setValue('missing-user')
    await wrapper.find('input[autocomplete="current-password"]').setValue('wrong')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.text()).toContain('Invalid username or password')
    expect(wrapper.text()).not.toContain('missing-user')
  })

  it('shows retry-after countdown for lockout responses without exposing account existence', async () => {
    vi.useFakeTimers()
    mockDoLogin.mockRejectedValue({
      response: {
        status: 429,
        headers: { 'retry-after': '3' },
      },
    })
    const wrapper = mountLogin()

    await wrapper.find('input[autocomplete="username"]').setValue('brian')
    await wrapper.find('input[autocomplete="current-password"]').setValue('wrong')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.text()).toContain('Too many attempts. Try again later. Retry in 3 seconds.')
    expect(wrapper.text()).not.toContain('brian')

    vi.advanceTimersByTime(1000)
    await flushPromises()

    expect(wrapper.text()).toContain('Retry in 2 seconds.')
  })

  it('renders enabled OIDC providers and redirects to provider authorization URL', async () => {
    mockGetOIDCPublicProviders.mockResolvedValue({
      data: {
        providers: [
          { id: 1, name: 'entra-work', displayName: 'Microsoft', providerType: 'entra' },
          { id: 2, name: 'pocket-id', displayName: 'Pocket ID', providerType: 'pocket_id' },
        ],
      },
    })
    const wrapper = mountLogin()
    await flushPromises()

    expect(wrapper.text()).toContain('Sign in with Microsoft')
    expect(wrapper.text()).toContain('Sign in with Pocket ID')

    await wrapper.find('section[aria-label="Alternate sign in"]').findAll('button')[0]?.trigger('click')
    await flushPromises()

    expect(mockStartOIDCLogin).toHaveBeenCalledWith(1, {
      redirectPath: '/',
      callbackPath: '/auth/oidc/callback/1',
    })
    expect(mockLocationAssign).toHaveBeenCalledWith('https://provider.example/authorize')
  })

  it.each([
    ['access_denied', 'Sign-in was cancelled or denied at the provider.'],
    ['oidc_validation_failed', 'The provider response could not be validated.'],
    ['account_linking_conflict', 'Sign in locally, then link the provider from Account Settings.'],
    ['provider_misconfiguration', 'The sign-in provider is not configured correctly.'],
  ])('shows distinct OIDC callback error category for %s', async (category, expected) => {
    mockRoute.query = { oidc_error: category }

    const wrapper = mountLogin()
    await flushPromises()

    expect(wrapper.text()).toContain(expected)
  })

  it('shows provider misconfiguration when OIDC login start fails with server error', async () => {
    mockGetOIDCPublicProviders.mockResolvedValue({
      data: {
        providers: [
          { id: 1, name: 'entra-work', displayName: 'Microsoft', providerType: 'entra' },
        ],
      },
    })
    mockStartOIDCLogin.mockRejectedValue({
      response: {
        status: 500,
        data: { error: 'provider discovery failed' },
      },
    })
    const wrapper = mountLogin()
    await flushPromises()

    await wrapper.find('section[aria-label="Alternate sign in"] button').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('The sign-in provider is not configured correctly.')
  })
})
