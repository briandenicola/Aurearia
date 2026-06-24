import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RegisterPage from '@/pages/RegisterPage.vue'

const mockPush = vi.fn()
const mockDoRegister = vi.fn()

vi.mock('/coin-logo.jpg', () => ({ default: '/coin-logo.jpg' }))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
  RouterLink: { template: '<a><slot /></a>' },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    doRegister: mockDoRegister,
  }),
}))

function mountRegister() {
  return mount(RegisterPage, {
    global: {
      stubs: {
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })
}

describe('RegisterPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows backend registration closed errors instead of duplicate username fallback', async () => {
    mockDoRegister.mockRejectedValue({
      response: {
        status: 403,
        data: { error: 'Registration is not available' },
      },
    })
    const wrapper = mountRegister()

    await wrapper.find('input[autocomplete="username"]').setValue('manager')
    await wrapper.find('input[type="email"]').setValue('manager@example.com')
    const passwordInputs = wrapper.findAll('input[type="password"]')
    await passwordInputs[0]!.setValue('password123')
    await passwordInputs[1]!.setValue('password123')
    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.text()).toContain('Registration is not available')
    expect(wrapper.text()).not.toContain('username may already exist')
  })
})
