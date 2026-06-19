import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LoginPage from '@/pages/LoginPage.vue'

const mockPush = vi.fn()
const mockDoLogin = vi.fn()
const mockDoWebAuthnLogin = vi.fn()
const mockWebAuthnCheck = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
  RouterLink: { template: '<a><slot /></a>' },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    doLogin: mockDoLogin,
    doWebAuthnLogin: mockDoWebAuthnLogin,
  }),
}))

vi.mock('@/api/client', () => ({
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
    vi.stubGlobal('localStorage', {
      getItem: vi.fn(() => null),
      setItem: vi.fn(),
      removeItem: vi.fn(),
    })
    Object.defineProperty(window, 'PublicKeyCredential', {
      value: undefined,
      configurable: true,
    })
    mockWebAuthnCheck.mockResolvedValue({ data: { available: false } })
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
})
