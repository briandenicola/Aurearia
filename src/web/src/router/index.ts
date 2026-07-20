import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior(to, _from, savedPosition) {
    if (savedPosition) return savedPosition
    if (to.hash) return { el: to.hash }
    return { top: 0 }
  },
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/pages/LoginPage.vue'),
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/pages/RegisterPage.vue'),
    },
    {
      path: '/',
      name: 'collection',
      component: () => import('@/pages/CollectionPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/coin/:id',
      name: 'coin-detail',
      component: () => import('@/pages/CoinDetailPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/coin/:id/journal',
      name: 'coin-detail-journal',
      component: () => import('@/pages/CoinDetailJournalPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/coin/:id/health',
      name: 'coin-detail-health',
      component: () => import('@/pages/CoinDetailHealthPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/coin/:id/notes',
      name: 'coin-detail-notes',
      component: () => import('@/pages/CoinDetailNotesPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/coin/:id/actions',
      name: 'coin-detail-actions',
      component: () => import('@/pages/CoinDetailActionsPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/coin/:id/analysis',
      name: 'coin-detail-analysis',
      component: () => import('@/pages/CoinDetailAnalysisPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/coin/:id/valuation',
      name: 'coin-detail-valuation',
      component: () => import('@/pages/CoinDetailValuationPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/add',
      name: 'add-coin',
      component: () => import('@/pages/AddCoinPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/quick-capture',
      name: 'quick-capture',
      component: () => import('@/pages/QuickCapturePage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/quick-capture/drafts',
      name: 'quick-capture-drafts',
      component: () => import('@/pages/QuickCaptureDraftsPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/quick-capture/drafts/:id',
      name: 'quick-capture-draft',
      component: () => import('@/pages/QuickCaptureDraftPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/lookup',
      name: 'lookup',
      component: () => import('@/pages/CoinLookupPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/edit/:id',
      name: 'edit-coin',
      component: () => import('@/pages/EditCoinPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/wishlist',
      name: 'wishlist',
      component: () => import('@/pages/WishlistPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/wishlist/search-alerts',
      name: 'wishlist-search-alerts',
      component: () => import('@/pages/WishlistAlertsPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/sold',
      name: 'sold',
      component: () => import('@/pages/SoldPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/auctions',
      name: 'auctions',
      component: () => import('@/pages/AuctionsPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/stats',
      name: 'stats',
      component: () => import('@/pages/StatsPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/stats/mint-map',
      name: 'stats-mint-map',
      component: () => import('@/pages/MintMapPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/stats/timeline',
      name: 'stats-timeline',
      component: () => import('@/pages/TimelinePage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/stats/health',
      name: 'stats-health',
      component: () => import('@/pages/StatsHealthPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/stats/value-trends',
      name: 'stats-value-trends',
      component: () => import('@/pages/StatsValueTrendsPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/stats/investment-breakdown',
      name: 'stats-investment-breakdown',
      component: () => import('@/pages/StatsInvestmentBreakdownPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/stats/distribution',
      name: 'stats-distribution',
      component: () => import('@/pages/CollectionDistributionPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/stats/emperors',
      name: 'stats-emperors',
      component: () => import('@/pages/EmperorTrackerPage.vue'),
      meta: { requiresAuth: true, requiresEmperorTracker: true },
    },
    {
      path: '/mint-map',
      name: 'mint-map',
      redirect: '/stats/mint-map',
    },
    {
      path: '/timeline',
      name: 'timeline',
      redirect: '/stats/timeline',
    },
    {
      path: '/notes',
      name: 'notes',
      component: () => import('@/pages/NotesPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('@/pages/SettingsPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/settings/oidc/link/callback/:providerId',
      name: 'oidc-link-callback',
      component: () => import('@/pages/OIDCLinkCallbackPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/auth/oidc/callback/:providerId',
      name: 'oidc-login-callback',
      component: () => import('@/pages/OIDCLoginCallbackPage.vue'),
    },
    {
      path: '/admin',
      name: 'admin',
      component: () => import('@/pages/AdminPage.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/process-image',
      name: 'process-image',
      redirect: { path: '/settings', query: { tab: 'process' } },
    },
    {
      path: '/followers',
      name: 'followers',
      component: () => import('@/pages/FollowersPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/followers/:username/gallery',
      name: 'follower-gallery',
      component: () => import('@/pages/FollowerGalleryPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/followers/:username/coins/:coinId',
      name: 'follower-coin-detail',
      component: () => import('@/pages/FollowerCoinDetailPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/notifications',
      name: 'notifications',
      component: () => import('@/pages/NotificationsPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/showcases',
      name: 'showcases',
      component: () => import('@/pages/ShowcasesPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/showcases/:id/edit',
      name: 'showcase-edit',
      component: () => import('@/pages/ShowcaseEditPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/s/:slug',
      name: 'public-showcase',
      component: () => import('@/pages/PublicShowcasePage.vue'),
    },
    {
      path: '/calendar',
      name: 'calendar',
      component: () => import('@/pages/CalendarPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/tray',
      name: 'tray',
      component: () => import('@/pages/TrayViewPage.vue'),
      meta: { requiresAuth: true },
    },
    // Set routes - placeholder for Phase 2 and Phase 3 implementation
    {
      path: '/sets',
      name: 'sets',
      component: () => import('@/pages/SetsPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/sets/:id',
      name: 'set-detail',
      component: () => import('@/pages/SetDetailPage.vue'),
      meta: { requiresAuth: true },
    },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login' }
  }
  if (to.meta.requiresAdmin && !auth.isAdmin) {
    return { name: 'collection' }
  }
  if (to.meta.requiresEmperorTracker && !auth.user?.emperorTrackerEnabled) {
    return { name: 'stats' }
  }
})

export default router
