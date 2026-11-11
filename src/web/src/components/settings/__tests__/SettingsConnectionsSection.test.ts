import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SettingsConnectionsSection from '@/components/settings/SettingsConnectionsSection.vue'

const mockTestPaperless = vi.fn()
const authUser: { paperlessConfigured: boolean } = { paperlessConfigured: true }

vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: authUser }) }))
vi.mock('@/api/client', () => ({ testPaperlessConnection: () => mockTestPaperless() }))
vi.mock('@/composables/useSettingsProfile', () => ({
  useSettingsProfile: () => ({
    nbUsername: ref(''), nbPassword: ref(''), cngUsername: ref(''), cngPassword: ref(''),
    parcelAppAPIKey: ref(''), paperlessUrl: ref('http://paperless-ngx:8000'), paperlessApiToken: ref(''), paperlessTags: ref('coins, shopping'),
    pushoverKey: ref(''), pushoverTesting: ref(false), pushoverTestMsg: ref(''), pushoverTestError: ref(false),
    handleTestPushover: vi.fn(), profileMsg: ref(''), profileError: ref(false), profileSaving: ref(false),
    nbValidating: ref(false), nbValidationError: ref(''), cngValidating: ref(false), cngValidationError: ref(''),
    handleSaveProfile: vi.fn(),
  }),
}))

describe('SettingsConnectionsSection paperless-ngx test', () => {
  beforeEach(() => {
    mockTestPaperless.mockReset()
    authUser.paperlessConfigured = true
  })

  it('does not claim a connection just because settings are saved', () => {
    const wrapper = mount(SettingsConnectionsSection)
    expect(wrapper.text()).not.toContain('Paperless-ngx connected')
    expect(wrapper.text()).toContain('URL and token saved.')
  })

  it('shows success when the server reaches paperless-ngx', async () => {
    mockTestPaperless.mockResolvedValue({ data: { ok: true } })
    const wrapper = mount(SettingsConnectionsSection)
    await wrapper.get('[data-testid="paperless-test-connection"]').trigger('click')
    await flushPromises()
    const result = wrapper.get('[data-testid="paperless-test-result"]')
    expect(result.text()).toContain('Connected')
    expect(result.classes()).toContain('text-gain')
  })

  it('shows the server-reported cause when the test fails', async () => {
    const cause = 'The Aurearia server could not resolve "paperless-ngx".'
    mockTestPaperless.mockRejectedValue({ response: { data: { error: cause } } })
    const wrapper = mount(SettingsConnectionsSection)
    await wrapper.get('[data-testid="paperless-test-connection"]').trigger('click')
    await flushPromises()
    const result = wrapper.get('[data-testid="paperless-test-result"]')
    expect(result.text()).toBe(cause)
    expect(result.classes()).toContain('text-loss')
  })

  it('shows the saved paperless tags', () => {
    const wrapper = mount(SettingsConnectionsSection)
    expect((wrapper.get('[data-testid="paperless-tags"]').element as HTMLInputElement).value).toBe('coins, shopping')
  })

  it('hides the test button until a URL and token are saved', () => {
    authUser.paperlessConfigured = false
    const wrapper = mount(SettingsConnectionsSection)
    expect(wrapper.find('[data-testid="paperless-test-connection"]').exists()).toBe(false)
  })
})
