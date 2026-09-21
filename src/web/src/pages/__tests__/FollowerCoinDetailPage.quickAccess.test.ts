import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import FollowerCoinDetailPage from '../FollowerCoinDetailPage.vue'
import CoinDetailHeaderActions from '@/components/coin/CoinDetailHeaderActions.vue'
import { getFollowingCoinDetail } from '@/api/client'
import { useQuickAccess } from '@/composables/useQuickAccess'
import type { LimitedCoin } from '@/types'

const isPwa = ref(false)

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { username: 'other-collector', coinId: '42' } }),
  useRouter: () => ({ push: vi.fn() }),
}))
vi.mock('@/api/client', () => ({
  getPublicProfile: vi.fn(async () => ({ data: { id: 9 } })),
  getFollowingCoinDetail: vi.fn(async () => ({
    data: {
      id: 42, name: 'Followed collection denarius', category: 'Roman',
      denomination: 'Denarius', ruler: '', era: '', material: 'Silver',
      grade: '', images: [],
    } satisfies LimitedCoin,
  })),
  addComment: vi.fn(),
  deleteComment: vi.fn(),
  rateCoin: vi.fn(),
}))
vi.mock('@/composables/useQuickAccess', () => ({
  useQuickAccess: vi.fn(() => ({
    error: ref(''), pin: vi.fn(), unpin: vi.fn(),
    isPinned: () => ref(false), isBusy: () => ref(false),
  })),
}))
vi.mock('@/composables/usePwa', () => ({
  usePwa: () => ({ isPwa: isPwa.value }),
}))
vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ showToast: vi.fn() }),
}))

describe('Follower coin Quick Access exclusion', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it.each([false, true])('keeps owner pin controls off a loaded follower page (PWA=%s)', async pwa => {
    isPwa.value = pwa
    const follower = mount(FollowerCoinDetailPage)
    try {
      await flushPromises()
      expect(getFollowingCoinDetail).toHaveBeenCalledWith(9, 42)
      expect(follower.get('h1').text()).toBe('Followed collection denarius')
      expect(follower.findComponent(CoinDetailHeaderActions).exists()).toBe(false)
      expect(follower.find('[aria-label*="Quick Access"]').exists()).toBe(false)
      expect(useQuickAccess).not.toHaveBeenCalled()

      const owner = mount(CoinDetailHeaderActions, {
        props: { coinId: 42, isWishlist: false, isSold: false },
        global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } },
      })
      try {
        if (pwa) await owner.get('button[aria-label="Open overflow actions"]').trigger('click')
        expect(owner.get('button[aria-label="Pin coin to Quick Access"]').exists()).toBe(true)
      } finally {
        owner.unmount()
      }
    } finally {
      follower.unmount()
    }
  })
})
