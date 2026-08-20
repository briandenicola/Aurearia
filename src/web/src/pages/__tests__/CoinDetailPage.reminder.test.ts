/**
 * Focused tests for Feature 355 UX revision:
 * Purchase Reminder Date shown as a detail-grid row (not a pill/strip)
 * immediately after the Purchase Price row.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
import CoinDetailPage from '../CoinDetailPage.vue'
import { buildWishlistAureusTarget } from '@/test/fixtures/coins'
import type { PurchaseReminder } from '@/types'

// Wishlist coin with a purchase price so we can assert row ordering
const wishlistCoin = buildWishlistAureusTarget({ purchasePrice: 150 })

const fetchCoin = vi.fn()
const routerPush = vi.fn()

vi.mock('@/stores/coins', () => ({
  useCoinsStore: () => ({
    loading: false,
    currentCoin: wishlistCoin,
    fetchCoin,
  }),
}))

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: () => ({ params: { id: String(wishlistCoin.id) } }),
    useRouter: () => ({ push: routerPush }),
  }
})

vi.mock('@/api/client', () => ({
  createCoinReference: vi.fn(),
  deleteCoin: vi.fn(),
  deleteCoinReference: vi.fn(),
  duplicateCoin: vi.fn(),
  listCatalogs: vi.fn().mockResolvedValue([]),
  purchaseCoin: vi.fn(),
  sellCoin: vi.fn(),
  updateCoinReference: vi.fn(),
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ showConfirm: vi.fn(), showAlert: vi.fn() }),
}))

vi.mock('@/composables/useCoinShareCard', () => ({
  useCoinShareCard: () => ({
    sharing: ref(false),
    shareCoinCard: vi.fn().mockResolvedValue({ mode: 'downloaded' }),
  }),
}))

// Mutable reminder state shared across tests
const mockReminder = ref<PurchaseReminder | null>(null)

vi.mock('@/composables/usePurchaseReminder', () => ({
  usePurchaseReminder: () => ({
    reminder: mockReminder,
    loading: ref(false),
    saving: ref(false),
    error: ref(''),
    fetchReminder: vi.fn(),
    saveReminder: vi.fn().mockResolvedValue(null),
    cancelReminder: vi.fn(),
  }),
  // Simplified formatting for test assertions
  formatReminderDateValue: (d: string) => {
    const [y, m, day] = d.split('-')
    return `${m}/${day}/${y}`
  },
  getBrowserTimezone: () => 'America/Chicago',
  todayDateString: () => '2026-08-20',
}))

const routerLinkStub = { props: ['to'], template: '<a :href="to"><slot /></a>' }

function buildActiveReminder(): PurchaseReminder {
  return {
    id: 42,
    coinId: wishlistCoin.id,
    remindDate: '2026-09-01',
    timezone: 'America/Chicago',
    status: 'pending',
    createdAt: '2026-08-20T00:00:00Z',
    updatedAt: '2026-08-20T00:00:00Z',
  }
}

function mountPage() {
  return mount(CoinDetailPage, {
    global: {
      stubs: {
        RouterLink: routerLinkStub,
        SellModal: true,
        PurchaseModal: true,
        PurchaseReminderModal: true,
        ImageLightbox: true,
        CoinTagsSection: true,
        // CoinDetailMetadataTable is NOT stubbed — required to render rows
        CoinListingStatus: true,
        CoinReferencesSection: true,
        AuthenticatedImage: true,
        SafeExternalLink: { template: '<a><slot /></a>' },
      },
    },
  })
}

describe('CoinDetailPage — Feature 355 reminder detail row', () => {
  beforeEach(() => {
    fetchCoin.mockReset()
    routerPush.mockReset()
    mockReminder.value = null
  })

  it('renders a Purchase Reminder Date row when an active reminder exists', async () => {
    mockReminder.value = buildActiveReminder()
    const wrapper = mountPage()
    await flushPromises()

    const rows = wrapper.findAll('.metadata-row')
    const reminderRow = rows.find(r => r.text().includes('Purchase Reminder Date'))
    expect(reminderRow, 'reminder metadata row should be present').toBeDefined()
  })

  it('displays the formatted date as plain value text (not a pill or chip)', async () => {
    mockReminder.value = buildActiveReminder()
    const wrapper = mountPage()
    await flushPromises()

    const rows = wrapper.findAll('.metadata-row')
    const reminderRow = rows.find(r => r.text().includes('Purchase Reminder Date'))!

    // Value uses .row-value — no pill/chip class
    const valueEl = reminderRow.find('.row-value')
    expect(valueEl.exists()).toBe(true)
    expect(valueEl.text()).toContain('09/01/2026')
    expect(valueEl.classes()).not.toContain('chip-sm')
    expect(valueEl.classes()).not.toContain('chip')
  })

  it('places the reminder row immediately after the Purchase Price row', async () => {
    mockReminder.value = buildActiveReminder()
    const wrapper = mountPage()
    await flushPromises()

    const rows = wrapper.findAll('.metadata-row')
    const labels = rows.map(r => r.find('.row-label').exists() ? r.find('.row-label').text() : '')

    const purchasePriceIdx = labels.findIndex(l => l.includes('Purchase Price'))
    const reminderIdx = labels.findIndex(l => l.includes('Purchase Reminder Date'))

    expect(purchasePriceIdx, 'Purchase Price row must be present').toBeGreaterThanOrEqual(0)
    expect(reminderIdx, 'reminder row must be directly after Purchase Price').toBe(purchasePriceIdx + 1)
  })

  it('renders an Edit button inside the reminder row', async () => {
    mockReminder.value = buildActiveReminder()
    const wrapper = mountPage()
    await flushPromises()

    const rows = wrapper.findAll('.metadata-row')
    const reminderRow = rows.find(r => r.text().includes('Purchase Reminder Date'))!
    const editBtn = reminderRow.find('button')
    expect(editBtn.exists()).toBe(true)
    expect(editBtn.text()).toBe('Edit')
  })

  it('clicking Edit inside the reminder row opens PurchaseReminderModal', async () => {
    mockReminder.value = buildActiveReminder()
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.find('purchase-reminder-modal-stub').exists()).toBe(false)

    const rows = wrapper.findAll('.metadata-row')
    const reminderRow = rows.find(r => r.text().includes('Purchase Reminder Date'))!
    await reminderRow.find('button').trigger('click')

    expect(wrapper.find('purchase-reminder-modal-stub').exists()).toBe(true)
  })

  it('does not render the old inline strip with gold chip style when reminder is active', async () => {
    mockReminder.value = buildActiveReminder()
    const wrapper = mountPage()
    await flushPromises()

    // Old strip had .chip-sm with inline background accent-gold-dim
    const goldPills = wrapper.findAll('.chip-sm').filter(el =>
      (el.attributes('style') ?? '').includes('accent-gold-dim')
    )
    expect(goldPills).toHaveLength(0)
  })

  it('does not render the reminder row when no reminder is active', async () => {
    mockReminder.value = null
    const wrapper = mountPage()
    await flushPromises()

    const rows = wrapper.findAll('.metadata-row')
    const reminderRow = rows.find(r => r.text().includes('Purchase Reminder Date'))
    expect(reminderRow).toBeUndefined()
  })
})