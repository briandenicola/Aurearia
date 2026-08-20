import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import CoinCard from '../CoinCard.vue'
import type { Coin, PurchaseReminder } from '@/types'

// Minimal router stub
const mockPush = () => {}
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
}))

function buildWishlistCoin(id = 1): Coin {
  return {
    id,
    name: 'Augustus Aureus',
    category: 'Roman',
    denomination: 'Aureus',
    ruler: 'Augustus',
    era: 'ancient',
    mint: 'Rome',
    material: 'Gold',
    weightGrams: null,
    diameterMm: null,
    grade: '',
    obverseInscription: '',
    reverseInscription: '',
    obverseDescription: '',
    reverseDescription: '',
    rarityRating: '',
    purchasePrice: null,
    currentValue: null,
    purchaseDate: null,
    purchaseLocation: '',
    vendorSku: '',
    vendorInvoice: '',
    storageLocationId: null,
    storageLocation: null,
    notes: '',
    aiAnalysis: '',
    obverseAnalysis: '',
    reverseAnalysis: '',
    referenceUrl: '',
    referenceText: '',
    isWishlist: true,
    isSold: false,
    soldPrice: null,
    soldDate: null,
    soldTo: '',
    isPrivate: false,
    listingStatus: 'unlisted',
    listingCheckedAt: null,
    listingCheckReason: '',
    userId: 1,
    images: [],
    createdAt: '',
    updatedAt: '',
  }
}

function buildReminder(remindDate: string, status: PurchaseReminder['status'] = 'pending'): PurchaseReminder {
  return {
    id: 1,
    coinId: 1,
    remindDate,
    timezone: 'America/Chicago',
    status,
    createdAt: '2026-09-01T10:00:00Z',
    updatedAt: '2026-09-01T10:00:00Z',
  }
}

function mountCard(props: { coin: Coin; wishlist?: boolean; activeReminder?: PurchaseReminder | null }) {
  return mount(CoinCard, {
    props: { wishlist: true, ...props },
    global: {
      stubs: {
        AuthenticatedImage: true,
        SafeExternalLink: true,
        Coins: true,
        ShoppingCart: true,
        Check: true,
      },
    },
  })
}

describe('CoinCard reminder badge', () => {
  it('shows a reminder badge when wishlist coin has an active pending reminder', () => {
    const wrapper = mountCard({
      coin: buildWishlistCoin(),
      activeReminder: buildReminder('2026-09-25'),
    })
    expect(wrapper.text()).toContain('Due Sep 25')
  })

  it('shows "Due Today" for a reminder on today or a past date', () => {
    // Use a very old date so it is always "past"
    const wrapper = mountCard({
      coin: buildWishlistCoin(),
      activeReminder: buildReminder('2020-01-01'),
    })
    expect(wrapper.text()).toContain('Due Today')
  })

  it('does not show a badge when activeReminder is null', () => {
    const wrapper = mountCard({ coin: buildWishlistCoin(), activeReminder: null })
    expect(wrapper.text()).not.toContain('Due')
  })

  it('does not show a badge when reminder status is cancelled', () => {
    const wrapper = mountCard({
      coin: buildWishlistCoin(),
      activeReminder: buildReminder('2026-09-25', 'cancelled'),
    })
    expect(wrapper.text()).not.toContain('Due Sep 25')
  })

  it('does not show reminder badge on non-wishlist coins', () => {
    const coin = { ...buildWishlistCoin(), isWishlist: false }
    const wrapper = mountCard({ coin, wishlist: false, activeReminder: buildReminder('2026-09-25') })
    expect(wrapper.text()).not.toContain('Due Sep 25')
  })
})
