import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
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
      path: '/add',
      name: 'add-coin',
      component: () => import('@/pages/AddCoinPage.vue'),
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
      path: '/timeline',
      name: 'timeline',
      component: () => import('@/pages/TimelinePage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('@/pages/SettingsPage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin',
      name: 'admin',
      component: () => import('@/pages/AdminPage.vue'),
      meta: { requiresAuth: true },
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
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login' }
  }
})

export default router
