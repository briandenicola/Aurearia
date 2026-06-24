import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminSecuritySection from '@/components/admin/AdminSecuritySection.vue'
import type { UserInfo } from '@/types'

const mockGetSecuritySummary = vi.fn()
const mockGetSecurityEvents = vi.fn()
const mockGetSecurityIpRules = vi.fn()
const mockCreateSecurityIpRule = vi.fn()
const mockDeleteSecurityIpRule = vi.fn()
const mockGetSecurityExposureCheck = vi.fn()
const mockUnlockUser = vi.fn()

vi.mock('@/api/client', () => ({
  getSecuritySummary: () => mockGetSecuritySummary(),
  getSecurityEvents: (filters?: Record<string, unknown>) => mockGetSecurityEvents(filters),
  getSecurityIpRules: () => mockGetSecurityIpRules(),
  createSecurityIpRule: (payload: Record<string, unknown>) => mockCreateSecurityIpRule(payload),
  deleteSecurityIpRule: (id: number) => mockDeleteSecurityIpRule(id),
  getSecurityExposureCheck: () => mockGetSecurityExposureCheck(),
  unlockUser: (id: number) => mockUnlockUser(id),
}))

const baseUser: UserInfo = {
  id: 1,
  username: 'admin',
  role: 'admin',
  email: 'admin@example.com',
  avatarPath: '',
  isPublic: false,
  bio: '',
  zipCode: '',
  emailMissing: false,
  numisBidsUsername: '',
  numisBidsConfigured: false,
  createdAt: '2026-01-01T00:00:00Z',
}

function mountSection(users: UserInfo[] = []) {
  return mount(AdminSecuritySection, {
    props: {
      users,
      registrationMode: 'invite',
    },
  })
}

describe('AdminSecuritySection', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetSecuritySummary.mockResolvedValue({
      data: { failedLogins: 3, lockedAccounts: 1, activeBans: 2, recentEvents: 7 },
    })
    mockGetSecurityEvents.mockResolvedValue({
      data: {
        events: [
          {
            id: 10,
            timestamp: '2026-06-19T12:00:00Z',
            type: 'login',
            severity: 'warning',
            outcome: 'failure',
            username: 'brian',
            ip: '203.0.113.5',
            message: 'Failed login',
          },
        ],
      },
    })
    mockGetSecurityIpRules.mockResolvedValue({
      data: { rules: [{ id: 4, cidr: '203.0.113.0/24', reason: 'bot traffic', expiresAt: null, createdBy: 'admin' }] },
    })
    mockGetSecurityExposureCheck.mockResolvedValue({
      data: {
        publicIp: '198.51.100.4',
        proxy: true,
        cors: false,
        corsWarning: 'CORS allows a broad origin',
        webAuthn: true,
        publicAppUrl: true,
        registration: true,
        agentToken: true,
      },
    })
    mockCreateSecurityIpRule.mockResolvedValue({ data: {} })
    mockDeleteSecurityIpRule.mockResolvedValue({ data: {} })
    mockUnlockUser.mockResolvedValue({ data: {} })
  })

  it('shows loading state while security data loads', async () => {
    mockGetSecuritySummary.mockReturnValue(new Promise(() => {}))

    const wrapper = mountSection()
    await nextTick()

    expect(wrapper.text()).toContain('Loading...')
  })

  it('loads summary, exposure, events, and IP rules', async () => {
    const wrapper = mountSection([{ ...baseUser, id: 2, username: 'locked', lockedUntil: '2999-01-01T00:00:00Z' }])
    await flushPromises()

    expect(wrapper.text()).toContain('Failed Logins')
    expect(wrapper.text()).toContain('3')
    expect(wrapper.text()).toContain('API sees your IP as')
    expect(wrapper.text()).toContain('203.0.113.0/24')
    expect(wrapper.text()).toContain('Failed login')
    expect(wrapper.text()).toContain('locked')
  })

  it('normalizes backend security response shapes from the Go admin API', async () => {
    mockGetSecuritySummary.mockResolvedValue({
      data: {
        summary: {
          loginFailures: 5,
          activeIpRuleCount: 2,
        },
        backupStatus: 'restore drill complete',
      },
    })
    mockGetSecurityEvents.mockResolvedValue({
      data: {
        events: [
          {
            id: 11,
            createdAt: '2026-06-19T12:00:00Z',
            type: 'password_login_failure',
            username: 'alice',
            clientIp: '198.51.100.10',
            message: 'failed password login',
          },
        ],
      },
    })
    mockGetSecurityIpRules.mockResolvedValue({
      data: {
        ipRules: [{ id: 5, cidr: '198.51.100.0/24', reason: 'credential stuffing', expiresAt: null, createdBy: 1 }],
      },
    })

    const wrapper = mountSection()
    await flushPromises()

    expect(wrapper.text()).toContain('5')
    expect(wrapper.text()).toContain('198.51.100.10')
    expect(wrapper.text()).toContain('198.51.100.0/24')
  })

  it('renders empty events and IP rule states', async () => {
    mockGetSecurityEvents.mockResolvedValue({ data: { events: [] } })
    mockGetSecurityIpRules.mockResolvedValue({ data: { rules: [] } })

    const wrapper = mountSection()
    await flushPromises()

    expect(wrapper.text()).toContain('No security events match the current filters.')
    expect(wrapper.text()).toContain('No active manual IP bans.')
  })

  it('shows an error when loading fails', async () => {
    mockGetSecuritySummary.mockRejectedValue(new Error('backend unavailable'))

    const wrapper = mountSection()
    await flushPromises()

    expect(wrapper.text()).toContain('Failed to load security data')
  })

  it('applies security event filters', async () => {
    const wrapper = mountSection()
    await flushPromises()

    await wrapper.find('input[placeholder="Type"]').setValue('login')
    await wrapper.find('input[placeholder="User"]').setValue('brian')
    await wrapper.find('input[placeholder="IP"]').setValue('203.0.113.5')
    await wrapper.find('input[type="date"]').setValue('2026-06-19')
    await wrapper.find('form.filters-grid').trigger('submit.prevent')
    await flushPromises()

    expect(mockGetSecurityEvents).toHaveBeenLastCalledWith(expect.objectContaining({
      type: 'login',
      username: 'brian',
      clientIp: '203.0.113.5',
      since: expect.any(String),
      limit: 50,
    }))
  })

  it('marks security event dates with narrow-safe hooks', async () => {
    const wrapper = mountSection()
    await flushPromises()

    expect(wrapper.find('input[type="date"]').classes()).toContain('date-filter-input')
    expect(wrapper.find('.table-wrap').exists()).toBe(true)
    expect(wrapper.find('.date-cell').text()).toBeTruthy()
  })

  it('adds and deletes IP bans through the API client', async () => {
    const wrapper = mountSection()
    await flushPromises()

    await wrapper.find('input[placeholder^="CIDR"]').setValue('198.51.100.9')
    await wrapper.find('input[placeholder^="Duration"]').setValue('24h')
    await wrapper.find('input[placeholder="Reason"]').setValue('credential stuffing')
    await wrapper.find('form.ban-form').trigger('submit.prevent')
    await flushPromises()

    expect(mockCreateSecurityIpRule).toHaveBeenCalledWith({
      cidr: '198.51.100.9',
      durationMinutes: 1440,
      reason: 'credential stuffing',
    })

    await wrapper.find('button.btn-danger').trigger('click')
    await flushPromises()

    expect(mockDeleteSecurityIpRule).toHaveBeenCalledWith(4)
  })
})
